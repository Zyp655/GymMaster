// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package job

import (
	"context"

	v1 "gymmaster-api/api/job/v1"
)

type IJobV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Runs(ctx context.Context, req *v1.RunsReq) (res *v1.RunsRes, err error)
	Run(ctx context.Context, req *v1.RunReq) (res *v1.RunRes, err error)
	TestEmail(ctx context.Context, req *v1.TestEmailReq) (res *v1.TestEmailRes, err error)
}
