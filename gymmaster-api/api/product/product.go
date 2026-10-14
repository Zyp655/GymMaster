// =================================================================================
// Interface controller – viết tay theo mẫu api/branch/branch.go.
// =================================================================================

package product

import (
	"context"

	v1 "gymmaster-api/api/product/v1"
)

type IProductV1 interface {
	CategoryList(ctx context.Context, req *v1.CategoryListReq) (res *v1.CategoryListRes, err error)
	CategoryCreate(ctx context.Context, req *v1.CategoryCreateReq) (res *v1.CategoryCreateRes, err error)
	CategoryUpdate(ctx context.Context, req *v1.CategoryUpdateReq) (res *v1.CategoryUpdateRes, err error)
	CategoryDelete(ctx context.Context, req *v1.CategoryDeleteReq) (res *v1.CategoryDeleteRes, err error)

	SupplierList(ctx context.Context, req *v1.SupplierListReq) (res *v1.SupplierListRes, err error)
	SupplierOptions(ctx context.Context, req *v1.SupplierOptionsReq) (res *v1.SupplierOptionsRes, err error)
	SupplierGet(ctx context.Context, req *v1.SupplierGetReq) (res *v1.SupplierGetRes, err error)
	SupplierCreate(ctx context.Context, req *v1.SupplierCreateReq) (res *v1.SupplierCreateRes, err error)
	SupplierUpdate(ctx context.Context, req *v1.SupplierUpdateReq) (res *v1.SupplierUpdateRes, err error)
	SupplierDelete(ctx context.Context, req *v1.SupplierDeleteReq) (res *v1.SupplierDeleteRes, err error)

	List(ctx context.Context, req *v1.ProductListReq) (res *v1.ProductListRes, err error)
	Get(ctx context.Context, req *v1.ProductGetReq) (res *v1.ProductGetRes, err error)
	Create(ctx context.Context, req *v1.ProductCreateReq) (res *v1.ProductCreateRes, err error)
	Update(ctx context.Context, req *v1.ProductUpdateReq) (res *v1.ProductUpdateRes, err error)
	Delete(ctx context.Context, req *v1.ProductDeleteReq) (res *v1.ProductDeleteRes, err error)
}
