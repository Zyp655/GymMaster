package notification

import (
	"context"

	v1 "gymmaster-api/api/notification/v1"
)

func (c *ControllerV1) Read(ctx context.Context, req *v1.ReadReq) (res *v1.ReadRes, err error) {
	return c.notificationService.MarkRead(ctx, req.ID)
}

func (c *MemberControllerV1) MemberRead(ctx context.Context, req *v1.MemberReadReq) (res *v1.MemberReadRes, err error) {
	return c.notificationService.MarkRead(ctx, req.ID)
}
