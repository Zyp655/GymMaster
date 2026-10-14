package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) TrainerSchedule(ctx context.Context, req *v1.TrainerScheduleReq) (res *v1.TrainerScheduleRes, err error) {
	items, err := c.svc.TrainerScheduleAdmin(ctx, req)
	if err != nil {
		return nil, err
	}
	out := v1.TrainerScheduleRes(items)
	return &out, nil
}
