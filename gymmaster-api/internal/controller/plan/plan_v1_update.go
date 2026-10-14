package plan

import (
	"context"

	v1 "gymmaster-api/api/plan/v1"
)

func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	return c.svc.Update(ctx, req.ID, req.PlanInput)
}
