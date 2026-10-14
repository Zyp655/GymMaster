package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type AttendanceItem struct {
	ID           uint       `json:"id"`
	StaffID      uint       `json:"staff_id"`
	StaffCode    string     `json:"staff_code,omitempty"`
	StaffName    string     `json:"staff_name,omitempty"`
	Position     string     `json:"position,omitempty"`
	BranchID     uint       `json:"branch_id"`
	BranchName   string     `json:"branch_name,omitempty"`
	ShiftID      *uint      `json:"shift_id"`
	CheckInAt    time.Time  `json:"check_in_at"`
	CheckOutAt   *time.Time `json:"check_out_at"`
	LateMinutes  int        `json:"late_minutes"`
	EarlyMinutes int        `json:"early_minutes"`
	Method       string     `json:"method"`
	Note         string     `json:"note"`
	Action       string     `json:"action,omitempty"` // CHECK_IN | CHECK_OUT
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type PayrollLineItem struct {
	ID              uint      `json:"id"`
	PayrollID       uint      `json:"payroll_id"`
	StaffID         uint      `json:"staff_id"`
	StaffCode       string    `json:"staff_code,omitempty"`
	StaffName       string    `json:"staff_name,omitempty"`
	Position        string    `json:"position,omitempty"`
	BaseSalary      float64   `json:"base_salary"`
	WorkDays        float64   `json:"work_days"`
	OTHours         float64   `json:"ot_hours"`
	PTCommission    float64   `json:"pt_commission"`
	SalesCommission float64   `json:"sales_commission"`
	Bonus           float64   `json:"bonus"`
	Penalty         float64   `json:"penalty"`
	Total           float64   `json:"total"`
	Note            string    `json:"note"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PayrollSummary struct {
	ID          uint       `json:"id"`
	BranchID    uint       `json:"branch_id"`
	BranchName  string     `json:"branch_name,omitempty"`
	Period      string     `json:"period"`
	Status      string     `json:"status"`
	StaffCount  int        `json:"staff_count"`
	TotalAmount float64    `json:"total_amount"`
	LockedBy    *uint      `json:"locked_by"`
	LockedAt    *time.Time `json:"locked_at"`
	PaidBy      *uint      `json:"paid_by"`
	PaidAt      *time.Time `json:"paid_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PayrollDetail struct {
	PayrollSummary
	Items []*PayrollLineItem `json:"items"`
}

type PayrollItemUpdateInput struct {
	ID         uint     `json:"id"`
	StaffID    uint     `json:"staff_id"`
	BaseSalary *float64 `json:"base_salary"`
	WorkDays   *float64 `json:"work_days"`
	OTHours    *float64 `json:"ot_hours"`
	Bonus      *float64 `json:"bonus"`
	Penalty    *float64 `json:"penalty"`
	Note       *string  `json:"note"`
}

type PayslipItem struct {
	PayrollLineItem
	BranchID   uint       `json:"branch_id"`
	BranchName string     `json:"branch_name,omitempty"`
	Period     string     `json:"period"`
	Status     string     `json:"status"`
	PaidAt     *time.Time `json:"paid_at"`
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ScanAttendanceReq struct {
	g.Meta   `path:"/attendances/scan" method:"post" tags:"Chấm công & Lương" summary:"Chấm công vào/ra cho nhân viên" perm:"payroll.manage,shift.manage"`
	StaffID  uint   `json:"staff_id"`
	BranchID uint   `json:"branch_id" v:"required|min:1"`
	Method   string `json:"method"    d:"QR" v:"in:QR,MANUAL"`
	Note     string `json:"note"`
}
type ScanAttendanceRes = AttendanceItem

type ListAttendancesReq struct {
	g.Meta   `path:"/attendances" method:"get" tags:"Chấm công & Lương" summary:"Danh sách chấm công nhân viên" perm:"payroll.view,payroll.manage"`
	BranchID uint   `json:"branch_id" in:"query"`
	StaffID  uint   `json:"staff_id"  in:"query"`
	From     string `json:"from"      in:"query"`
	To       string `json:"to"        in:"query"`
}
type ListAttendancesRes []*AttendanceItem

type GeneratePayrollReq struct {
	g.Meta   `path:"/payrolls/generate" method:"post" tags:"Chấm công & Lương" summary:"Tổng hợp & tạo bảng lương tháng" perm:"payroll.manage"`
	BranchID uint   `json:"branch_id" v:"required|min:1"`
	Period   string `json:"period"    v:"required"`
}
type GeneratePayrollRes = PayrollDetail

type ListPayrollsReq struct {
	g.Meta   `path:"/payrolls" method:"get" tags:"Chấm công & Lương" summary:"Danh sách bảng lương" perm:"payroll.view,payroll.manage"`
	BranchID uint   `json:"branch_id" in:"query"`
	Period   string `json:"period"    in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListPayrollsRes []*PayrollSummary

type GetPayrollReq struct {
	g.Meta `path:"/payrolls/{id}" method:"get" tags:"Chấm công & Lương" summary:"Chi tiết bảng lương" perm:"payroll.view,payroll.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetPayrollRes = PayrollDetail

type UpdatePayrollReq struct {
	g.Meta `path:"/payrolls/{id}" method:"put" tags:"Chấm công & Lương" summary:"Cập nhật dòng lương (khi đang DRAFT)" perm:"payroll.manage"`
	ID     uint                     `json:"id"    in:"path" v:"required|min:1"`
	Items  []PayrollItemUpdateInput `json:"items"`
}
type UpdatePayrollRes = PayrollDetail

type LockPayrollReq struct {
	g.Meta `path:"/payrolls/{id}/lock" method:"post" tags:"Chấm công & Lương" summary:"Khóa bảng lương" perm:"payroll.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type LockPayrollRes = PayrollDetail

type PayPayrollReq struct {
	g.Meta `path:"/payrolls/{id}/pay" method:"post" tags:"Chấm công & Lương" summary:"Xác nhận đã chi lương" perm:"payroll.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type PayPayrollRes = PayrollDetail

type ExportPayrollReq struct {
	g.Meta `path:"/payrolls/{id}/export" method:"get" tags:"Chấm công & Lương" summary:"Xuất bảng lương CSV/Excel" perm:"payroll.view,payroll.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ExportPayrollRes struct {
	PayrollID   uint               `json:"payroll_id"`
	BranchID    uint               `json:"branch_id"`
	BranchName  string             `json:"branch_name"`
	Period      string             `json:"period"`
	Status      string             `json:"status"`
	TotalAmount float64            `json:"total_amount"`
	CSV         string             `json:"csv"`
	Items       []*PayrollLineItem `json:"items"`
}

type MyPayslipsReq struct {
	g.Meta `path:"/my-payslips" method:"get" tags:"Chấm công & Lương" summary:"Phiếu lương của nhân viên đang đăng nhập"`
}
type MyPayslipsRes []*PayslipItem
