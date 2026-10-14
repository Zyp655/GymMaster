package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) LockPortalAccount(ctx context.Context, req *v1.LockPortalAccountReq) (res *v1.LockPortalAccountRes, err error) {
	return nil, c.memberService.LockPortalAccount(ctx, req.ID)
}
