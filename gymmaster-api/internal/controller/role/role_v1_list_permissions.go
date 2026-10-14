package role

import (
	"context"

	v1 "gymmaster-api/api/role/v1"
)

func (c *ControllerV1) ListPermissions(ctx context.Context, req *v1.ListPermissionsReq) (res *v1.ListPermissionsRes, err error) {
	perms, err := c.roleService.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.ListPermissionsRes(perms)
	return &out, nil
}
