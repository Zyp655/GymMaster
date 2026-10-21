package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// ---- Đối tượng trả về ----------------------------------------------------------

// CheckinMember là thông tin hội viên hiển thị ở màn hình quầy.
type CheckinMember struct {
	ID        uint   `json:"id"`
	Code      string `json:"code"`
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
	Phone     string `json:"phone"`
	Status    string `json:"status" dc:"ACTIVE | INACTIVE | BANNED"`
}

// CheckinSubscription là gói được dùng để vào (hoặc gói liên quan khi bị từ chối).
type CheckinSubscription struct {
	ID                uint   `json:"id"`
	PlanName          string `json:"plan_name"`
	Type              string `json:"type"     dc:"TIME | SESSION"`
	Status            string `json:"status"   dc:"ACTIVE | FROZEN"`
	EndDate           string `json:"end_date" dc:"YYYY-MM-DD"`
	DaysLeft          int    `json:"days_left" dc:"Số ngày còn lại đến hết end_date (0 = hết hạn hôm nay)"`
	RemainingSessions *int   `json:"remaining_sessions" dc:"Số buổi còn lại SAU lần check-in này (gói SESSION); null với gói TIME"`
}

// CheckinRef là bản ghi check-in vừa tạo (hoặc lần check-in trước khi bị DUPLICATE).
type CheckinRef struct {
	ID          uint      `json:"id"`
	CheckedInAt time.Time `json:"checked_in_at"`
	BranchID    uint      `json:"branch_id"`
	BranchName  string    `json:"branch_name,omitempty"`
}

// ScanResult là kết quả quét QR / check-in thủ công (HTTP 200 cho cả ALLOWED và DENIED).
type ScanResult struct {
	Result          string               `json:"result"       dc:"ALLOWED | DENIED"`
	DenyReason      string               `json:"deny_reason"  dc:"Rỗng khi ALLOWED"`
	DenyMessage     string               `json:"deny_message" dc:"Thông điệp tiếng Việt cho lễ tân"`
	Member          *CheckinMember       `json:"member"       dc:"null khi QR hỏng không xác định được hội viên"`
	Subscription    *CheckinSubscription `json:"subscription"`
	Checkin         *CheckinRef          `json:"checkin"          dc:"Bản ghi đã lưu; null khi không ghi DB (QR hỏng hoàn toàn)"`
	PreviousCheckin *CheckinRef          `json:"previous_checkin" dc:"Lần check-in trước (chỉ khi DUPLICATE)"`
	Warnings        []string             `json:"warnings"`
}

// CheckinItem là một dòng lịch sử ra vào.
type CheckinItem struct {
	ID              uint       `json:"id"`
	MemberID        uint       `json:"member_id"`
	MemberCode      string     `json:"member_code"`
	MemberName      string     `json:"member_name"`
	MemberPhone     string     `json:"member_phone"`
	MemberAvatarURL string     `json:"member_avatar_url"`
	BranchID        uint       `json:"branch_id"`
	BranchName      string     `json:"branch_name"`
	SubscriptionID  *uint      `json:"subscription_id"`
	PlanName        *string    `json:"plan_name"`
	PlanType        *string    `json:"plan_type"`
	CheckedInAt     time.Time  `json:"checked_in_at"`
	CheckedOutAt    *time.Time `json:"checked_out_at"`
	Method          string     `json:"method"       dc:"QR | MANUAL | DEVICE"`
	Result          string     `json:"result"       dc:"ALLOWED | DENIED"`
	DenyReason      string     `json:"deny_reason"`
	DenyMessage     string     `json:"deny_message"`
	StaffID         *uint      `json:"staff_id"`
	StaffName       *string    `json:"staff_name"`
	Note            string     `json:"note"`
	DeviceID        *uint      `json:"device_id,omitempty"`
	DayPassID       *uint      `json:"day_pass_id,omitempty"     dc:"Lượt vào bằng vé ngày / vé khách (DP-...)"`
	TrialPassID     *uint      `json:"trial_pass_id,omitempty"   dc:"Lượt vào bằng vé tập thử (TP-...)"`
}

// PresentItem là hội viên đang trong phòng tập.
type PresentItem struct {
	CheckinID       uint      `json:"checkin_id"`
	MemberID        uint      `json:"member_id"`
	MemberCode      string    `json:"member_code"`
	MemberName      string    `json:"member_name"`
	MemberPhone     string    `json:"member_phone"`
	MemberAvatarURL string    `json:"member_avatar_url"`
	BranchID        uint      `json:"branch_id"`
	BranchName      string    `json:"branch_name"`
	PlanName        *string   `json:"plan_name"`
	CheckedInAt     time.Time `json:"checked_in_at"`
	Minutes         int       `json:"minutes" dc:"Số phút đã ở trong phòng tập"`
}

// ---- Request -------------------------------------------------------------------

type MemberQRReq struct {
	g.Meta `path:"/members/{id}/qr" method:"get" tags:"Check-in" summary:"Lấy mã QR động của hội viên (in/hiển thị tại quầy)" perm:"checkin.scan"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type MemberQRRes struct {
	MemberID   uint      `json:"member_id"`
	Token      string    `json:"token"       dc:"Chuỗi mã hóa vào QR"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTLSeconds int       `json:"ttl_seconds"`
}

type ScanReq struct {
	g.Meta   `path:"/checkins/scan" method:"post" tags:"Check-in" summary:"Quét QR check-in" perm:"checkin.scan"`
	Token    string `json:"token"     v:"required|max-length:200#Vui lòng quét mã QR|Mã QR không hợp lệ"`
	BranchID uint   `json:"branch_id" v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
}
type ScanRes = ScanResult

type ManualReq struct {
	g.Meta   `path:"/checkins/manual" method:"post" tags:"Check-in" summary:"Check-in thủ công (hội viên quên điện thoại)" perm:"checkin.scan"`
	MemberID uint   `json:"member_id" v:"required|min:1#Vui lòng chọn hội viên|Vui lòng chọn hội viên"`
	BranchID uint   `json:"branch_id" v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	Note     string `json:"note"      v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type ManualRes = ScanResult

type CheckoutReq struct {
	g.Meta `path:"/checkins/{id}/checkout" method:"post" tags:"Check-in" summary:"Ghi nhận hội viên ra về" perm:"checkin.scan"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type CheckoutRes = CheckinItem

type PresentReq struct {
	g.Meta   `path:"/checkins/present" method:"get" tags:"Check-in" summary:"Hội viên đang trong phòng tập (hôm nay, chưa checkout)" perm:"checkin.view,checkin.scan"`
	BranchID uint `json:"branch_id" in:"query" dc:"Bỏ trống = mọi chi nhánh trong phạm vi"`
}
type PresentRes []*PresentItem

type ListReq struct {
	g.Meta `path:"/checkins" method:"get" tags:"Check-in" summary:"Lịch sử ra vào" perm:"checkin.view"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query"`
	MemberID uint   `json:"member_id" in:"query"`
	Result   string `json:"result"    in:"query" v:"in:ALLOWED,DENIED#Kết quả không hợp lệ"`
	Method   string `json:"method"    in:"query" v:"in:QR,MANUAL,DEVICE#Phương thức không hợp lệ"`
	From     string `json:"from"      in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"YYYY-MM-DD (giờ VN), tính cả ngày"`
	To       string `json:"to"        in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"YYYY-MM-DD (giờ VN), tính cả ngày"`
}
type ListRes = dto.PageRes[*CheckinItem]

type StatsReq struct {
	g.Meta   `path:"/checkins/stats" method:"get" tags:"Check-in" summary:"Thống kê ra vào trong ngày (dashboard quầy)" perm:"checkin.view,checkin.scan"`
	BranchID uint   `json:"branch_id" in:"query" dc:"Bỏ trống = mọi chi nhánh trong phạm vi"`
	Date     string `json:"date"      in:"query" v:"date-format:Y-m-d#Ngày phải có dạng YYYY-MM-DD" dc:"YYYY-MM-DD, mặc định hôm nay"`
}

type HourStat struct {
	Hour    int   `json:"hour" dc:"0-23 (giờ VN)"`
	Allowed int64 `json:"allowed"`
	Denied  int64 `json:"denied"`
}

type ReasonStat struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Count   int64  `json:"count"`
}

type StatsRes struct {
	Date          string        `json:"date"`
	Total         int64         `json:"total"`
	Allowed       int64         `json:"allowed"`
	Denied        int64         `json:"denied"`
	UniqueMembers int64         `json:"unique_members" dc:"Số hội viên khác nhau được vào"`
	Present       int64         `json:"present"        dc:"Đang trong phòng tập (vào trong ngày, chưa checkout)"`
	ByHour        []*HourStat   `json:"by_hour"        dc:"Đủ 24 phần tử, giờ 0..23"`
	DenyReasons   []*ReasonStat `json:"deny_reasons"`
}
