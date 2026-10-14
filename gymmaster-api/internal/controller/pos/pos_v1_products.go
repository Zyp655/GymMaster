package pos

import (
	"context"

	v1 "gymmaster-api/api/pos/v1"
)

func (c *ControllerV1) Products(ctx context.Context, req *v1.ProductsReq) (res *v1.ProductsRes, err error) {
	items, err := c.posService.Products(ctx, req)
	if err != nil {
		return nil, err
	}
	out := v1.ProductsRes(items)
	return &out, nil
}

func (c *ControllerV1) ByBarcode(ctx context.Context, req *v1.ByBarcodeReq) (res *v1.ByBarcodeRes, err error) {
	return c.posService.ByBarcode(ctx, req.Code, req.BranchID)
}
