package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/pos/v1"
	"gymmaster-api/internal/entity"
)

func posOrder(branch uint, member *uint, items [][2]int, pays ...*v1.OrderPayment) *v1.CreateOrderReq {
	req := &v1.CreateOrderReq{BranchID: branch, MemberID: member, Payments: pays}
	for _, it := range items {
		req.Items = append(req.Items, &v1.OrderItem{ProductID: uint(it[0]), Quantity: it[1]})
	}
	return req
}

func posCash(v float64) *v1.OrderPayment {
	return &v1.OrderPayment{Amount: v, Method: entity.PaymentCash}
}
func posTransfer(v float64) *v1.OrderPayment {
	return &v1.OrderPayment{Amount: v, Method: entity.PaymentTransfer}
}
func posUptr(v uint) *uint { return &v }

func TestPosUnitPrice(t *testing.T) {
	withMember := &entity.Product{SellPrice: 10000, MemberPrice: posPtrF(8000)}
	noMember := &entity.Product{SellPrice: 10000}
	assert.Equal(t, 8000.0, posUnitPrice(withMember, true))
	assert.Equal(t, 10000.0, posUnitPrice(withMember, false))
	assert.Equal(t, 10000.0, posUnitPrice(noMember, true), "không có giá hội viên → giá bán")
}

func TestPosCreateOrder_MemberPrice(t *testing.T) {
	ctx := subOwnerCtx()
	cases := []struct {
		name   string
		member *uint
		unit   float64
		total  float64
	}{
		{"hội viên ACTIVE → giá hội viên", posUptr(1), 8000, 2*8000 + 1950000},
		{"hội viên không ACTIVE → giá bán", posUptr(2), 10000, 2*10000 + 1950000},
		{"khách vãng lai → giá bán", nil, 10000, 2*10000 + 1950000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fx := newPosFixture()
			res, err := fx.svc.CreateOrder(ctx, posOrder(1, c.member, [][2]int{{1, 2}, {2, 1}}, posCash(c.total)))
			require.NoError(t, err)
			assert.Equal(t, c.unit, res.Invoice.Items[0].UnitPrice)
			assert.Equal(t, 1950000.0, res.Invoice.Items[1].UnitPrice, "sản phẩm không có giá hội viên → giá bán")
			assert.Equal(t, c.total, res.Invoice.Total)
			assert.Equal(t, entity.InvoiceStatusPaid, res.Invoice.Status)
			assert.Equal(t, entity.InvoiceSourcePOS, res.Invoice.Source)
			assert.Equal(t, 0.0, res.ChangeAmount)
			assert.Equal(t, 18, fx.stock.levels[stockKey{1, 1}])
			assert.Equal(t, 2, fx.stock.levels[stockKey{2, 1}])
			sales := fx.stock.txsOf(entity.StockSale)
			require.Len(t, sales, 2)
			assert.Equal(t, -2, sales[0].Quantity)
			assert.Equal(t, 18, sales[0].BalanceAfter)
			assert.Equal(t, 5000.0, sales[0].UnitCost, "dòng SALE ghi giá vốn hiện tại")
			assert.Equal(t, res.Invoice.InvoiceNo, sales[0].VoucherNo)
		})
	}
}

func TestPosCreateOrder_PaymentMustBeFull(t *testing.T) {
	fx := newPosFixture()
	_, err := fx.svc.CreateOrder(subOwnerCtx(), posOrder(1, nil, [][2]int{{1, 3}}, posCash(10000), posTransfer(10000)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Khách thanh toán chưa đủ")
	assert.Contains(t, err.Error(), "còn thiếu 10.000 đ")
	assert.Equal(t, 20, fx.stock.levels[stockKey{1, 1}], "không trừ kho khi thanh toán thiếu")
	assert.Empty(t, fx.inv.invoices, "không lập hóa đơn khi thanh toán thiếu")

	_, err = fx.svc.CreateOrder(subOwnerCtx(), posOrder(1, nil, [][2]int{{1, 3}}))
	require.Error(t, err, "không có thanh toán")
}

func TestPosCreateOrder_ChangeAmount(t *testing.T) {
	fx := newPosFixture()
	// Tổng 30.000: chuyển khoản 10.000 + tiền mặt 50.000 → thừa 30.000, chỉ ghi nhận 20.000 tiền mặt.
	res, err := fx.svc.CreateOrder(subOwnerCtx(), posOrder(1, nil, [][2]int{{1, 3}}, posTransfer(10000), posCash(50000)))
	require.NoError(t, err)
	assert.Equal(t, 60000.0, res.ReceivedAmount)
	assert.Equal(t, 30000.0, res.ChangeAmount)
	assert.Equal(t, 30000.0, res.Invoice.PaidAmount)
	require.Len(t, res.Invoice.Payments, 2)
	assert.Equal(t, 10000.0, res.Invoice.Payments[0].Amount)
	assert.Equal(t, entity.PaymentTransfer, res.Invoice.Payments[0].Method)
	assert.Equal(t, 20000.0, res.Invoice.Payments[1].Amount)
	assert.NotNil(t, res.Print)
	assert.Equal(t, "Ba mươi nghìn đồng", res.Print.TotalInWords)

	// Chuyển khoản dư → từ chối (chỉ tiền mặt được trả lại tiền thừa).
	_, err = fx.svc.CreateOrder(subOwnerCtx(), posOrder(1, nil, [][2]int{{1, 1}}, posTransfer(20000)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Chỉ tiền mặt mới được đưa thừa")
}

func TestPosPlanPayments(t *testing.T) {
	pays, rcv, chg, err := posPlanPayments(100000, []*v1.OrderPayment{posCash(50000), posTransfer(30000), posCash(50000)})
	require.NoError(t, err)
	assert.Equal(t, 130000.0, rcv)
	assert.Equal(t, 30000.0, chg)
	var sum float64
	for _, p := range pays {
		sum += p.Amount
	}
	assert.Equal(t, 100000.0, sum, "Σ ghi nhận = total")
	assert.Equal(t, 20000.0, pays[2].Amount, "giảm tiền mặt từ khoản cuối")

	pays, _, chg, err = posPlanPayments(0, []*v1.OrderPayment{posCash(5000)})
	require.NoError(t, err)
	assert.Empty(t, pays)
	assert.Equal(t, 5000.0, chg)
}

func TestPosCreateOrder_Shortage(t *testing.T) {
	fx := newPosFixture()
	_, err := fx.svc.CreateOrder(subOwnerCtx(), posOrder(1, nil, [][2]int{{2, 2}, {2, 2}}, posCash(10000000)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Whey ON Gold Standard 5lbs")
	assert.Contains(t, err.Error(), "còn 3 hộp, cần 4", "dòng trùng sản phẩm được gộp")
	assert.Equal(t, 3, fx.stock.levels[stockKey{2, 1}])
	assert.Empty(t, fx.inv.invoices)

	_, err = fx.svc.CreateOrder(subOwnerCtx(), posOrder(1, nil, [][2]int{{3, 1}}, posCash(1000)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ngừng kinh doanh")
}

func TestPosCreateOrder_LocksInOrder(t *testing.T) {
	fx := newPosFixture()
	_, err := fx.svc.CreateOrder(subOwnerCtx(), posOrder(1, nil, [][2]int{{2, 1}, {1, 1}}, posCash(1960000)))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(fx.stock.lockLog), 2)
	assert.Equal(t, []stockKey{{1, 1}, {2, 1}}, fx.stock.lockLog[:2], "khóa theo product_id tăng dần")
}

func TestPosReturn_NotExceedSold(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	// 3 chai × 10.000 = 30.000
	res, err := fx.svc.CreateOrder(ctx, posOrder(1, nil, [][2]int{{1, 3}}, posCash(30000)))
	require.NoError(t, err)
	id := res.Invoice.ID

	ret := func(qty int) (*v1.ReturnRes, error) {
		return fx.svc.Return(ctx, &v1.ReturnReq{InvoiceID: id, RefundMethod: entity.PaymentCash, Reason: "Khách đổi ý",
			Items: []*v1.ReturnItem{{ProductID: 1, Quantity: qty}}})
	}
	_, err = ret(4)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chỉ còn có thể trả 3")

	r1, err := ret(2)
	require.NoError(t, err)
	assert.Equal(t, 20000.0, r1.RefundAmount)
	assert.Equal(t, 19, fx.stock.levels[stockKey{1, 1}], "17 + 2 hoàn kho")
	assert.Equal(t, 1, r1.Lines[0].ReturnableQty)
	assert.Equal(t, 20000.0, r1.Invoice.RefundedAmount)

	_, err = ret(2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chỉ còn có thể trả 1 (đã bán 3, đã trả 2)")

	r2, err := ret(1)
	require.NoError(t, err)
	assert.Equal(t, 10000.0, r2.RefundAmount)
	assert.Equal(t, entity.InvoiceStatusRefunded, r2.Invoice.Status)
	returns := fx.stock.txsOf(entity.StockSaleReturn)
	require.Len(t, returns, 2)
	assert.Equal(t, 5000.0, returns[0].UnitCost, "hoàn kho theo giá vốn lúc bán")

	_, err = fx.svc.Return(ctx, &v1.ReturnReq{InvoiceID: id, RefundMethod: entity.PaymentCash, Reason: "x",
		Items: []*v1.ReturnItem{{ProductID: 2, Quantity: 1}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không có trong hóa đơn")
}

func TestPosReturnRefund_Cumulative(t *testing.T) {
	// 3 sản phẩm thành tiền 100.000 (sau khuyến mãi): trả lần lượt 1+1+1 → tổng hoàn đúng 100.000.
	var sum float64
	for i := 0; i < 3; i++ {
		sum += posReturnRefund(100000, 3, i, 1)
	}
	assert.Equal(t, 100000.0, sum)
	assert.Equal(t, 33333.0, posReturnRefund(100000, 3, 0, 1))
	assert.Equal(t, 0.0, posReturnRefund(100000, 0, 0, 1))
}

func TestPosVoidHandler_Restock(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	// Hóa đơn POS chưa thu (lập tay qua InvoiceService) + xuất kho theo hóa đơn → hủy → hoàn kho.
	inv, err := fx.invoices.CreateInvoice(ctx, InvoiceDraft{BranchID: 1, Source: entity.InvoiceSourcePOS,
		Items: []InvoiceItemDraft{{ItemType: entity.InvoiceItemProduct, RefID: posUptr(1), Description: "Aquafina", Quantity: 4, UnitPrice: 10000}}})
	require.NoError(t, err)
	_, err = fx.stockSvc.applyStockChange(ctx, 1, 1, -4, entity.StockSale, stockRef{RefType: entity.StockRefInvoice, RefID: &inv.ID})
	require.NoError(t, err)
	assert.Equal(t, 16, fx.stock.levels[stockKey{1, 1}])

	require.NoError(t, fx.invoices.Void(ctx, inv.ID, "Khách không lấy"))
	assert.Equal(t, 20, fx.stock.levels[stockKey{1, 1}])
	returns := fx.stock.txsOf(entity.StockSaleReturn)
	require.Len(t, returns, 1)
	assert.Equal(t, 4, returns[0].Quantity)

	// Idempotent: gọi lại handler không nhập trùng.
	items, _ := fx.inv.Items(ctx, inv.ID)
	require.NoError(t, fx.svc.onInvoiceVoid(ctx, inv, &items[0]))
	assert.Equal(t, 20, fx.stock.levels[stockKey{1, 1}])
}

func TestPosScope(t *testing.T) {
	fx := newPosFixture()
	// Lễ tân chi nhánh 2 không bán được ở chi nhánh 1.
	_, err := fx.svc.CreateOrder(subStaffCtx(), posOrder(1, nil, [][2]int{{1, 1}}, posCash(10000)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không có quyền")
}
