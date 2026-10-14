package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) Retention(ctx context.Context, req *v1.RetentionReq) (res *v1.RetentionRes, err error) {
	return c.reportService.Retention(ctx, req)
}
