package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) CommissionList(ctx context.Context, req *v1.CommissionListReq) (res *v1.CommissionListRes, err error) {
	return c.svc.ListCommissions(ctx, req)
}
