package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type LeadActivity struct {
	ID          uint      `json:"id"`
	LeadID      uint      `json:"lead_id"`
	Type        string    `json:"type"`
	Content     string    `json:"content"`
	CreatedBy   *uint     `json:"created_by"`
	CreatorName string    `json:"creator_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type TrialPass struct {
	ID         uint       `json:"id"`
	LeadID     uint       `json:"lead_id"`
	BranchID   uint       `json:"branch_id"`
	BranchName string     `json:"branch_name,omitempty"`
	ValidDate  string     `json:"valid_date"`
	QRToken    string     `json:"qr_token"`
	UsedAt     *time.Time `json:"used_at"`
	CreatedBy  *uint      `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Lead struct {
	ID               uint            `json:"id"`
	BranchID         uint            `json:"branch_id"`
	BranchName       string          `json:"branch_name,omitempty"`
	FullName         string          `json:"full_name"`
	Phone            string          `json:"phone"`
	Email            string          `json:"email"`
	Source           string          `json:"source"`
	InterestPlanID   *uint           `json:"interest_plan_id"`
	InterestPlanName string          `json:"interest_plan_name,omitempty"`
	Status           string          `json:"status"`
	AssignedTo       *uint           `json:"assigned_to"`
	AssignedName     string          `json:"assigned_name,omitempty"`
	AssignedToName   string          `json:"assigned_to_name,omitempty"`
	TrialDate        *string         `json:"trial_date"`
	MemberID         *uint           `json:"member_id"`
	MemberCode       string          `json:"member_code,omitempty"`
	Note             string          `json:"note"`
	Activities       []*LeadActivity `json:"activities,omitempty"`
	TrialPasses      []*TrialPass    `json:"trial_passes,omitempty"`
	TrialPass        *TrialPass      `json:"trial_pass,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type LeadInput struct {
	BranchID       uint   `json:"branch_id"        v:"required|min:1#Vui lòng chọn chi nhánh"`
	FullName       string `json:"full_name"        v:"required|length:2,150#Vui lòng nhập họ tên|Họ tên từ 2 đến 150 ký tự"`
	Phone          string `json:"phone"            v:"required|length:9,20#Vui lòng nhập số điện thoại|Số điện thoại không hợp lệ"`
	Email          string `json:"email"`
	Source         string `json:"source"           d:"WALK_IN" v:"in:WEBSITE,WALK_IN,REFERRAL,FACEBOOK,OTHER#Nguồn không hợp lệ"`
	InterestPlanID *uint  `json:"interest_plan_id"`
	Status         string `json:"status"           d:"NEW"     v:"in:NEW,CONTACTED,TRIAL,CONVERTED,LOST#Trạng thái không hợp lệ"`
	AssignedTo     *uint  `json:"assigned_to"`
	TrialDate      string `json:"trial_date"`
	Note           string `json:"note"`
}

type PublicClassSchedule struct {
	ID             uint      `json:"id"`
	ClassTypeID    uint      `json:"class_type_id"`
	ClassTypeCode  string    `json:"class_type_code"`
	ClassTypeName  string    `json:"class_type_name"`
	ClassTypeColor string    `json:"class_type_color"`
	Level          string    `json:"level"`
	BranchID       uint      `json:"branch_id"`
	BranchName     string    `json:"branch_name"`
	RoomID         uint      `json:"room_id"`
	RoomName       string    `json:"room_name"`
	TrainerID      uint      `json:"trainer_id"`
	TrainerName    string    `json:"trainer_name"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	Capacity       int       `json:"capacity"`
	BookedCount    int       `json:"booked_count"`
	// AvailableSlots = capacity − booked_count (không âm) – số chỗ còn trống để đặt.
	AvailableSlots int    `json:"available_slots"`
	Status         string `json:"status"`
}

type AtRiskMember struct {
	ID            uint       `json:"id"`
	MemberID      uint       `json:"member_id"`
	MemberCode    string     `json:"member_code"`
	FullName      string     `json:"full_name"`
	Phone         string     `json:"phone"`
	Email         string     `json:"email"`
	BranchID      uint       `json:"branch_id"`
	BranchName    string     `json:"branch_name"`
	PlanName      string     `json:"plan_name"`
	EndDate       string     `json:"end_date"`
	LastCheckinAt *time.Time `json:"last_checkin_at"`
	DaysInactive  int        `json:"days_inactive"`
	AbsentDays    int        `json:"absent_days"`
}

type LeadCountItem struct {
	Key    string `json:"key"`
	Status string `json:"status,omitempty"`
	Source string `json:"source,omitempty"`
	Count  int64  `json:"count"`
}

// ---- Public (/v1/public) --------------------------------------------------------

type PublicCreateLeadReq struct {
	g.Meta         `path:"/leads" method:"post" tags:"Công khai" summary:"Đăng ký tập thử từ trang chủ"`
	BranchID       uint   `json:"branch_id"        v:"required|min:1#Vui lòng chọn chi nhánh"`
	FullName       string `json:"full_name"        v:"required|length:2,150#Vui lòng nhập họ tên|Họ tên từ 2 đến 150 ký tự"`
	Phone          string `json:"phone"            v:"required|length:9,20#Vui lòng nhập số điện thoại|Số điện thoại không hợp lệ"`
	Email          string `json:"email"`
	InterestPlanID *uint  `json:"interest_plan_id"`
	Note           string `json:"note"`
}
type PublicCreateLeadRes = Lead

type PublicClassSchedulesReq struct {
	g.Meta   `path:"/class-schedules" method:"get" tags:"Công khai" summary:"Lịch lớp học công khai cho landing page"`
	BranchID uint   `json:"branch_id" in:"query"`
	From     string `json:"from"      in:"query"`
	To       string `json:"to"        in:"query"`
}
type PublicClassSchedulesRes []*PublicClassSchedule

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListLeadsReq struct {
	g.Meta `path:"/leads" method:"get" tags:"CRM & Chăm sóc" summary:"Danh sách khách tiềm năng" perm:"lead.view,lead.manage"`
	dto.PageReq
	BranchID   uint   `json:"branch_id"   in:"query"`
	Status     string `json:"status"      in:"query" v:"in:NEW,CONTACTED,TRIAL,CONVERTED,LOST"`
	Source     string `json:"source"      in:"query" v:"in:WEBSITE,WALK_IN,REFERRAL,FACEBOOK,OTHER"`
	AssignedTo uint   `json:"assigned_to" in:"query"`
	Search     string `json:"search"      in:"query"`
}
type ListLeadsRes = dto.PageRes[*Lead]

type GetLeadReq struct {
	g.Meta `path:"/leads/{id}" method:"get" tags:"CRM & Chăm sóc" summary:"Chi tiết khách tiềm năng" perm:"lead.view,lead.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetLeadRes = Lead

type CreateLeadReq struct {
	g.Meta `path:"/leads" method:"post" tags:"CRM & Chăm sóc" summary:"Thêm khách tiềm năng" perm:"lead.manage"`
	LeadInput
}
type CreateLeadRes = Lead

// LeadUpdateInput: giống LeadInput nhưng KHÔNG có giá trị mặc định cho source/status –
// bỏ trống thì giữ nguyên giá trị hiện tại (không bị reset về WALK_IN/NEW).
type LeadUpdateInput struct {
	BranchID       uint   `json:"branch_id"        v:"required|min:1#Vui lòng chọn chi nhánh"`
	FullName       string `json:"full_name"        v:"required|length:2,150#Vui lòng nhập họ tên|Họ tên từ 2 đến 150 ký tự"`
	Phone          string `json:"phone"            v:"required|length:9,20#Vui lòng nhập số điện thoại|Số điện thoại không hợp lệ"`
	Email          string `json:"email"`
	Source         string `json:"source"           v:"in:WEBSITE,WALK_IN,REFERRAL,FACEBOOK,OTHER#Nguồn không hợp lệ"`
	InterestPlanID *uint  `json:"interest_plan_id"`
	Status         string `json:"status"           v:"in:NEW,CONTACTED,TRIAL,CONVERTED,LOST#Trạng thái không hợp lệ"`
	AssignedTo     *uint  `json:"assigned_to"`
	TrialDate      string `json:"trial_date"`
	Note           string `json:"note"`
}

type UpdateLeadReq struct {
	g.Meta `path:"/leads/{id}" method:"put" tags:"CRM & Chăm sóc" summary:"Cập nhật khách tiềm năng" perm:"lead.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	LeadUpdateInput
}
type UpdateLeadRes = Lead

type UpdateLeadStatusReq struct {
	g.Meta `path:"/leads/{id}/status" method:"patch" tags:"CRM & Chăm sóc" summary:"Chuyển trạng thái khách tiềm năng (Kanban)" perm:"lead.manage"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Status string `json:"status"           v:"required|in:NEW,CONTACTED,TRIAL,CONVERTED,LOST#Vui lòng chọn trạng thái|Trạng thái không hợp lệ"`
}
type UpdateLeadStatusRes = Lead

type DeleteLeadReq struct {
	g.Meta `path:"/leads/{id}" method:"delete" tags:"CRM & Chăm sóc" summary:"Xóa khách tiềm năng" perm:"lead.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteLeadRes struct{}

type ListLeadActivitiesReq struct {
	g.Meta `path:"/leads/{id}/activities" method:"get" tags:"CRM & Chăm sóc" summary:"Danh sách nhật ký chăm sóc khách tiềm năng" perm:"lead.view,lead.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ListLeadActivitiesRes []*LeadActivity

type AddLeadActivityReq struct {
	g.Meta  `path:"/leads/{id}/activities" method:"post" tags:"CRM & Chăm sóc" summary:"Ghi nhật ký chăm sóc khách tiềm năng" perm:"lead.manage"`
	ID      uint   `json:"id"      in:"path" v:"required|min:1"`
	Type    string `json:"type"              v:"required|in:CALL,SMS,VISIT,NOTE#Vui lòng chọn hình thức liên hệ|Hình thức liên hệ không hợp lệ"`
	Content string `json:"content"           v:"required#Vui lòng nhập nội dung chăm sóc"`
}
type AddLeadActivityRes = LeadActivity

type IssueTrialPassReq struct {
	g.Meta    `path:"/leads/{id}/trial-pass" method:"post" tags:"CRM & Chăm sóc" summary:"Cấp vé tập thử (QR 1 ngày)" perm:"lead.manage"`
	ID        uint   `json:"id"         in:"path" v:"required|min:1"`
	ValidDate string `json:"valid_date"`
}
type IssueTrialPassRes = TrialPass

type ConvertLeadReq struct {
	g.Meta  `path:"/leads/{id}/convert" method:"post" tags:"CRM & Chăm sóc" summary:"Chuyển đổi khách tiềm năng thành hội viên" perm:"lead.manage"`
	ID      uint   `json:"id"      in:"path" v:"required|min:1"`
	Gender  string `json:"gender"            d:"OTHER" v:"in:MALE,FEMALE,OTHER"`
	Dob     string `json:"dob"`
	Address string `json:"address"`
}
type ConvertLeadRes = Lead

type AtRiskMembersReq struct {
	g.Meta       `path:"/members/at-risk" method:"get" tags:"CRM & Chăm sóc" summary:"Danh sách hội viên có nguy cơ nghỉ tập" perm:"lead.view,lead.manage,member.view"`
	BranchID     uint `json:"branch_id"     in:"query"`
	DaysInactive int  `json:"days_inactive" in:"query" d:"14"`
	Days         int  `json:"days"          in:"query"`
}
type AtRiskMembersRes []*AtRiskMember

type LeadsReportReq struct {
	g.Meta   `path:"/reports/leads" method:"get" tags:"CRM & Chăm sóc" summary:"Báo cáo phễu chuyển đổi khách tiềm năng" perm:"lead.view,lead.manage,report.view"`
	BranchID uint   `json:"branch_id" in:"query"`
	From     string `json:"from"      in:"query"`
	To       string `json:"to"        in:"query"`
}
type LeadsReportRes struct {
	Total          int64            `json:"total"`
	Converted      int64            `json:"converted"`
	ConversionRate float64          `json:"conversion_rate"`
	ByStatus       []*LeadCountItem `json:"by_status"`
	BySource       []*LeadCountItem `json:"by_source"`
}
