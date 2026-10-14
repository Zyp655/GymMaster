package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) SessionCancel(ctx context.Context, req *v1.SessionCancelReq) (res *v1.SessionCancelRes, err error) {
	return c.svc.CancelSession(ctx, req.ID, req.Reason, req.CountAsUsed)
}
