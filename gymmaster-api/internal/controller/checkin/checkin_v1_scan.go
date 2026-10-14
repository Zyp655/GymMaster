package checkin

import (
	"context"

	v1 "gymmaster-api/api/checkin/v1"
)

func (c *ControllerV1) Scan(ctx context.Context, req *v1.ScanReq) (res *v1.ScanRes, err error) {
	return c.checkinService.Scan(ctx, req.Token, req.BranchID)
}
