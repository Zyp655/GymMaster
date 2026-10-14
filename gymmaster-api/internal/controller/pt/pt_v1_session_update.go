package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) SessionUpdate(ctx context.Context, req *v1.SessionUpdateReq) (res *v1.SessionUpdateRes, err error) {
	return c.svc.RescheduleSession(ctx, req.ID, req.StartAt)
}
