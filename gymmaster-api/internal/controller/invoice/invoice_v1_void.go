package invoice

import (
	"context"

	v1 "gymmaster-api/api/invoice/v1"
)

func (c *ControllerV1) Void(ctx context.Context, req *v1.VoidReq) (res *v1.VoidRes, err error) {
	if err = c.svc.Void(ctx, req.ID, req.Reason); err != nil {
		return nil, err
	}
	return c.svc.Get(ctx, req.ID)
}
