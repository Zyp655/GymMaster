package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/inventory/v1"
	productv1 "gymmaster-api/api/product/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

func TestStockImport_RejectsOutOfRangeQuantityAndCost(t *testing.T) {
	fx := newPosFixture()
	ctx := subOwnerCtx()
	_, err := fx.stockSvc.Import(ctx, &v1.ImportReq{BranchID: 1,
		Items: []*v1.ImportItem{{ProductID: 1, Quantity: 1, UnitCost: 1e13}}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "đơn giá 1e13 → 400 (trước đây 500 numeric overflow)")
	_, err = fx.stockSvc.Import(ctx, &v1.ImportReq{BranchID: 1,
		Items: []*v1.ImportItem{{ProductID: 1, Quantity: stockMaxLineQty + 1, UnitCost: 1000}}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "số lượng vượt giới hạn dòng")
	_, err = fx.stockSvc.Transfer(ctx, &v1.TransferReq{FromBranchID: 1, ToBranchID: 2,
		Items: []*v1.TransferItem{{ProductID: 1, Quantity: stockMaxLineQty + 1}}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	tooMany := stockMaxLevel + 1
	_, err = fx.stockSvc.Adjust(ctx, &v1.AdjustReq{BranchID: 1, Items: []*v1.AdjustItem{{ProductID: 1, ActualQuantity: &tooMany}}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Tổng tồn sau nhập vượt giới hạn → 400, không đổi tồn/giá vốn.
	fx.stock.levels[stockKey{1, 1}] = stockMaxLevel - 10
	_, err = fx.stockSvc.Import(ctx, &v1.ImportReq{BranchID: 1,
		Items: []*v1.ImportItem{{ProductID: 1, Quantity: 11, UnitCost: 1000}}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Equal(t, stockMaxLevel-10, fx.stock.levels[stockKey{1, 1}])

	// Biên hợp lệ vẫn nhập được.
	fx.stock.levels[stockKey{1, 1}] = 0
	_, err = fx.stockSvc.Import(ctx, &v1.ImportReq{BranchID: 1,
		Items: []*v1.ImportItem{{ProductID: 1, Quantity: stockMaxLineQty, UnitCost: stockMaxUnitCost}}})
	require.NoError(t, err)
}

// productUpdateFakeRepo: đủ phương thức cho productService.Update/Get.
type productUpdateFakeRepo struct {
	*stockFakeProductRepo
	saved *entity.Product
}

func (f *productUpdateFakeRepo) ExistsSKU(context.Context, string, uint) (bool, error) {
	return false, nil
}
func (f *productUpdateFakeRepo) UpdateProduct(_ context.Context, p *entity.Product) error {
	cp := *p
	f.saved = &cp
	return nil
}
func (f *productUpdateFakeRepo) FindProductRow(_ context.Context, id uint, _ uint) (*repository.ProductRow, error) {
	p := f.products[id]
	if f.saved != nil && f.saved.ID == id {
		p = f.saved
	}
	return &repository.ProductRow{Product: *p}, nil
}
func (f *productUpdateFakeRepo) ProductStocks(context.Context, uint, []uint) ([]repository.ProductStockRow, error) {
	return nil, nil
}

func TestProductUpdate_IgnoresClientCostPrice(t *testing.T) {
	fx := newPosFixture()
	fx.products.products[1].CostPrice = 6000 // giá vốn bình quân sau nhập kho
	repo := &productUpdateFakeRepo{stockFakeProductRepo: fx.products}
	svc := NewProductService(repo, nil)
	res, err := svc.Update(subOwnerCtx(), 1, productv1.ProductInput{SKU: "AQUA500", Name: "Aquafina 500ml (mới)",
		Unit: "chai", CostPrice: 1, SellPrice: 11000, Status: entity.ProductStatusActive})
	require.NoError(t, err)
	require.NotNil(t, repo.saved)
	assert.Equal(t, 6000.0, repo.saved.CostPrice, "PUT không ghi đè giá vốn bình quân")
	assert.Equal(t, 6000.0, res.CostPrice)
	assert.Equal(t, 11000.0, repo.saved.SellPrice)
}
