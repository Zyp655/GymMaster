package role

import (
	"context"

	v1 "gymmaster-api/api/role/v1"
)

func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	return nil, c.roleService.Delete(ctx, req.ID)
}
