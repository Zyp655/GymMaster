// =================================================================================
// Interface controller – viết tay theo mẫu api/branch/branch.go.
// =================================================================================

package pos

import (
	"context"

	v1 "gymmaster-api/api/pos/v1"
)

type IPosV1 interface {
	Products(ctx context.Context, req *v1.ProductsReq) (res *v1.ProductsRes, err error)
	ByBarcode(ctx context.Context, req *v1.ByBarcodeReq) (res *v1.ByBarcodeRes, err error)
	CreateOrder(ctx context.Context, req *v1.CreateOrderReq) (res *v1.CreateOrderRes, err error)
	ListOrders(ctx context.Context, req *v1.ListOrdersReq) (res *v1.ListOrdersRes, err error)
	GetOrder(ctx context.Context, req *v1.GetOrderReq) (res *v1.GetOrderRes, err error)
	Return(ctx context.Context, req *v1.ReturnReq) (res *v1.ReturnRes, err error)
	Summary(ctx context.Context, req *v1.SummaryReq) (res *v1.SummaryRes, err error)
}
