package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) EndFreeze(ctx context.Context, req *v1.EndFreezeReq) (res *v1.EndFreezeRes, err error) {
	return c.svc.EndFreezeEarly(ctx, req.ID, req.Note)
}
