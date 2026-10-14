package staff

import (
	"context"

	v1 "gymmaster-api/api/staff/v1"
)

func (c *ControllerV1) Unlock(ctx context.Context, req *v1.UnlockReq) (res *v1.UnlockRes, err error) {
	return c.staffService.Unlock(ctx, req.ID)
}
