package checkin

import (
	"context"

	v1 "gymmaster-api/api/checkin/v1"
)

func (c *ControllerV1) MemberQR(ctx context.Context, req *v1.MemberQRReq) (res *v1.MemberQRRes, err error) {
	return c.checkinService.MemberQR(ctx, req.ID)
}
