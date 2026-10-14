package checkin

import (
	"context"

	v1 "gymmaster-api/api/checkin/v1"
)

func (c *ControllerV1) Checkout(ctx context.Context, req *v1.CheckoutReq) (res *v1.CheckoutRes, err error) {
	return c.checkinService.Checkout(ctx, req.ID)
}
