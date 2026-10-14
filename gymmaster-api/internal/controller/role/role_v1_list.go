package role

import (
	"context"

	v1 "gymmaster-api/api/role/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	roles, err := c.roleService.List(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.ListRes(roles)
	return &out, nil
}
