package notification

import (
	"context"

	v1 "gymmaster-api/api/notification/v1"
)

func (c *ControllerV1) ReadAll(ctx context.Context, req *v1.ReadAllReq) (res *v1.ReadAllRes, err error) {
	return c.notificationService.MarkAllRead(ctx)
}

func (c *MemberControllerV1) MemberReadAll(ctx context.Context, req *v1.MemberReadAllReq) (res *v1.MemberReadAllRes, err error) {
	return c.notificationService.MarkAllRead(ctx)
}
