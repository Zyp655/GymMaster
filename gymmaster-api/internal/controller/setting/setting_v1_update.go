package setting

import (
	"context"

	v1 "gymmaster-api/api/setting/v1"
)

func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	return c.svc.Update(ctx, req.Key, req.Value)
}
