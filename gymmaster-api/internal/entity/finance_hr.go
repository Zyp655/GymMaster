package entity

import (
	"math"
	"time"
)

// Sprint 20: Chấm công & bảng lương.
const (
	AttendanceMethodQR     = "QR"
	AttendanceMethodManual = "MANUAL"

	PayrollStatusDraft  = "DRAFT"
	PayrollStatusLocked = "LOCKED"
	PayrollStatusPaid   = "PAID"
)

type StaffAttendance struct {
	ID           uint `gorm:"primaryKey;autoIncrement"`
	StaffID      uint `gorm:"not null"`
	BranchID     uint `gorm:"not null"`
	ShiftID      *uint
	CheckInAt    time.Time `gorm:"not null"`
	CheckOutAt   *time.Time
	LateMinutes  int    `gorm:"not null;default:0"`
	EarlyMinutes int    `gorm:"not null;default:0"`
	Method       string `gorm:"type:varchar(20);not null;default:'MANUAL'"`
	Note         string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (StaffAttendance) TableName() string { return "staff_attendances" }

type Payroll struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	BranchID  uint   `gorm:"not null"`
	Period    string `gorm:"type:char(7);not null"`
	Status    string `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	LockedBy  *uint
	LockedAt  *time.Time
	PaidBy    *uint
	PaidAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Payroll) TableName() string { return "payrolls" }

type PayrollItem struct {
	ID              uint    `gorm:"primaryKey;autoIncrement"`
	PayrollID       uint    `gorm:"not null"`
	StaffID         uint    `gorm:"not null"`
	BaseSalary      float64 `gorm:"type:numeric(14,2);not null;default:0"`
	WorkDays        float64 `gorm:"type:numeric(5,2);not null;default:0"`
	OTHours         float64 `gorm:"column:ot_hours;type:numeric(5,2);not null;default:0"`
	PTCommission    float64 `gorm:"column:pt_commission;type:numeric(14,2);not null;default:0"`
	SalesCommission float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Bonus           float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Penalty         float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Total           float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Note            string  `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (PayrollItem) TableName() string { return "payroll_items" }

// PayrollStandardWorkDays: số ngày công chuẩn một tháng. Lương cơ bản được trả theo ngày công thực tế:
// lương CB thực nhận = BaseSalary × min(WorkDays, 26) / 26 (làm tròn đồng) – PLAN.md Sprint 20.
const PayrollStandardWorkDays = 26.0

// BasePay: lương cơ bản thực nhận theo ngày công (không vượt quá lương cơ bản hợp đồng).
func (it *PayrollItem) BasePay() float64 {
	if it.BaseSalary <= 0 || it.WorkDays <= 0 {
		return 0
	}
	days := math.Min(it.WorkDays, PayrollStandardWorkDays)
	return math.Round(it.BaseSalary * days / PayrollStandardWorkDays)
}

// RecalcTotal: thực lĩnh = lương CB theo ngày công + hoa hồng PT + hoa hồng bán gói + thưởng − phạt.
func (it *PayrollItem) RecalcTotal() {
	it.Total = it.BasePay() + it.PTCommission + it.SalesCommission + it.Bonus - it.Penalty
}

// Sprint 21: Chi phí, P&L, đối soát ca & trả góp.
const (
	ExpenseTypeFixed    = "FIXED"
	ExpenseTypeVariable = "VARIABLE"

	ExpenseStatusPending  = "PENDING"
	ExpenseStatusApproved = "APPROVED"
	ExpenseStatusRejected = "REJECTED"

	CashSessionOpen   = "OPEN"
	CashSessionClosed = "CLOSED"

	InstallmentPending = "PENDING"
	InstallmentPaid    = "PAID"
	InstallmentOverdue = "OVERDUE"
)

type ExpenseCategory struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"type:varchar(150);not null;unique"`
	Type      string `gorm:"type:varchar(20);not null;default:'VARIABLE'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ExpenseCategory) TableName() string { return "expense_categories" }

type Expense struct {
	ID               uint      `gorm:"primaryKey;autoIncrement"`
	BranchID         uint      `gorm:"not null"`
	CategoryID       uint      `gorm:"not null"`
	Amount           float64   `gorm:"type:numeric(14,2);not null"`
	SpentAt          time.Time `gorm:"type:date;not null"`
	Description      string    `gorm:"type:varchar(500);not null;default:''"`
	AttachmentURL    string    `gorm:"column:attachment_url;type:varchar(500);not null;default:''"`
	MaintenanceLogID *uint
	CreatedBy        *uint
	ApprovedBy       *uint
	Status           string `gorm:"type:varchar(20);not null;default:'PENDING'"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Expense) TableName() string { return "expenses" }

type CashSession struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	BranchID     uint      `gorm:"not null"`
	StaffID      uint      `gorm:"not null"`
	OpenedAt     time.Time `gorm:"not null"`
	ClosedAt     *time.Time
	OpeningCash  float64 `gorm:"type:numeric(14,2);not null;default:0"`
	ExpectedCash float64 `gorm:"type:numeric(14,2);not null;default:0"`
	CountedCash  float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Difference   float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Note         string  `gorm:"type:varchar(500);not null;default:''"`
	Status       string  `gorm:"type:varchar(20);not null;default:'OPEN'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (CashSession) TableName() string { return "cash_sessions" }

type InstallmentPlan struct {
	ID           uint    `gorm:"primaryKey;autoIncrement"`
	InvoiceID    uint    `gorm:"not null;unique"`
	MemberID     uint    `gorm:"not null"`
	Total        float64 `gorm:"type:numeric(14,2);not null"`
	Installments int     `gorm:"not null"`
	CreatedBy    *uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (InstallmentPlan) TableName() string { return "installment_plans" }

type InstallmentSchedule struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	PlanID     uint      `gorm:"not null"`
	Seq        int       `gorm:"not null"`
	DueDate    time.Time `gorm:"type:date;not null"`
	Amount     float64   `gorm:"type:numeric(14,2);not null"`
	PaidAmount float64   `gorm:"type:numeric(14,2);not null;default:0"`
	Status     string    `gorm:"type:varchar(20);not null;default:'PENDING'"`
	PaidAt     *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (InstallmentSchedule) TableName() string { return "installment_schedules" }

// Sprint 26: Chỉ tiêu doanh số KPI.
const (
	KPIMetricRevenue    = "REVENUE"
	KPIMetricNewMembers = "NEW_MEMBERS"
	KPIMetricRenewals   = "RENEWALS"
	KPIMetricPTSales    = "PT_SALES"
)

type SalesTarget struct {
	ID          uint `gorm:"primaryKey;autoIncrement"`
	BranchID    uint `gorm:"not null"`
	StaffID     *uint
	Period      string  `gorm:"type:char(7);not null"`
	Metric      string  `gorm:"type:varchar(30);not null"`
	TargetValue float64 `gorm:"type:numeric(14,2);not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (SalesTarget) TableName() string { return "sales_targets" }
