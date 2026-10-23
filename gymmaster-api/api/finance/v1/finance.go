package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/pkg/einvoice"
)

type ExpenseCategoryItem struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ExpenseItem struct {
	ID               uint      `json:"id"`
	BranchID         uint      `json:"branch_id"`
	BranchName       string    `json:"branch_name,omitempty"`
	CategoryID       uint      `json:"category_id"`
	CategoryName     string    `json:"category_name,omitempty"`
	CategoryType     string    `json:"category_type,omitempty"`
	Amount           float64   `json:"amount"`
	SpentAt          string    `json:"spent_at"`
	Description      string    `json:"description"`
	AttachmentURL    string    `json:"attachment_url"`
	MaintenanceLogID *uint     `json:"maintenance_log_id"`
	CreatedBy        *uint     `json:"created_by"`
	CreatorName      string    `json:"creator_name,omitempty"`
	ApprovedBy       *uint     `json:"approved_by"`
	ApproverName     string    `json:"approver_name,omitempty"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CashSessionItem struct {
	ID           uint       `json:"id"`
	BranchID     uint       `json:"branch_id"`
	BranchName   string     `json:"branch_name,omitempty"`
	StaffID      uint       `json:"staff_id"`
	StaffName    string     `json:"staff_name,omitempty"`
	OpenedAt     time.Time  `json:"opened_at"`
	ClosedAt     *time.Time `json:"closed_at"`
	OpeningCash  float64    `json:"opening_cash"`
	ExpectedCash float64    `json:"expected_cash"`
	CountedCash  float64    `json:"counted_cash"`
	Difference   float64    `json:"difference"`
	Note         string     `json:"note"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type InstallmentScheduleItem struct {
	ID         uint       `json:"id"`
	PlanID     uint       `json:"plan_id"`
	InvoiceID  uint       `json:"invoice_id,omitempty"`
	InvoiceNo  string     `json:"invoice_no,omitempty"`
	MemberID   uint       `json:"member_id,omitempty"`
	MemberCode string     `json:"member_code,omitempty"`
	MemberName string     `json:"member_name,omitempty"`
	Seq        int        `json:"seq"`
	DueDate    string     `json:"due_date"`
	Amount     float64    `json:"amount"`
	PaidAmount float64    `json:"paid_amount"`
	Status     string     `json:"status"`
	PaidAt     *time.Time `json:"paid_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type InstallmentPlanItem struct {
	ID           uint                       `json:"id"`
	InvoiceID    uint                       `json:"invoice_id"`
	InvoiceNo    string                     `json:"invoice_no,omitempty"`
	MemberID     uint                       `json:"member_id"`
	MemberCode   string                     `json:"member_code,omitempty"`
	MemberName   string                     `json:"member_name,omitempty"`
	Total        float64                    `json:"total"`
	Installments int                        `json:"installments"`
	CreatedBy    *uint                      `json:"created_by"`
	Schedules    []*InstallmentScheduleItem `json:"schedules"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
}

// ---- Admin: Danh mục & Chi phí (/expense-categories, /expenses) -----------------

type ListExpenseCategoriesReq struct {
	g.Meta `path:"/expense-categories" method:"get" tags:"Tài chính & Kế toán" summary:"Danh mục chi phí vận hành" perm:"expense.view,expense.manage"`
}
type ListExpenseCategoriesRes []*ExpenseCategoryItem

type CreateExpenseCategoryReq struct {
	g.Meta `path:"/expense-categories" method:"post" tags:"Tài chính & Kế toán" summary:"Tạo danh mục chi phí" perm:"expense.manage"`
	Name   string `json:"name" v:"required|max-length:150#Vui lòng nhập tên danh mục chi phí|Tên danh mục tối đa 150 ký tự"`
	Type   string `json:"type" d:"VARIABLE" v:"in:FIXED,VARIABLE#Loại danh mục chi phí không hợp lệ"`
}
type CreateExpenseCategoryRes = ExpenseCategoryItem

type ListExpensesReq struct {
	g.Meta     `path:"/expenses" method:"get" tags:"Tài chính & Kế toán" summary:"Danh sách chi phí vận hành" perm:"expense.view,expense.manage"`
	BranchID   uint   `json:"branch_id"   in:"query"`
	CategoryID uint   `json:"category_id" in:"query"`
	Status     string `json:"status"      in:"query"`
	From       string `json:"from"        in:"query"`
	To         string `json:"to"          in:"query"`
}
type ListExpensesRes []*ExpenseItem

type CreateExpenseReq struct {
	g.Meta           `path:"/expenses" method:"post" tags:"Tài chính & Kế toán" summary:"Ghi nhận chi phí vận hành" perm:"expense.manage"`
	BranchID         uint    `json:"branch_id"          v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	CategoryID       uint    `json:"category_id"        v:"required|min:1#Vui lòng chọn danh mục chi phí|Vui lòng chọn danh mục chi phí"`
	Amount           float64 `json:"amount"             v:"required|min:1|max:999999999999#Vui lòng nhập số tiền chi phí|Số tiền chi phí phải lớn hơn 0|Số tiền chi phí vượt quá giới hạn cho phép"`
	SpentAt          string  `json:"spent_at"`
	Description      string  `json:"description"        v:"max-length:500#Mô tả tối đa 500 ký tự"`
	AttachmentURL    string  `json:"attachment_url"     v:"max-length:500#Đường dẫn chứng từ tối đa 500 ký tự"`
	MaintenanceLogID *uint   `json:"maintenance_log_id"`
}
type CreateExpenseRes = ExpenseItem

type UpdateExpenseReq struct {
	g.Meta        `path:"/expenses/{id}" method:"put" tags:"Tài chính & Kế toán" summary:"Cập nhật chi phí vận hành" perm:"expense.manage"`
	ID            uint     `json:"id"             in:"path" v:"required|min:1"`
	CategoryID    uint     `json:"category_id"`
	Amount        *float64 `json:"amount"         v:"min:1|max:999999999999#Số tiền chi phí phải lớn hơn 0|Số tiền chi phí vượt quá giới hạn cho phép"`
	SpentAt       string   `json:"spent_at"`
	Description   *string  `json:"description"    v:"max-length:500#Mô tả tối đa 500 ký tự"`
	AttachmentURL *string  `json:"attachment_url" v:"max-length:500#Đường dẫn chứng từ tối đa 500 ký tự"`
	// Status chỉ để tương thích ngược: PUT không đổi được trạng thái – dùng POST /expenses/{id}/approve.
	Status string `json:"status" dc:"Không dùng – duyệt/từ chối qua POST /expenses/{id}/approve"`
}
type UpdateExpenseRes = ExpenseItem

type DeleteExpenseReq struct {
	g.Meta `path:"/expenses/{id}" method:"delete" tags:"Tài chính & Kế toán" summary:"Xóa khoản chi phí" perm:"expense.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteExpenseRes struct{}

type ApproveExpenseReq struct {
	g.Meta `path:"/expenses/{id}/approve" method:"post" tags:"Tài chính & Kế toán" summary:"Duyệt / Từ chối chi phí" perm:"expense.manage"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Status string `json:"status" d:"APPROVED" v:"in:APPROVED,REJECTED#Trạng thái duyệt không hợp lệ"`
}
type ApproveExpenseRes = ExpenseItem

// ---- Admin: Đối soát ca thu ngân (/cash-sessions) -------------------------------

type OpenCashSessionReq struct {
	g.Meta      `path:"/cash-sessions/open" method:"post" tags:"Tài chính & Kế toán" summary:"Mở ca thu ngân" perm:"cash_session.manage"`
	BranchID    uint    `json:"branch_id"    v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	OpeningCash float64 `json:"opening_cash" v:"min:0|max:999999999999#Tiền mặt đầu ca không được âm|Tiền mặt đầu ca vượt quá giới hạn cho phép"`
	Note        string  `json:"note"         v:"max-length:500#Ghi chú tối đa 500 ký tự"`
}
type OpenCashSessionRes = CashSessionItem

type CloseCashSessionReq struct {
	g.Meta      `path:"/cash-sessions/{id}/close" method:"post" tags:"Tài chính & Kế toán" summary:"Đóng ca thu ngân & đối soát tiền mặt" perm:"cash_session.manage"`
	ID          uint    `json:"id"           in:"path" v:"required|min:1"`
	CountedCash float64 `json:"counted_cash" v:"min:0|max:999999999999#Tiền mặt đếm thực tế không được âm|Tiền mặt đếm thực tế vượt quá giới hạn cho phép"`
	Note        string  `json:"note"         v:"max-length:500#Ghi chú tối đa 500 ký tự"`
}
type CloseCashSessionRes = CashSessionItem

type ListCashSessionsReq struct {
	g.Meta   `path:"/cash-sessions" method:"get" tags:"Tài chính & Kế toán" summary:"Danh sách ca thu ngân" perm:"cash_session.manage,expense.view"`
	BranchID uint   `json:"branch_id" in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListCashSessionsRes []*CashSessionItem

// ---- Admin: Báo cáo P&L (/reports/profit-loss) ----------------------------------

type ProfitLossReportReq struct {
	g.Meta   `path:"/reports/profit-loss" method:"get" tags:"Tài chính & Kế toán" summary:"Báo cáo lãi lỗ (P&L)" perm:"expense.view,report.view"`
	BranchID uint   `json:"branch_id" in:"query"`
	From     string `json:"from"      in:"query"`
	To       string `json:"to"        in:"query"`
}
type ProfitLossReportRes struct {
	From              string  `json:"from"`
	To                string  `json:"to"`
	MembershipRevenue float64 `json:"membership_revenue"`
	PTRevenue         float64 `json:"pt_revenue"`
	POSRevenue        float64 `json:"pos_revenue"`
	LockerRevenue     float64 `json:"locker_revenue"`
	OtherRevenue      float64 `json:"other_revenue"`
	GrossRevenue      float64 `json:"gross_revenue"`
	Refunds           float64 `json:"refunds"`
	NetRevenue        float64 `json:"net_revenue"`
	POSCost           float64 `json:"pos_cost"`
	OperatingExpenses float64 `json:"operating_expenses"`
	PayrollExpenses   float64 `json:"payroll_expenses"`
	TotalExpenses     float64 `json:"total_expenses"`
	NetProfit         float64 `json:"net_profit"`
	// ExpensesByCategory: chi phí vận hành đã duyệt theo danh mục (tổng = operating_expenses).
	ExpensesByCategory []*ExpenseByCategoryItem `json:"expenses_by_category"`
}

type ExpenseByCategoryItem struct {
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	CategoryType string  `json:"category_type"`
	Amount       float64 `json:"amount"`
}

// ---- Admin: Trả góp & Hóa đơn điện tử -------------------------------------------

type CreateInstallmentPlanReq struct {
	g.Meta       `path:"/invoices/{id}/installments" method:"post" tags:"Tài chính & Kế toán" summary:"Tạo kế hoạch trả góp cho hóa đơn" perm:"invoice.collect"`
	ID           uint   `json:"id"             in:"path" v:"required|min:1"`
	Installments int    `json:"installments"   v:"required|between:2,12#Vui lòng nhập số kỳ trả góp|Số kỳ trả góp phải từ 2 đến 12 kỳ"`
	FirstDueDate string `json:"first_due_date"`
}
type CreateInstallmentPlanRes = InstallmentPlanItem

type ListInstallmentsReq struct {
	g.Meta    `path:"/installments" method:"get" tags:"Tài chính & Kế toán" summary:"Danh sách các kỳ trả góp" perm:"invoice.view"`
	Status    string `json:"status"     in:"query"`
	MemberID  uint   `json:"member_id"  in:"query"`
	InvoiceID uint   `json:"invoice_id" in:"query"`
}
type ListInstallmentsRes []*InstallmentScheduleItem

type PayInstallmentScheduleReq struct {
	g.Meta `path:"/installments/{id}/pay" method:"post" tags:"Tài chính & Kế toán" summary:"Thu tiền một kỳ trả góp" perm:"invoice.collect"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Method string `json:"method" d:"CASH" dc:"CASH | TRANSFER | CARD (không phân biệt hoa/thường)"`
	Note   string `json:"note"`
}
type PayInstallmentScheduleRes = InstallmentScheduleItem

type IssueEInvoiceReq struct {
	g.Meta `path:"/invoices/{id}/einvoice" method:"post" tags:"Tài chính & Kế toán" summary:"Xuất hóa đơn điện tử" perm:"invoice.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type IssueEInvoiceRes = einvoice.IssuedEInvoice

// ---- Member (/v1/member) --------------------------------------------------------

type MemberInstallmentsReq struct {
	g.Meta `path:"/installments" method:"get" tags:"Cổng hội viên" summary:"Kế hoạch trả góp của tôi" role:"MEMBER"`
}
type MemberInstallmentsRes []*InstallmentPlanItem
