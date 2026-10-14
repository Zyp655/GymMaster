package job

import (
	"context"

	v1 "gymmaster-api/api/job/v1"
)

func (c *ControllerV1) Runs(ctx context.Context, req *v1.RunsReq) (res *v1.RunsRes, err error) {
	return c.jobService.Runs(ctx, req)
}
