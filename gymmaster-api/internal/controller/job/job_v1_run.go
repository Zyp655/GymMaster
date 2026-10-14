package job

import (
	"context"

	v1 "gymmaster-api/api/job/v1"
)

func (c *ControllerV1) Run(ctx context.Context, req *v1.RunReq) (res *v1.RunRes, err error) {
	return c.jobService.Run(ctx, req.Name)
}
