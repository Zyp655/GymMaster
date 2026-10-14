package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) Freeze(ctx context.Context, req *v1.FreezeReq) (res *v1.FreezeRes, err error) {
	return c.svc.RequestFreeze(ctx, req.ID, req.FreezeInput)
}
