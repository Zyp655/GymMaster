package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// subFreezeDays = end_date − start_date + 1 (giống cột GENERATED days).
func subFreezeDays(f *entity.SubscriptionFreeze) int {
	return subDaysBetween(f.StartDate, f.EndDate) + 1
}

// subFreezeApplied: bảo lưu APPROVED đang được áp dụng (gói FROZEN và đã tới ngày bắt đầu).
func subFreezeApplied(sub *entity.Subscription, f *entity.SubscriptionFreeze, today time.Time) bool {
	return f.Status == entity.FreezeStatusApproved && sub.Status == entity.SubStatusFrozen && !subNorm(f.StartDate).After(subNorm(today))
}

// subFreezeCommitted trả số ngày / số lần bảo lưu đã dùng hoặc đã đăng ký (chờ duyệt, đã duyệt chưa áp dụng)
// của gói – dùng để kiểm tra giới hạn max_freeze_days / max_freeze_times.
func subFreezeCommitted(sub *entity.Subscription, freezes []entity.SubscriptionFreeze, today time.Time) (days, times int) {
	days, times = sub.FreezeDaysUsed, sub.FreezeTimesUsed
	for i := range freezes {
		f := &freezes[i]
		switch {
		case f.Status == entity.FreezeStatusPending,
			f.Status == entity.FreezeStatusApproved && !subFreezeApplied(sub, f, today):
			days += subFreezeDays(f)
			times++
		}
	}
	return days, times
}

// subValidateFreeze kiểm tra yêu cầu bảo lưu [start, end] cho gói (hàm thuần).
func subValidateFreeze(sub *entity.Subscription, plan *entity.MembershipPlan, freezes []entity.SubscriptionFreeze, start, end, today time.Time) error {
	if sub.Status != entity.SubStatusActive {
		return rescode.New(rescode.BadRequest, "Chỉ bảo lưu được gói đang hoạt động")
	}
	if plan.MaxFreezeDays <= 0 || plan.MaxFreezeTimes <= 0 {
		return rescode.New(rescode.BadRequest, "Gói tập này không cho phép bảo lưu")
	}
	if start.Before(subNorm(today)) {
		return rescode.New(rescode.BadRequest, "Ngày bắt đầu bảo lưu phải từ hôm nay trở đi")
	}
	if end.Before(start) {
		return rescode.New(rescode.BadRequest, "Ngày kết thúc bảo lưu phải sau hoặc bằng ngày bắt đầu")
	}
	if start.Before(subNorm(sub.StartDate)) || start.After(subNorm(sub.EndDate)) {
		return rescode.New(rescode.BadRequest, "Ngày bắt đầu bảo lưu phải nằm trong thời hạn gói ("+subFmt(sub.StartDate)+" – "+subFmt(sub.EndDate)+")")
	}
	for i := range freezes {
		f := &freezes[i]
		if f.Status != entity.FreezeStatusPending && f.Status != entity.FreezeStatusApproved {
			continue
		}
		if !start.After(subNorm(f.EndDate)) && !end.Before(subNorm(f.StartDate)) {
			return rescode.New(rescode.Conflict, "Khoảng bảo lưu bị trùng với yêu cầu bảo lưu khác ("+subFmt(f.StartDate)+" – "+subFmt(f.EndDate)+")")
		}
	}
	days := subDaysBetween(start, end) + 1
	usedDays, usedTimes := subFreezeCommitted(sub, freezes, today)
	if usedTimes+1 > plan.MaxFreezeTimes {
		return rescode.New(rescode.BadRequest, fmt.Sprintf("Gói tập đã dùng hết %d lần bảo lưu cho phép", plan.MaxFreezeTimes))
	}
	if usedDays+days > plan.MaxFreezeDays {
		return rescode.New(rescode.BadRequest, fmt.Sprintf("Vượt quá số ngày bảo lưu cho phép: tối đa %d ngày, đã dùng/đăng ký %d ngày, còn %d ngày",
			plan.MaxFreezeDays, usedDays, max(0, plan.MaxFreezeDays-usedDays)))
	}
	return nil
}

// subApplyFreeze áp bảo lưu vào gói: FROZEN, end_date += days, cộng số ngày/lần đã dùng (hàm thuần).
func subApplyFreeze(sub *entity.Subscription, days int) {
	sub.Status = entity.SubStatusFrozen
	sub.EndDate = subAddDays(sub.EndDate, days)
	sub.FreezeDaysUsed += days
	sub.FreezeTimesUsed++
}

// subEndFreezeEarly kết thúc sớm bảo lưu đang áp dụng tại ngày today (hội viên quay lại tập từ hôm nay).
// Hoàn lại số ngày chưa dùng: end_date −= unused, freeze_days_used −= unused. Nếu bảo lưu mới bắt đầu
// đúng hôm nay thì coi như chưa dùng: trả lại cả số lần (cancelled = true). Trả số ngày hoàn lại và ngày kết thúc mới
// của bảo lưu (khi không bị hủy).
func subEndFreezeEarly(sub *entity.Subscription, f *entity.SubscriptionFreeze, today time.Time) (unused int, newEnd time.Time, cancelled bool) {
	today = subNorm(today)
	unused = max(0, subDaysBetween(today, f.EndDate)+1)
	if !subNorm(f.StartDate).Before(today) {
		unused, cancelled = subFreezeDays(f), true
		sub.FreezeTimesUsed = max(0, sub.FreezeTimesUsed-1)
	}
	sub.Status = entity.SubStatusActive
	sub.EndDate = subAddDays(sub.EndDate, -unused)
	sub.FreezeDaysUsed = max(0, sub.FreezeDaysUsed-unused)
	newEnd = subAddDays(today, -1)
	if newEnd.After(subNorm(f.EndDate)) {
		newEnd = subNorm(f.EndDate)
	}
	return unused, newEnd, cancelled
}

// ---- Nghiệp vụ ------------------------------------------------------------------

func (s *subscriptionService) RequestFreeze(ctx context.Context, id uint, in v1.FreezeInput) (*v1.Freeze, error) {
	sub, err := s.loadSub(ctx, id, false)
	if err != nil {
		return nil, err
	}
	return s.requestFreeze(ctx, sub, in)
}

func (s *subscriptionService) RequestFreezeByMember(ctx context.Context, memberID, subscriptionID uint, in v1.FreezeInput) (*v1.Freeze, error) {
	sub, err := s.repo.FindByID(ctx, subscriptionID, false)
	if err != nil || sub.MemberID != memberID {
		return nil, errSubNotFound
	}
	return s.requestFreeze(ctx, sub, in)
}

func (s *subscriptionService) requestFreeze(ctx context.Context, sub *entity.Subscription, in v1.FreezeInput) (*v1.Freeze, error) {
	start, err := subParseDate(in.StartDate, "Ngày bắt đầu bảo lưu")
	if err != nil {
		return nil, err
	}
	end, err := subParseDate(in.EndDate, "Ngày kết thúc bảo lưu")
	if err != nil {
		return nil, err
	}
	plan, err := s.repo.FindPlan(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	today := s.today()
	var freezeID uint
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		sub, err := s.repo.FindByID(ctx, sub.ID, true)
		if err != nil {
			return err
		}
		freezes, err := s.subFreezes(ctx, sub.ID)
		if err != nil {
			return err
		}
		if err := subValidateFreeze(sub, plan, freezes, start, end, today); err != nil {
			return err
		}
		f := &entity.SubscriptionFreeze{
			SubscriptionID: sub.ID, StartDate: start, EndDate: end, Reason: invoiceTruncate(strings.TrimSpace(in.Reason), 255),
			Status: entity.FreezeStatusPending, RequestedBy: reqctx.UserID(ctx),
		}
		if err := s.repo.CreateFreeze(ctx, f); err != nil {
			return err
		}
		days := subFreezeDays(f)
		if err := s.event(ctx, sub.ID, entity.SubEventFreezeRequested, map[string]any{
			"freeze_id": f.ID, "start_date": subFmt(start), "end_date": subFmt(end), "days": days, "reason": f.Reason,
		}); err != nil {
			return err
		}
		s.logAudit(ctx, AuditFreezeRequest, auditEntityFreeze, f.ID, nil, map[string]any{
			"subscription_id": sub.ID, "start_date": subFmt(start), "end_date": subFmt(end), "days": days, "reason": f.Reason,
		})
		freezeID = f.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	row, err := s.repo.FindFreezeRow(ctx, freezeID)
	if err != nil {
		return nil, err
	}
	if s.notifier != nil {
		branchID := row.BranchID
		body := fmt.Sprintf("%s (%s) xin bảo lưu gói %s từ %s đến %s (%d ngày).", row.MemberName, row.MemberCode, row.PlanName,
			invoiceDescDate(row.StartDate), invoiceDescDate(row.EndDate), subFreezeDays(&row.SubscriptionFreeze))
		if err := s.notifier.NotifyPermission(ctx, consts.PermSubscriptionApproveFreeze, &branchID, NotifyFreezeRequest,
			"Yêu cầu bảo lưu gói tập", body, "/admin/subscription-freezes?status=PENDING"); err != nil {
			subWarn(ctx, "notify freeze request: %v", err)
		}
	}
	return toFreezeRes(*row), nil
}

// subFreezes trả các yêu cầu bảo lưu (entity) của gói.
func (s *subscriptionService) subFreezes(ctx context.Context, subID uint) ([]entity.SubscriptionFreeze, error) {
	rows, err := s.repo.FreezesBySubscription(ctx, subID)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, func(r repository.FreezeRow) entity.SubscriptionFreeze { return r.SubscriptionFreeze }), nil
}

// lockFreeze khóa yêu cầu bảo lưu và gói của nó, kiểm tra phạm vi chi nhánh.
func (s *subscriptionService) lockFreeze(ctx context.Context, freezeID uint) (*entity.SubscriptionFreeze, *entity.Subscription, error) {
	f, err := s.repo.FindFreeze(ctx, freezeID, true)
	if err != nil {
		return nil, nil, subMapNotFound(err, errFreezeNotFound)
	}
	sub, err := s.loadSub(ctx, f.SubscriptionID, true)
	if err != nil {
		return nil, nil, err
	}
	return f, sub, nil
}

// applyFreeze áp bảo lưu f vào gói sub (đã khóa) và ghi sự kiện FROZEN.
func (s *subscriptionService) applyFreeze(ctx context.Context, sub *entity.Subscription, f *entity.SubscriptionFreeze) error {
	days := subFreezeDays(f)
	oldEnd := sub.EndDate
	subApplyFreeze(sub, days)
	if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{
		"status": sub.Status, "end_date": sub.EndDate, "freeze_days_used": sub.FreezeDaysUsed, "freeze_times_used": sub.FreezeTimesUsed,
	}); err != nil {
		return err
	}
	return s.event(ctx, sub.ID, entity.SubEventFrozen, map[string]any{
		"freeze_id": f.ID, "start_date": subFmt(f.StartDate), "end_date": subFmt(f.EndDate), "days": days,
		"old_end_date": subFmt(oldEnd), "new_end_date": subFmt(sub.EndDate),
	})
}

// unfreeze kết thúc bảo lưu đã hết hạn: gói ACTIVE, bảo lưu ENDED, sự kiện UNFROZEN.
func (s *subscriptionService) unfreeze(ctx context.Context, sub *entity.Subscription, f *entity.SubscriptionFreeze) error {
	if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{"status": entity.SubStatusActive}); err != nil {
		return err
	}
	if err := s.repo.UpdateFreezeFields(ctx, f.ID, map[string]any{"status": entity.FreezeStatusEnded}); err != nil {
		return err
	}
	return s.event(ctx, sub.ID, entity.SubEventUnfrozen, map[string]any{"freeze_id": f.ID, "end_date": subFmt(sub.EndDate)})
}

func (s *subscriptionService) ApproveFreeze(ctx context.Context, freezeID uint, note string) (*v1.Freeze, error) {
	today := s.today()
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		f, sub, err := s.lockFreeze(ctx, freezeID)
		if err != nil {
			return err
		}
		if f.Status != entity.FreezeStatusPending {
			return rescode.New(rescode.Conflict, "Yêu cầu bảo lưu đã được xử lý")
		}
		if sub.Status != entity.SubStatusActive && sub.Status != entity.SubStatusFrozen {
			return rescode.New(rescode.BadRequest, "Gói tập không còn hoạt động, hãy từ chối yêu cầu")
		}
		if subNorm(f.EndDate).Before(today) {
			return rescode.New(rescode.BadRequest, "Yêu cầu bảo lưu đã quá thời gian bảo lưu, hãy từ chối")
		}
		now := s.now()
		f.Status, f.ReviewedBy, f.ReviewedAt, f.ReviewNote = entity.FreezeStatusApproved, reqctx.UserID(ctx), &now, invoiceTruncate(strings.TrimSpace(note), 255)
		if err := s.repo.UpdateFreezeFields(ctx, f.ID, map[string]any{
			"status": f.Status, "reviewed_by": f.ReviewedBy, "reviewed_at": now, "review_note": f.ReviewNote,
		}); err != nil {
			return err
		}
		applied := false
		if !subNorm(f.StartDate).After(today) && sub.Status == entity.SubStatusActive {
			if err := s.applyFreeze(ctx, sub, f); err != nil {
				return err
			}
			applied = true
		}
		s.logAudit(ctx, AuditFreezeApprove, auditEntityFreeze, f.ID, map[string]any{"status": entity.FreezeStatusPending},
			map[string]any{"status": f.Status, "note": f.ReviewNote, "applied": applied, "subscription_end_date": subFmt(sub.EndDate)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.freezeResult(ctx, freezeID, true)
}

func (s *subscriptionService) RejectFreeze(ctx context.Context, freezeID uint, note string) (*v1.Freeze, error) {
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		f, sub, err := s.lockFreeze(ctx, freezeID)
		if err != nil {
			return err
		}
		if f.Status != entity.FreezeStatusPending {
			return rescode.New(rescode.Conflict, "Yêu cầu bảo lưu đã được xử lý")
		}
		now := s.now()
		note = invoiceTruncate(strings.TrimSpace(note), 255)
		if err := s.repo.UpdateFreezeFields(ctx, f.ID, map[string]any{
			"status": entity.FreezeStatusRejected, "reviewed_by": reqctx.UserID(ctx), "reviewed_at": now, "review_note": note,
		}); err != nil {
			return err
		}
		if err := s.event(ctx, sub.ID, entity.SubEventFreezeRejected, map[string]any{"freeze_id": f.ID, "note": note}); err != nil {
			return err
		}
		s.logAudit(ctx, AuditFreezeReject, auditEntityFreeze, f.ID, map[string]any{"status": entity.FreezeStatusPending},
			map[string]any{"status": entity.FreezeStatusRejected, "note": note})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.freezeResult(ctx, freezeID, false)
}

func (s *subscriptionService) EndFreezeEarly(ctx context.Context, freezeID uint, note string) (*v1.Freeze, error) {
	today := s.today()
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		f, sub, err := s.lockFreeze(ctx, freezeID)
		if err != nil {
			return err
		}
		before := map[string]any{"status": f.Status, "end_date": subFmt(f.EndDate), "subscription_end_date": subFmt(sub.EndDate)}
		after := map[string]any{"note": strings.TrimSpace(note)}
		switch {
		case f.Status == entity.FreezeStatusPending || (f.Status == entity.FreezeStatusApproved && !subFreezeApplied(sub, f, today)):
			// Chưa áp dụng → hủy yêu cầu, không ảnh hưởng gói.
			if err := s.repo.UpdateFreezeFields(ctx, f.ID, map[string]any{"status": entity.FreezeStatusCancelled}); err != nil {
				return err
			}
			after["status"] = entity.FreezeStatusCancelled
		case f.Status == entity.FreezeStatusApproved:
			unused, newEnd, cancelled := subEndFreezeEarly(sub, f, today)
			if err := s.repo.UpdateFields(ctx, sub.ID, map[string]any{
				"status": sub.Status, "end_date": sub.EndDate, "freeze_days_used": sub.FreezeDaysUsed, "freeze_times_used": sub.FreezeTimesUsed,
			}); err != nil {
				return err
			}
			fields := map[string]any{"status": entity.FreezeStatusEnded, "end_date": newEnd}
			if cancelled {
				fields = map[string]any{"status": entity.FreezeStatusCancelled}
			}
			if err := s.repo.UpdateFreezeFields(ctx, f.ID, fields); err != nil {
				return err
			}
			if err := s.event(ctx, sub.ID, entity.SubEventUnfrozen, map[string]any{
				"freeze_id": f.ID, "early": true, "returned_days": unused, "end_date": subFmt(sub.EndDate),
			}); err != nil {
				return err
			}
			after["status"], after["returned_days"], after["subscription_end_date"] = fields["status"], unused, subFmt(sub.EndDate)
		default:
			return rescode.New(rescode.BadRequest, "Yêu cầu bảo lưu đã kết thúc hoặc bị từ chối")
		}
		s.logAudit(ctx, AuditFreezeEnd, auditEntityFreeze, f.ID, before, after)
		return nil
	})
	if err != nil {
		return nil, err
	}
	row, err := s.repo.FindFreezeRow(ctx, freezeID)
	if err != nil {
		return nil, err
	}
	return toFreezeRes(*row), nil
}

// freezeResult đọc lại yêu cầu bảo lưu và thông báo kết quả cho hội viên có tài khoản cổng.
func (s *subscriptionService) freezeResult(ctx context.Context, freezeID uint, approved bool) (*v1.Freeze, error) {
	row, err := s.repo.FindFreezeRow(ctx, freezeID)
	if err != nil {
		return nil, err
	}
	if s.notifier != nil {
		if m, err := s.repo.FindMember(ctx, row.MemberID); err == nil && m.UserID != nil {
			result := "đã được duyệt"
			if !approved {
				result = "đã bị từ chối"
			}
			body := fmt.Sprintf("Yêu cầu bảo lưu gói %s từ %s đến %s %s.", row.PlanName, invoiceDescDate(row.StartDate), invoiceDescDate(row.EndDate), result)
			if row.ReviewNote != "" {
				body += " Ghi chú: " + row.ReviewNote
			}
			if err := s.notifier.Notify(ctx, []uint{*m.UserID}, NotifyFreezeResult, "Kết quả yêu cầu bảo lưu", body, "/member/subscriptions"); err != nil {
				subWarn(ctx, "notify freeze result: %v", err)
			}
		}
	}
	return toFreezeRes(*row), nil
}

var freezeOrderable = []string{"id", "start_date", "end_date", "created_at", "status"}

func (s *subscriptionService) ListFreezes(ctx context.Context, req *v1.FreezeListReq) (*v1.FreezeListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	q := req.PageReq.ToQuery(freezeOrderable, "id")
	q.OrderBy = "f." + q.OrderBy
	rows, total, err := s.repo.ListFreezes(ctx, q, repository.FreezeListFilter{BranchIDs: branchIDs, Status: req.Status, SubscriptionID: req.SubscriptionID})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.FreezeRow) *v1.Freeze { return toFreezeRes(r) }), total, q), nil
}
