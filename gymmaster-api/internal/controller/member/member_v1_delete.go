package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	return nil, c.memberService.Delete(ctx, req.ID)
}
