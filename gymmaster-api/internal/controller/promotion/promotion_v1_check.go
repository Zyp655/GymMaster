package promotion

import (
	"context"

	v1 "gymmaster-api/api/promotion/v1"
)

func (c *ControllerV1) Check(ctx context.Context, req *v1.CheckReq) (res *v1.CheckRes, err error) {
	return c.svc.Check(ctx, req.Code, req.AppliesTo, req.Amount)
}
