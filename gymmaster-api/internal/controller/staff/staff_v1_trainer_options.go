package staff

import (
	"context"

	v1 "gymmaster-api/api/staff/v1"
)

func (c *ControllerV1) TrainerOptions(ctx context.Context, req *v1.TrainerOptionsReq) (res *v1.TrainerOptionsRes, err error) {
	items, err := c.staffService.TrainerOptions(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	out := v1.TrainerOptionsRes(items)
	return &out, nil
}
