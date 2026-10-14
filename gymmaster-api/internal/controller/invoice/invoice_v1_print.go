package invoice

import (
	"context"

	v1 "gymmaster-api/api/invoice/v1"
)

func (c *ControllerV1) Print(ctx context.Context, req *v1.PrintReq) (res *v1.PrintRes, err error) {
	return c.svc.Print(ctx, req.ID)
}
