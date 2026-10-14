package pos

import (
	"context"

	v1 "gymmaster-api/api/pos/v1"
)

func (c *ControllerV1) CreateOrder(ctx context.Context, req *v1.CreateOrderReq) (res *v1.CreateOrderRes, err error) {
	return c.posService.CreateOrder(ctx, req)
}

func (c *ControllerV1) ListOrders(ctx context.Context, req *v1.ListOrdersReq) (res *v1.ListOrdersRes, err error) {
	return c.posService.ListOrders(ctx, req)
}

func (c *ControllerV1) GetOrder(ctx context.Context, req *v1.GetOrderReq) (res *v1.GetOrderRes, err error) {
	return c.posService.GetOrder(ctx, req.InvoiceID)
}

func (c *ControllerV1) Return(ctx context.Context, req *v1.ReturnReq) (res *v1.ReturnRes, err error) {
	return c.posService.Return(ctx, req)
}

func (c *ControllerV1) Summary(ctx context.Context, req *v1.SummaryReq) (res *v1.SummaryRes, err error) {
	return c.posService.Summary(ctx, req)
}
