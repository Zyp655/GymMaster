package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/pos/v1"
	"gymmaster-api/internal/entity"
)

// FIN-POS-30: hàng khách trả nhập lại kho theo giá vốn lúc bán → giá vốn bình quân được tính lại.
func TestPosReturn_ReweightsAverageCost(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	res, err := fx.svc.CreateOrder(ctx, posOrder(1, nil, [][2]int{{1, 3}}, posCash(30000)))
	require.NoError(t, err)
	saleCost := fx.stock.txsOf(entity.StockSale)[0].UnitCost

	// Sau khi bán, giá vốn hiện tại tăng (vd phiếu nhập mới giá cao hơn).
	fx.stock.products.products[1].CostPrice = saleCost + 1000
	before, _ := fx.stock.products.TotalStock(ctx, 1)

	_, err = fx.svc.Return(ctx, &v1.ReturnReq{InvoiceID: res.Invoice.ID, RefundMethod: entity.PaymentCash, Reason: "Khách đổi ý",
		Items: []*v1.ReturnItem{{ProductID: 1, Quantity: 2}}})
	require.NoError(t, err)

	want := stockWeightedCost(before, saleCost+1000, 2, saleCost)
	assert.Equal(t, want, fx.stock.products.products[1].CostPrice)
	assert.Less(t, fx.stock.products.products[1].CostPrice, saleCost+1000, "giá vốn bình quân giảm về phía giá vốn hàng trả")
}
