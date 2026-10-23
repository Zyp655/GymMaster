package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type TrainerAvailabilityItem struct {
	ID            uint      `json:"id"`
	TrainerID     uint      `json:"trainer_id"`
	BranchID      uint      `json:"branch_id"`
	BranchName    string    `json:"branch_name,omitempty"`
	DOW           int       `json:"dow"`
	StartTime     string    `json:"start_time"`
	EndTime       string    `json:"end_time"`
	EffectiveFrom *string   `json:"effective_from"`
	EffectiveTo   *string   `json:"effective_to"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TrainerAvailabilitySlotInput struct {
	BranchID      uint   `json:"branch_id"`
	DOW           int    `json:"dow"        v:"between:0,6#Thứ trong tuần từ 0 (CN) đến 6 (Thứ 7)"`
	StartTime     string `json:"start_time" v:"required#Vui lòng nhập giờ bắt đầu"`
	EndTime       string `json:"end_time"   v:"required#Vui lòng nhập giờ kết thúc"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
}

type LeaveRequestItem struct {
	ID           uint       `json:"id"`
	StaffID      uint       `json:"staff_id"`
	StaffName    string     `json:"staff_name,omitempty"`
	BranchID     *uint      `json:"branch_id"`
	BranchName   string     `json:"branch_name,omitempty"`
	FromAt       time.Time  `json:"from_at"`
	ToAt         time.Time  `json:"to_at"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"`
	ApprovedBy   *uint      `json:"approved_by"`
	ApproverName string     `json:"approver_name,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at"`
	ReviewNote   string     `json:"review_note"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ImpactedClassSchedule struct {
	ID                    uint      `json:"id"`
	ClassTypeName         string    `json:"class_type_name"`
	BranchID              uint      `json:"branch_id"`
	BranchName            string    `json:"branch_name"`
	RoomName              string    `json:"room_name"`
	TrainerID             uint      `json:"trainer_id"`
	TrainerName           string    `json:"trainer_name"`
	SubstituteTrainerID   *uint     `json:"substitute_trainer_id"`
	SubstituteTrainerName string    `json:"substitute_trainer_name,omitempty"`
	StartAt               time.Time `json:"start_at"`
	EndAt                 time.Time `json:"end_at"`
	BookedCount           int       `json:"booked_count"`
	Status                string    `json:"status"`
}

type ImpactedPTSession struct {
	ID                  uint      `json:"id"`
	ContractID          uint      `json:"contract_id"`
	MemberID            uint      `json:"member_id"`
	MemberCode          string    `json:"member_code"`
	MemberName          string    `json:"member_name"`
	TrainerID           uint      `json:"trainer_id"`
	TrainerName         string    `json:"trainer_name"`
	OriginalTrainerID   *uint     `json:"original_trainer_id"`
	OriginalTrainerName string    `json:"original_trainer_name,omitempty"`
	BranchID            uint      `json:"branch_id"`
	BranchName          string    `json:"branch_name"`
	StartAt             time.Time `json:"start_at"`
	EndAt               time.Time `json:"end_at"`
	Status              string    `json:"status"`
}

type BranchHolidayItem struct {
	ID                    uint      `json:"id"`
	BranchID              *uint     `json:"branch_id"`
	BranchName            string    `json:"branch_name,omitempty"`
	Date                  string    `json:"date"`
	Name                  string    `json:"name"`
	Closed                bool      `json:"closed"`
	OpenTime              *string   `json:"open_time"`
	CloseTime             *string   `json:"close_time"`
	ExtendSubs            bool      `json:"extend_subs"`
	ExtendedSubscriptions int64     `json:"extended_subscriptions,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	// Chỉ có trong phản hồi thêm/sửa: buổi lớp / buổi PT đã xếp (SCHEDULED) rơi vào ngày nghỉ (hoặc ngoài giờ đặc
	// biệt) – cần hủy, dời lịch hoặc thông báo cho hội viên. Lịch đã xếp KHÔNG tự hủy.
	AffectedClassSchedules []*ImpactedClassSchedule `json:"affected_class_schedules,omitempty"`
	AffectedPTSessions     []*ImpactedPTSession     `json:"affected_pt_sessions,omitempty"`
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

type GetTrainerAvailabilityReq struct {
	g.Meta   `path:"/trainers/{id}/availability" method:"get" tags:"Lịch rảnh & Nghỉ lễ" summary:"Xem khung giờ rảnh trong tuần của HLV" perm:"leave.manage,pt.view,pt.schedule"`
	ID       uint `json:"id"        in:"path"  v:"required|min:1"`
	BranchID uint `json:"branch_id" in:"query"`
}
type GetTrainerAvailabilityRes = []*TrainerAvailabilityItem

type SaveTrainerAvailabilityReq struct {
	g.Meta   `path:"/trainers/{id}/availability" method:"put" tags:"Lịch rảnh & Nghỉ lễ" summary:"Thay toàn bộ khung giờ rảnh trong tuần của HLV tại MỘT chi nhánh (danh sách rỗng = không giới hạn giờ nhận PT)" perm:"leave.manage,pt.schedule"`
	ID       uint                           `json:"id"        in:"path" v:"required|min:1"`
	BranchID uint                           `json:"branch_id" v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	Slots    []TrainerAvailabilitySlotInput `json:"slots"`
	Items    []TrainerAvailabilitySlotInput `json:"items"`
}
type SaveTrainerAvailabilityRes = []*TrainerAvailabilityItem

type ListLeaveRequestsReq struct {
	g.Meta `path:"/leave-requests" method:"get" tags:"Lịch rảnh & Nghỉ lễ" summary:"Danh sách đơn xin nghỉ phép của nhân viên/HLV" perm:"leave.manage,pt.view,shift.view"`
	dto.PageReq
	StaffID  uint   `json:"staff_id"  in:"query"`
	BranchID uint   `json:"branch_id" in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListLeaveRequestsRes = dto.PageRes[*LeaveRequestItem]

type CreateLeaveRequestReq struct {
	g.Meta   `path:"/leave-requests" method:"post" tags:"Lịch rảnh & Nghỉ lễ" summary:"Gửi đơn xin nghỉ phép" perm:"leave.manage,pt.schedule,shift.view"`
	StaffID  uint   `json:"staff_id"`
	BranchID *uint  `json:"branch_id"`
	FromAt   string `json:"from_at" v:"required#Vui lòng chọn thời gian bắt đầu nghỉ"`
	ToAt     string `json:"to_at"   v:"required#Vui lòng chọn thời gian kết thúc nghỉ"`
	Reason   string `json:"reason"  v:"required|max-length:500#Vui lòng nhập lý do xin nghỉ"`
}
type CreateLeaveRequestRes = LeaveRequestItem

type ApproveLeaveRequestReq struct {
	g.Meta     `path:"/leave-requests/{id}/approve" method:"post" tags:"Lịch rảnh & Nghỉ lễ" summary:"Duyệt đơn nghỉ phép" perm:"leave.manage"`
	ID         uint   `json:"id"          in:"path" v:"required|min:1"`
	ReviewNote string `json:"review_note"           v:"max-length:255"`
}
type ApproveLeaveRequestRes = LeaveRequestItem

type RejectLeaveRequestReq struct {
	g.Meta     `path:"/leave-requests/{id}/reject" method:"post" tags:"Lịch rảnh & Nghỉ lễ" summary:"Từ chối đơn nghỉ phép" perm:"leave.manage"`
	ID         uint   `json:"id"          in:"path" v:"required|min:1"`
	ReviewNote string `json:"review_note"           v:"max-length:255"`
}
type RejectLeaveRequestRes = LeaveRequestItem

type GetLeaveImpactsReq struct {
	g.Meta `path:"/leave-requests/{id}/impacts" method:"get" tags:"Lịch rảnh & Nghỉ lễ" summary:"Xem các buổi lớp và buổi PT bị ảnh hưởng bởi đơn nghỉ phép" perm:"leave.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetLeaveImpactsRes struct {
	LeaveRequest   *LeaveRequestItem        `json:"leave_request"`
	ClassSchedules []*ImpactedClassSchedule `json:"class_schedules"`
	PTSessions     []*ImpactedPTSession     `json:"pt_sessions"`
}

type SubstituteClassTrainerReq struct {
	g.Meta              `path:"/class-schedules/{id}/substitute" method:"post" tags:"Lịch rảnh & Nghỉ lễ" summary:"Phân công HLV dạy thay buổi lớp nhóm" perm:"leave.manage,class.manage"`
	ID                  uint   `json:"id"                    in:"path" v:"required|min:1"`
	SubstituteTrainerID uint   `json:"substitute_trainer_id"`
	TrainerID           uint   `json:"trainer_id"`
	Note                string `json:"note"                            v:"max-length:255"`
}
type SubstituteClassTrainerRes = ImpactedClassSchedule

type ReassignPTSessionReq struct {
	g.Meta       `path:"/pt-sessions/{id}/reassign" method:"post" tags:"Lịch rảnh & Nghỉ lễ" summary:"Chuyển buổi PT sang HLV dạy thay" perm:"leave.manage,pt.schedule"`
	ID           uint   `json:"id"             in:"path" v:"required|min:1"`
	TrainerID    uint   `json:"trainer_id"`
	NewTrainerID uint   `json:"new_trainer_id"`
	Note         string `json:"note"`
}
type ReassignPTSessionRes = ImpactedPTSession

type ListBranchHolidaysReq struct {
	g.Meta `path:"/branch-holidays" method:"get" tags:"Lịch rảnh & Nghỉ lễ" summary:"Danh sách lịch nghỉ lễ của phòng tập" perm:"holiday.manage,branch.view"`
	dto.PageReq
	BranchID uint `json:"branch_id" in:"query"`
	Year     int  `json:"year"      in:"query"`
}
type ListBranchHolidaysRes = dto.PageRes[*BranchHolidayItem]

type CreateBranchHolidayReq struct {
	g.Meta     `path:"/branch-holidays" method:"post" tags:"Lịch rảnh & Nghỉ lễ" summary:"Thêm ngày nghỉ lễ (tự động cộng bù ngày gói tập nếu chọn)" perm:"holiday.manage"`
	BranchID   *uint   `json:"branch_id"`
	Date       string  `json:"date"        v:"required#Vui lòng chọn ngày nghỉ lễ"`
	Name       string  `json:"name"        v:"required|max-length:150#Vui lòng nhập tên ngày lễ"`
	Closed     *bool   `json:"closed"`
	OpenTime   *string `json:"open_time"  dc:"HH:mm – bắt buộc khi closed=false và muốn áp giờ đặc biệt"`
	CloseTime  *string `json:"close_time" dc:"HH:mm – phải sau open_time"`
	ExtendSubs bool    `json:"extend_subs"`
}
type CreateBranchHolidayRes = BranchHolidayItem

type UpdateBranchHolidayReq struct {
	g.Meta    `path:"/branch-holidays/{id}" method:"put" tags:"Lịch rảnh & Nghỉ lễ" summary:"Sửa ngày nghỉ lễ / giờ đặc biệt (không cộng bù lại gói tập)" perm:"holiday.manage"`
	ID        uint    `json:"id"         in:"path" v:"required|min:1"`
	BranchID  *uint   `json:"branch_id"`
	Date      string  `json:"date"       v:"required#Vui lòng chọn ngày nghỉ lễ"`
	Name      string  `json:"name"       v:"required|max-length:150#Vui lòng nhập tên ngày lễ"`
	Closed    *bool   `json:"closed"`
	OpenTime  *string `json:"open_time"  dc:"HH:mm"`
	CloseTime *string `json:"close_time" dc:"HH:mm"`
}
type UpdateBranchHolidayRes = BranchHolidayItem

type DeleteBranchHolidayReq struct {
	g.Meta `path:"/branch-holidays/{id}" method:"delete" tags:"Lịch rảnh & Nghỉ lễ" summary:"Xóa lịch nghỉ lễ" perm:"holiday.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteScheduleExtRes struct{}

// ---- Public (/v1/public) --------------------------------------------------------

type PublicHolidaysReq struct {
	g.Meta   `path:"/holidays" method:"get" tags:"Công khai" summary:"Lịch nghỉ lễ sắp tới của phòng tập"`
	BranchID uint `json:"branch_id" in:"query"`
}
type PublicHolidaysRes []*BranchHolidayItem
