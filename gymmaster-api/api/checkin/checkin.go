// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package checkin

import (
	"context"

	v1 "gymmaster-api/api/checkin/v1"
)

type ICheckinV1 interface {
	MemberQR(ctx context.Context, req *v1.MemberQRReq) (res *v1.MemberQRRes, err error)
	Scan(ctx context.Context, req *v1.ScanReq) (res *v1.ScanRes, err error)
	Manual(ctx context.Context, req *v1.ManualReq) (res *v1.ManualRes, err error)
	Checkout(ctx context.Context, req *v1.CheckoutReq) (res *v1.CheckoutRes, err error)
	Present(ctx context.Context, req *v1.PresentReq) (res *v1.PresentRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Stats(ctx context.Context, req *v1.StatsReq) (res *v1.StatsRes, err error)
}
