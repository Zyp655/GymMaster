package invoice

import (
	"context"

	v1 "gymmaster-api/api/invoice/v1"
	"gymmaster-api/internal/service"
)

func (c *ControllerV1) Collect(ctx context.Context, req *v1.CollectReq) (res *v1.CollectRes, err error) {
	if _, err = c.svc.AddPayment(ctx, req.ID, service.PaymentInput{
		Amount: req.Amount, Method: req.Method, Reference: req.Reference, Note: req.Note,
	}); err != nil {
		return nil, err
	}
	return c.svc.Get(ctx, req.ID)
}
