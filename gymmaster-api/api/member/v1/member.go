package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// SubscriptionSummary tóm tắt gói đang hiệu lực (ACTIVE/FROZEN, ngày hết hạn xa nhất) của hội viên.
type SubscriptionSummary struct {
	ID                uint   `json:"id"`
	PlanID            uint   `json:"plan_id"`
	PlanName          string `json:"plan_name"`
	Status            string `json:"status"     dc:"ACTIVE | FROZEN"`
	StartDate         string `json:"start_date" dc:"YYYY-MM-DD"`
	EndDate           string `json:"end_date"   dc:"YYYY-MM-DD"`
	RemainingSessions *int   `json:"remaining_sessions" dc:"Số buổi còn lại (gói theo buổi), null với gói theo thời gian"`
	DaysLeft          int    `json:"days_left"  dc:"Số ngày còn lại tính đến hết end_date (0 = hết hạn hôm nay)"`
}

// MemberItem là một dòng trong danh sách hội viên.
type MemberItem struct {
	ID                 uint                 `json:"id"`
	MemberCode         string               `json:"member_code"`
	FullName           string               `json:"full_name"`
	Gender             string               `json:"gender"`
	Dob                *string              `json:"dob"`
	Phone              string               `json:"phone"`
	Email              string               `json:"email"`
	AvatarURL          string               `json:"avatar_url"`
	HomeBranchID       uint                 `json:"home_branch_id"`
	HomeBranchName     string               `json:"home_branch_name"`
	Source             string               `json:"source"`
	Status             string               `json:"status"`
	HasPortalAccount   bool                 `json:"has_portal_account"`
	ActiveSubscription *SubscriptionSummary `json:"active_subscription"`
	CreatedAt          time.Time            `json:"created_at"`
}

type BranchRef struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type PortalAccount struct {
	UserID      uint       `json:"user_id"`
	Email       string     `json:"email"`
	Status      string     `json:"status" dc:"ACTIVE | LOCKED"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

type MemberStats struct {
	TotalCheckins int64      `json:"total_checkins" dc:"Số lần check-in thành công"`
	LastCheckinAt *time.Time `json:"last_checkin_at"`
}

// Member là hồ sơ chi tiết hội viên.
type Member struct {
	ID                    uint                 `json:"id"`
	MemberCode            string               `json:"member_code"`
	FullName              string               `json:"full_name"`
	Gender                string               `json:"gender"`
	Dob                   *string              `json:"dob"`
	Phone                 string               `json:"phone"`
	Email                 string               `json:"email"`
	IDCard                string               `json:"id_card"`
	Address               string               `json:"address"`
	AvatarURL             string               `json:"avatar_url"`
	EmergencyContactName  string               `json:"emergency_contact_name"`
	EmergencyContactPhone string               `json:"emergency_contact_phone"`
	HomeBranchID          uint                 `json:"home_branch_id"`
	HomeBranch            *BranchRef           `json:"home_branch"`
	Source                string               `json:"source"`
	Status                string               `json:"status"`
	Note                  string               `json:"note"`
	BookingBlockedUntil   *time.Time           `json:"booking_blocked_until"`
	PortalAccount         *PortalAccount       `json:"portal_account"`
	LatestMetric          *BodyMetric          `json:"latest_metric"`
	ActiveSubscription    *SubscriptionSummary `json:"active_subscription"`
	Stats                 MemberStats          `json:"stats"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

// MemberInput là body dùng chung cho tạo/sửa hội viên.
type MemberInput struct {
	FullName              string `json:"full_name"               v:"required|max-length:150#Vui lòng nhập họ tên|Họ tên tối đa 150 ký tự"`
	Gender                string `json:"gender"                  v:"in:MALE,FEMALE,OTHER#Giới tính không hợp lệ" d:"OTHER"`
	Dob                   string `json:"dob"                     v:"date-format:Y-m-d#Ngày sinh phải có dạng YYYY-MM-DD" dc:"YYYY-MM-DD, tùy chọn"`
	Phone                 string `json:"phone"                   v:"required#Vui lòng nhập số điện thoại" dc:"SĐT Việt Nam, chuẩn hóa về 0xxxxxxxxx"`
	Email                 string `json:"email"                   v:"email|max-length:150#Email không hợp lệ|Email tối đa 150 ký tự"`
	IDCard                string `json:"id_card"                 v:"regex:^[0-9]{9,12}$#Số CCCD/CMND gồm 9-12 chữ số"`
	Address               string `json:"address"                 v:"max-length:255#Địa chỉ tối đa 255 ký tự"`
	AvatarURL             string `json:"avatar_url"              v:"max-length:500#URL ảnh quá dài"`
	EmergencyContactName  string `json:"emergency_contact_name"  v:"max-length:150#Tên người liên hệ tối đa 150 ký tự"`
	EmergencyContactPhone string `json:"emergency_contact_phone" dc:"SĐT người liên hệ khẩn cấp, tùy chọn"`
	HomeBranchID          uint   `json:"home_branch_id"          v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	Source                string `json:"source"                  v:"in:WALK_IN,REFERRAL,FACEBOOK,WEBSITE,OTHER#Nguồn không hợp lệ" d:"WALK_IN"`
	Status                string `json:"status"                  v:"in:ACTIVE,INACTIVE,BANNED#Trạng thái không hợp lệ" d:"ACTIVE"`
	Note                  string `json:"note"                    v:"max-length:2000#Ghi chú tối đa 2000 ký tự"`
}

type ListReq struct {
	g.Meta `path:"/members" method:"get" tags:"Hội viên" summary:"Danh sách hội viên" perm:"member.view"`
	dto.PageReq
	BranchID           uint   `json:"branch_id"           in:"query" dc:"Chi nhánh chính"`
	Status             string `json:"status"              in:"query" v:"in:ACTIVE,INACTIVE,BANNED#Trạng thái không hợp lệ"`
	Gender             string `json:"gender"              in:"query" v:"in:MALE,FEMALE,OTHER#Giới tính không hợp lệ"`
	Source             string `json:"source"              in:"query" v:"in:WALK_IN,REFERRAL,FACEBOOK,WEBSITE,OTHER#Nguồn không hợp lệ"`
	SubscriptionStatus string `json:"subscription_status" in:"query" v:"in:active,expiring,none#Tình trạng gói không hợp lệ" dc:"active | expiring (≤7 ngày) | none"`
}
type ListRes = dto.PageRes[*MemberItem]

type OptionsReq struct {
	g.Meta   `path:"/members/options" method:"get" tags:"Hội viên" summary:"Tìm nhanh hội viên (tối đa 20) cho ô chọn" perm:"member.view,subscription.sell,checkin.scan,booking.manage,pt.sell,pos.sell"`
	Keyword  string `json:"keyword"   in:"query" dc:"Tên, SĐT hoặc mã hội viên"`
	BranchID uint   `json:"branch_id" in:"query" dc:"Chỉ lấy hội viên thuộc chi nhánh này"`
}
type MemberOption struct {
	ID         uint   `json:"id"`
	MemberCode string `json:"member_code"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	AvatarURL  string `json:"avatar_url"`
	Status     string `json:"status"`
}
type OptionsRes []*MemberOption

type GetReq struct {
	g.Meta `path:"/members/{id}" method:"get" tags:"Hội viên" summary:"Hồ sơ hội viên" perm:"member.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = Member

type CreateReq struct {
	g.Meta `path:"/members" method:"post" tags:"Hội viên" summary:"Thêm hội viên" perm:"member.create"`
	MemberInput
}
type CreateRes = Member

type UpdateReq struct {
	g.Meta `path:"/members/{id}" method:"put" tags:"Hội viên" summary:"Cập nhật hội viên" perm:"member.update"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	MemberInput
}
type UpdateRes = Member

type DeleteReq struct {
	g.Meta `path:"/members/{id}" method:"delete" tags:"Hội viên" summary:"Xóa hội viên (xóa mềm)" perm:"member.delete"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}

type CreatePortalAccountReq struct {
	g.Meta   `path:"/members/{id}/portal-account" method:"post" tags:"Hội viên" summary:"Cấp (hoặc mở lại) tài khoản cổng hội viên" perm:"member.update"`
	ID       uint   `json:"id"       in:"path" v:"required|min:1"`
	Email    string `json:"email"    v:"required|email|max-length:150#Vui lòng nhập email|Email không hợp lệ|Email tối đa 150 ký tự"`
	Password string `json:"password" v:"required|length:8,72#Vui lòng nhập mật khẩu|Mật khẩu từ 8 đến 72 ký tự"`
}
type CreatePortalAccountRes = PortalAccount

type LockPortalAccountReq struct {
	g.Meta `path:"/members/{id}/portal-account" method:"delete" tags:"Hội viên" summary:"Khóa tài khoản cổng hội viên" perm:"member.update"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type LockPortalAccountRes struct{}

type ResetQRReq struct {
	g.Meta `path:"/members/{id}/reset-qr" method:"post" tags:"Hội viên" summary:"Đặt lại bí mật QR (vô hiệu mã QR cũ)" perm:"member.update"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ResetQRRes struct{}
