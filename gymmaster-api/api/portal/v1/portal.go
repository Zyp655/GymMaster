// Package v1 định nghĩa API cổng hội viên (/v1/member/*). Mọi API khai báo role:"MEMBER";
// hội viên được xác định từ tài khoản đăng nhập (members.user_id), không nhận member_id từ client.
package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	checkinv1 "gymmaster-api/api/checkin/v1"
	classv1 "gymmaster-api/api/class/v1"
	invoicev1 "gymmaster-api/api/invoice/v1"
	memberv1 "gymmaster-api/api/member/v1"
	ptv1 "gymmaster-api/api/pt/v1"
	subv1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/dto"
)

// ---- Đối tượng ------------------------------------------------------------------

// Branch: chi nhánh (kèm giờ mở cửa) – dùng cho hồ sơ và bộ chọn chi nhánh xem lịch lớp.
type Branch struct {
	ID        uint   `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Phone     string `json:"phone"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsHome    bool   `json:"is_home" dc:"Chi nhánh chính của hội viên"`
}

// Subscription: gói tập của hội viên + hạn mức bảo lưu của gói + các yêu cầu bảo lưu.
type Subscription struct {
	*subv1.Subscription
	IncludesClasses bool      `json:"includes_classes"`
	MaxFreezeDays   int       `json:"max_freeze_days"`
	MaxFreezeTimes  int       `json:"max_freeze_times"`
	Freezes         []*Freeze `json:"freezes" dc:"Mới nhất trước"`
}

// Freeze: yêu cầu bảo lưu (rút gọn cho hội viên).
type Freeze struct {
	ID             uint       `json:"id"`
	SubscriptionID uint       `json:"subscription_id"`
	StartDate      string     `json:"start_date"`
	EndDate        string     `json:"end_date"`
	Days           int        `json:"days"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status" dc:"PENDING | APPROVED | REJECTED | ENDED | CANCELLED"`
	ReviewNote     string     `json:"review_note"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Profile: hồ sơ hội viên + chi nhánh chính + gói hiện tại.
type Profile struct {
	Member *memberv1.Member `json:"member" dc:"note (ghi chú nội bộ) luôn rỗng"`
	Branch *Branch          `json:"branch"`
	// CurrentSubscription: gói ACTIVE/FROZEN đang dùng (start_date ≤ hôm nay, hết hạn xa nhất);
	// không có thì gói ACTIVE sắp bắt đầu sớm nhất; null nếu không có gói nào.
	CurrentSubscription *Subscription `json:"current_subscription"`
	// HasValidSubscription: có gói ACTIVE đang trong hạn (gói theo buổi còn buổi) – vào cửa được.
	HasValidSubscription bool `json:"has_valid_subscription"`
}

// ScheduleItem: buổi lớp kèm trạng thái đặt chỗ của chính hội viên.
type ScheduleItem struct {
	*classv1.ClassSchedule
	MyBooking      *classv1.ClassBooking `json:"my_booking"      dc:"Lượt đặt còn hiệu lực của hội viên (BOOKED/WAITLIST/ATTENDED/NO_SHOW) hoặc null"`
	CancelDeadline time.Time             `json:"cancel_deadline" dc:"Hội viên tự hủy được trước thời điểm này"`
	Bookable       bool                  `json:"bookable"        dc:"Trong khoảng được đặt trước (booking.advance_days), chưa bắt đầu và hội viên không bị khóa đặt lớp"`
}

// Booking: lượt đặt lớp của hội viên kèm hạn tự hủy.
type Booking struct {
	*classv1.ClassBooking
	CancelDeadline time.Time `json:"cancel_deadline"`
	CanCancel      bool      `json:"can_cancel" dc:"BOOKED/WAITLIST, lớp chưa hủy và chưa quá hạn hủy"`
}

// PTSession: buổi PT kèm hạn tự hủy.
type PTSession struct {
	*ptv1.Session
	CancelDeadline time.Time `json:"cancel_deadline"`
	CanCancel      bool      `json:"can_cancel"`
}

// Slot: một khung giờ 30 phút trong ngày.
type Slot struct {
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	Available bool      `json:"available"`
	Reason    string    `json:"reason" dc:"PAST | TRAINER_BUSY | MEMBER_BUSY | OUT_OF_CONTRACT | ''"`
}

// ---- Hồ sơ ----------------------------------------------------------------------

type GetProfileReq struct {
	g.Meta `path:"/profile" method:"get" tags:"Cổng hội viên" summary:"Hồ sơ của tôi (kèm chi nhánh, gói hiện tại)" role:"MEMBER"`
}
type GetProfileRes = Profile

type UpdateProfileReq struct {
	g.Meta                `path:"/profile" method:"put" tags:"Cổng hội viên" summary:"Cập nhật hồ sơ (email, địa chỉ, liên hệ khẩn cấp, ảnh đại diện)" role:"MEMBER"`
	Email                 string `json:"email"                   v:"email|max-length:150#Email không hợp lệ|Email tối đa 150 ký tự"`
	Address               string `json:"address"                 v:"max-length:255#Địa chỉ tối đa 255 ký tự"`
	EmergencyContactName  string `json:"emergency_contact_name"  v:"max-length:150#Tên người liên hệ tối đa 150 ký tự"`
	EmergencyContactPhone string `json:"emergency_contact_phone" v:"max-length:20#Số điện thoại tối đa 20 ký tự"`
	AvatarURL             string `json:"avatar_url"              v:"max-length:500#Đường dẫn ảnh quá dài"`
}
type UpdateProfileRes = Profile

type UploadReq struct {
	g.Meta `path:"/uploads" method:"post" mime:"multipart/form-data" tags:"Cổng hội viên" summary:"Tải ảnh đại diện (jpg/png/webp, ≤ 2 MB)" role:"MEMBER"`
	File   *ghttp.UploadFile `json:"file" type:"file" v:"required#Vui lòng chọn tệp ảnh"`
}
type UploadRes struct {
	URL string `json:"url"`
}

// ---- QR -------------------------------------------------------------------------

type QRReq struct {
	g.Meta `path:"/qr" method:"get" tags:"Cổng hội viên" summary:"Mã QR vào cửa (động)" role:"MEMBER"`
}
type QRRes struct {
	Token                string    `json:"token"`
	ExpiresAt            time.Time `json:"expires_at"`
	TTL                  int       `json:"ttl"         dc:"Thời gian sống (giây)"`
	ServerTime           time.Time `json:"server_time" dc:"Giờ máy chủ lúc cấp – FE dùng để bù lệch đồng hồ"`
	HasValidSubscription bool      `json:"has_valid_subscription"`
}

// ---- Gói tập --------------------------------------------------------------------

type ListSubscriptionsReq struct {
	g.Meta `path:"/subscriptions" method:"get" tags:"Cổng hội viên" summary:"Gói tập của tôi (mới nhất trước)" role:"MEMBER"`
}
type ListSubscriptionsRes []*Subscription

type RequestFreezeReq struct {
	g.Meta `path:"/subscriptions/{id}/freezes" method:"post" tags:"Cổng hội viên" summary:"Yêu cầu bảo lưu gói" role:"MEMBER"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	subv1.FreezeInput
}
type RequestFreezeRes = Freeze

// ---- Check-in -------------------------------------------------------------------

type ListCheckinsReq struct {
	g.Meta `path:"/checkins" method:"get" tags:"Cổng hội viên" summary:"Lịch sử ra vào của tôi" role:"MEMBER"`
	dto.PageReq
}
type ListCheckinsRes = dto.PageRes[*checkinv1.CheckinItem]

// ---- Lớp nhóm -------------------------------------------------------------------

type ListBranchesReq struct {
	g.Meta `path:"/branches" method:"get" tags:"Cổng hội viên" summary:"Chi nhánh tôi được dùng (theo phạm vi gói)" role:"MEMBER"`
}
type ListBranchesRes []*Branch

type ListClassesReq struct {
	g.Meta   `path:"/classes" method:"get" tags:"Cổng hội viên" summary:"Lịch lớp có thể đặt (kèm trạng thái đặt của tôi)" role:"MEMBER"`
	BranchID uint   `json:"branch_id" in:"query" dc:"Mặc định: chi nhánh chính (hoặc chi nhánh đầu tiên trong phạm vi)"`
	From     string `json:"from"      in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"Mặc định hôm nay"`
	To       string `json:"to"        in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"Mặc định from + booking.advance_days; tối đa 31 ngày"`
}
type ListClassesRes struct {
	BranchID            uint            `json:"branch_id"`
	From                string          `json:"from"`
	To                  string          `json:"to"`
	AdvanceDays         int             `json:"advance_days"          dc:"booking.advance_days"`
	CancelBeforeMinutes int             `json:"cancel_before_minutes" dc:"booking.cancel_before_minutes"`
	BookingBlockedUntil *time.Time      `json:"booking_blocked_until" dc:"Hội viên bị khóa đặt lớp đến thời điểm này (vắng mặt nhiều lần); null = không bị khóa"`
	Items               []*ScheduleItem `json:"items"`
}

type BookClassReq struct {
	g.Meta     `path:"/classes/{scheduleId}/book" method:"post" tags:"Cổng hội viên" summary:"Đặt chỗ lớp (hết chỗ → danh sách chờ)" role:"MEMBER"`
	ScheduleID uint `json:"scheduleId" in:"path" v:"required|min:1"`
}
type BookClassRes = Booking

type ListBookingsReq struct {
	g.Meta `path:"/bookings" method:"get" tags:"Cổng hội viên" summary:"Lớp tôi đã đặt" role:"MEMBER"`
	From   string `json:"from" in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"Mặc định 30 ngày trước"`
	To     string `json:"to"   in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"Mặc định 30 ngày tới; tối đa 1 năm"`
}
type ListBookingsRes []*Booking

type CancelBookingReq struct {
	g.Meta `path:"/bookings/{id}/cancel" method:"post" tags:"Cổng hội viên" summary:"Hủy lượt đặt lớp (trước hạn hủy)" role:"MEMBER"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type CancelBookingRes = Booking

// ---- PT -------------------------------------------------------------------------

type ListPTContractsReq struct {
	g.Meta `path:"/pt-contracts" method:"get" tags:"Cổng hội viên" summary:"Hợp đồng PT của tôi" role:"MEMBER"`
}
type ListPTContractsRes []*ptv1.Contract

type ListPTSessionsReq struct {
	g.Meta `path:"/pt-sessions" method:"get" tags:"Cổng hội viên" summary:"Buổi PT của tôi" role:"MEMBER"`
	From   string `json:"from" in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"Mặc định 30 ngày trước"`
	To     string `json:"to"   in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"Mặc định 30 ngày tới; tối đa 62 ngày"`
}
type ListPTSessionsRes struct {
	CancelBeforeMinutes int          `json:"cancel_before_minutes" dc:"pt.cancel_before_minutes"`
	Items               []*PTSession `json:"items"`
}

type BookPTSessionReq struct {
	g.Meta     `path:"/pt-sessions" method:"post" tags:"Cổng hội viên" summary:"Đặt buổi PT" role:"MEMBER"`
	ContractID uint   `json:"contract_id" v:"required|min:1#Vui lòng chọn hợp đồng PT|Vui lòng chọn hợp đồng PT"`
	StartAt    string `json:"start_at"    v:"required#Vui lòng chọn giờ bắt đầu" dc:"RFC3339 hoặc YYYY-MM-DDTHH:mm (giờ VN)"`
}
type BookPTSessionRes = PTSession

type CancelPTSessionReq struct {
	g.Meta `path:"/pt-sessions/{id}/cancel" method:"post" tags:"Cổng hội viên" summary:"Hủy buổi PT (trước hạn hủy)" role:"MEMBER"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Reason string `json:"reason" v:"max-length:255#Lý do tối đa 255 ký tự"`
}
type CancelPTSessionRes = PTSession

type TrainerAvailabilityReq struct {
	g.Meta `path:"/pt-contracts/{id}/trainer-availability" method:"get" tags:"Cổng hội viên" summary:"Khung giờ trống của HLV trong ngày" role:"MEMBER"`
	ID     uint   `json:"id"   in:"path"  v:"required|min:1"`
	Date   string `json:"date" in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn ngày|Ngày phải có dạng YYYY-MM-DD"`
}
type TrainerAvailabilityRes struct {
	Date           string  `json:"date"`
	ContractID     uint    `json:"contract_id"`
	TrainerID      uint    `json:"trainer_id"`
	TrainerName    string  `json:"trainer_name"`
	BranchID       uint    `json:"branch_id"`
	BranchName     string  `json:"branch_name"`
	OpenTime       string  `json:"open_time"`
	CloseTime      string  `json:"close_time"`
	SessionMinutes int     `json:"session_minutes"`
	Slots          []*Slot `json:"slots" dc:"Bước 30 phút, buổi phải kết thúc trước giờ đóng cửa"`
}

// ---- Chỉ số cơ thể / hóa đơn ----------------------------------------------------

type ListBodyMetricsReq struct {
	g.Meta `path:"/body-metrics" method:"get" tags:"Cổng hội viên" summary:"Chỉ số cơ thể của tôi (tăng dần theo thời gian đo)" role:"MEMBER"`
}
type ListBodyMetricsRes = memberv1.ListBodyMetricsRes

type ListInvoicesReq struct {
	g.Meta `path:"/invoices" method:"get" tags:"Cổng hội viên" summary:"Hóa đơn của tôi" role:"MEMBER"`
	dto.PageReq
}
type ListInvoicesRes = invoicev1.ListRes

type GetInvoiceReq struct {
	g.Meta `path:"/invoices/{id}" method:"get" tags:"Cổng hội viên" summary:"Chi tiết hóa đơn của tôi" role:"MEMBER"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetInvoiceRes = invoicev1.InvoiceDetail
