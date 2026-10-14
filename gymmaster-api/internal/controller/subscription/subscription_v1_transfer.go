package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) Transfer(ctx context.Context, req *v1.TransferReq) (res *v1.TransferRes, err error) {
	return c.svc.Transfer(ctx, req.ID, req.TransferInput)
}
