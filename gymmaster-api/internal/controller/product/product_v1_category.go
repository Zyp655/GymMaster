package product

import (
	"context"

	v1 "gymmaster-api/api/product/v1"
)

func (c *ControllerV1) CategoryList(ctx context.Context, req *v1.CategoryListReq) (res *v1.CategoryListRes, err error) {
	items, err := c.productService.ListCategories(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}
	out := v1.CategoryListRes(items)
	return &out, nil
}

func (c *ControllerV1) CategoryCreate(ctx context.Context, req *v1.CategoryCreateReq) (res *v1.CategoryCreateRes, err error) {
	return c.productService.CreateCategory(ctx, req.CategoryInput)
}

func (c *ControllerV1) CategoryUpdate(ctx context.Context, req *v1.CategoryUpdateReq) (res *v1.CategoryUpdateRes, err error) {
	return c.productService.UpdateCategory(ctx, req.ID, req.CategoryInput)
}

func (c *ControllerV1) CategoryDelete(ctx context.Context, req *v1.CategoryDeleteReq) (res *v1.CategoryDeleteRes, err error) {
	return &v1.CategoryDeleteRes{}, c.productService.DeleteCategory(ctx, req.ID)
}
