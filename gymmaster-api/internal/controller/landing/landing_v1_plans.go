package landing

import (
	"context"

	v1 "gymmaster-api/api/landing/v1"
)

func (c *ControllerV1) Plans(ctx context.Context, req *v1.PlansReq) (res *v1.PlansRes, err error) {
	return c.svc.Plans(ctx)
}
