package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// ClassBooking là một lượt đặt chỗ, kèm thông tin hội viên và buổi lớp.
type ClassBooking struct {
	ID               uint       `json:"id"`
	ScheduleID       uint       `json:"schedule_id"`
	MemberID         uint       `json:"member_id"`
	MemberCode       string     `json:"member_code"`
	MemberName       string     `json:"member_name"`
	MemberPhone      string     `json:"member_phone"`
	SubscriptionID   *uint      `json:"subscription_id"`
	Status           string     `json:"status"`
	WaitlistPosition *int       `json:"waitlist_position"`
	Source           string     `json:"source"`
	BookedAt         time.Time  `json:"booked_at"`
	CancelledAt      *time.Time `json:"cancelled_at"`
	AttendedAt       *time.Time `json:"attended_at"`
	ClassTypeID      uint       `json:"class_type_id"`
	ClassTypeName    string     `json:"class_type_name"`
	ClassTypeColor   string     `json:"class_type_color"`
	BranchID         uint       `json:"branch_id"`
	BranchName       string     `json:"branch_name"`
	RoomName         string     `json:"room_name"`
	TrainerID        uint       `json:"trainer_id"`
	TrainerName      string     `json:"trainer_name"`
	StartAt          time.Time  `json:"start_at"`
	EndAt            time.Time  `json:"end_at"`
	ScheduleStatus   string     `json:"schedule_status"`
}

type ListScheduleBookingsReq struct {
	g.Meta `path:"/class-schedules/{id}/bookings" method:"get" tags:"Lớp nhóm" summary:"Danh sách đặt chỗ / waitlist / điểm danh của buổi lớp" perm:"class.view,booking.manage"`
	ID     uint   `json:"id"     in:"path"  v:"required|min:1"`
	Status string `json:"status" in:"query" v:"in:BOOKED,WAITLIST,CANCELLED,ATTENDED,NO_SHOW#Trạng thái không hợp lệ"`
}
type ListScheduleBookingsRes []*ClassBooking

type CreateBookingReq struct {
	g.Meta   `path:"/class-schedules/{id}/bookings" method:"post" tags:"Lớp nhóm" summary:"Đặt chỗ hộ hội viên tại quầy" perm:"booking.manage"`
	ID       uint `json:"id"        in:"path" v:"required|min:1"`
	MemberID uint `json:"member_id" v:"required|min:1#Vui lòng chọn hội viên|Vui lòng chọn hội viên"`
}
type CreateBookingRes = ClassBooking

type CancelBookingReq struct {
	g.Meta `path:"/class-bookings/{id}/cancel" method:"post" tags:"Lớp nhóm" summary:"Hủy đặt chỗ hộ hội viên" perm:"booking.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type CancelBookingRes = ClassBooking

type ListMemberBookingsReq struct {
	g.Meta   `path:"/class-bookings" method:"get" tags:"Lớp nhóm" summary:"Lịch sử đặt lớp của một hội viên" perm:"class.view,booking.manage"`
	MemberID uint   `json:"member_id" in:"query" v:"required|min:1#Vui lòng chọn hội viên|Vui lòng chọn hội viên"`
	From     string `json:"from"      in:"query" v:"date-format:Y-m-d#Ngày phải có dạng YYYY-MM-DD" dc:"Mặc định: 30 ngày trước"`
	To       string `json:"to"        in:"query" v:"date-format:Y-m-d#Ngày phải có dạng YYYY-MM-DD" dc:"Mặc định: 30 ngày tới"`
}
type ListMemberBookingsRes []*ClassBooking

type AttendanceItem struct {
	BookingID uint   `json:"booking_id" v:"required|min:1#Thiếu booking_id|Thiếu booking_id"`
	Status    string `json:"status"     v:"required|in:ATTENDED,NO_SHOW#Thiếu trạng thái|Trạng thái phải là ATTENDED hoặc NO_SHOW"`
}

type AttendanceReq struct {
	g.Meta `path:"/class-schedules/{id}/attendance" method:"put" tags:"Lớp nhóm" summary:"Điểm danh buổi lớp" perm:"booking.manage"`
	ID     uint              `json:"id"    in:"path" v:"required|min:1"`
	Items  []*AttendanceItem `json:"items" v:"required#Vui lòng chọn hội viên cần điểm danh"`
}
type AttendanceRes struct {
	Updated int `json:"updated"`
}
