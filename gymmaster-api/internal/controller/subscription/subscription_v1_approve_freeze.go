package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) ApproveFreeze(ctx context.Context, req *v1.ApproveFreezeReq) (res *v1.ApproveFreezeRes, err error) {
	return c.svc.ApproveFreeze(ctx, req.ID, req.Note)
}
