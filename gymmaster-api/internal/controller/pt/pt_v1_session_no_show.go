package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) SessionNoShow(ctx context.Context, req *v1.SessionNoShowReq) (res *v1.SessionNoShowRes, err error) {
	return c.svc.MarkNoShow(ctx, req.ID, req.Note)
}
