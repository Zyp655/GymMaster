package service

import (
	"context"
	"fmt"

	v1 "gymmaster-api/api/pt/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

var ptCommissionOrderable = []string{"id", "period", "amount", "created_at", "status", "paid_at"}

// commissionTrainer: HLV (chỉ có vai trò TRAINER) chỉ xem hoa hồng của chính mình.
func commissionTrainer(ctx context.Context, requested uint) (uint, error) {
	if !ptTrainerOnly(ctx) {
		return requested, nil
	}
	if requested != 0 && requested != ptSelf(ctx) {
		return 0, errPTOnlyOwnData
	}
	return ptSelf(ctx), nil
}

func (s *ptService) ListCommissions(ctx context.Context, req *v1.CommissionListReq) (*v1.CommissionListRes, error) {
	if req.Period != "" {
		if err := ptValidPeriod(req.Period); err != nil {
			return nil, err
		}
	}
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	trainerID, err := commissionTrainer(ctx, req.TrainerID)
	if err != nil {
		return nil, err
	}
	q := req.PageReq.ToQuery(ptCommissionOrderable, "id")
	q.OrderBy = "tc." + q.OrderBy
	rows, total, err := s.repo.ListCommissions(ctx, q, repository.PTCommissionFilter{
		BranchIDs: branchIDs, TrainerID: trainerID, Period: req.Period, Status: req.Status, SourceType: req.SourceType,
	})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.PTCommissionRow) *v1.Commission {
		return &v1.Commission{
			ID: r.ID, TrainerID: r.TrainerID, TrainerName: r.TrainerName, BranchID: r.BranchID, BranchName: r.BranchName,
			SourceType: r.SourceType, SourceID: r.SourceID, SessionStart: r.SessionStartAt, ContractID: r.ContractID, MemberName: r.MemberName,
			BaseAmount: r.BaseAmount, Rate: r.Rate, Amount: r.Amount, Period: r.Period, Status: r.Status,
			PaidAt: r.PaidAt, PaidBy: r.PaidBy, PaidByName: r.PaidByName, CreatedAt: r.CreatedAt,
		}
	}), total, q), nil
}

func (s *ptService) CommissionSummary(ctx context.Context, req *v1.CommissionSummaryReq) (*v1.CommissionSummaryRes, error) {
	if err := ptValidPeriod(req.Period); err != nil {
		return nil, err
	}
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	trainerID, err := commissionTrainer(ctx, 0)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.CommissionSummary(ctx, repository.PTCommissionFilter{BranchIDs: branchIDs, TrainerID: trainerID, Period: req.Period})
	if err != nil {
		return nil, err
	}
	res := &v1.CommissionSummaryRes{Period: req.Period, BranchID: req.BranchID, Items: make([]*v1.CommissionSummaryItem, 0, len(rows))}
	for _, r := range rows {
		res.Items = append(res.Items, &v1.CommissionSummaryItem{
			TrainerID: r.TrainerID, TrainerName: r.TrainerName, StaffCode: r.StaffCode, Sessions: r.Sessions,
			BaseTotal: r.BaseTotal, TotalAmount: r.TotalAmount, PaidAmount: r.PaidAmount, PendingAmount: r.PendingAmount,
		})
		res.Sessions += r.Sessions
		res.TotalAmount += r.TotalAmount
		res.PaidAmount += r.PaidAmount
		res.PendingAmount += r.PendingAmount
	}
	return res, nil
}

func (s *ptService) PayCommissions(ctx context.Context, req *v1.CommissionPayReq) (*v1.CommissionPayRes, error) {
	if err := ptValidPeriod(req.Period); err != nil {
		return nil, err
	}
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền chi trả hoa hồng tại chi nhánh này")
	}
	name, err := s.repo.UserName(ctx, req.TrainerID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy huấn luyện viên")
	}
	at := s.now()
	res := &v1.CommissionPayRes{TrainerID: req.TrainerID, TrainerName: name, Period: req.Period, PaidAt: at}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		n, amount, err := s.repo.PayCommissions(ctx, req.TrainerID, req.Period, branchIDs, reqctx.UserID(ctx), at)
		if err != nil {
			return err
		}
		if n == 0 {
			return rescode.New(rescode.BadRequest, fmt.Sprintf("Không có hoa hồng nào chờ chi trả riêng của HLV %s trong kỳ %s (khoản đã đưa vào bảng lương sẽ được chi cùng lương)", name, req.Period))
		}
		res.Count, res.Amount = n, roundMoneyVND(amount)
		s.logAudit(ctx, ptActPay, ptAuditCommission, req.TrainerID, nil, map[string]any{
			"trainer_id": req.TrainerID, "period": req.Period, "branch_ids": branchIDs, "count": n, "amount": res.Amount,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.notify(ctx, []uint{req.TrainerID}, NotifyCommissionPaid, "Đã chi trả hoa hồng",
		fmt.Sprintf("Hoa hồng kỳ %s: %d buổi, tổng %s đã được chi trả.", req.Period, res.Count, formatMoneyVND(res.Amount)),
		"/admin/commissions?period="+req.Period)
	return res, nil
}
