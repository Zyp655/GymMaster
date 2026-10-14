package shift

import (
	"context"

	v1 "gymmaster-api/api/shift/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	items, err := c.shiftService.List(ctx, req)
	if err != nil {
		return nil, err
	}
	out := v1.ListRes(items)
	return &out, nil
}
