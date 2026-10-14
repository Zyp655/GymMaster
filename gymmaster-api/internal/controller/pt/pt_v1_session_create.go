package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) SessionCreate(ctx context.Context, req *v1.SessionCreateReq) (res *v1.SessionCreateRes, err error) {
	return c.svc.CreateSession(ctx, req.ContractID, req.StartAt)
}
