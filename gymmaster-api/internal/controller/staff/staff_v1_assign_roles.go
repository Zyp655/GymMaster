package staff

import (
	"context"

	v1 "gymmaster-api/api/staff/v1"
)

func (c *ControllerV1) AssignRoles(ctx context.Context, req *v1.AssignRolesReq) (res *v1.AssignRolesRes, err error) {
	return c.staffService.AssignRoles(ctx, req.ID, req.Roles)
}
