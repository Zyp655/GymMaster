package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) ClassOccupancy(ctx context.Context, req *v1.ClassOccupancyReq) (res *v1.ClassOccupancyRes, err error) {
	return c.reportService.ClassOccupancy(ctx, req)
}
