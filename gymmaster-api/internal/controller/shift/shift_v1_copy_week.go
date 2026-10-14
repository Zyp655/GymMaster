package shift

import (
	"context"

	v1 "gymmaster-api/api/shift/v1"
)

func (c *ControllerV1) CopyWeek(ctx context.Context, req *v1.CopyWeekReq) (res *v1.CopyWeekRes, err error) {
	return c.shiftService.CopyWeek(ctx, req)
}
