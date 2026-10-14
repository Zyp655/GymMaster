package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) RejectFreeze(ctx context.Context, req *v1.RejectFreezeReq) (res *v1.RejectFreezeRes, err error) {
	return c.svc.RejectFreeze(ctx, req.ID, req.Note)
}
