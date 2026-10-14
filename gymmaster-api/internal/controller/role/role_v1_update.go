package role

import (
	"context"

	v1 "gymmaster-api/api/role/v1"
)

func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	return c.roleService.Update(ctx, req)
}
