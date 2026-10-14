// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package auditlog

import (
	"context"

	v1 "gymmaster-api/api/auditlog/v1"
)

type IAuditlogV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error)
	Options(ctx context.Context, req *v1.OptionsReq) (res *v1.OptionsRes, err error)
}
