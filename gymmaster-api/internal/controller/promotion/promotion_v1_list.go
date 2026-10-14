package promotion

import (
	"context"

	v1 "gymmaster-api/api/promotion/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	return c.svc.List(ctx, req)
}
