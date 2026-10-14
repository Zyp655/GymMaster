package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

func subCode(err error) int { return codeOf(err) }

func TestSubDates(t *testing.T) {
	assert.Equal(t, subD(11, 12), subEndDate(subD(10, 14), 30))
	assert.Equal(t, subD(10, 14), subEndDate(subD(10, 14), 1))
	assert.Equal(t, 29, subDaysBetween(subD(10, 14), subD(11, 12)))
	assert.Equal(t, subD(10, 14), subChainStart(subTestToday, nil))
	assert.Equal(t, subD(11, 13), subChainStart(subTestToday, &entity.Subscription{EndDate: subD(11, 12)}))
	// Gói cũ hết hạn hôm qua → không bắt đầu trong quá khứ.
	assert.Equal(t, subD(10, 14), subChainStart(subTestToday, &entity.Subscription{EndDate: subD(10, 12)}))
	// Ngày VN: 23:30 ngày 13/10 UTC = 06:30 ngày 14/10 giờ VN.
	assert.Equal(t, subD(10, 14), subDateOf(time.Date(2026, 10, 13, 23, 30, 0, 0, time.UTC)))
}

func TestSubSell_ChainingAndValidation(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()

	a, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 2, BranchID: 1}, &v1.PaymentInput{Amount: 1350000, Method: "CASH"})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-14", a.StartDate)
	assert.Equal(t, "2027-01-11", a.EndDate)
	assert.Equal(t, entity.SubStatusActive, a.Status)
	assert.Nil(t, a.PreviousID)
	assert.Equal(t, []string{entity.SubEventCreated, entity.SubEventActivated}, fx.repo.eventTypes(a.ID))

	// Gói ALL_BRANCHES mới nối tiếp gói ALL_BRANCHES đang chạy.
	b, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 3, BranchID: 2}, nil)
	require.NoError(t, err)
	assert.Equal(t, "2027-01-12", b.StartDate)
	assert.Equal(t, "2028-01-11", b.EndDate)
	require.NotNil(t, b.PreviousID)
	assert.Equal(t, a.ID, *b.PreviousID)
	assert.Equal(t, entity.SubStatusPending, b.Status, "chưa thu tiền thì PENDING")

	// Khác phạm vi (SINGLE_BRANCH) → bắt đầu hôm nay.
	c, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 4, BranchID: 1}, nil)
	require.NoError(t, err)
	assert.Equal(t, "2026-10-14", c.StartDate)
	assert.Equal(t, "2026-12-12", c.EndDate)
	require.NotNil(t, c.TotalSessions)
	assert.Equal(t, 12, *c.TotalSessions)
	assert.Equal(t, 12, *c.RemainingSessions)

	// start_date truyền vào → không nối gói.
	d, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 2, BranchID: 1, StartDate: "2026-11-01"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "2026-11-01", d.StartDate)
	assert.Nil(t, d.PreviousID)

	_, err = fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 2, BranchID: 1, StartDate: "2026-10-13"}, nil)
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "ngày bắt đầu trong quá khứ")
	_, err = fx.svc.Sell(ctx, v1.SellInput{MemberID: 3, PlanID: 2, BranchID: 1}, nil)
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "hội viên không hoạt động")
	fx.repo.plans[1].Status = "INACTIVE"
	_, err = fx.svc.Sell(ctx, v1.SellInput{MemberID: 2, PlanID: 1, BranchID: 1}, nil)
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "gói ngừng bán")
	_, err = fx.svc.Sell(subStaffCtx(), v1.SellInput{MemberID: 2, PlanID: 2, BranchID: 1}, nil)
	assert.Equal(t, rescode.PermissionDenied.Code(), subCode(err), "chi nhánh ngoài phạm vi")
	_, err = fx.svc.Sell(ctx, v1.SellInput{MemberID: 2, PlanID: 2, BranchID: 1}, &v1.PaymentInput{Amount: 2000000, Method: "CASH"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "thu vượt tổng tiền")
}

func TestSubQuote(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	q, err := fx.svc.Quote(ctx, v1.SellInput{MemberID: 1, PlanID: 2, BranchID: 1, PromotionCode: "chaodong"})
	require.NoError(t, err)
	assert.Equal(t, 1350000.0, q.Subtotal)
	assert.Equal(t, 135000.0, q.Discount)
	assert.Equal(t, 1215000.0, q.Total)
	require.NotNil(t, q.Promotion)
	assert.Equal(t, "CHAODONG", q.Promotion.Code)
	assert.Equal(t, "2026-10-14", q.StartDate)
	assert.False(t, q.Chained)
	assert.Empty(t, fx.repo.subs, "quote không ghi DB")
	assert.Empty(t, fx.invRep.invoices)
	assert.Equal(t, 0, fx.promos.items[1].UsedCount)

	// M1 (500.000) đạt min_amount nhưng S12 cũng đạt; mã chỉ áp gói tập.
	_, err = fx.svc.Quote(ctx, v1.SellInput{MemberID: 1, PlanID: 2, BranchID: 1, PromotionCode: "SAI"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
}

func TestSubSell_PromotionAndPartialPayments(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	sub, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 2, BranchID: 1, PromotionCode: "CHAODONG"},
		&v1.PaymentInput{Amount: 500000, Method: "CASH"})
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusPending, sub.Status)
	require.NotNil(t, sub.Invoice)
	assert.Equal(t, 1215000.0, sub.Invoice.Total)
	assert.Equal(t, 135000.0, sub.Invoice.Discount)
	assert.Equal(t, entity.InvoiceStatusPartial, sub.Invoice.Status)
	assert.Equal(t, 1215000.0, sub.Price, "giá trị gói = số tiền thực bán")
	assert.Equal(t, 1, fx.promos.items[1].UsedCount)
	assert.Equal(t, entity.InvoiceItemMembership, sub.Invoice.Items[0].ItemType)
	assert.Equal(t, sub.ID, *sub.Invoice.Items[0].RefID)

	_, err = fx.inv.AddPayment(ctx, *sub.InvoiceID, PaymentInput{Amount: 715001, Method: "CASH"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "không thu quá số còn nợ")
	assert.Equal(t, entity.SubStatusPending, fx.repo.subs[sub.ID].Status)

	_, err = fx.inv.AddPayment(ctx, *sub.InvoiceID, PaymentInput{Amount: 715000, Method: "TRANSFER"})
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[sub.ID].Status)
	assert.Equal(t, entity.InvoiceStatusPaid, fx.invRep.invoices[*sub.InvoiceID].Status)
	assert.Equal(t, []string{entity.SubEventCreated, entity.SubEventActivated}, fx.repo.eventTypes(sub.ID))
}

func TestSubVoidInvoiceCancelsPending(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	sub, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 1, BranchID: 1}, nil)
	require.NoError(t, err)
	require.NoError(t, fx.inv.Void(ctx, *sub.InvoiceID, "Khách đổi ý"))
	assert.Equal(t, entity.SubStatusCancelled, fx.repo.subs[sub.ID].Status)

	// Hủy gói PENDING chưa thu → hóa đơn bị void.
	sub2, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 1, BranchID: 1}, nil)
	require.NoError(t, err)
	got, err := fx.svc.Cancel(ctx, sub2.ID, v1.CancelInput{Reason: "Không tập nữa"})
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusCancelled, got.Status)
	assert.Equal(t, entity.InvoiceStatusVoid, fx.invRep.invoices[*sub2.InvoiceID].Status)
}

func TestSubCancelWithRefund(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	sub, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 2, BranchID: 1}, &v1.PaymentInput{Amount: 1350000, Method: "CASH"})
	require.NoError(t, err)
	_, err = fx.svc.Cancel(ctx, sub.ID, v1.CancelInput{Reason: "Chuyển công tác", Refund: &v1.RefundInput{Amount: 2000000, Method: "CASH"}})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
	got, err := fx.svc.Cancel(ctx, sub.ID, v1.CancelInput{Reason: "Chuyển công tác", Refund: &v1.RefundInput{Amount: 1000000, Method: "CASH"}})
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusCancelled, got.Status)
	assert.Equal(t, 1000000.0, fx.invRep.invoices[*sub.InvoiceID].RefundedAmount)
	_, err = fx.svc.Cancel(ctx, sub.ID, v1.CancelInput{Reason: "lần 2"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
}

func TestSubRemainingValue(t *testing.T) {
	// TIME: 30 ngày 01/10–30/10, giá 600.000, hôm nay 21/10 → còn 10 ngày → 200.000.
	s := &entity.Subscription{StartDate: subD(10, 1), EndDate: subD(10, 30), Price: 600000}
	v, total, rem := subRemaining(s, entity.PlanTypeTime, subD(10, 21))
	assert.Equal(t, 200000.0, v)
	assert.Equal(t, 30, total)
	assert.Equal(t, 10, rem)
	// Chưa bắt đầu → còn nguyên giá trị; đã quá hạn → 0.
	v, _, _ = subRemaining(s, entity.PlanTypeTime, subD(9, 1))
	assert.Equal(t, 600000.0, v)
	v, _, _ = subRemaining(s, entity.PlanTypeTime, subD(11, 5))
	assert.Equal(t, 0.0, v)
	// SESSION: còn 3/12 buổi giá 600.000 → 150.000 (không phụ thuộc ngày).
	s = &entity.Subscription{StartDate: subD(10, 1), EndDate: subD(11, 29), Price: 600000, TotalSessions: subIntPtr(12), RemainingSessions: subIntPtr(3)}
	v, _, _ = subRemaining(s, entity.PlanTypeSession, subD(10, 2))
	assert.Equal(t, 150000.0, v)
}

func TestSubUpgrade(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	old := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(10, 5), EndDate: subD(11, 3),
		Price: 600000, Status: entity.SubStatusActive})

	_, err := fx.svc.UpgradeQuote(ctx, old.ID, 1)
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "gói mới phải khác")

	q, err := fx.svc.UpgradeQuote(ctx, old.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, 30, q.TotalDays)
	assert.Equal(t, 21, q.RemainingDays) // 14/10 → 03/11
	assert.Equal(t, 420000.0, q.RemainingValue)
	assert.Equal(t, 930000.0, q.AmountDue)
	assert.Equal(t, "2026-10-14", q.StartDate)
	assert.Equal(t, "2027-01-11", q.EndDate)

	got, err := fx.svc.Upgrade(ctx, old.ID, v1.UpgradeInput{PlanID: 2})
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusPending, got.Status)
	assert.Equal(t, old.ID, *got.PreviousID)
	assert.Equal(t, 930000.0, got.Invoice.Total)
	assert.Equal(t, 420000.0, got.Invoice.Items[0].Discount)
	assert.Contains(t, got.Invoice.Items[0].Description, "khấu trừ")
	assert.Equal(t, entity.SubStatusCancelled, fx.repo.subs[old.ID].Status)
	assert.Contains(t, fx.repo.eventTypes(old.ID), entity.SubEventUpgraded)

	// Hủy hóa đơn nâng cấp (chưa thu) → khôi phục gói cũ.
	require.NoError(t, fx.inv.Void(ctx, *got.InvoiceID, "Đổi ý"))
	assert.Equal(t, entity.SubStatusCancelled, fx.repo.subs[got.ID].Status)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[old.ID].Status)

	// SESSION: còn 6/12 buổi, giá 600.000 → khấu trừ 300.000; gói mới đắt hơn giá trị → thu chênh lệch.
	sess := fx.addSub(entity.Subscription{MemberID: 2, PlanID: 4, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(11, 29),
		Price: 600000, Status: entity.SubStatusActive, TotalSessions: subIntPtr(12), RemainingSessions: subIntPtr(6)})
	q, err = fx.svc.UpgradeQuote(ctx, sess.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, 300000.0, q.RemainingValue)
	assert.Equal(t, 200000.0, q.AmountDue)

	// Giá trị còn lại ≥ giá gói mới → hóa đơn 0đ, gói mới ACTIVE ngay.
	big := fx.addSub(entity.Subscription{MemberID: 2, PlanID: 3, BranchID: 1, StartDate: subD(10, 14), EndDate: subD(10, 13).AddDate(1, 0, 0),
		Price: 4800000, Status: entity.SubStatusActive})
	got, err = fx.svc.Upgrade(ctx, big.ID, v1.UpgradeInput{PlanID: 2})
	require.NoError(t, err)
	assert.Equal(t, 0.0, got.Invoice.Total)
	assert.Equal(t, entity.SubStatusActive, got.Status)

	// Chỉ nâng cấp gói ACTIVE.
	fx.repo.subs[sess.ID].Status = entity.SubStatusFrozen
	_, err = fx.svc.UpgradeQuote(ctx, sess.ID, 1)
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
}

func TestSubRenew(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	old := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(10, 30),
		Price: 500000, Status: entity.SubStatusActive})
	got, err := fx.svc.Renew(ctx, old.ID, v1.RenewInput{Payment: &v1.PaymentInput{Amount: 500000, Method: "CASH"}})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-31", got.StartDate)
	assert.Equal(t, "2026-11-29", got.EndDate)
	assert.Equal(t, old.ID, *got.PreviousID)
	assert.Equal(t, entity.SubStatusActive, got.Status)
	assert.Contains(t, fx.repo.eventTypes(old.ID), entity.SubEventRenewed)

	// Gói đã hết hạn → gia hạn bắt đầu hôm nay.
	exp := fx.addSub(entity.Subscription{MemberID: 2, PlanID: 1, BranchID: 1, StartDate: subD(9, 1), EndDate: subD(9, 30),
		Price: 500000, Status: entity.SubStatusExpired})
	q, err := fx.svc.RenewQuote(ctx, exp.ID, v1.RenewInput{})
	require.NoError(t, err)
	assert.Equal(t, "2026-10-14", q.StartDate)
	assert.Equal(t, exp.ID, *q.PreviousID)

	fx.repo.subs[exp.ID].Status = entity.SubStatusCancelled
	_, err = fx.svc.RenewQuote(ctx, exp.ID, v1.RenewInput{})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
}

func TestSubFreeze_Limits(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	m1 := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(10, 30), Price: 500000, Status: entity.SubStatusActive})

	_, err := fx.svc.RequestFreeze(ctx, m1.ID, v1.FreezeInput{StartDate: "2026-10-15", EndDate: "2026-10-22"}) // 8 > 7 ngày
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
	_, err = fx.svc.RequestFreeze(ctx, m1.ID, v1.FreezeInput{StartDate: "2026-10-13", EndDate: "2026-10-15"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "bắt đầu trong quá khứ")
	_, err = fx.svc.RequestFreeze(ctx, m1.ID, v1.FreezeInput{StartDate: "2026-10-20", EndDate: "2026-10-18"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "kết thúc trước bắt đầu")
	_, err = fx.svc.RequestFreeze(ctx, m1.ID, v1.FreezeInput{StartDate: "2026-11-01", EndDate: "2026-11-03"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "ngoài thời hạn gói")

	f, err := fx.svc.RequestFreeze(ctx, m1.ID, v1.FreezeInput{StartDate: "2026-10-16", EndDate: "2026-10-22", Reason: "Ốm"})
	require.NoError(t, err)
	assert.Equal(t, 7, f.Days)
	assert.Equal(t, entity.FreezeStatusPending, f.Status)
	_, err = fx.svc.RequestFreeze(ctx, m1.ID, v1.FreezeInput{StartDate: "2026-10-25", EndDate: "2026-10-25"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "hết số lần (1)")

	// M12: 60 ngày / 3 lần – cộng dồn cả yêu cầu đang chờ và số đã dùng.
	m12 := fx.addSub(entity.Subscription{MemberID: 2, PlanID: 3, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(10, 1).AddDate(1, 0, -1),
		Price: 4800000, Status: entity.SubStatusActive, FreezeDaysUsed: 30, FreezeTimesUsed: 1})
	_, err = fx.svc.RequestFreeze(ctx, m12.ID, v1.FreezeInput{StartDate: "2026-10-20", EndDate: "2026-11-08"}) // 20 ngày → 50
	require.NoError(t, err)
	_, err = fx.svc.RequestFreeze(ctx, m12.ID, v1.FreezeInput{StartDate: "2026-11-05", EndDate: "2026-11-06"})
	assert.Equal(t, rescode.Conflict.Code(), subCode(err), "trùng khoảng")
	_, err = fx.svc.RequestFreeze(ctx, m12.ID, v1.FreezeInput{StartDate: "2026-12-01", EndDate: "2026-12-11"}) // 11 ngày → 61
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "vượt số ngày")
	_, err = fx.svc.RequestFreeze(ctx, m12.ID, v1.FreezeInput{StartDate: "2026-12-01", EndDate: "2026-12-10"}) // 10 ngày → 60, lần 3
	require.NoError(t, err)
	_, err = fx.svc.RequestFreeze(ctx, m12.ID, v1.FreezeInput{StartDate: "2027-01-01", EndDate: "2027-01-01"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "hết lần")

	// Gói không cho bảo lưu / gói không ACTIVE.
	s12 := fx.addSub(entity.Subscription{MemberID: 2, PlanID: 4, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(11, 29), Price: 600000, Status: entity.SubStatusActive})
	_, err = fx.svc.RequestFreeze(ctx, s12.ID, v1.FreezeInput{StartDate: "2026-10-20", EndDate: "2026-10-21"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
	fx.repo.subs[m1.ID].Status = entity.SubStatusPending
	_, err = fx.svc.RequestFreeze(ctx, m1.ID, v1.FreezeInput{StartDate: "2026-10-27", EndDate: "2026-10-27"})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
	// Cổng hội viên: gói của người khác → không tìm thấy.
	_, err = fx.svc.RequestFreezeByMember(ctx, 1, m12.ID, v1.FreezeInput{StartDate: "2027-02-01", EndDate: "2027-02-01"})
	assert.Equal(t, rescode.NotFound.Code(), subCode(err))
}

func TestSubFreeze_ApplyAndEnd(t *testing.T) {
	// Hàm thuần.
	s := &entity.Subscription{Status: entity.SubStatusActive, StartDate: subD(10, 1), EndDate: subD(10, 30)}
	subApplyFreeze(s, 10)
	assert.Equal(t, entity.SubStatusFrozen, s.Status)
	assert.Equal(t, subD(11, 9), s.EndDate)
	assert.Equal(t, 10, s.FreezeDaysUsed)
	assert.Equal(t, 1, s.FreezeTimesUsed)
	f := &entity.SubscriptionFreeze{Status: entity.FreezeStatusApproved, StartDate: subD(10, 10), EndDate: subD(10, 19)}
	unused, newEnd, cancelled := subEndFreezeEarly(s, f, subD(10, 15))
	assert.Equal(t, 5, unused)
	assert.False(t, cancelled)
	assert.Equal(t, subD(10, 14), newEnd)
	assert.Equal(t, subD(11, 4), s.EndDate)
	assert.Equal(t, 5, s.FreezeDaysUsed)
	assert.Equal(t, entity.SubStatusActive, s.Status)

	fx := newSubFixture()
	ctx := subOwnerCtx()
	sub := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 3, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(12, 31),
		Price: 4800000, Status: entity.SubStatusActive})

	// Bảo lưu bắt đầu hôm nay → duyệt là áp ngay.
	fr, err := fx.svc.RequestFreeze(ctx, sub.ID, v1.FreezeInput{StartDate: "2026-10-14", EndDate: "2026-10-23"})
	require.NoError(t, err)
	_, err = fx.svc.ApproveFreeze(subStaffCtx(), fr.ID, "")
	assert.Equal(t, rescode.PermissionDenied.Code(), subCode(err), "duyệt ngoài phạm vi chi nhánh")
	got, err := fx.svc.ApproveFreeze(ctx, fr.ID, "OK")
	require.NoError(t, err)
	assert.Equal(t, entity.FreezeStatusApproved, got.Status)
	assert.Equal(t, entity.SubStatusFrozen, fx.repo.subs[sub.ID].Status)
	assert.Equal(t, subD(1, 10).AddDate(1, 0, 0), fx.repo.subs[sub.ID].EndDate) // 31/12 + 10
	assert.Equal(t, 10, fx.repo.subs[sub.ID].FreezeDaysUsed)
	_, err = fx.svc.ApproveFreeze(ctx, fr.ID, "")
	assert.Equal(t, rescode.Conflict.Code(), subCode(err))

	// Cron: hết bảo lưu (24/10) → ACTIVE, freeze ENDED.
	n, err := fx.svc.ApplyDueFreezes(context.Background(), subD(10, 24))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[sub.ID].Status)
	assert.Equal(t, entity.FreezeStatusEnded, fx.repo.freezes[fr.ID].Status)

	// Bảo lưu tương lai: duyệt → chưa áp; cron đến ngày → FROZEN; kết thúc sớm → hoàn ngày chưa dùng.
	fx.setDay(subD(10, 25))
	fr2, err := fx.svc.RequestFreeze(ctx, sub.ID, v1.FreezeInput{StartDate: "2026-11-01", EndDate: "2026-11-20"})
	require.NoError(t, err)
	_, err = fx.svc.ApproveFreeze(ctx, fr2.ID, "")
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[sub.ID].Status)
	n, err = fx.svc.ApplyDueFreezes(context.Background(), subD(11, 1))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	endFrozen := fx.repo.subs[sub.ID].EndDate
	assert.Equal(t, subD(1, 30).AddDate(1, 0, 0), endFrozen) // 10/01/2027 + 20
	assert.Equal(t, 30, fx.repo.subs[sub.ID].FreezeDaysUsed)
	assert.Equal(t, 2, fx.repo.subs[sub.ID].FreezeTimesUsed)

	fx.setDay(subD(11, 6))
	got, err = fx.svc.EndFreezeEarly(ctx, fr2.ID, "Đi tập lại")
	require.NoError(t, err)
	assert.Equal(t, entity.FreezeStatusEnded, got.Status)
	assert.Equal(t, "2026-11-05", got.EndDate)
	assert.Equal(t, 5, got.Days)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[sub.ID].Status)
	assert.Equal(t, subAddDays(endFrozen, -15), fx.repo.subs[sub.ID].EndDate) // 06/11–20/11 = 15 ngày hoàn lại
	assert.Equal(t, 15, fx.repo.subs[sub.ID].FreezeDaysUsed)

	// Từ chối + hủy yêu cầu chưa áp dụng.
	fr3, err := fx.svc.RequestFreeze(ctx, sub.ID, v1.FreezeInput{StartDate: "2026-12-01", EndDate: "2026-12-02"})
	require.NoError(t, err)
	got, err = fx.svc.RejectFreeze(ctx, fr3.ID, "Không đủ lý do")
	require.NoError(t, err)
	assert.Equal(t, entity.FreezeStatusRejected, got.Status)
	fr4, err := fx.svc.RequestFreeze(ctx, sub.ID, v1.FreezeInput{StartDate: "2026-12-01", EndDate: "2026-12-02"})
	require.NoError(t, err)
	got, err = fx.svc.EndFreezeEarly(ctx, fr4.ID, "")
	require.NoError(t, err)
	assert.Equal(t, entity.FreezeStatusCancelled, got.Status)
	_, err = fx.svc.EndFreezeEarly(ctx, fr4.ID, "")
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
}

func TestSubTransfer(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	old := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(10, 5), EndDate: subD(11, 3),
		Price: 600000, Status: entity.SubStatusActive, FreezeDaysUsed: 2, FreezeTimesUsed: 1})

	_, err := fx.svc.Transfer(ctx, old.ID, v1.TransferInput{ToMemberID: 1})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "chuyển cho chính mình")
	_, err = fx.svc.Transfer(ctx, old.ID, v1.TransferInput{ToMemberID: 3})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err), "người nhận không hoạt động")

	got, err := fx.svc.Transfer(ctx, old.ID, v1.TransferInput{ToMemberID: 2})
	require.NoError(t, err)
	assert.Equal(t, uint(2), got.MemberID)
	assert.Equal(t, entity.SubStatusPending, got.Status, "chờ thu phí chuyển nhượng")
	assert.Equal(t, "2026-10-14", got.StartDate)
	assert.Equal(t, "2026-11-03", got.EndDate)
	assert.Equal(t, 420000.0, got.Price, "giá trị còn lại 21/30 ngày")
	assert.Equal(t, 2, got.FreezeDaysUsed)
	assert.Equal(t, old.ID, *got.PreviousID)
	assert.Equal(t, entity.SubStatusTransferred, fx.repo.subs[old.ID].Status)
	require.NotNil(t, got.Invoice)
	assert.Equal(t, 100000.0, got.Invoice.Total)
	assert.Equal(t, entity.InvoiceItemFee, got.Invoice.Items[0].ItemType)
	assert.Equal(t, uint(2), *got.Invoice.MemberID)

	_, err = fx.inv.AddPayment(ctx, *got.InvoiceID, PaymentInput{Amount: 100000, Method: "CASH"})
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[got.ID].Status)

	// Gói theo buổi, phí 0 → ACTIVE ngay, chuyển số buổi còn lại.
	sess := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 4, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(11, 29),
		Price: 600000, Status: entity.SubStatusActive, TotalSessions: subIntPtr(12), RemainingSessions: subIntPtr(5)})
	got, err = fx.svc.Transfer(ctx, sess.ID, v1.TransferInput{ToMemberID: 2})
	require.NoError(t, err)
	assert.Equal(t, entity.SubStatusActive, got.Status)
	assert.Nil(t, got.Invoice)
	assert.Equal(t, 5, *got.RemainingSessions)
	assert.Equal(t, 5, *got.TotalSessions)
	assert.Equal(t, 250000.0, got.Price)

	// Hủy hóa đơn phí chuyển nhượng (chưa thu) → hoàn tác: gói người nhận CANCELLED, gói cũ ACTIVE lại.
	old2 := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 2, BranchID: 1, StartDate: subD(10, 1), EndDate: subD(12, 29),
		Price: 1350000, Status: entity.SubStatusActive})
	got, err = fx.svc.Transfer(ctx, old2.ID, v1.TransferInput{ToMemberID: 2})
	require.NoError(t, err)
	require.NoError(t, fx.inv.Void(ctx, *got.InvoiceID, "Không chuyển nữa"))
	assert.Equal(t, entity.SubStatusCancelled, fx.repo.subs[got.ID].Status)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[old2.ID].Status)

	// Gói không ACTIVE.
	_, err = fx.svc.Transfer(ctx, old.ID, v1.TransferInput{ToMemberID: 2})
	assert.Equal(t, rescode.BadRequest.Code(), subCode(err))
}

func TestSubExpireDue(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	expActive := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(9, 14), EndDate: subD(10, 13), Price: 1, Status: entity.SubStatusActive})
	expFrozen := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(9, 1), EndDate: subD(10, 10), Price: 1, Status: entity.SubStatusFrozen})
	endsToday := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(9, 15), EndDate: subD(10, 14), Price: 1, Status: entity.SubStatusActive})
	cancelled := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(9, 1), EndDate: subD(9, 30), Price: 1, Status: entity.SubStatusCancelled})

	// PENDING bán ngày 09/10 (quá 3 ngày) chưa thu → hủy + void; bán 12/10 → giữ; bán 09/10 đã thu 1 phần → giữ.
	fx.setDay(subD(10, 9))
	stale, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 2, PlanID: 1, BranchID: 1}, nil)
	require.NoError(t, err)
	partial, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 2, PlanID: 2, BranchID: 1}, &v1.PaymentInput{Amount: 1000, Method: "CASH"})
	require.NoError(t, err)
	fx.setDay(subD(10, 12))
	fresh, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 2, PlanID: 3, BranchID: 1}, nil)
	require.NoError(t, err)

	fx.setDay(subTestToday)
	n, err := fx.svc.ExpireDue(context.Background(), subTestToday)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, entity.SubStatusExpired, fx.repo.subs[expActive.ID].Status)
	assert.Equal(t, entity.SubStatusExpired, fx.repo.subs[expFrozen.ID].Status)
	assert.Equal(t, entity.SubStatusActive, fx.repo.subs[endsToday.ID].Status)
	assert.Equal(t, entity.SubStatusCancelled, fx.repo.subs[cancelled.ID].Status)
	assert.Equal(t, entity.SubStatusCancelled, fx.repo.subs[stale.ID].Status)
	assert.Equal(t, entity.InvoiceStatusVoid, fx.invRep.invoices[*stale.InvoiceID].Status)
	assert.Equal(t, entity.SubStatusPending, fx.repo.subs[partial.ID].Status)
	assert.Equal(t, entity.SubStatusPending, fx.repo.subs[fresh.ID].Status)
	assert.Contains(t, fx.repo.eventTypes(expActive.ID), entity.SubEventExpired)

	// Chạy lại không đổi gì.
	n, err = fx.svc.ExpireDue(context.Background(), subTestToday)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestSubListExpiring(t *testing.T) {
	fx := newSubFixture()
	a := fx.addSub(entity.Subscription{MemberID: 1, PlanID: 1, BranchID: 1, StartDate: subD(9, 22), EndDate: subD(10, 21), Status: entity.SubStatusActive})
	fx.addSub(entity.Subscription{MemberID: 2, PlanID: 1, BranchID: 1, StartDate: subD(9, 22), EndDate: subD(10, 20), Status: entity.SubStatusActive})
	b := fx.addSub(entity.Subscription{MemberID: 2, PlanID: 1, BranchID: 1, StartDate: subD(9, 16), EndDate: subD(10, 15), Status: entity.SubStatusActive})
	rows, err := fx.svc.ListExpiring(context.Background(), []int{7, 1}, subTestToday)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	got := map[uint]int{}
	for _, r := range rows {
		got[r.SubscriptionID] = r.DaysLeft
	}
	assert.Equal(t, map[uint]int{a.ID: 7, b.ID: 1}, got)
}
