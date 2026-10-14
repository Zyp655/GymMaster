package portal

import (
	"context"

	v1 "gymmaster-api/api/portal/v1"
)

func (c *ControllerV1) ListPTContracts(ctx context.Context, _ *v1.ListPTContractsReq) (res *v1.ListPTContractsRes, err error) {
	items, err := c.portalService.PTContracts(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.ListPTContractsRes(items)
	return &out, nil
}

func (c *ControllerV1) ListPTSessions(ctx context.Context, req *v1.ListPTSessionsReq) (res *v1.ListPTSessionsRes, err error) {
	return c.portalService.PTSessions(ctx, req.From, req.To)
}

func (c *ControllerV1) BookPTSession(ctx context.Context, req *v1.BookPTSessionReq) (res *v1.BookPTSessionRes, err error) {
	return c.portalService.BookPTSession(ctx, req.ContractID, req.StartAt)
}

func (c *ControllerV1) CancelPTSession(ctx context.Context, req *v1.CancelPTSessionReq) (res *v1.CancelPTSessionRes, err error) {
	return c.portalService.CancelPTSession(ctx, req.ID, req.Reason)
}

func (c *ControllerV1) TrainerAvailability(ctx context.Context, req *v1.TrainerAvailabilityReq) (res *v1.TrainerAvailabilityRes, err error) {
	return c.portalService.TrainerAvailability(ctx, req.ID, req.Date)
}
