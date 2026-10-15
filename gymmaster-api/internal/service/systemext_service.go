package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	v1 "gymmaster-api/api/systemext/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type SystemExtService struct {
	repo     *repository.SystemExtRepository
	notifier Notifier
}

func NewSystemExtService(repo *repository.SystemExtRepository, notifier Notifier) *SystemExtService {
	return &SystemExtService{repo: repo, notifier: notifier}
}

func (s *SystemExtService) GlobalSearch(ctx context.Context, q string) (*v1.GlobalSearchRes, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return &v1.GlobalSearchRes{Items: []*v1.SearchHit{}}, nil
	}
	like := "%" + q + "%"
	hits := make([]*v1.SearchHit, 0, 16)

	var members []entity.Member
	_ = s.repo.DB().WithContext(ctx).
		Where("full_name ILIKE ? OR phone ILIKE ? OR member_code ILIKE ?", like, like, like).
		Limit(6).Find(&members).Error
	for _, m := range members {
		hits = append(hits, &v1.SearchHit{
			ID:       m.ID,
			Category: "MEMBER",
			Title:    fmt.Sprintf("%s (%s)", m.FullName, m.MemberCode),
			Subtitle: fmt.Sprintf("SĐT: %s · Trạng thái: %s", m.Phone, m.Status),
			Link:     fmt.Sprintf("/app/members/%d", m.ID),
		})
	}

	var invoices []entity.Invoice
	_ = s.repo.DB().WithContext(ctx).
		Where("invoice_no ILIKE ? OR note ILIKE ?", like, like).
		Limit(5).Find(&invoices).Error
	for _, inv := range invoices {
		hits = append(hits, &v1.SearchHit{
			ID:       inv.ID,
			Category: "INVOICE",
			Title:    fmt.Sprintf("Hóa đơn %s", inv.InvoiceNo),
			Subtitle: fmt.Sprintf("Tổng: %.0fđ · Trạng thái: %s", inv.Total, inv.Status),
			Link:     "/app/invoices",
		})
	}

	var products []entity.Product
	_ = s.repo.DB().WithContext(ctx).
		Where("name ILIKE ? OR sku ILIKE ?", like, like).
		Limit(5).Find(&products).Error
	for _, p := range products {
		hits = append(hits, &v1.SearchHit{
			ID:       p.ID,
			Category: "PRODUCT",
			Title:    fmt.Sprintf("%s (%s)", p.Name, p.SKU),
			Subtitle: fmt.Sprintf("Giá bán: %.0fđ", p.SellPrice),
			Link:     "/app/products",
		})
	}

	var plans []entity.MembershipPlan
	_ = s.repo.DB().WithContext(ctx).
		Where("name ILIKE ? OR code ILIKE ?", like, like).
		Limit(4).Find(&plans).Error
	for _, pl := range plans {
		hits = append(hits, &v1.SearchHit{
			ID:       pl.ID,
			Category: "PLAN",
			Title:    fmt.Sprintf("%s (%s)", pl.Name, pl.Code),
			Subtitle: fmt.Sprintf("Giá gói: %.0fđ", pl.Price),
			Link:     "/app/plans",
		})
	}

	return &v1.GlobalSearchRes{Items: hits}, nil
}

func (s *SystemExtService) BulkMemberAction(ctx context.Context, req *v1.BulkMemberActionReq) (*v1.BulkMemberActionRes, error) {
	if len(req.MemberIDs) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Chưa chọn danh sách hội viên")
	}
	affected := 0
	switch req.Action {
	case "NOTIFY":
		title := strings.TrimSpace(req.Title)
		if title == "" {
			title = "Thông báo từ trung tâm GymMaster"
		}
		msg := strings.TrimSpace(req.Message)
		if msg == "" {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập nội dung thông báo")
		}
		var members []entity.Member
		_ = s.repo.DB().WithContext(ctx).Where("id IN ?", req.MemberIDs).Find(&members).Error
		var userIDs []uint
		for _, m := range members {
			if m.UserID != nil {
				userIDs = append(userIDs, *m.UserID)
			}
			affected++
		}
		if len(userIDs) > 0 && s.notifier != nil {
			_ = s.notifier.Notify(ctx, userIDs, NotifySystem, title, msg, "/me/notifications")
		}
	case "TRANSFER_BRANCH":
		if req.ToBranchID == 0 {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh chuyển đến")
		}
		for _, mid := range req.MemberIDs {
			_, err := s.CreateBranchTransfer(ctx, &v1.CreateBranchTransferReq{
				MemberID:   mid,
				ToBranchID: req.ToBranchID,
				Reason:     req.Message,
			})
			if err == nil {
				affected++
			}
		}
	case "ACTIVATE", "DEACTIVATE":
		status := entity.MemberStatusActive
		if req.Action == "DEACTIVATE" {
			status = entity.MemberStatusInactive
		}
		res := s.repo.DB().WithContext(ctx).Model(&entity.Member{}).Where("id IN ?", req.MemberIDs).Update("status", status)
		affected = int(res.RowsAffected)
	}
	return &v1.BulkMemberActionRes{
		Affected: affected,
		Message:  fmt.Sprintf("Đã thực hiện %s cho %d hội viên", req.Action, affected),
	}, nil
}

func (s *SystemExtService) ListBranchTransfers(ctx context.Context) (v1.ListBranchTransfersRes, error) {
	list, err := s.repo.ListBranchTransfers(ctx)
	if err != nil {
		return nil, err
	}
	res := make(v1.ListBranchTransfersRes, 0, len(list))
	for _, tr := range list {
		res = append(res, s.enrichTransfer(ctx, tr))
	}
	return res, nil
}

func (s *SystemExtService) CreateBranchTransfer(ctx context.Context, req *v1.CreateBranchTransferReq) (*v1.CreateBranchTransferRes, error) {
	var member entity.Member
	if err := s.repo.DB().WithContext(ctx).First(&member, req.MemberID).Error; err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
	}
	if member.HomeBranchID == req.ToBranchID {
		return nil, rescode.New(rescode.BadRequest, "Hội viên đã thuộc chi nhánh này")
	}

	now := time.Now()
	var approvedBy *uint
	if auth := reqctx.Auth(ctx); auth != nil {
		approvedBy = &auth.UserID
	}

	var tr entity.MemberBranchTransfer
	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var activeSub entity.Subscription
		var subID *uint
		if err := tx.Where("member_id = ? AND status = 'ACTIVE'", member.ID).Order("id DESC").First(&activeSub).Error; err == nil {
			subID = &activeSub.ID
			_ = tx.Model(&entity.Subscription{}).Where("id = ?", activeSub.ID).Update("branch_id", req.ToBranchID).Error
		}
		tr = entity.MemberBranchTransfer{
			MemberID:       member.ID,
			FromBranchID:   member.HomeBranchID,
			ToBranchID:     req.ToBranchID,
			SubscriptionID: subID,
			TransferFee:    req.TransferFee,
			Reason:         strings.TrimSpace(req.Reason),
			Status:         "APPROVED",
			ApprovedBy:     approvedBy,
			ApprovedAt:     &now,
			CreatedAt:      now,
		}
		if err := tx.Create(&tr).Error; err != nil {
			return err
		}
		return tx.Model(&entity.Member{}).Where("id = ?", member.ID).Update("home_branch_id", req.ToBranchID).Error
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateBranchTransferRes{BranchTransferItem: s.enrichTransfer(ctx, tr)}, nil
}

func (s *SystemExtService) enrichTransfer(ctx context.Context, tr entity.MemberBranchTransfer) *v1.BranchTransferItem {
	var m entity.Member
	var fb, tb entity.Branch
	_ = s.repo.DB().WithContext(ctx).First(&m, tr.MemberID).Error
	_ = s.repo.DB().WithContext(ctx).First(&fb, tr.FromBranchID).Error
	_ = s.repo.DB().WithContext(ctx).First(&tb, tr.ToBranchID).Error
	return &v1.BranchTransferItem{
		ID:             tr.ID,
		MemberID:       tr.MemberID,
		MemberName:     m.FullName,
		MemberCode:     m.MemberCode,
		FromBranchID:   tr.FromBranchID,
		FromBranchName: fb.Name,
		ToBranchID:     tr.ToBranchID,
		ToBranchName:   tb.Name,
		SubscriptionID: tr.SubscriptionID,
		TransferFee:    tr.TransferFee,
		Reason:         tr.Reason,
		Status:         tr.Status,
		ApprovedAt:     tr.ApprovedAt,
		CreatedAt:      tr.CreatedAt,
	}
}

func (s *SystemExtService) RefundOnlineTxn(ctx context.Context, req *v1.RefundOnlineTxnReq) (*v1.RefundOnlineTxnRes, error) {
	var txn entity.OnlinePaymentTxn
	if err := s.repo.DB().WithContext(ctx).First(&txn, req.ID).Error; err != nil {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy giao dịch VNPay")
	}
	if txn.Status != entity.OnlineTxnSuccess {
		return nil, rescode.New(rescode.BadRequest, "Chỉ giao dịch VNPay đã thành công (SUCCESS) mới có thể hoàn tiền")
	}
	amount := req.Amount
	if amount <= 0 || amount > txn.Amount {
		amount = txn.Amount
	}

	var actorID *uint
	if auth := reqctx.Auth(ctx); auth != nil {
		actorID = &auth.UserID
	}
	now := time.Now()
	gatewayRef := fmt.Sprintf("VNP-REF-%d-%s", txn.ID, now.Format("150405"))
	var refund entity.Refund

	err := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		refund = entity.Refund{
			InvoiceID:   txn.InvoiceID,
			OnlineTxnID: &txn.ID,
			Amount:      amount,
			Method:      entity.PaymentVNPay,
			Status:      "SUCCESS",
			GatewayRef:  gatewayRef,
			Reason:      strings.TrimSpace(req.Reason),
			RefundedBy:  actorID,
			CreatedAt:   now,
		}
		if err := tx.Create(&refund).Error; err != nil {
			return err
		}
		return tx.Model(&entity.Invoice{}).Where("id = ?", txn.InvoiceID).Updates(map[string]any{
			"refunded_amount": gorm.Expr("refunded_amount + ?", amount),
			"status":          entity.InvoiceStatusRefunded,
			"updated_at":      now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &v1.RefundOnlineTxnRes{
		RefundID:   refund.ID,
		InvoiceID:  refund.InvoiceID,
		Amount:     refund.Amount,
		Status:     refund.Status,
		GatewayRef: refund.GatewayRef,
		CreatedAt:  refund.CreatedAt,
	}, nil
}

// ==================== LEGACY DATA IMPORT (DRY-RUN & COMMIT) ====================

func ValidateLegacyImportRows(rows []v1.LegacyImportRow) (int, []v1.LegacyImportErrorItem) {
	seenPhones := make(map[string]int)
	var errs []v1.LegacyImportErrorItem
	okCount := 0

	for idx, r := range rows {
		rowNum := idx + 1
		phone := strings.TrimSpace(r.Phone)
		name := strings.TrimSpace(r.FullName)
		if name == "" {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: phone, Message: "Thiếu họ tên hội viên"})
			continue
		}
		if len(phone) < 9 || len(phone) > 20 {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: phone, Message: "Số điện thoại không hợp lệ (9–20 ký tự)"})
			continue
		}
		if prevRow, dup := seenPhones[phone]; dup {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: phone, Message: fmt.Sprintf("Trùng số điện thoại với dòng %d trong file", prevRow)})
			continue
		}
		seenPhones[phone] = rowNum

		if r.SubStartDate != "" && r.SubEndDate != "" {
			st, err1 := time.Parse("2006-01-02", r.SubStartDate)
			et, err2 := time.Parse("2006-01-02", r.SubEndDate)
			if err1 != nil || err2 != nil || et.Before(st) {
				errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: phone, Message: "Ngày bắt đầu/kết thúc gói tập không hợp lệ"})
				continue
			}
		}
		if r.DebtAmount < 0 {
			errs = append(errs, v1.LegacyImportErrorItem{Row: rowNum, Phone: phone, Message: "Công nợ không được âm"})
			continue
		}
		okCount++
	}
	return okCount, errs
}

func (s *SystemExtService) ListLegacyImports(ctx context.Context) (v1.ListLegacyImportJobsRes, error) {
	list, err := s.repo.ListLegacyImports(ctx)
	if err != nil {
		return nil, err
	}
	res := make(v1.ListLegacyImportJobsRes, 0, len(list))
	for _, j := range list {
		res = append(res, s.toImportJobItem(ctx, j))
	}
	return res, nil
}

func (s *SystemExtService) RunLegacyImport(ctx context.Context, req *v1.RunLegacyImportReq) (*v1.RunLegacyImportRes, error) {
	mode := req.Mode
	if mode == "" {
		mode = entity.ImportModeDryRun
	}
	okCount, valErrs := ValidateLegacyImportRows(req.Rows)

	if mode == entity.ImportModeCommit && len(valErrs) == 0 && s.repo.DB() != nil {
		_ = s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var defaultPlan entity.MembershipPlan
			_ = tx.Where("status = 'ACTIVE'").Order("id ASC").First(&defaultPlan).Error

			for _, r := range req.Rows {
				gender := strings.ToUpper(strings.TrimSpace(r.Gender))
				if gender != "MALE" && gender != "FEMALE" {
					gender = "OTHER"
				}
				m := entity.Member{
					FullName:     strings.TrimSpace(r.FullName),
					Phone:        strings.TrimSpace(r.Phone),
					Email:        strings.TrimSpace(r.Email),
					Gender:       gender,
					HomeBranchID: req.BranchID,
					Source:       "OTHER",
					Status:       entity.MemberStatusActive,
					Note:         "Chuyển đổi từ hệ thống cũ (IMPORT)",
				}
				if err := tx.Where("phone = ?", m.Phone).FirstOrCreate(&m).Error; err != nil {
					continue
				}

				if defaultPlan.ID > 0 && r.SubStartDate != "" && r.SubEndDate != "" {
					st, _ := time.Parse("2006-01-02", r.SubStartDate)
					et, _ := time.Parse("2006-01-02", r.SubEndDate)
					sub := entity.Subscription{
						MemberID:          m.ID,
						PlanID:            defaultPlan.ID,
						BranchID:          req.BranchID,
						StartDate:         st,
						EndDate:           et,
						RemainingSessions: r.RemainingSessions,
						Price:             defaultPlan.Price,
						Status:            entity.SubStatusActive,
						Note:              "IMPORT",
					}
					_ = tx.Create(&sub).Error
				}

				if r.DebtAmount > 0 {
					inv := entity.Invoice{
						MemberID:   &m.ID,
						BranchID:   req.BranchID,
						Subtotal:   r.DebtAmount,
						Total:      r.DebtAmount,
						PaidAmount: 0,
						Status:     entity.InvoiceStatusUnpaid,
						Source:     entity.InvoiceSourceCounter,
						Note:       "Công nợ chuyển giao từ hệ thống cũ (IMPORT)",
						IssuedAt:   time.Now(),
					}
					_ = tx.Create(&inv).Error
				}
			}
			return nil
		})
	}

	errBytes, _ := json.Marshal(valErrs)
	var actorID *uint
	if auth := reqctx.Auth(ctx); auth != nil {
		actorID = &auth.UserID
	}
	fileName := strings.TrimSpace(req.FileName)
	if fileName == "" {
		fileName = "legacy-import.xlsx"
	}
	job := &entity.LegacyImportJob{
		BranchID:    req.BranchID,
		FileName:    fileName,
		Mode:        mode,
		Status:      "COMPLETED",
		TotalRows:   len(req.Rows),
		OkRows:      okCount,
		ErrorRows:   len(valErrs),
		SummaryJSON: datatypes.JSON([]byte(fmt.Sprintf(`{"mode":%q,"ok":%d,"err":%d}`, mode, okCount, len(valErrs)))),
		ErrorsJSON:  datatypes.JSON(errBytes),
		CreatedBy:   actorID,
		CreatedAt:   time.Now(),
	}
	if s.repo != nil && s.repo.DB() != nil {
		_ = s.repo.CreateLegacyImportJob(ctx, job)
	}
	return &v1.RunLegacyImportRes{LegacyImportJobItem: s.toImportJobItem(ctx, *job)}, nil
}

func (s *SystemExtService) toImportJobItem(ctx context.Context, j entity.LegacyImportJob) *v1.LegacyImportJobItem {
	var errs []v1.LegacyImportErrorItem
	_ = json.Unmarshal(j.ErrorsJSON, &errs)
	if errs == nil {
		errs = []v1.LegacyImportErrorItem{}
	}
	branchName := ""
	if s.repo != nil && s.repo.DB() != nil {
		var b entity.Branch
		if err := s.repo.DB().WithContext(ctx).First(&b, j.BranchID).Error; err == nil {
			branchName = b.Name
		}
	}
	return &v1.LegacyImportJobItem{
		ID:         j.ID,
		BranchID:   j.BranchID,
		BranchName: branchName,
		FileName:   j.FileName,
		Mode:       j.Mode,
		Status:     j.Status,
		TotalRows:  j.TotalRows,
		OkRows:     j.OkRows,
		ErrorRows:  j.ErrorRows,
		Errors:     errs,
		CreatedAt:  j.CreatedAt,
	}
}

// ==================== SETUP WIZARD (SPRINT 30 / MỤC 15.2) ====================

func (s *SystemExtService) GetSetupWizardStatus(ctx context.Context) (*v1.GetSetupWizardStatusRes, error) {
	db := s.repo.DB().WithContext(ctx)
	var branchCount, roomCount, planCount, staffCount, settingCount int64
	_ = db.Model(&entity.Branch{}).Count(&branchCount).Error
	_ = db.Model(&entity.Room{}).Count(&roomCount).Error
	_ = db.Model(&entity.MembershipPlan{}).Count(&planCount).Error
	_ = db.Model(&entity.User{}).Count(&staffCount).Error
	_ = db.Model(&entity.Setting{}).Count(&settingCount).Error

	var wizardSkipped entity.Setting
	skipped := db.Where("key = ?", "setup_wizard_skipped").First(&wizardSkipped).Error == nil

	steps := []*v1.SetupWizardStepStatus{
		{Key: "CENTER_INFO", Title: "1. Thông tin trung tâm Fitness", Completed: settingCount > 0, Count: settingCount, Route: "/app/settings"},
		{Key: "BRANCHES", Title: "2. Khởi tạo chi nhánh hoạt động", Completed: branchCount > 0, Count: branchCount, Route: "/app/branches"},
		{Key: "ROOMS", Title: "3. Thiết lập phòng tập & khu vực", Completed: roomCount > 0, Count: roomCount, Route: "/app/rooms"},
		{Key: "PLANS", Title: "4. Cấu hình gói tập & gói PT mẫu", Completed: planCount > 0, Count: planCount, Route: "/app/plans"},
		{Key: "STAFF", Title: "5. Tài khoản nhân sự & Huấn luyện viên", Completed: staffCount > 1, Count: staffCount, Route: "/app/staff"},
		{Key: "RULES", Title: "6. Quy định vận hành (Giờ mở cửa & Hủy lớp)", Completed: settingCount > 0, Count: settingCount, Route: "/app/settings"},
	}
	allDone := true
	for _, st := range steps {
		if !st.Completed {
			allDone = false
			break
		}
	}
	return &v1.GetSetupWizardStatusRes{
		Completed: allDone,
		Skipped:   skipped,
		Steps:     steps,
	}, nil
}

func (s *SystemExtService) CompleteSetupWizard(ctx context.Context, req *v1.CompleteSetupWizardReq) (*v1.CompleteSetupWizardRes, error) {
	db := s.repo.DB().WithContext(ctx)
	saveKV := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			return
		}
		raw, _ := json.Marshal(v)
		st := entity.Setting{Key: k, Value: datatypes.JSON(raw), UpdatedAt: time.Now()}
		_ = db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&st).Error
	}
	saveKV("center_name", req.CenterName)
	saveKV("center_hotline", req.Hotline)
	saveKV("opening_hours", req.OpeningHours)
	if req.CancelClassBefore > 0 {
		saveKV("class_cancel_before_hours", fmt.Sprintf("%d", req.CancelClassBefore))
	}
	if req.Skip {
		saveKV("setup_wizard_skipped", "true")
	}
	status, err := s.GetSetupWizardStatus(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.CompleteSetupWizardRes{GetSetupWizardStatusRes: status}, nil
}
