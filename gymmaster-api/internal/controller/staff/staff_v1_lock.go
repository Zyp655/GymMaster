package staff

import (
	"context"

	v1 "gymmaster-api/api/staff/v1"
)

func (c *ControllerV1) Lock(ctx context.Context, req *v1.LockReq) (res *v1.LockRes, err error) {
	return c.staffService.Lock(ctx, req.ID)
}
