package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/entity"
)

func schedFixture(invID uint, amounts ...float64) []entity.InstallmentSchedule {
	out := make([]entity.InstallmentSchedule, len(amounts))
	for i, a := range amounts {
		out[i] = entity.InstallmentSchedule{ID: invID*100 + uint(i) + 1, PlanID: invID, Seq: i + 1, Amount: a, Status: entity.InstallmentPending}
	}
	return out
}

func TestInvoiceResyncInstallments(t *testing.T) {
	now := time.Now()
	// Kỳ 1 đã thu 1tr; hóa đơn 3tr đã thu 1tr + thu trực tiếp 500k → còn nợ 1.5tr chia lại cho 2 kỳ còn mở.
	s := schedFixture(1, 1000000, 1000000, 1000000)
	s[0].PaidAmount, s[0].Status = 1000000, entity.InstallmentPaid
	inv := &entity.Invoice{Total: 3000000, PaidAmount: 1500000, Status: entity.InvoiceStatusPartial}
	ch := invoiceResyncInstallments(s, inv, installmentSettled, now)
	require.Len(t, ch, 2)
	assert.Equal(t, 750000.0, ch[0].Amount)
	assert.Equal(t, 750000.0, ch[1].Amount)

	// Đã khớp → không đổi.
	inv.PaidAmount = 1000000
	assert.Empty(t, invoiceResyncInstallments(s, inv, installmentSettled, now))

	// Thu đủ trực tiếp → kỳ còn mở SETTLED; xóa nợ → CANCELLED; hóa đơn VOID/REFUNDED → CANCELLED.
	inv.PaidAmount, inv.Status = 3000000, entity.InvoiceStatusPaid
	for _, c := range invoiceResyncInstallments(s, inv, installmentSettled, now) {
		assert.Equal(t, installmentSettled, c.Status)
	}
	for _, c := range invoiceResyncInstallments(s, inv, installmentCancelled, now) {
		assert.Equal(t, installmentCancelled, c.Status)
	}
	inv.Status = entity.InvoiceStatusRefunded
	for _, c := range invoiceResyncInstallments(s, inv, installmentSettled, now) {
		assert.Equal(t, installmentCancelled, c.Status)
	}
	// Còn nợ ít hơn số kỳ (1đ cho 2 kỳ) → kỳ được chia 0đ SETTLED.
	inv = &entity.Invoice{Total: 3000000, PaidAmount: 2999999, Status: entity.InvoiceStatusPartial}
	ch = invoiceResyncInstallments(s, inv, installmentSettled, now)
	require.Len(t, ch, 2)
	assert.Equal(t, installmentSettled, ch[0].Status)
	assert.Equal(t, 1.0, ch[1].Amount)
}

// Thu trực tiếp / hủy / xóa nợ / hoàn tiền trên hóa đơn có kế hoạch trả góp → các kỳ được đồng bộ.
func TestInvoiceFlowsSyncInstallments(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	sub, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 3, BranchID: 1}, nil) // 4.8tr
	require.NoError(t, err)
	invID := *sub.InvoiceID
	fx.invRep.schedules[invID] = schedFixture(invID, 1600000, 1600000, 1600000)

	_, err = fx.inv.AddPayment(ctx, invID, PaymentInput{Amount: 1000000, Method: "CASH"})
	require.NoError(t, err)
	sc := fx.invRep.schedules[invID]
	assert.Equal(t, 3800000.0, sc[0].Amount+sc[1].Amount+sc[2].Amount, "chia lại số còn nợ")

	_, err = fx.inv.AddPayment(ctx, invID, PaymentInput{Amount: 3800000, Method: "TRANSFER"})
	require.NoError(t, err)
	for _, s := range fx.invRep.schedules[invID] {
		assert.Equal(t, installmentSettled, s.Status)
	}

	// Hủy gói đã thu một phần → xóa nợ ghi riêng, kỳ chưa thu bị hủy.
	sub2, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 2, PlanID: 2, BranchID: 1}, &v1.PaymentInput{Amount: 350000, Method: "CASH"})
	require.NoError(t, err)
	inv2 := *sub2.InvoiceID
	fx.invRep.schedules[inv2] = schedFixture(inv2, 500000, 500000)
	_, err = fx.svc.Cancel(ctx, sub2.ID, v1.CancelInput{Reason: "Chuyển nhà"})
	require.NoError(t, err)
	i2 := fx.invRep.invoices[inv2]
	assert.Equal(t, 1000000.0, i2.WriteOffAmount)
	assert.Equal(t, 0.0, i2.Discount, "xóa nợ không tính là giảm giá")
	assert.Equal(t, 350000.0, i2.Total)
	assert.Contains(t, i2.WriteOffReason, "Chuyển nhà")
	for _, s := range fx.invRep.schedules[inv2] {
		assert.Equal(t, installmentCancelled, s.Status)
	}

	// Hóa đơn chưa thu bị hủy → kỳ CANCELLED.
	sub3, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 2, PlanID: 1, BranchID: 2}, nil)
	require.NoError(t, err)
	inv3 := *sub3.InvoiceID
	fx.invRep.schedules[inv3] = schedFixture(inv3, 250000, 250000)
	require.NoError(t, fx.inv.Void(ctx, inv3, "Nhập sai"))
	for _, s := range fx.invRep.schedules[inv3] {
		assert.Equal(t, installmentCancelled, s.Status)
	}
}

// Nâng cấp sau chuyển nhượng: gốc khấu trừ là giá trị đã trả (sau hoàn) mang theo từ gói gốc,
// không phải giá niêm yết và không phải hóa đơn phí chuyển nhượng.
func TestSubUpgradeAfterTransferUsesCarriedPaidBase(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	sub, err := fx.svc.Sell(ctx, v1.SellInput{MemberID: 1, PlanID: 1, BranchID: 1}, &v1.PaymentInput{Amount: 500000, Method: "CASH"})
	require.NoError(t, err)
	require.Equal(t, entity.SubStatusActive, sub.Status)
	_, err = fx.inv.Refund(ctx, *sub.InvoiceID, RefundInput{Amount: 200000, Method: "CASH", Reason: "Giảm giá bù"})
	require.NoError(t, err)

	got, err := fx.svc.Transfer(ctx, sub.ID, v1.TransferInput{ToMemberID: 2})
	require.NoError(t, err)
	nsub := fx.repo.subs[got.ID]
	require.NotNil(t, nsub.CarriedPaidBase)
	assert.Equal(t, 300000.0, *nsub.CarriedPaidBase)
	_, err = fx.inv.AddPayment(ctx, *got.InvoiceID, PaymentInput{Amount: 100000, Method: "CASH"})
	require.NoError(t, err)

	q, err := fx.svc.UpgradeQuote(ctx, got.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, 300000.0, q.RemainingValue)
	assert.Equal(t, 1050000.0, q.AmountDue)

	// Hoàn phí chuyển nhượng không làm giảm gốc khấu trừ.
	_, err = fx.inv.Refund(ctx, *got.InvoiceID, RefundInput{Amount: 50000, Method: "CASH", Reason: "x"})
	require.NoError(t, err)
	q, err = fx.svc.UpgradeQuote(ctx, got.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, 300000.0, q.RemainingValue)
}
