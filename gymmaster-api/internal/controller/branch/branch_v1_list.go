package branch

import (
	"context"

	v1 "gymmaster-api/api/branch/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	return c.branchService.List(ctx, req)
}
