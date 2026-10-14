package notification

import (
	"context"

	v1 "gymmaster-api/api/notification/v1"
)

func (c *ControllerV1) UnreadCount(ctx context.Context, req *v1.UnreadCountReq) (res *v1.UnreadCountRes, err error) {
	return c.notificationService.UnreadCount(ctx)
}

func (c *MemberControllerV1) MemberUnreadCount(ctx context.Context, req *v1.MemberUnreadCountReq) (res *v1.MemberUnreadCountRes, err error) {
	return c.notificationService.UnreadCount(ctx)
}
