package setting

import (
	"context"

	v1 "gymmaster-api/api/setting/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	items, err := c.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.ListRes(items)
	return &out, nil
}
