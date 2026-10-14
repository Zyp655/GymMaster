// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package landing

import (
	"context"

	v1 "gymmaster-api/api/landing/v1"
)

type ILandingV1 interface {
	Plans(ctx context.Context, req *v1.PlansReq) (res *v1.PlansRes, err error)
	Branches(ctx context.Context, req *v1.BranchesReq) (res *v1.BranchesRes, err error)
}
