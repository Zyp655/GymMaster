package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) SessionList(ctx context.Context, req *v1.SessionListReq) (res *v1.SessionListRes, err error) {
	items, err := c.svc.ListSessions(ctx, req)
	if err != nil {
		return nil, err
	}
	out := v1.SessionListRes(items)
	return &out, nil
}
