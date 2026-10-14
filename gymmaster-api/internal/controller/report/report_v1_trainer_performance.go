package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) TrainerPerformance(ctx context.Context, req *v1.TrainerPerformanceReq) (res *v1.TrainerPerformanceRes, err error) {
	return c.reportService.TrainerPerformance(ctx, req)
}
