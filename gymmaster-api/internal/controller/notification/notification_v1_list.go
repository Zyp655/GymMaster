package notification

import (
	"context"

	v1 "gymmaster-api/api/notification/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	return c.notificationService.List(ctx, req.ListFilter)
}

func (c *MemberControllerV1) MemberList(ctx context.Context, req *v1.MemberListReq) (res *v1.MemberListRes, err error) {
	return c.notificationService.List(ctx, req.ListFilter)
}
