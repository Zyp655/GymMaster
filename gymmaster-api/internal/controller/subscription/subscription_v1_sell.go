package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) Sell(ctx context.Context, req *v1.SellReq) (res *v1.SellRes, err error) {
	return c.svc.Sell(ctx, req.SellInput, req.Payment)
}
