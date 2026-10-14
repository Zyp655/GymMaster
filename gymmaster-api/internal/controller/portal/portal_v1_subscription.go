package portal

import (
	"context"

	v1 "gymmaster-api/api/portal/v1"
)

func (c *ControllerV1) ListSubscriptions(ctx context.Context, _ *v1.ListSubscriptionsReq) (res *v1.ListSubscriptionsRes, err error) {
	items, err := c.portalService.Subscriptions(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.ListSubscriptionsRes(items)
	return &out, nil
}

func (c *ControllerV1) RequestFreeze(ctx context.Context, req *v1.RequestFreezeReq) (res *v1.RequestFreezeRes, err error) {
	return c.portalService.RequestFreeze(ctx, req.ID, req.FreezeInput)
}

func (c *ControllerV1) ListCheckins(ctx context.Context, req *v1.ListCheckinsReq) (res *v1.ListCheckinsRes, err error) {
	return c.portalService.Checkins(ctx, req.PageReq)
}

func (c *ControllerV1) ListBodyMetrics(ctx context.Context, _ *v1.ListBodyMetricsReq) (res *v1.ListBodyMetricsRes, err error) {
	items, err := c.portalService.BodyMetrics(ctx)
	if err != nil {
		return nil, err
	}
	return &items, nil
}

func (c *ControllerV1) ListInvoices(ctx context.Context, req *v1.ListInvoicesReq) (res *v1.ListInvoicesRes, err error) {
	return c.portalService.Invoices(ctx, req.PageReq)
}

func (c *ControllerV1) GetInvoice(ctx context.Context, req *v1.GetInvoiceReq) (res *v1.GetInvoiceRes, err error) {
	return c.portalService.Invoice(ctx, req.ID)
}
