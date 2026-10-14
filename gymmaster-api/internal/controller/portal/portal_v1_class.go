package portal

import (
	"context"

	v1 "gymmaster-api/api/portal/v1"
)

func (c *ControllerV1) ListBranches(ctx context.Context, _ *v1.ListBranchesReq) (res *v1.ListBranchesRes, err error) {
	items, err := c.portalService.Branches(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.ListBranchesRes(items)
	return &out, nil
}

func (c *ControllerV1) ListClasses(ctx context.Context, req *v1.ListClassesReq) (res *v1.ListClassesRes, err error) {
	return c.portalService.Classes(ctx, req.BranchID, req.From, req.To)
}

func (c *ControllerV1) BookClass(ctx context.Context, req *v1.BookClassReq) (res *v1.BookClassRes, err error) {
	return c.portalService.BookClass(ctx, req.ScheduleID)
}

func (c *ControllerV1) ListBookings(ctx context.Context, req *v1.ListBookingsReq) (res *v1.ListBookingsRes, err error) {
	items, err := c.portalService.Bookings(ctx, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := v1.ListBookingsRes(items)
	return &out, nil
}

func (c *ControllerV1) CancelBooking(ctx context.Context, req *v1.CancelBookingReq) (res *v1.CancelBookingRes, err error) {
	return c.portalService.CancelBooking(ctx, req.ID)
}
