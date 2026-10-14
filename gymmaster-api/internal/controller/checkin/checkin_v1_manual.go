package checkin

import (
	"context"

	v1 "gymmaster-api/api/checkin/v1"
)

func (c *ControllerV1) Manual(ctx context.Context, req *v1.ManualReq) (res *v1.ManualRes, err error) {
	return c.checkinService.Manual(ctx, req.MemberID, req.BranchID, req.Note)
}
