package plan

import (
	"context"

	v1 "gymmaster-api/api/plan/v1"
)

func (c *ControllerV1) UpdateStatus(ctx context.Context, req *v1.UpdateStatusReq) (res *v1.UpdateStatusRes, err error) {
	return c.svc.UpdateStatus(ctx, req.ID, req.Status)
}
