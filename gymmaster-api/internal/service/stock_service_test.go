package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/inventory/v1"
	"gymmaster-api/internal/entity"
)

func TestStockWeightedCost(t *testing.T) {
	assert.Equal(t, 7500.0, stockWeightedCost(20, 6000, 20, 9000))
	assert.Equal(t, 9000.0, stockWeightedCost(0, 6000, 5, 9000), "hết hàng → giá nhập mới")
	assert.Equal(t, 6333.33, stockWeightedCost(2, 5000, 1, 9000), "làm tròn 2 chữ số")
	assert.Equal(t, 6000.0, stockWeightedCost(10, 6000, 0, 9000))
}

func TestStockImport_WeightedAverageCost(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	// Aquafina: giá vốn 5.000, tồn 20 (CN1) + 10 (CN2) = 30. Nhập CN1 30 chai giá 7.000
	// → (30×5.000 + 30×7.000) / 60 = 6.000 (bình quân trên tổng tồn mọi chi nhánh).
	v, err := fx.stockSvc.Import(ctx, &v1.ImportReq{BranchID: 1, SupplierID: posUptr(1),
		Items: []*v1.ImportItem{{ProductID: 1, Quantity: 30, UnitCost: 7000}}, Note: "Nhập đầu tháng"})
	require.NoError(t, err)
	assert.Equal(t, 6000.0, fx.products.products[1].CostPrice)
	assert.Equal(t, 50, fx.stock.levels[stockKey{1, 1}])
	assert.Equal(t, "IMPORT", v.Kind)
	assert.Regexp(t, `^NK\d{6}0001$`, v.VoucherNo)
	assert.Equal(t, 30, v.TotalQuantity)
	assert.Equal(t, 210000.0, v.TotalValue)
	require.Len(t, v.Lines, 1)
	assert.Equal(t, 7000.0, v.Lines[0].UnitCost)
	assert.Equal(t, 50, v.Lines[0].BalanceAfter)

	// Hai dòng cùng sản phẩm trong một phiếu: tính lần lượt.
	_, err = fx.stockSvc.Import(ctx, &v1.ImportReq{BranchID: 2,
		Items: []*v1.ImportItem{{ProductID: 1, Quantity: 40, UnitCost: 6000}, {ProductID: 1, Quantity: 100, UnitCost: 9000}}})
	require.NoError(t, err)
	// (60×6000 + 40×6000)/100 = 6000 → (100×6000 + 100×9000)/200 = 7500
	assert.Equal(t, 7500.0, fx.products.products[1].CostPrice)

	_, err = fx.stockSvc.Import(ctx, &v1.ImportReq{BranchID: 1, SupplierID: posUptr(9), Items: []*v1.ImportItem{{ProductID: 1, Quantity: 1}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nhà cung cấp không tồn tại")
}

func TestStockAdjust_Difference(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	actual := func(v int) *int { return &v }
	// CN1: Aquafina 20 → thực tế 17 (−3); Whey 3 → thực tế 3 (khớp, bỏ qua); hàng cũ 5 → 6 (+1).
	v, err := fx.stockSvc.Adjust(ctx, &v1.AdjustReq{BranchID: 1, Note: "Kiểm kê cuối tháng", Items: []*v1.AdjustItem{
		{ProductID: 1, ActualQuantity: actual(17)}, {ProductID: 2, ActualQuantity: actual(3)}, {ProductID: 3, ActualQuantity: actual(6)},
	}})
	require.NoError(t, err)
	assert.Equal(t, entity.StockAdjust, v.Kind)
	assert.Regexp(t, `^KK\d{6}0001$`, v.VoucherNo)
	require.Len(t, v.Lines, 2, "dòng không lệch không ghi thẻ kho")
	assert.Equal(t, -3, v.Lines[0].Quantity)
	assert.Equal(t, 17, v.Lines[0].BalanceAfter)
	assert.Equal(t, 1, v.Lines[1].Quantity)
	assert.Equal(t, 17, fx.stock.levels[stockKey{1, 1}])
	assert.Equal(t, 6, fx.stock.levels[stockKey{3, 1}])
	assert.Equal(t, "Kiểm kê cuối tháng", v.Note)

	_, err = fx.stockSvc.Adjust(ctx, &v1.AdjustReq{BranchID: 1, Items: []*v1.AdjustItem{{ProductID: 1, ActualQuantity: actual(17)}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không có chênh lệch")

	_, err = fx.stockSvc.Adjust(ctx, &v1.AdjustReq{BranchID: 1, Items: []*v1.AdjustItem{{ProductID: 1}}})
	require.Error(t, err)
}

func TestStockTransfer(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	v, err := fx.stockSvc.Transfer(ctx, &v1.TransferReq{FromBranchID: 2, ToBranchID: 1,
		Items: []*v1.TransferItem{{ProductID: 1, Quantity: 4}}, Note: "Bổ sung quầy"})
	require.NoError(t, err)
	assert.Equal(t, "TRANSFER", v.Kind)
	assert.Equal(t, uint(2), v.BranchID)
	require.NotNil(t, v.ToBranchID)
	assert.Equal(t, uint(1), *v.ToBranchID)
	assert.Equal(t, 4, v.TotalQuantity)
	assert.Equal(t, 6, fx.stock.levels[stockKey{1, 2}])
	assert.Equal(t, 24, fx.stock.levels[stockKey{1, 1}])
	assert.Equal(t, []stockKey{{1, 1}, {1, 2}}, fx.stock.lockLog[:2], "khóa (product_id, branch_id) tăng dần")

	_, err = fx.stockSvc.Transfer(ctx, &v1.TransferReq{FromBranchID: 2, ToBranchID: 1, Items: []*v1.TransferItem{{ProductID: 1, Quantity: 7}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "còn 6 chai, cần 7")

	_, err = fx.stockSvc.Transfer(ctx, &v1.TransferReq{FromBranchID: 1, ToBranchID: 1, Items: []*v1.TransferItem{{ProductID: 1, Quantity: 1}}})
	require.Error(t, err)
}

func TestStockLowStockAlert_OnlyWhenCrossing(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	fx.stock.levels[stockKey{1, 1}] = 8 // min_stock = 5
	sell := func(q int) {
		_, err := fx.stockSvc.applyStockChange(ctx, 1, 1, -q, entity.StockSale, stockRef{})
		require.NoError(t, err)
	}
	sell(2) // 8 → 6: vẫn ≥ 5
	assert.Empty(t, fx.notifier.calls)
	sell(1) // 6 → 5: bằng ngưỡng, chưa thấp
	assert.Empty(t, fx.notifier.calls)
	sell(1) // 5 → 4: vừa vượt ngưỡng → cảnh báo
	require.Len(t, fx.notifier.calls, 1)
	assert.Equal(t, "inventory.view|1|LOW_STOCK|Tồn kho thấp: Nước suối Aquafina 500ml", fx.notifier.calls[0])
	sell(1) // 4 → 3: đã thấp từ trước → không báo lại
	assert.Len(t, fx.notifier.calls, 1)

	_, err := fx.stockSvc.applyStockChange(ctx, 1, 1, 10, entity.StockImport, stockRef{}) // 3 → 13
	require.NoError(t, err)
	assert.Len(t, fx.notifier.calls, 1, "nhập hàng không cảnh báo")
	sell(9) // 13 → 4: vượt ngưỡng lần nữa → báo
	assert.Len(t, fx.notifier.calls, 2)

	assert.True(t, stockCrossedLow(5, 4, 5))
	assert.False(t, stockCrossedLow(4, 3, 5))
	assert.False(t, stockCrossedLow(10, 0, 0), "min_stock = 0 → không cảnh báo")
}

func TestStockApplyChange_NoNegative(t *testing.T) {
	fx := newPosFixture()
	_, err := fx.stockSvc.applyStockChange(subOwnerCtx(), 2, 1, -4, entity.StockSale, stockRef{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "còn 3 hộp, cần 4")
	assert.Equal(t, 3, fx.stock.levels[stockKey{2, 1}])
	assert.Empty(t, fx.stock.txs)
}

func TestStockSortKeys(t *testing.T) {
	keys := stockSortKeys([]stockKey{{2, 1}, {1, 3}, {1, 1}, {2, 1}})
	assert.Equal(t, []stockKey{{1, 1}, {1, 3}, {2, 1}}, keys)
}
