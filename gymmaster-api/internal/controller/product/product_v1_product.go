package product

import (
	"context"

	v1 "gymmaster-api/api/product/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ProductListReq) (res *v1.ProductListRes, err error) {
	return c.productService.List(ctx, req)
}

func (c *ControllerV1) Get(ctx context.Context, req *v1.ProductGetReq) (res *v1.ProductGetRes, err error) {
	return c.productService.Get(ctx, req.ID)
}

func (c *ControllerV1) Create(ctx context.Context, req *v1.ProductCreateReq) (res *v1.ProductCreateRes, err error) {
	return c.productService.Create(ctx, req.ProductInput)
}

func (c *ControllerV1) Update(ctx context.Context, req *v1.ProductUpdateReq) (res *v1.ProductUpdateRes, err error) {
	return c.productService.Update(ctx, req.ID, req.ProductInput)
}

func (c *ControllerV1) Delete(ctx context.Context, req *v1.ProductDeleteReq) (res *v1.ProductDeleteRes, err error) {
	return &v1.ProductDeleteRes{}, c.productService.Delete(ctx, req.ID)
}
