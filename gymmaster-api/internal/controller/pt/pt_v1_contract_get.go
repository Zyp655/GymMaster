package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) ContractGet(ctx context.Context, req *v1.ContractGetReq) (res *v1.ContractGetRes, err error) {
	return c.svc.GetContract(ctx, req.ID)
}
