package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) ResetQR(ctx context.Context, req *v1.ResetQRReq) (res *v1.ResetQRRes, err error) {
	return nil, c.memberService.ResetQR(ctx, req.ID)
}
