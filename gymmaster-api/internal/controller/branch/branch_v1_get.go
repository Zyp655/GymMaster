package branch

import (
	"context"

	v1 "gymmaster-api/api/branch/v1"
)

func (c *ControllerV1) Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error) {
	return c.branchService.Get(ctx, req.ID)
}
