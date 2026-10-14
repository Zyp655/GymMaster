package checkin

import (
	"context"

	v1 "gymmaster-api/api/checkin/v1"
)

func (c *ControllerV1) Stats(ctx context.Context, req *v1.StatsReq) (res *v1.StatsRes, err error) {
	return c.checkinService.Stats(ctx, req.BranchID, req.Date)
}
