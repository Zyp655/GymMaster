package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) UpgradeQuote(ctx context.Context, req *v1.UpgradeQuoteReq) (res *v1.UpgradeQuoteRes, err error) {
	return c.svc.UpgradeQuote(ctx, req.ID, req.PlanID)
}
