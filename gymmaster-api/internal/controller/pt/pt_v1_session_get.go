package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) SessionGet(ctx context.Context, req *v1.SessionGetReq) (res *v1.SessionGetRes, err error) {
	return c.svc.GetSession(ctx, req.ID)
}
