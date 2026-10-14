package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) CreatePortalAccount(ctx context.Context, req *v1.CreatePortalAccountReq) (res *v1.CreatePortalAccountRes, err error) {
	return c.memberService.CreatePortalAccount(ctx, req.ID, req.Email, req.Password)
}
