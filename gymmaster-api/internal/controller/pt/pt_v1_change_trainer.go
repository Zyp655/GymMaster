package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) ChangeTrainer(ctx context.Context, req *v1.ChangeTrainerReq) (res *v1.ChangeTrainerRes, err error) {
	return c.svc.ChangeTrainer(ctx, req.ID, req.ChangeTrainerInput)
}
