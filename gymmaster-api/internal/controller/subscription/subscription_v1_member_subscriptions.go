package subscription

import (
	"context"

	v1 "gymmaster-api/api/subscription/v1"
)

func (c *ControllerV1) MemberSubscriptions(ctx context.Context, req *v1.MemberSubscriptionsReq) (res *v1.MemberSubscriptionsRes, err error) {
	items, err := c.svc.ListByMember(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	out := v1.MemberSubscriptionsRes(items)
	return &out, nil
}
