package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/pt/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

type ptPay = v1.PaymentInput

func ptSell(memberID, packageID, trainerID uint) v1.SellInput {
	return v1.SellInput{MemberID: memberID, PTPackageID: packageID, TrainerID: trainerID, BranchID: 1}
}

func ptRFC(t time.Time) string { return t.Format(time.RFC3339) }

func TestPTCommissionCalc(t *testing.T) {
	base, amount := ptCommission(3500000, 10, 10)
	assert.Equal(t, 350000.0, base)
	assert.Equal(t, 35000.0, amount)
	// Làm tròn VND: 1.000.000 / 7 = 142.857,14 → 142.857; × 12,5% = 17.857,125 → 17.857.
	base, amount = ptCommission(1000000, 7, 12.5)
	assert.Equal(t, 142857.0, base)
	assert.Equal(t, 17857.0, amount)
	_, amount = ptCommission(3500000, 10, 0)
	assert.Equal(t, 0.0, amount)
	// Kỳ theo giờ VN: 18:00 UTC ngày 31/10 = 01:00 ngày 01/11 giờ VN.
	assert.Equal(t, "2026-11", ptPeriod(time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC)))
	assert.Equal(t, "2026-10", ptPeriod(time.Date(2026, 10, 31, 16, 59, 0, 0, time.UTC)))
	assert.NoError(t, ptValidPeriod("2026-10"))
	assert.Error(t, ptValidPeriod("2026-13"))
}

func TestPTCancelOutcome(t *testing.T) {
	start := ptVN(10, 20, 8, 0)
	// Đúng hạn (trước 12 giờ) → hủy, không tính buổi.
	st, late, err := ptCancelOutcome(start, start.Add(-13*time.Hour), 720, true, false)
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionCancelled, st)
	assert.False(t, late)
	// Đúng mốc hạn chót vẫn là đúng hạn.
	_, late, _ = ptCancelOutcome(start, start.Add(-12*time.Hour), 720, false, true)
	assert.False(t, late)
	// Muộn: nhân viên chọn tính buổi → NO_SHOW; không chọn → CANCELLED; hội viên → bị từ chối.
	st, late, err = ptCancelOutcome(start, start.Add(-time.Hour), 720, true, false)
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionNoShow, st)
	assert.True(t, late)
	st, _, _ = ptCancelOutcome(start, start.Add(-time.Hour), 720, false, false)
	assert.Equal(t, entity.PTSessionCancelled, st)
	_, _, err = ptCancelOutcome(start, start.Add(-time.Hour), 720, false, true)
	assert.Equal(t, 40000, codeOf(err))
	assert.Contains(t, err.Error(), "12 giờ")
}

func TestPTSell_PendingThenPaidActivates(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()

	q, err := fx.svc.Quote(ctx, ptSell(1, 1, 6))
	require.NoError(t, err)
	assert.Equal(t, "2026-10-14", q.StartDate)
	assert.Equal(t, "2026-11-27", q.EndDate, "45 ngày tính cả ngày bắt đầu")
	assert.Equal(t, 3500000.0, q.Total)
	assert.Equal(t, 350000.0, q.PricePerSession)
	assert.Empty(t, fx.repo.contracts, "quote không ghi DB")

	res, err := fx.svc.Sell(ctx, ptSell(1, 1, 6), &ptPay{Amount: 1000000, Method: "CASH"})
	require.NoError(t, err)
	assert.Equal(t, entity.PTContractPending, res.Status, "thu một phần → PENDING")
	assert.Equal(t, 10, res.TotalSessions)
	assert.Equal(t, 60, res.SessionMinutes)
	assert.Equal(t, "2026-11-27", res.EndDate)
	assert.Equal(t, 3500000.0, res.Price)
	require.NotNil(t, res.Invoice)
	require.Len(t, res.Invoice.Items, 1)
	assert.Equal(t, entity.InvoiceItemPT, res.Invoice.Items[0].ItemType)
	assert.Equal(t, res.ID, *res.Invoice.Items[0].RefID)
	assert.Equal(t, entity.InvoiceStatusPartial, res.Invoice.Status)

	// Thu nốt → PaidHandler "PT" kích hoạt hợp đồng.
	_, err = fx.inv.AddPayment(ctx, *res.InvoiceID, PaymentInput{Amount: 2500000, Method: "TRANSFER"})
	require.NoError(t, err)
	assert.Equal(t, entity.PTContractActive, fx.repo.contracts[res.ID].Status)

	// Thu đủ ngay khi bán → ACTIVE luôn.
	res2, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 3, PTPackageID: 1, TrainerID: 7, BranchID: 1, StartDate: "2026-10-20"},
		&ptPay{Amount: 3500000, Method: "CARD"})
	require.NoError(t, err)
	assert.Equal(t, entity.PTContractActive, res2.Status)
	assert.Equal(t, "2026-10-20", res2.StartDate)
	assert.Equal(t, "2026-12-03", res2.EndDate)
}

func TestPTSell_Validation(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	cases := []struct {
		name string
		in   v1.SellInput
		code int
	}{
		{"hội viên không hoạt động", ptSell(2, 1, 6), 40000},
		{"hội viên không tồn tại", ptSell(99, 1, 6), 40400},
		{"gói ngừng bán", ptSell(1, 2, 6), 40000},
		{"gói không tồn tại", ptSell(1, 99, 6), 40400},
		{"HLV không thuộc chi nhánh", ptSell(1, 1, 14), 40000},
		{"không phải HLV", ptSell(1, 1, 99), 40000},
		{"ngày bắt đầu quá khứ", v1.SellInput{MemberID: 1, PTPackageID: 1, TrainerID: 6, BranchID: 1, StartDate: "2026-10-13"}, 40000},
	}
	for _, c := range cases {
		_, err := fx.svc.Sell(ctx, c.in, nil)
		assert.Equal(t, c.code, codeOf(err), c.name)
	}
	// Lễ tân chi nhánh 2 bán tại chi nhánh 1 → 403.
	_, err := fx.svc.Sell(subStaffCtx(), ptSell(1, 1, 6), nil)
	assert.Equal(t, 40300, codeOf(err))
	assert.Empty(t, fx.repo.contracts)
}

func TestPTCancelContract_VoidAndRefund(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	res, err := fx.svc.Sell(ctx, ptSell(1, 1, 6), nil)
	require.NoError(t, err)

	// PENDING, chưa thu → void hóa đơn → VoidHandler "PT" chuyển CANCELLED.
	out, err := fx.svc.CancelContract(ctx, res.ID, v1.CancelContractInput{Reason: "Khách đổi ý"})
	require.NoError(t, err)
	assert.Equal(t, entity.PTContractCancelled, out.Status)
	assert.Equal(t, entity.InvoiceStatusVoid, fx.invRep.invoices[*res.InvoiceID].Status)

	// ACTIVE có buổi đặt → hoàn tiền + hủy buổi.
	id := fx.activeContract(1, 1, 6)
	s, err := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 8, 0)))
	require.NoError(t, err)
	out, err = fx.svc.CancelContract(ctx, id, v1.CancelContractInput{Reason: "Chuyển nhà", Refund: &v1.RefundInput{Amount: 1000000, Method: "CASH"}})
	require.NoError(t, err)
	assert.Equal(t, entity.PTContractCancelled, out.Status)
	assert.Equal(t, 1000000.0, fx.invRep.invoices[*out.InvoiceID].RefundedAmount)
	assert.Equal(t, entity.PTSessionCancelled, fx.repo.sessions[s.ID].Status)
	_, err = fx.svc.CancelContract(ctx, id, v1.CancelContractInput{Reason: "x"})
	assert.Equal(t, 40000, codeOf(err), "đã hủy thì không hủy lại")
}

func TestPTBook_Rules(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 1, 6)

	s, err := fx.svc.CreateSession(ctx, id, "2026-10-15T08:00")
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionScheduled, s.Status)
	assert.Equal(t, ptVN(10, 15, 9, 0), s.EndAt.In(subVNLoc), "end = start + session_minutes")
	assert.Equal(t, entity.PTSourceCounter, s.Source)
	assert.True(t, fx.notifier.notifiedUser(101, NotifyPTScheduled), "hội viên có tài khoản cổng nhận thông báo")
	assert.True(t, fx.notifier.notifiedUser(6, NotifyPTScheduled))

	bad := []struct {
		name  string
		start string
		code  int
	}{
		{"quá khứ", ptRFC(ptVN(10, 14, 8, 0)), 40000},
		{"ngoài giờ mở cửa", ptRFC(ptVN(10, 15, 5, 0)), 40000},
		{"kết thúc sau giờ đóng cửa", ptRFC(ptVN(10, 15, 21, 30)), 40000},
		{"ngoài hạn hợp đồng", ptRFC(ptVN(11, 28, 8, 0)), 40000},
		{"hội viên trùng buổi PT của mình", ptRFC(ptVN(10, 15, 8, 30)), 40900},
		{"giờ không hợp lệ", "abc", 40000},
	}
	for _, c := range bad {
		_, err := fx.svc.CreateSession(ctx, id, c.start)
		assert.Equal(t, c.code, codeOf(err), c.name)
	}

	// HLV trùng buổi PT với hội viên khác → 409.
	other := fx.activeContract(3, 1, 6)
	_, err = fx.svc.CreateSession(ctx, other, ptRFC(ptVN(10, 15, 8, 30)))
	assert.Equal(t, 40900, codeOf(err))
	assert.Contains(t, err.Error(), "HLV")

	// HLV có lịch dạy lớp nhóm → 409.
	fx.repo.classes = append(fx.repo.classes, ptFakeSlot{id: 1, ownerID: 6, start: ptVN(10, 16, 18, 0), end: ptVN(10, 16, 19, 0)})
	_, err = fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 18, 30)))
	assert.Equal(t, 40900, codeOf(err))
	assert.Contains(t, err.Error(), "lớp nhóm")
	// Lớp đã hủy không tính.
	fx.repo.classes[0].cancelled = true
	_, err = fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 18, 30)))
	require.NoError(t, err)

	// Hội viên đã đặt lớp nhóm trùng giờ → 409.
	fx.repo.bookings = append(fx.repo.bookings, ptFakeSlot{ownerID: 1, start: ptVN(10, 17, 7, 0), end: ptVN(10, 17, 8, 0)})
	_, err = fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 17, 7, 30)))
	assert.Equal(t, 40900, codeOf(err))
	assert.Contains(t, err.Error(), "Hội viên")

	// Hợp đồng PENDING → không đặt được.
	pending, err := fx.svc.Sell(ctx, ptSell(3, 1, 7), nil)
	require.NoError(t, err)
	_, err = fx.svc.CreateSession(ctx, pending.ID, ptRFC(ptVN(10, 18, 8, 0)))
	assert.Equal(t, 40000, codeOf(err))

	// EXCLUDE trong DB (đặt đồng thời lọt qua kiểm tra) → 409.
	err = ptMapOverlap(fx.repo.exclusion(&entity.PTSession{TrainerID: 6, StartAt: ptVN(10, 15, 8, 0), EndAt: ptVN(10, 15, 9, 0), Status: entity.PTSessionScheduled}))
	assert.Equal(t, 40900, codeOf(err))
}

func TestPTBook_Quota(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 3, 6) // 2 buổi × 90 phút

	a, err := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 8, 0)))
	require.NoError(t, err)
	assert.Equal(t, ptVN(10, 15, 9, 30), a.EndAt.In(subVNLoc))
	b, err := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 8, 0)))
	require.NoError(t, err)
	_, err = fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 17, 8, 0)))
	assert.Equal(t, 40000, codeOf(err), "đã đặt đủ 2 buổi")
	assert.Contains(t, err.Error(), "hết số buổi")

	// Hủy đúng hạn 1 buổi → trả lại hạn mức.
	_, err = fx.svc.CancelSession(ctx, b.ID, "Bận", false)
	require.NoError(t, err)
	_, err = fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 17, 8, 0)))
	require.NoError(t, err)

	// Hoàn thành 1 buổi: used 1 + scheduled 1 = 2 → vẫn hết hạn mức.
	fx.clock = ptVN(10, 15, 9, 0)
	_, err = fx.svc.CompleteSession(ctx, a.ID, "", "")
	require.NoError(t, err)
	_, err = fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 18, 8, 0)))
	assert.Equal(t, 40000, codeOf(err))

	detail, err := fx.svc.GetContract(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, 1, detail.UsedSessions)
	assert.Equal(t, 1, detail.ScheduledSessions)
	assert.Equal(t, 0, detail.BookableSessions)
	assert.Equal(t, 1, detail.RemainingSessions)
}

func TestPTComplete_CommissionAndUsage(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 3, 7) // 2 buổi, giá 1.000.000, HLV 7 hoa hồng 12,5%
	a, err := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 8, 0)))
	require.NoError(t, err)
	b, err := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 8, 0)))
	require.NoError(t, err)

	// Trước giờ bắt đầu → không cho hoàn thành.
	_, err = fx.svc.CompleteSession(ctx, a.ID, "", "")
	assert.Equal(t, 40000, codeOf(err))

	fx.clock = ptVN(10, 15, 9, 0)
	// HLV khác không được xác nhận buổi của HLV 7.
	_, err = fx.svc.CompleteSession(ptTrainerCtx(6), a.ID, "", "")
	assert.Equal(t, 40300, codeOf(err))
	// HLV phụ trách tự xác nhận.
	done, err := fx.svc.CompleteSession(ptTrainerCtx(7), a.ID, "Squat 5x5", "Tiến bộ tốt")
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionCompleted, done.Status)
	assert.True(t, done.Counted)
	assert.Equal(t, "Squat 5x5", done.WorkoutNote)
	assert.Equal(t, 1, fx.repo.contracts[id].UsedSessions)
	assert.Equal(t, entity.PTContractActive, fx.repo.contracts[id].Status)

	require.Len(t, fx.repo.commissions, 1)
	c := fx.repo.commissions[0]
	assert.Equal(t, uint(7), c.TrainerID)
	assert.Equal(t, entity.CommissionSourcePTSession, c.SourceType)
	assert.Equal(t, a.ID, c.SourceID)
	assert.Equal(t, 500000.0, c.BaseAmount)
	assert.Equal(t, 12.5, c.Rate)
	assert.Equal(t, 62500.0, c.Amount)
	assert.Equal(t, "2026-10", c.Period)
	assert.Equal(t, entity.CommissionPending, c.Status)

	// Hoàn thành lần 2 → 409, không sinh thêm hoa hồng.
	_, err = fx.svc.CompleteSession(ctx, a.ID, "", "")
	assert.Equal(t, 40900, codeOf(err))
	assert.Len(t, fx.repo.commissions, 1)

	// Buổi cuối → hợp đồng COMPLETED.
	fx.clock = ptVN(10, 16, 10, 0)
	_, err = fx.svc.CompleteSession(ctx, b.ID, "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, fx.repo.contracts[id].UsedSessions)
	assert.Equal(t, entity.PTContractCompleted, fx.repo.contracts[id].Status)
	assert.Len(t, fx.repo.commissions, 2)
}

func TestPTCommission_UniqueSourceGuard(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 1, 6)
	s, err := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 8, 0)))
	require.NoError(t, err)
	// Đã có hoa hồng cho buổi này (UNIQUE(source)) → hoàn thành bị chặn 409 và rollback (fake không có rollback,
	// nhưng service phải trả lỗi Conflict).
	fx.repo.commissions = append(fx.repo.commissions, &entity.TrainerCommission{SourceType: entity.CommissionSourcePTSession, SourceID: s.ID, TrainerID: 6})
	fx.clock = ptVN(10, 15, 9, 0)
	_, err = fx.svc.CompleteSession(ctx, s.ID, "", "")
	assert.Equal(t, 40900, codeOf(err))
}

func TestPTCancelSession_OnTimeAndLate(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 1, 6)
	a, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 8, 0)))
	b, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 14, 18, 0)))
	c, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 14, 20, 0)))
	d, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 17, 8, 0)))

	// Hội viên hủy đúng hạn (trước 12 giờ) qua cổng → CANCELLED, không tính.
	res, err := fx.svc.CancelSessionByMember(context.Background(), 1, a.ID, "Ốm")
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionCancelled, res.Status)
	assert.False(t, res.Counted)
	assert.Equal(t, 0, fx.repo.contracts[id].UsedSessions)

	// Hội viên hủy muộn (còn 9 giờ) → bị từ chối.
	_, err = fx.svc.CancelSessionByMember(context.Background(), 1, b.ID, "Bận")
	assert.Equal(t, 40000, codeOf(err))
	// Hội viên khác không thấy buổi.
	_, err = fx.svc.CancelSessionByMember(context.Background(), 3, d.ID, "x")
	assert.Equal(t, 40400, codeOf(err))

	// Lễ tân hủy muộn, không tính buổi.
	res, err = fx.svc.CancelSession(ctx, b.ID, "Khách báo ốm", false)
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionCancelled, res.Status)
	assert.Equal(t, 0, fx.repo.contracts[id].UsedSessions)

	// Lễ tân hủy muộn, tính là đã dùng buổi → NO_SHOW, used +1, không hoa hồng.
	res, err = fx.svc.CancelSession(ctx, c.ID, "Báo hủy sát giờ", true)
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionNoShow, res.Status)
	assert.True(t, res.Counted)
	assert.Equal(t, 1, fx.repo.contracts[id].UsedSessions)
	assert.Empty(t, fx.repo.commissions)

	// count_as_used khi hủy đúng hạn bị bỏ qua.
	res, err = fx.svc.CancelSession(ctx, d.ID, "Đổi lịch", true)
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionCancelled, res.Status)
	assert.Equal(t, 1, fx.repo.contracts[id].UsedSessions)

	// Hủy lại buổi đã hủy → 400.
	_, err = fx.svc.CancelSession(ctx, d.ID, "x", false)
	assert.Equal(t, 40000, codeOf(err))
}

func TestPTNoShowAndCron(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 1, 6)
	a, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 8, 0)))
	b, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 10, 0)))
	c, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 16, 0)))

	// Vắng mặt thủ công: tính buổi, không hoa hồng.
	fx.clock = ptVN(10, 15, 9, 30)
	res, err := fx.svc.MarkNoShow(ctx, a.ID, "")
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionNoShow, res.Status)
	assert.True(t, res.Counted)
	assert.Equal(t, 1, fx.repo.contracts[id].UsedSessions)
	assert.Empty(t, fx.repo.commissions)
	_, err = fx.svc.MarkNoShow(ctx, a.ID, "")
	assert.Equal(t, 40900, codeOf(err))

	// Cron 13:30: buổi 10:00-11:00 quá 2 giờ → NO_SHOW chưa tính; buổi 16:00 chưa tới.
	n, err := fx.svc.MarkPTNoShows(context.Background(), ptVN(10, 15, 13, 30))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	sb := fx.repo.sessions[b.ID]
	assert.Equal(t, entity.PTSessionNoShow, sb.Status)
	assert.False(t, sb.Counted())
	assert.Equal(t, 1, fx.repo.contracts[id].UsedSessions, "cron không trừ buổi")
	assert.Equal(t, entity.PTSessionScheduled, fx.repo.sessions[c.ID].Status)
	assert.True(t, fx.notifier.notifiedUser(6, NotifyPTNoShow))
	// Chạy lại không đánh dấu thêm.
	n, err = fx.svc.MarkPTNoShows(context.Background(), ptVN(10, 15, 13, 30))
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Quản lý xử lý: HLV quên xác nhận → hoàn thành buổi đã bị cron đánh dấu.
	fx.clock = ptVN(10, 15, 14, 0)
	res, err = fx.svc.CompleteSession(ctx, b.ID, "Cardio", "")
	require.NoError(t, err)
	assert.Equal(t, entity.PTSessionCompleted, res.Status)
	assert.Equal(t, "", res.CancelReason)
	assert.Equal(t, 2, fx.repo.contracts[id].UsedSessions)
	assert.Len(t, fx.repo.commissions, 1)
}

func TestPTReschedule(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 1, 6)
	a, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 8, 0)))
	_, _ = fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 8, 0)))

	// Dời sang giờ chồng lên chính nó vẫn được (loại trừ chính buổi đó).
	res, err := fx.svc.RescheduleSession(ctx, a.ID, ptRFC(ptVN(10, 15, 8, 30)))
	require.NoError(t, err)
	assert.Equal(t, ptVN(10, 15, 9, 30), res.EndAt.In(subVNLoc))
	// Trùng buổi khác của hội viên → 409.
	_, err = fx.svc.RescheduleSession(ctx, a.ID, ptRFC(ptVN(10, 16, 8, 30)))
	assert.Equal(t, 40900, codeOf(err))
	// Buổi đã bắt đầu → không đổi được.
	fx.clock = ptVN(10, 15, 8, 45)
	_, err = fx.svc.RescheduleSession(ctx, a.ID, ptRFC(ptVN(10, 17, 8, 0)))
	assert.Equal(t, 40000, codeOf(err))
}

func TestPTChangeTrainer(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 1, 6)
	past, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 14, 10, 0)))
	a, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 8, 0)))
	b, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 8, 0)))
	// HLV 7 dạy lớp trùng buổi b.
	fx.repo.classes = append(fx.repo.classes, ptFakeSlot{id: 5, ownerID: 7, start: ptVN(10, 16, 8, 0), end: ptVN(10, 16, 9, 0)})
	fx.clock = ptVN(10, 14, 11, 0)

	chk, err := fx.svc.CheckChangeTrainer(ctx, id, v1.ChangeTrainerInput{TrainerID: 7})
	require.NoError(t, err)
	assert.False(t, chk.OK)
	assert.Equal(t, 2, chk.MovedSessions)
	require.Len(t, chk.Conflicts, 1)
	assert.Equal(t, b.ID, chk.Conflicts[0].SessionID)

	_, err = fx.svc.ChangeTrainer(ctx, id, v1.ChangeTrainerInput{TrainerID: 7})
	assert.Equal(t, 40900, codeOf(err))
	assert.Contains(t, err.Error(), "#")
	assert.Equal(t, uint(6), fx.repo.contracts[id].TrainerID, "trùng → không đổi gì")

	_, err = fx.svc.ChangeTrainer(ctx, id, v1.ChangeTrainerInput{TrainerID: 6})
	assert.Equal(t, 40000, codeOf(err), "trùng HLV hiện tại")
	_, err = fx.svc.ChangeTrainer(ctx, id, v1.ChangeTrainerInput{TrainerID: 14})
	assert.Equal(t, 40000, codeOf(err), "HLV khác chi nhánh")

	fx.repo.classes = nil
	res, err := fx.svc.ChangeTrainer(ctx, id, v1.ChangeTrainerInput{TrainerID: 7})
	require.NoError(t, err)
	assert.Equal(t, uint(7), res.TrainerID)
	assert.Equal(t, uint(7), fx.repo.sessions[a.ID].TrainerID)
	assert.Equal(t, uint(7), fx.repo.sessions[b.ID].TrainerID)
	assert.Equal(t, uint(6), fx.repo.sessions[past.ID].TrainerID, "buổi đã qua giữ HLV cũ")
}

func TestPTExpireContracts(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 3, 6) // 30 ngày: 14/10 → 12/11
	inside, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(11, 12, 20, 0)))
	// Buổi sau hạn (dữ liệu cũ) → cron hủy.
	fx.repo.sessions[9001] = &entity.PTSession{ID: 9001, ContractID: id, TrainerID: 6, MemberID: 1, BranchID: 1,
		StartAt: ptVN(11, 13, 8, 0), EndAt: ptVN(11, 13, 9, 0), Status: entity.PTSessionScheduled}

	n, err := fx.svc.ExpireContracts(context.Background(), ptVN(11, 12, 23, 0))
	require.NoError(t, err)
	assert.Equal(t, 0, n, "còn trong hạn")

	n, err = fx.svc.ExpireContracts(context.Background(), ptVN(11, 13, 0, 5))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, entity.PTContractExpired, fx.repo.contracts[id].Status)
	assert.Equal(t, entity.PTSessionCancelled, fx.repo.sessions[9001].Status)
	assert.Equal(t, entity.PTSessionScheduled, fx.repo.sessions[inside.ID].Status, "buổi trong hạn giữ nguyên để xác nhận")

	// Hợp đồng hết hạn vẫn xác nhận được buổi trong hạn.
	fx.clock = ptVN(11, 13, 9, 0)
	_, err = fx.svc.CompleteSession(ctx, inside.ID, "", "")
	require.NoError(t, err)
	assert.Equal(t, 1, fx.repo.contracts[id].UsedSessions)
}

func TestPTTrainerOnlyVisibility(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	mine := fx.activeContract(1, 1, 6)
	other := fx.activeContract(3, 1, 7)
	s7, _ := fx.svc.CreateSession(ctx, other, ptRFC(ptVN(10, 15, 8, 0)))
	_, _ = fx.svc.CreateSession(ctx, mine, ptRFC(ptVN(10, 15, 10, 0)))

	tctx := ptTrainerCtx(6)
	list, err := fx.svc.ListContracts(tctx, &v1.ContractListReq{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, mine, list.Items[0].ID)
	_, err = fx.svc.ListContracts(tctx, &v1.ContractListReq{TrainerID: 7})
	assert.Equal(t, 40300, codeOf(err))
	_, err = fx.svc.GetContract(tctx, other)
	assert.Equal(t, 40300, codeOf(err))
	_, err = fx.svc.GetSession(tctx, s7.ID)
	assert.Equal(t, 40300, codeOf(err))

	sessions, err := fx.svc.ListSessions(tctx, &v1.SessionListReq{From: "2026-10-14", To: "2026-10-20"})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, uint(6), sessions[0].TrainerID)

	// Người có vai trò khác (quản lý) thấy tất cả.
	all, err := fx.svc.ListSessions(ctx, &v1.SessionListReq{From: "2026-10-14", To: "2026-10-20"})
	require.NoError(t, err)
	assert.Len(t, all, 2)
	_, err = fx.svc.ListSessions(ctx, &v1.SessionListReq{From: "2026-10-01", To: "2026-12-31"})
	assert.Equal(t, 40000, codeOf(err), "quá 62 ngày")

	// Lịch dạy: gộp PT + lớp nhóm, sắp theo giờ.
	fx.repo.classes = append(fx.repo.classes, ptFakeSlot{id: 3, ownerID: 6, start: ptVN(10, 15, 7, 0), end: ptVN(10, 15, 8, 0)})
	items, err := fx.svc.TrainerScheduleAdmin(tctx, &v1.TrainerScheduleReq{From: "2026-10-14", To: "2026-10-20"})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "CLASS", items[0].Type)
	assert.Equal(t, "PT", items[1].Type)
	_, err = fx.svc.TrainerScheduleAdmin(tctx, &v1.TrainerScheduleReq{TrainerID: 7, From: "2026-10-14", To: "2026-10-20"})
	assert.Equal(t, 40300, codeOf(err))
}

func TestPTMemberPortal(t *testing.T) {
	fx := newPTFixture()
	id := fx.activeContract(1, 1, 6)
	other := fx.activeContract(3, 1, 7)

	contracts, err := fx.svc.ListMemberContracts(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, contracts, 1)
	assert.Equal(t, id, contracts[0].ID)

	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 101, Roles: []string{consts.RoleMember}})
	s, err := fx.svc.BookSessionByMember(memberCtx, 1, id, ptVN(10, 20, 18, 0))
	require.NoError(t, err)
	assert.Equal(t, entity.PTSourcePortal, s.Source)
	assert.False(t, fx.notifier.notifiedUser(101, NotifyPTScheduled), "không tự thông báo cho chính người đặt")
	assert.True(t, fx.notifier.notifiedUser(6, NotifyPTScheduled))
	_, err = fx.svc.BookSessionByMember(memberCtx, 1, other, ptVN(10, 21, 18, 0))
	assert.Equal(t, 40400, codeOf(err), "hợp đồng của người khác")

	sessions, err := fx.svc.ListMemberSessions(context.Background(), 1, ptVN(10, 14, 0, 0), ptVN(10, 28, 0, 0))
	require.NoError(t, err)
	require.Len(t, sessions, 1)
}

func TestPTCommissionPayAndSummary(t *testing.T) {
	fx := newPTFixture()
	ctx := subOwnerCtx()
	id := fx.activeContract(1, 1, 6)
	a, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 15, 8, 0)))
	b, _ := fx.svc.CreateSession(ctx, id, ptRFC(ptVN(10, 16, 8, 0)))
	fx.clock = ptVN(10, 17, 8, 0)
	_, err := fx.svc.CompleteSession(ctx, a.ID, "", "")
	require.NoError(t, err)
	_, err = fx.svc.CompleteSession(ctx, b.ID, "", "")
	require.NoError(t, err)

	sum, err := fx.svc.CommissionSummary(ctx, &v1.CommissionSummaryReq{Period: "2026-10"})
	require.NoError(t, err)
	require.Len(t, sum.Items, 1)
	assert.Equal(t, 2, sum.Items[0].Sessions)
	assert.Equal(t, 70000.0, sum.Items[0].TotalAmount)
	assert.Equal(t, 70000.0, sum.Items[0].PendingAmount)

	// HLV không có quyền xem của người khác.
	_, err = fx.svc.ListCommissions(ptTrainerCtx(7), &v1.CommissionListReq{TrainerID: 6})
	assert.Equal(t, 40300, codeOf(err))
	mine, err := fx.svc.ListCommissions(ptTrainerCtx(7), &v1.CommissionListReq{})
	require.NoError(t, err)
	assert.Empty(t, mine.Items)

	paid, err := fx.svc.PayCommissions(ctx, &v1.CommissionPayReq{TrainerID: 6, Period: "2026-10"})
	require.NoError(t, err)
	assert.Equal(t, 2, paid.Count)
	assert.Equal(t, 70000.0, paid.Amount)
	assert.True(t, fx.notifier.notifiedUser(6, NotifyCommissionPaid))
	for _, c := range fx.repo.commissions {
		assert.Equal(t, entity.CommissionPaid, c.Status)
		require.NotNil(t, c.PaidBy)
	}
	_, err = fx.svc.PayCommissions(ctx, &v1.CommissionPayReq{TrainerID: 6, Period: "2026-10"})
	assert.Equal(t, 40000, codeOf(err), "không còn gì để chi")
	sum, _ = fx.svc.CommissionSummary(ctx, &v1.CommissionSummaryReq{Period: "2026-10"})
	assert.Equal(t, 70000.0, sum.PaidAmount)
	assert.Equal(t, 0.0, sum.PendingAmount)
	_, err = fx.svc.CommissionSummary(ctx, &v1.CommissionSummaryReq{Period: "2026-13"})
	assert.Equal(t, 40000, codeOf(err))
}

// Đảm bảo lỗi nghiệp vụ dùng đúng gói rescode (tránh lỗi nội bộ lọt ra client).
func TestPTErrorsAreBusinessErrors(t *testing.T) {
	fx := newPTFixture()
	_, err := fx.svc.GetContract(subOwnerCtx(), 12345)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
	_, err = fx.svc.GetSession(subOwnerCtx(), 12345)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}
