package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) Overview(ctx context.Context, req *v1.OverviewReq) (res *v1.OverviewRes, err error) {
	return c.reportService.Overview(ctx, req)
}
