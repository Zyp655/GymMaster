package inventory

import (
	"context"

	v1 "gymmaster-api/api/inventory/v1"
)

func (c *ControllerV1) Stock(ctx context.Context, req *v1.StockReq) (res *v1.StockRes, err error) {
	return c.stockService.Stock(ctx, req)
}

func (c *ControllerV1) Transactions(ctx context.Context, req *v1.TransactionListReq) (res *v1.TransactionListRes, err error) {
	return c.stockService.Transactions(ctx, req)
}

func (c *ControllerV1) LowStock(ctx context.Context, req *v1.LowStockReq) (res *v1.LowStockRes, err error) {
	items, err := c.stockService.LowStock(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	out := v1.LowStockRes(items)
	return &out, nil
}
