package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

func TestNumberToVietnameseWords(t *testing.T) {
	cases := map[int64]string{
		0:             "không",
		5:             "năm",
		10:            "mười",
		15:            "mười lăm",
		21:            "hai mươi mốt",
		24:            "hai mươi tư",
		25:            "hai mươi lăm",
		105:           "một trăm linh năm",
		110:           "một trăm mười",
		1000:          "một nghìn",
		1001:          "một nghìn không trăm linh một",
		100000:        "một trăm nghìn",
		500000:        "năm trăm nghìn",
		1050000:       "một triệu không trăm năm mươi nghìn",
		1215000:       "một triệu hai trăm mười lăm nghìn",
		1350000:       "một triệu ba trăm năm mươi nghìn",
		4800000:       "bốn triệu tám trăm nghìn",
		1000001:       "một triệu không trăm linh một",
		2000000000:    "hai tỷ",
		1500000000000: "một nghìn năm trăm tỷ",
	}
	for n, want := range cases {
		assert.Equal(t, want, NumberToVietnameseWords(n), "n=%d", n)
	}
	assert.Equal(t, "Một triệu hai trăm mười lăm nghìn đồng", VNDInWords(1215000))
	assert.Equal(t, "Không đồng", VNDInWords(0))
}

func TestInvoiceAllocateDiscount(t *testing.T) {
	items := []entity.InvoiceItem{{Amount: 1000}, {Amount: 2000}, {Amount: 3001}, {Amount: 0}}
	invoiceAllocateDiscount(items, 1001)
	var sum float64
	for _, it := range items {
		assert.GreaterOrEqual(t, it.Amount, 0.0)
		sum += it.Discount
	}
	assert.Equal(t, 1001.0, sum)
	assert.Equal(t, 0.0, items[3].Discount)

	// Giảm nhiều hơn tổng → chỉ giảm tối đa bằng tổng.
	items = []entity.InvoiceItem{{Amount: 100}, {Amount: 50}}
	invoiceAllocateDiscount(items, 1000)
	assert.Equal(t, 0.0, items[0].Amount+items[1].Amount)
}

func TestInvoiceBuildItems(t *testing.T) {
	items, subtotal, disc, err := invoiceBuildItems([]InvoiceItemDraft{
		{ItemType: entity.InvoiceItemProduct, Description: "Nước", Quantity: 3, UnitPrice: 10000.4},
		{ItemType: entity.InvoiceItemMembership, Description: "Gói", UnitPrice: 500000, Discount: 600000},
	})
	require.NoError(t, err)
	assert.Equal(t, 530000.0, subtotal)
	assert.Equal(t, 500000.0, disc, "giảm theo dòng không vượt thành tiền")
	assert.Equal(t, 30000.0, items[0].Amount)
	assert.Equal(t, 1, items[1].Quantity)
	assert.Equal(t, 0.0, items[1].Amount)

	_, _, _, err = invoiceBuildItems(nil)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, _, _, err = invoiceBuildItems([]InvoiceItemDraft{{ItemType: "XYZ", Description: "a"}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestInvoice_PromotionAppliedOnCreate(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	member := uint(1)
	draft := InvoiceDraft{MemberID: &member, BranchID: 1, PromotionCode: "chaodong", AppliesTo: entity.AppliesMembership,
		Items: []InvoiceItemDraft{
			{ItemType: entity.InvoiceItemMembership, Description: "M3", UnitPrice: 1350000},
			{ItemType: entity.InvoiceItemOther, Description: "Khăn", UnitPrice: 150000},
		}}

	pv, err := fx.inv.Preview(ctx, draft)
	require.NoError(t, err)
	assert.Equal(t, 1500000.0, pv.Subtotal)
	assert.Equal(t, 150000.0, pv.Discount)
	assert.Equal(t, 0, fx.promos.items[1].UsedCount, "preview không tiêu thụ lượt")

	inv, err := fx.inv.CreateInvoice(ctx, draft)
	require.NoError(t, err)
	assert.Equal(t, 1500000.0, inv.Subtotal)
	assert.Equal(t, 150000.0, inv.Discount)
	assert.Equal(t, 1350000.0, inv.Total)
	assert.Equal(t, entity.InvoiceStatusUnpaid, inv.Status)
	require.NotNil(t, inv.PromotionID)
	assert.Equal(t, 1, fx.promos.items[1].UsedCount)
	assert.Equal(t, 135000.0, inv.Items[0].Discount)
	assert.Equal(t, 15000.0, inv.Items[1].Discount)
	assert.Equal(t, inv.Total, inv.Items[0].Amount+inv.Items[1].Amount)

	// Mã không tồn tại / thiếu loại đơn.
	_, err = fx.inv.CreateInvoice(ctx, InvoiceDraft{MemberID: &member, BranchID: 1, PromotionCode: "KHONGCO", AppliesTo: entity.AppliesMembership, Items: draft.Items})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = fx.inv.CreateInvoice(ctx, InvoiceDraft{MemberID: &member, BranchID: 1, PromotionCode: "CHAODONG", Items: draft.Items})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	// Hóa đơn ngoài phạm vi chi nhánh.
	_, err = fx.inv.CreateInvoice(subStaffCtx(), InvoiceDraft{MemberID: &member, BranchID: 1, Items: draft.Items})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}

func TestInvoice_PaymentsNoOverpayAndPaidHandler(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	var paidCalls int
	fx.inv.RegisterPaidHandler(entity.InvoiceItemPT, func(_ context.Context, inv *entity.Invoice, it *entity.InvoiceItem) error {
		paidCalls++
		assert.Equal(t, entity.InvoiceStatusPaid, inv.Status)
		return nil
	})
	member := uint(2)
	inv, err := fx.inv.CreateInvoice(ctx, InvoiceDraft{MemberID: &member, BranchID: 1,
		Items: []InvoiceItemDraft{{ItemType: entity.InvoiceItemPT, Description: "PT 10", UnitPrice: 3500000}}})
	require.NoError(t, err)

	_, err = fx.inv.AddPayment(ctx, inv.ID, PaymentInput{Amount: 0, Method: entity.PaymentCash})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = fx.inv.AddPayment(ctx, inv.ID, PaymentInput{Amount: 100, Method: "BITCOIN"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	got, err := fx.inv.AddPayment(ctx, inv.ID, PaymentInput{Amount: 1000000, Method: entity.PaymentCash})
	require.NoError(t, err)
	assert.Equal(t, entity.InvoiceStatusPartial, got.Status)
	assert.Equal(t, 0, paidCalls)

	_, err = fx.inv.AddPayment(ctx, inv.ID, PaymentInput{Amount: 2500001, Method: entity.PaymentCash})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2.500.000")

	got, err = fx.inv.AddPayment(ctx, inv.ID, PaymentInput{Amount: 2500000, Method: entity.PaymentTransfer, Reference: "FT1"})
	require.NoError(t, err)
	assert.Equal(t, entity.InvoiceStatusPaid, got.Status)
	assert.Equal(t, 3500000.0, fx.invRep.invoices[inv.ID].PaidAmount)
	assert.Equal(t, 1, paidCalls)
	assert.Len(t, fx.invRep.payments[inv.ID], 2)

	_, err = fx.inv.AddPayment(ctx, inv.ID, PaymentInput{Amount: 1, Method: entity.PaymentCash})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Nhân viên chi nhánh khác không thu được.
	inv2, err := fx.inv.CreateInvoice(ctx, InvoiceDraft{MemberID: &member, BranchID: 1,
		Items: []InvoiceItemDraft{{ItemType: entity.InvoiceItemOther, Description: "x", UnitPrice: 1000}}})
	require.NoError(t, err)
	_, err = fx.inv.AddPayment(subStaffCtx(), inv2.ID, PaymentInput{Amount: 1000, Method: entity.PaymentCash})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}

func TestInvoice_ZeroTotalIsPaidImmediately(t *testing.T) {
	fx := newSubFixture()
	called := false
	fx.inv.RegisterPaidHandler(entity.InvoiceItemOther, func(context.Context, *entity.Invoice, *entity.InvoiceItem) error {
		called = true
		return nil
	})
	member := uint(1)
	inv, err := fx.inv.CreateInvoice(subOwnerCtx(), InvoiceDraft{MemberID: &member, BranchID: 1,
		Items: []InvoiceItemDraft{{ItemType: entity.InvoiceItemOther, Description: "Quà tặng", UnitPrice: 0}}})
	require.NoError(t, err)
	assert.Equal(t, entity.InvoiceStatusPaid, inv.Status)
	assert.True(t, called)
}

func TestInvoice_VoidOnlyWhenUnpaid(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	var voided []uint
	fx.inv.RegisterVoidHandler(entity.InvoiceItemOther, func(_ context.Context, inv *entity.Invoice, _ *entity.InvoiceItem) error {
		voided = append(voided, inv.ID)
		return nil
	})
	member := uint(1)
	mk := func(promo string) *entity.Invoice {
		inv, err := fx.inv.CreateInvoice(ctx, InvoiceDraft{MemberID: &member, BranchID: 1, PromotionCode: promo, AppliesTo: entity.AppliesMembership,
			Items: []InvoiceItemDraft{{ItemType: entity.InvoiceItemOther, Description: "x", UnitPrice: 600000}}})
		require.NoError(t, err)
		return inv
	}
	a := mk("CHAODONG")
	assert.Equal(t, 1, fx.promos.items[1].UsedCount)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(fx.inv.Void(ctx, a.ID, " ")))
	require.NoError(t, fx.inv.Void(ctx, a.ID, "Khách không mua nữa"))
	assert.Equal(t, entity.InvoiceStatusVoid, fx.invRep.invoices[a.ID].Status)
	assert.Equal(t, 0, fx.promos.items[1].UsedCount, "trả lại lượt khuyến mãi")
	assert.Equal(t, []uint{a.ID}, voided)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(fx.inv.Void(ctx, a.ID, "lần 2")))
	_, err := fx.inv.AddPayment(ctx, a.ID, PaymentInput{Amount: 1, Method: entity.PaymentCash})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	b := mk("")
	_, err = fx.inv.AddPayment(ctx, b.ID, PaymentInput{Amount: 1000, Method: entity.PaymentCash})
	require.NoError(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(fx.inv.Void(ctx, b.ID, "đã thu")))
	assert.Equal(t, entity.InvoiceStatusPartial, fx.invRep.invoices[b.ID].Status)
}

func TestInvoice_Refund(t *testing.T) {
	fx := newSubFixture()
	ctx := subOwnerCtx()
	member := uint(1)
	inv, err := fx.inv.CreateInvoice(ctx, InvoiceDraft{MemberID: &member, BranchID: 1,
		Items: []InvoiceItemDraft{{ItemType: entity.InvoiceItemOther, Description: "x", UnitPrice: 1350000}}})
	require.NoError(t, err)

	_, err = fx.inv.Refund(ctx, inv.ID, RefundInput{Amount: 1, Method: entity.PaymentCash, Reason: "r"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "chưa thu thì không hoàn")

	_, err = fx.inv.AddPayment(ctx, inv.ID, PaymentInput{Amount: 1350000, Method: entity.PaymentCash})
	require.NoError(t, err)

	_, err = fx.inv.Refund(ctx, inv.ID, RefundInput{Amount: 1500000, Method: entity.PaymentCash, Reason: "r"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	_, err = fx.inv.Refund(ctx, inv.ID, RefundInput{Amount: 100, Method: entity.PaymentCash, Reason: ""})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	got, err := fx.inv.Refund(ctx, inv.ID, RefundInput{Amount: 350000, Method: entity.PaymentCash, Reason: "Hoàn một phần"})
	require.NoError(t, err)
	assert.Equal(t, entity.InvoiceStatusPaid, got.Status)
	assert.Equal(t, 350000.0, got.RefundedAmount)

	got, err = fx.inv.Refund(ctx, inv.ID, RefundInput{Amount: 1000000, Method: entity.PaymentTransfer, Reason: "Hoàn nốt"})
	require.NoError(t, err)
	assert.Equal(t, entity.InvoiceStatusRefunded, got.Status)
	assert.Len(t, fx.invRep.refunds[inv.ID], 2)

	_, err = fx.inv.Refund(ctx, inv.ID, RefundInput{Amount: 1, Method: entity.PaymentCash, Reason: "thêm"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestInvoiceSummarize(t *testing.T) {
	s := invoiceSummarize([]repository.PaymentMethodTotal{{Method: "CASH", Amount: 100, Count: 2}, {Method: "CARD", Amount: 50, Count: 1}})
	assert.Equal(t, 150.0, s.TotalAmount)
	assert.Equal(t, int64(3), s.Count)
	require.Len(t, s.ByMethod, 3)
	assert.Equal(t, "TRANSFER", s.ByMethod[1].Method)
	assert.Equal(t, 0.0, s.ByMethod[1].Amount)
}
