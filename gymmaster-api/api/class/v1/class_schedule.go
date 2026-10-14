package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type ClassSchedule struct {
	ID             uint      `json:"id"`
	ClassTypeID    uint      `json:"class_type_id"`
	ClassTypeCode  string    `json:"class_type_code"`
	ClassTypeName  string    `json:"class_type_name"`
	ClassTypeColor string    `json:"class_type_color"`
	BranchID       uint      `json:"branch_id"`
	BranchName     string    `json:"branch_name"`
	RoomID         uint      `json:"room_id"`
	RoomName       string    `json:"room_name"`
	TrainerID      uint      `json:"trainer_id"`
	TrainerName    string    `json:"trainer_name"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	DurationMin    int       `json:"duration_min"`
	Capacity       int       `json:"capacity"`
	BookedCount    int       `json:"booked_count"`
	WaitlistCount  int       `json:"waitlist_count"`
	AttendedCount  int       `json:"attended_count"`
	AvailableSlots int       `json:"available_slots"`
	Status         string    `json:"status"`
	SeriesID       string    `json:"series_id"`
	CancelReason   string    `json:"cancel_reason"`
	Note           string    `json:"note"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ListScheduleReq struct {
	g.Meta      `path:"/class-schedules" method:"get" tags:"Lớp nhóm" summary:"Lịch lớp trong khoảng ngày (lịch tuần)" perm:"class.view"`
	BranchID    uint   `json:"branch_id"     in:"query" dc:"Lọc theo chi nhánh"`
	From        string `json:"from"          in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn ngày bắt đầu|Ngày phải có dạng YYYY-MM-DD"`
	To          string `json:"to"            in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn ngày kết thúc|Ngày phải có dạng YYYY-MM-DD"`
	TrainerID   uint   `json:"trainer_id"    in:"query" dc:"Lọc theo HLV. Người chỉ có vai trò HLV mặc định chỉ thấy lớp của mình"`
	ClassTypeID uint   `json:"class_type_id" in:"query"`
	RoomID      uint   `json:"room_id"       in:"query"`
	Status      string `json:"status"        in:"query" v:"in:SCHEDULED,CANCELLED,COMPLETED#Trạng thái không hợp lệ"`
}
type ListScheduleRes []*ClassSchedule

type GetScheduleReq struct {
	g.Meta `path:"/class-schedules/{id}" method:"get" tags:"Lớp nhóm" summary:"Chi tiết buổi lớp" perm:"class.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetScheduleRes = ClassSchedule

type CreateScheduleReq struct {
	g.Meta      `path:"/class-schedules" method:"post" tags:"Lớp nhóm" summary:"Tạo một buổi lớp" perm:"class.manage"`
	ClassTypeID uint   `json:"class_type_id" v:"required|min:1#Vui lòng chọn loại lớp|Vui lòng chọn loại lớp"`
	BranchID    uint   `json:"branch_id"     v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	RoomID      uint   `json:"room_id"       v:"required|min:1#Vui lòng chọn phòng|Vui lòng chọn phòng"`
	TrainerID   uint   `json:"trainer_id"    v:"required|min:1#Vui lòng chọn HLV|Vui lòng chọn HLV"`
	StartAt     string `json:"start_at"      v:"required#Vui lòng nhập giờ bắt đầu" dc:"RFC3339, VD: 2026-10-20T18:00:00+07:00"`
	DurationMin int    `json:"duration_min"  v:"between:15,240#Thời lượng từ 15 đến 240 phút" dc:"Mặc định theo loại lớp"`
	Capacity    int    `json:"capacity"      v:"min:1#Sức chứa phải lớn hơn 0" dc:"Mặc định = min(sức chứa mặc định của loại lớp, sức chứa phòng)"`
	Note        string `json:"note"          v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type CreateScheduleRes = ClassSchedule

type CreateRecurringReq struct {
	g.Meta      `path:"/class-schedules/recurring" method:"post" tags:"Lớp nhóm" summary:"Tạo lịch lớp lặp theo tuần" perm:"class.manage"`
	ClassTypeID uint   `json:"class_type_id" v:"required|min:1#Vui lòng chọn loại lớp|Vui lòng chọn loại lớp"`
	BranchID    uint   `json:"branch_id"     v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	RoomID      uint   `json:"room_id"       v:"required|min:1#Vui lòng chọn phòng|Vui lòng chọn phòng"`
	TrainerID   uint   `json:"trainer_id"    v:"required|min:1#Vui lòng chọn HLV|Vui lòng chọn HLV"`
	StartDate   string `json:"start_date"    v:"required|date-format:Y-m-d#Vui lòng chọn ngày bắt đầu|Ngày phải có dạng YYYY-MM-DD"`
	EndDate     string `json:"end_date"      v:"required|date-format:Y-m-d#Vui lòng chọn ngày kết thúc|Ngày phải có dạng YYYY-MM-DD"`
	Weekdays    []int  `json:"weekdays"      v:"required#Vui lòng chọn thứ trong tuần" dc:"1 = Thứ Hai … 7 = Chủ nhật"`
	StartTime   string `json:"start_time"    v:"required|regex:^([01][0-9]|2[0-3]):[0-5][0-9]$#Vui lòng nhập giờ bắt đầu|Giờ phải có dạng HH:mm"`
	DurationMin int    `json:"duration_min"  v:"between:15,240#Thời lượng từ 15 đến 240 phút" dc:"Mặc định theo loại lớp"`
	Capacity    int    `json:"capacity"      v:"min:1#Sức chứa phải lớn hơn 0" dc:"Mặc định = min(sức chứa mặc định của loại lớp, sức chứa phòng)"`
	Note        string `json:"note"          v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type RecurringSkipped struct {
	Date   string `json:"date"`
	Reason string `json:"reason"`
}
type CreateRecurringRes struct {
	SeriesID    string              `json:"series_id"`
	Created     int                 `json:"created"`
	ScheduleIDs []uint              `json:"schedule_ids"`
	Skipped     []*RecurringSkipped `json:"skipped"`
}

type UpdateScheduleReq struct {
	g.Meta      `path:"/class-schedules/{id}" method:"put" tags:"Lớp nhóm" summary:"Sửa buổi lớp (giờ, phòng, HLV, sức chứa)" perm:"class.manage"`
	ID          uint   `json:"id"           in:"path" v:"required|min:1"`
	RoomID      uint   `json:"room_id"      v:"required|min:1#Vui lòng chọn phòng|Vui lòng chọn phòng"`
	TrainerID   uint   `json:"trainer_id"   v:"required|min:1#Vui lòng chọn HLV|Vui lòng chọn HLV"`
	StartAt     string `json:"start_at"     v:"required#Vui lòng nhập giờ bắt đầu" dc:"RFC3339"`
	DurationMin int    `json:"duration_min" v:"required|between:15,240#Vui lòng nhập thời lượng|Thời lượng từ 15 đến 240 phút"`
	Capacity    int    `json:"capacity"     v:"required|min:1#Vui lòng nhập sức chứa|Sức chứa phải lớn hơn 0"`
	Note        string `json:"note"         v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type UpdateScheduleRes = ClassSchedule

type CancelScheduleReq struct {
	g.Meta `path:"/class-schedules/{id}/cancel" method:"post" tags:"Lớp nhóm" summary:"Hủy buổi lớp (một buổi hoặc cả chuỗi từ buổi này)" perm:"class.manage"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Reason string `json:"reason" v:"required|max-length:255#Vui lòng nhập lý do hủy|Lý do tối đa 255 ký tự"`
	Scope  string `json:"scope"  v:"in:one,following#Phạm vi hủy phải là one hoặc following" d:"one" dc:"one | following"`
}
type CancelScheduleRes struct {
	CancelledSchedules int    `json:"cancelled_schedules"`
	CancelledBookings  int    `json:"cancelled_bookings"`
	ScheduleIDs        []uint `json:"schedule_ids"`
}
