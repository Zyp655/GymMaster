package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) Upgrade(ctx context.Context, req *v1.UpgradeReq) (res *v1.UpgradeRes, err error) {
	return c.svc.Upgrade(ctx, req.ID, req.UpgradeInput)
}
