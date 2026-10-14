package shift

import (
	"context"

	v1 "gymmaster-api/api/shift/v1"
)

func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	return c.shiftService.Update(ctx, req.ID, req.ShiftInput)
}
