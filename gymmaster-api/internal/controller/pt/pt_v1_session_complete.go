package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) SessionComplete(ctx context.Context, req *v1.SessionCompleteReq) (res *v1.SessionCompleteRes, err error) {
	return c.svc.CompleteSession(ctx, req.ID, req.WorkoutNote, req.TrainerNote)
}
