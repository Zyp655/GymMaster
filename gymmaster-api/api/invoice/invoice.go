// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package invoice

import (
	"context"

	v1 "gymmaster-api/api/invoice/v1"
)

type IInvoiceV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error)
	Collect(ctx context.Context, req *v1.CollectReq) (res *v1.CollectRes, err error)
	Void(ctx context.Context, req *v1.VoidReq) (res *v1.VoidRes, err error)
	Refund(ctx context.Context, req *v1.RefundReq) (res *v1.RefundRes, err error)
	Print(ctx context.Context, req *v1.PrintReq) (res *v1.PrintRes, err error)
	PaymentList(ctx context.Context, req *v1.PaymentListReq) (res *v1.PaymentListRes, err error)
}
