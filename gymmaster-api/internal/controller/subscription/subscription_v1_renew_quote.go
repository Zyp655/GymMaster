package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) RenewQuote(ctx context.Context, req *v1.RenewQuoteReq) (res *v1.RenewQuoteRes, err error) {
	return c.svc.RenewQuote(ctx, req.ID, req.RenewInput)
}
