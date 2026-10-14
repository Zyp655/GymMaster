package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) CommissionSummary(ctx context.Context, req *v1.CommissionSummaryReq) (res *v1.CommissionSummaryRes, err error) {
	return c.svc.CommissionSummary(ctx, req)
}
