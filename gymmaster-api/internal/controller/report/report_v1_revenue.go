package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) Revenue(ctx context.Context, req *v1.RevenueReq) (res *v1.RevenueRes, err error) {
	return c.reportService.Revenue(ctx, req)
}
