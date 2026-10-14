package staff

import (
	"context"

	v1 "gymmaster-api/api/staff/v1"
)

func (c *ControllerV1) Options(ctx context.Context, req *v1.OptionsReq) (res *v1.OptionsRes, err error) {
	items, err := c.staffService.Options(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	out := v1.OptionsRes(items)
	return &out, nil
}
