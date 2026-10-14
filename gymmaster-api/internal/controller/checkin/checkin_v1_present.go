package checkin

import (
	"context"

	v1 "gymmaster-api/api/checkin/v1"
)

func (c *ControllerV1) Present(ctx context.Context, req *v1.PresentReq) (res *v1.PresentRes, err error) {
	items, err := c.checkinService.Present(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	return &items, nil
}
