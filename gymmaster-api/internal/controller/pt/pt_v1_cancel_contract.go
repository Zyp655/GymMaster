package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) CancelContract(ctx context.Context, req *v1.CancelContractReq) (res *v1.CancelContractRes, err error) {
	return c.svc.CancelContract(ctx, req.ID, req.CancelContractInput)
}
