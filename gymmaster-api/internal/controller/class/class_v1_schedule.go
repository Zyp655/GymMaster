package class

import (
	"context"

	v1 "gymmaster-api/api/class/v1"
)

func (c *ControllerV1) ListSchedule(ctx context.Context, req *v1.ListScheduleReq) (res *v1.ListScheduleRes, err error) {
	items, err := c.scheduleService.List(ctx, req)
	if err != nil {
		return nil, err
	}
	out := v1.ListScheduleRes(items)
	return &out, nil
}

func (c *ControllerV1) GetSchedule(ctx context.Context, req *v1.GetScheduleReq) (res *v1.GetScheduleRes, err error) {
	return c.scheduleService.Get(ctx, req.ID)
}

func (c *ControllerV1) CreateSchedule(ctx context.Context, req *v1.CreateScheduleReq) (res *v1.CreateScheduleRes, err error) {
	return c.scheduleService.Create(ctx, req)
}

func (c *ControllerV1) CreateRecurring(ctx context.Context, req *v1.CreateRecurringReq) (res *v1.CreateRecurringRes, err error) {
	return c.scheduleService.CreateRecurring(ctx, req)
}

func (c *ControllerV1) UpdateSchedule(ctx context.Context, req *v1.UpdateScheduleReq) (res *v1.UpdateScheduleRes, err error) {
	return c.scheduleService.Update(ctx, req)
}

func (c *ControllerV1) CancelSchedule(ctx context.Context, req *v1.CancelScheduleReq) (res *v1.CancelScheduleRes, err error) {
	return c.scheduleService.Cancel(ctx, req.ID, req.Reason, req.Scope)
}
