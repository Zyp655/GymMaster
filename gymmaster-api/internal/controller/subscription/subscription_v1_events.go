package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) Events(ctx context.Context, req *v1.EventsReq) (res *v1.EventsRes, err error) {
	items, err := c.svc.Events(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	out := v1.EventsRes(items)
	return &out, nil
}
