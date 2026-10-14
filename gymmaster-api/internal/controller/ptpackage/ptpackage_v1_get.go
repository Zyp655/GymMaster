package ptpackage

import (
	"context"

	v1 "gymmaster-api/api/ptpackage/v1"
)

func (c *ControllerV1) Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error) {
	return c.svc.Get(ctx, req.ID)
}
