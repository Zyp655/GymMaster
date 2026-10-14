package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) CheckinHeatmap(ctx context.Context, req *v1.CheckinHeatmapReq) (res *v1.CheckinHeatmapRes, err error) {
	return c.reportService.CheckinHeatmap(ctx, req)
}
