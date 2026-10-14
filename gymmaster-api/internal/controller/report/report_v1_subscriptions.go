package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) Subscriptions(ctx context.Context, req *v1.SubscriptionsReq) (res *v1.SubscriptionsRes, err error) {
	return c.reportService.Subscriptions(ctx, req)
}
