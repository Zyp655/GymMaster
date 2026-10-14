package product

import (
	"context"

	v1 "gymmaster-api/api/product/v1"
)

func (c *ControllerV1) SupplierList(ctx context.Context, req *v1.SupplierListReq) (res *v1.SupplierListRes, err error) {
	return c.productService.ListSuppliers(ctx, req)
}

func (c *ControllerV1) SupplierOptions(ctx context.Context, req *v1.SupplierOptionsReq) (res *v1.SupplierOptionsRes, err error) {
	items, err := c.productService.SupplierOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.SupplierOptionsRes(items)
	return &out, nil
}

func (c *ControllerV1) SupplierGet(ctx context.Context, req *v1.SupplierGetReq) (res *v1.SupplierGetRes, err error) {
	return c.productService.GetSupplier(ctx, req.ID)
}

func (c *ControllerV1) SupplierCreate(ctx context.Context, req *v1.SupplierCreateReq) (res *v1.SupplierCreateRes, err error) {
	return c.productService.CreateSupplier(ctx, req.SupplierInput)
}

func (c *ControllerV1) SupplierUpdate(ctx context.Context, req *v1.SupplierUpdateReq) (res *v1.SupplierUpdateRes, err error) {
	return c.productService.UpdateSupplier(ctx, req.ID, req.SupplierInput)
}

func (c *ControllerV1) SupplierDelete(ctx context.Context, req *v1.SupplierDeleteReq) (res *v1.SupplierDeleteRes, err error) {
	return &v1.SupplierDeleteRes{}, c.productService.DeleteSupplier(ctx, req.ID)
}
