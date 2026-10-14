package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) FreezeList(ctx context.Context, req *v1.FreezeListReq) (res *v1.FreezeListRes, err error) {
	return c.svc.ListFreezes(ctx, req)
}
