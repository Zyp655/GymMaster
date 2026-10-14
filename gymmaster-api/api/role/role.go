// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package role

import (
	"context"

	v1 "gymmaster-api/api/role/v1"
)

type IRoleV1 interface {
	ListPermissions(ctx context.Context, req *v1.ListPermissionsReq) (res *v1.ListPermissionsRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error)
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
}
