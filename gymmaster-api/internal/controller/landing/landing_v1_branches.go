package landing

import (
	"context"

	v1 "gymmaster-api/api/landing/v1"
)

func (c *ControllerV1) Branches(ctx context.Context, req *v1.BranchesReq) (res *v1.BranchesRes, err error) {
	items, err := c.svc.Branches(ctx)
	if err != nil {
		return nil, err
	}
	out := v1.BranchesRes(items)
	return &out, nil
}
