package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) ContractList(ctx context.Context, req *v1.ContractListReq) (res *v1.ContractListRes, err error) {
	return c.svc.ListContracts(ctx, req)
}
