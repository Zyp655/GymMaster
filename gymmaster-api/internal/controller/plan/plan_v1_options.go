package plan

import (
	"context"

	v1 "gymmaster-api/api/plan/v1"
)

func (c *ControllerV1) Options(ctx context.Context, req *v1.OptionsReq) (res *v1.OptionsRes, err error) {
	items, err := c.svc.Options(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.OptionsRes(items)
	return &out, nil
}
