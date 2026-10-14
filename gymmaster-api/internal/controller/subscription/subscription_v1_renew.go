package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) Renew(ctx context.Context, req *v1.RenewReq) (res *v1.RenewRes, err error) {
	return c.svc.Renew(ctx, req.ID, req.RenewInput)
}
