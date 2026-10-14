package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) CommissionPay(ctx context.Context, req *v1.CommissionPayReq) (res *v1.CommissionPayRes, err error) {
	return c.svc.PayCommissions(ctx, req)
}
