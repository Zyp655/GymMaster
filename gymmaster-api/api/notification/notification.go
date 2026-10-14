// =================================================================================
// Interface controller – có thể sinh lại bằng `make ctrl` (gf gen ctrl).
// =================================================================================

package notification

import (
	"context"

	v1 "gymmaster-api/api/notification/v1"
)

// INotificationV1: thông báo của nhân viên (/v1/admin/notifications).
type INotificationV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	UnreadCount(ctx context.Context, req *v1.UnreadCountReq) (res *v1.UnreadCountRes, err error)
	Read(ctx context.Context, req *v1.ReadReq) (res *v1.ReadRes, err error)
	ReadAll(ctx context.Context, req *v1.ReadAllReq) (res *v1.ReadAllRes, err error)
}

// IMemberNotificationV1: thông báo của hội viên (/v1/member/notifications).
type IMemberNotificationV1 interface {
	MemberList(ctx context.Context, req *v1.MemberListReq) (res *v1.MemberListRes, err error)
	MemberUnreadCount(ctx context.Context, req *v1.MemberUnreadCountReq) (res *v1.MemberUnreadCountRes, err error)
	MemberRead(ctx context.Context, req *v1.MemberReadReq) (res *v1.MemberReadRes, err error)
	MemberReadAll(ctx context.Context, req *v1.MemberReadAllReq) (res *v1.MemberReadAllRes, err error)
}
