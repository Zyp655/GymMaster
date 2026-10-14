package invoice

import (
	"context"

	v1 "gymmaster-api/api/invoice/v1"
)

func (c *ControllerV1) PaymentList(ctx context.Context, req *v1.PaymentListReq) (res *v1.PaymentListRes, err error) {
	return c.svc.ListPayments(ctx, req)
}
