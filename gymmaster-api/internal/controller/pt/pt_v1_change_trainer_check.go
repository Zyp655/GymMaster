package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) ChangeTrainerCheck(ctx context.Context, req *v1.ChangeTrainerCheckReq) (res *v1.ChangeTrainerCheckRes, err error) {
	return c.svc.CheckChangeTrainer(ctx, req.ID, req.ChangeTrainerInput)
}
