package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== GLOBAL SEARCH (CTRL+K) & BULK ACTION (MỤC 15.1) ====================

type SearchHit struct {
	ID       uint   `json:"id"`
	Category string `json:"category"` // MEMBER | INVOICE | PRODUCT | PLAN
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Link     string `json:"link"`
}

type GlobalSearchReq struct {
	g.Meta `path:"/search" method:"get" perm:"member.view,invoice.view,product.view,plan.view" tags:"Hệ thống – Tiện ích" summary:"Tìm kiếm nhanh toàn hệ thống (Ctrl+K)"`
	Q      string `json:"q" in:"query" v:"required|length:1,100#Vui lòng nhập từ khóa tìm kiếm|Từ khóa tìm kiếm tối đa 100 ký tự"`
}

type GlobalSearchRes struct {
	Items []*SearchHit `json:"items"`
}

type BulkMemberActionReq struct {
	g.Meta     `path:"/members/bulk-action" method:"post" perm:"member.update" tags:"Hệ thống – Tiện ích" summary:"Thao tác hàng loạt trên danh sách hội viên"`
	MemberIDs  []uint `json:"member_ids" v:"required"`
	Action     string `json:"action" v:"required|in:NOTIFY,TRANSFER_BRANCH,ACTIVATE,DEACTIVATE"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	ToBranchID uint   `json:"to_branch_id"`
}

type BulkMemberActionRes struct {
	Affected int    `json:"affected"`
	Message  string `json:"message"`
}

// ==================== CHUYỂN CHI NHÁNH CHÍNH & HOÀN TIỀN ONLINE (MỤC 15.1) ====================

type BranchTransferItem struct {
	ID             uint       `json:"id"`
	MemberID       uint       `json:"member_id"`
	MemberName     string     `json:"member_name"`
	MemberCode     string     `json:"member_code"`
	FromBranchID   uint       `json:"from_branch_id"`
	FromBranchName string     `json:"from_branch_name"`
	ToBranchID     uint       `json:"to_branch_id"`
	ToBranchName   string     `json:"to_branch_name"`
	SubscriptionID *uint      `json:"subscription_id,omitempty"`
	TransferFee    float64    `json:"transfer_fee"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ListBranchTransfersReq struct {
	g.Meta `path:"/branch-transfers" method:"get" perm:"member.view,subscription.transfer" tags:"Hệ thống – Tiện ích" summary:"Danh sách lịch sử chuyển chi nhánh chính"`
}

type ListBranchTransfersRes []*BranchTransferItem

type CreateBranchTransferReq struct {
	g.Meta      `path:"/branch-transfers" method:"post" perm:"member.update,subscription.transfer" tags:"Hệ thống – Tiện ích" summary:"Chuyển chi nhánh chính của hội viên"`
	MemberID    uint    `json:"member_id" v:"required|min:1"`
	ToBranchID  uint    `json:"to_branch_id" v:"required|min:1"`
	TransferFee float64 `json:"transfer_fee" v:"min:0"`
	Reason      string  `json:"reason"`
}

type CreateBranchTransferRes struct {
	*BranchTransferItem
}

type RefundOnlineTxnReq struct {
	g.Meta `path:"/online-payments/{id}/refund" method:"post" perm:"invoice.refund" tags:"Tài chính – Hóa đơn" summary:"Hoàn tiền giao dịch VNPay trực tuyến"`
	ID     uint    `json:"id" in:"path" v:"required|min:1"`
	Amount float64 `json:"amount"`
	Reason string  `json:"reason" v:"required|length:3,255"`
}

type RefundOnlineTxnRes struct {
	RefundID   uint      `json:"refund_id"`
	InvoiceID  uint      `json:"invoice_id"`
	Amount     float64   `json:"amount"`
	Status     string    `json:"status"`
	GatewayRef string    `json:"gateway_ref"`
	CreatedAt  time.Time `json:"created_at"`
}

// ==================== CHUYỂN ĐỔI DỮ LIỆU CŨ & WIZARD THIẾT LẬP (SPRINT 30 / MỤC 15.2) ====================

type LegacyImportRow struct {
	FullName            string  `json:"full_name"`
	Phone               string  `json:"phone"`
	Email               string  `json:"email"`
	Gender              string  `json:"gender"`
	PlanCode            string  `json:"plan_code"`
	SubStartDate        string  `json:"sub_start_date"`
	SubEndDate          string  `json:"sub_end_date"`
	RemainingSessions   *int    `json:"remaining_sessions"`
	PTPackageCode       string  `json:"pt_package_code"`
	PTRemainingSessions *int    `json:"pt_remaining_sessions"`
	DebtAmount          float64 `json:"debt_amount"`
}

type LegacyImportErrorItem struct {
	Row     int    `json:"row"`
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

type LegacyImportJobItem struct {
	ID         uint                    `json:"id"`
	BranchID   uint                    `json:"branch_id"`
	BranchName string                  `json:"branch_name"`
	FileName   string                  `json:"file_name"`
	Mode       string                  `json:"mode"`
	Status     string                  `json:"status"`
	TotalRows  int                     `json:"total_rows"`
	OkRows     int                     `json:"ok_rows"`
	ErrorRows  int                     `json:"error_rows"`
	Errors     []LegacyImportErrorItem `json:"errors"`
	CreatedAt  time.Time               `json:"created_at"`
}

type ListLegacyImportJobsReq struct {
	g.Meta `path:"/legacy-imports" method:"get" perm:"import.legacy,member.import" tags:"Hệ thống – Tiện ích" summary:"Danh sách phiên chuyển đổi dữ liệu cũ"`
}

type ListLegacyImportJobsRes []*LegacyImportJobItem

type RunLegacyImportReq struct {
	g.Meta   `path:"/legacy-imports" method:"post" perm:"import.legacy,member.import" tags:"Hệ thống – Tiện ích" summary:"Chạy thử (Dry-Run) hoặc Ghi chính thức (Commit) chuyển đổi dữ liệu cũ"`
	BranchID uint              `json:"branch_id" v:"required|min:1"`
	FileName string            `json:"file_name" d:"import-legacy.xlsx"`
	Mode     string            `json:"mode" d:"DRY_RUN" v:"in:DRY_RUN,COMMIT"`
	Rows     []LegacyImportRow `json:"rows" v:"required"`
}

type RunLegacyImportRes struct {
	*LegacyImportJobItem
}

type SetupWizardStepStatus struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Count     int64  `json:"count"`
	Route     string `json:"route"`
}

type GetSetupWizardStatusReq struct {
	g.Meta `path:"/setup-wizard/status" method:"get" perm:"setup.wizard,setting.manage,branch.view" tags:"Hệ thống – Tiện ích" summary:"Trạng thái 6 bước thiết lập ban đầu"`
}

type GetSetupWizardStatusRes struct {
	Completed bool                     `json:"completed"`
	Skipped   bool                     `json:"skipped"`
	Steps     []*SetupWizardStepStatus `json:"steps"`
}

type CompleteSetupWizardReq struct {
	g.Meta            `path:"/setup-wizard/complete" method:"post" perm:"setup.wizard,setting.manage" tags:"Hệ thống – Tiện ích" summary:"Lưu cấu hình nhanh hoặc đánh dấu hoàn tất Wizard"`
	CenterName        string `json:"center_name"`
	Hotline           string `json:"hotline"`
	OpeningHours      string `json:"opening_hours"`
	CancelClassBefore int    `json:"cancel_class_before_hours"`
	Skip              bool   `json:"skip"`
}

type CompleteSetupWizardRes struct {
	*GetSetupWizardStatusRes
}
