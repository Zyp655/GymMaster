package ptpackage

import (
	"context"

	v1 "gymmaster-api/api/ptpackage/v1"
)

func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	return c.svc.Create(ctx, req.PTPackageInput)
}
