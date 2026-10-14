package pt

import (
	"context"

	v1 "gymmaster-api/api/pt/v1"
)

func (c *ControllerV1) MemberContracts(ctx context.Context, req *v1.MemberContractsReq) (res *v1.MemberContractsRes, err error) {
	items, err := c.svc.ListByMember(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	out := v1.MemberContractsRes(items)
	return &out, nil
}
