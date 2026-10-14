package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type Notification struct {
	ID        uint       `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Link      string     `json:"link"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListFilter: tham số danh sách dùng chung cho nhân viên và hội viên.
type ListFilter struct {
	dto.PageReq
	Unread bool   `json:"unread" in:"query" dc:"true = chỉ thông báo chưa đọc"`
	Type   string `json:"type"   in:"query" dc:"Lọc theo loại thông báo"`
}

type UnreadCount struct {
	Count int64 `json:"count"`
}

type ReadAllResult struct {
	Updated int64 `json:"updated"`
}

// ---- Nhân viên (/v1/admin) – mọi tài khoản đã đăng nhập, không cần quyền riêng ----

type ListReq struct {
	g.Meta `path:"/notifications" method:"get" tags:"Thông báo" summary:"Thông báo của tôi (nhân viên)"`
	ListFilter
}
type ListRes = dto.PageRes[*Notification]

type UnreadCountReq struct {
	g.Meta `path:"/notifications/unread-count" method:"get" tags:"Thông báo" summary:"Số thông báo chưa đọc"`
}
type UnreadCountRes = UnreadCount

type ReadReq struct {
	g.Meta `path:"/notifications/{id}/read" method:"post" tags:"Thông báo" summary:"Đánh dấu đã đọc"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ReadRes = Notification

type ReadAllReq struct {
	g.Meta `path:"/notifications/read-all" method:"post" tags:"Thông báo" summary:"Đánh dấu tất cả đã đọc"`
}
type ReadAllRes = ReadAllResult

// ---- Hội viên (/v1/member) – role MEMBER ----

type MemberListReq struct {
	g.Meta `path:"/notifications" method:"get" tags:"Cổng hội viên" summary:"Thông báo của tôi (hội viên)" role:"MEMBER"`
	ListFilter
}
type MemberListRes = dto.PageRes[*Notification]

type MemberUnreadCountReq struct {
	g.Meta `path:"/notifications/unread-count" method:"get" tags:"Cổng hội viên" summary:"Số thông báo chưa đọc" role:"MEMBER"`
}
type MemberUnreadCountRes = UnreadCount

type MemberReadReq struct {
	g.Meta `path:"/notifications/{id}/read" method:"post" tags:"Cổng hội viên" summary:"Đánh dấu đã đọc" role:"MEMBER"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type MemberReadRes = Notification

type MemberReadAllReq struct {
	g.Meta `path:"/notifications/read-all" method:"post" tags:"Cổng hội viên" summary:"Đánh dấu tất cả đã đọc" role:"MEMBER"`
}
type MemberReadAllRes = ReadAllResult
