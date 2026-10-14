package invoice

import (
	"context"

	v1 "gymmaster-api/api/invoice/v1"
	"gymmaster-api/internal/service"
)

func (c *ControllerV1) Refund(ctx context.Context, req *v1.RefundReq) (res *v1.RefundRes, err error) {
	if _, err = c.svc.Refund(ctx, req.ID, service.RefundInput{
		Amount: req.Amount, Method: req.Method, Reason: req.Reason,
	}); err != nil {
		return nil, err
	}
	return c.svc.Get(ctx, req.ID)
}
