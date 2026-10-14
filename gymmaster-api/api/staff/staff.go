// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package staff

import (
	"context"

	v1 "gymmaster-api/api/staff/v1"
)

type IStaffV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error)
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	AssignRoles(ctx context.Context, req *v1.AssignRolesReq) (res *v1.AssignRolesRes, err error)
	Lock(ctx context.Context, req *v1.LockReq) (res *v1.LockRes, err error)
	Unlock(ctx context.Context, req *v1.UnlockReq) (res *v1.UnlockRes, err error)
	ResetPassword(ctx context.Context, req *v1.ResetPasswordReq) (res *v1.ResetPasswordRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	Options(ctx context.Context, req *v1.OptionsReq) (res *v1.OptionsRes, err error)
	TrainerOptions(ctx context.Context, req *v1.TrainerOptionsReq) (res *v1.TrainerOptionsRes, err error)
}
