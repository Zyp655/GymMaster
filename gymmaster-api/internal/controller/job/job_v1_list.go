package job

import (
	"context"

	v1 "gymmaster-api/api/job/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	list, err := c.jobService.List(ctx)
	if err != nil {
		return nil, err
	}
	return &list, nil
}
