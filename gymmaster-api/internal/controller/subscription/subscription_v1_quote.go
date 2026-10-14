package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) Quote(ctx context.Context, req *v1.QuoteReq) (res *v1.QuoteRes, err error) {
	return c.svc.Quote(ctx, req.SellInput)
}
