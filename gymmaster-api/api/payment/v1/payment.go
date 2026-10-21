package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// ---- Cổng hội viên (/v1/member/*, role MEMBER) ----------------------------------

type CreateInvoiceVNPayReq struct {
	g.Meta `path:"/invoices/{id}/vnpay" method:"post" role:"MEMBER" tags:"Cổng hội viên – Thanh toán" summary:"Tạo link thanh toán VNPay cho hóa đơn"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type PaymentURLRes struct {
	TxnRef     string    `json:"txn_ref"`
	PaymentURL string    `json:"payment_url"`
	Amount     float64   `json:"amount"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type CreateInvoiceVNPayRes = PaymentURLRes

type CheckoutSubscriptionReq struct {
	g.Meta        `path:"/subscriptions/checkout" method:"post" role:"MEMBER" tags:"Cổng hội viên – Thanh toán" summary:"Hội viên tự chọn mua gói tập và thanh toán VNPay"`
	PlanID        uint   `json:"plan_id"        v:"required|min:1"`
	BranchID      uint   `json:"branch_id"`
	PromotionCode string `json:"promotion_code"`
}

type CheckoutSubscriptionRes struct {
	SubscriptionID uint    `json:"subscription_id"`
	InvoiceID      uint    `json:"invoice_id"`
	InvoiceNo      string  `json:"invoice_no"`
	Amount         float64 `json:"amount"`
	TxnRef         string  `json:"txn_ref"`
	PaymentURL     string  `json:"payment_url"`
}

type CheckPromotionReq struct {
	g.Meta `path:"/promotions/check" method:"get" role:"MEMBER" tags:"Cổng hội viên – Thanh toán" summary:"Kiểm tra mã khuyến mãi khi mua gói"`
	Code   string  `json:"code"    in:"query" v:"required"`
	PlanID uint    `json:"plan_id" in:"query"`
	Amount float64 `json:"amount"  in:"query"`
}

type CheckPromotionRes struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Discount    float64 `json:"discount"`
	FinalAmount float64 `json:"final_amount"`
}

// ---- Công khai (/v1/public/*) ---------------------------------------------------

type VNPayIPNReq struct {
	g.Meta `path:"/payments/vnpay/ipn" method:"get" tags:"Công khai – Thanh toán" summary:"VNPay IPN webhook"`
}

type VNPayIPNRes struct {
	RspCode string `json:"RspCode"`
	Message string `json:"Message"`
}

type VNPayReturnReq struct {
	g.Meta `path:"/payments/vnpay/return" method:"get" tags:"Công khai – Thanh toán" summary:"Xác thực kết quả VNPay Return URL"`
}

type VNPayReturnRes struct {
	Success      bool    `json:"success"`
	TxnRef       string  `json:"txn_ref"`
	InvoiceID    uint    `json:"invoice_id"`
	InvoiceNo    string  `json:"invoice_no"`
	Amount       float64 `json:"amount"`
	ResponseCode string  `json:"response_code"`
	Message      string  `json:"message"`
}

// ---- Quản trị (/v1/admin/*) -----------------------------------------------------

type OnlineTxnItem struct {
	ID            uint       `json:"id"`
	InvoiceID     uint       `json:"invoice_id"`
	InvoiceNo     string     `json:"invoice_no"`
	MemberID      uint       `json:"member_id"`
	MemberCode    string     `json:"member_code"`
	MemberName    string     `json:"member_name"`
	TxnRef        string     `json:"txn_ref"`
	Amount        float64    `json:"amount"`
	Status        string     `json:"status"          dc:"PENDING | SUCCESS | FAILED | EXPIRED | NEEDS_REFUND (đã nhận tiền, cần hoàn) | REFUNDED"`
	ProviderTxnNo string     `json:"provider_txn_no"`
	BankCode      string     `json:"bank_code"`
	CreatedAt     time.Time  `json:"created_at"`
	PaidAt        *time.Time `json:"paid_at"`
	// RecordedAmount: phần đã ghi nhận vào hóa đơn; Amount − RecordedAmount (chưa hoàn) là phần cần hoàn cho khách.
	RecordedAmount float64 `json:"recorded_amount"`
	RefundedAmount float64 `json:"refunded_amount"`
	Refundable     float64 `json:"refundable"      dc:"Số còn có thể hoàn = amount − refunded_amount"`
	Note           string  `json:"note"`
}

type ListOnlineTxnsReq struct {
	g.Meta `path:"/online-payments" method:"get" perm:"invoice.view" tags:"Tài chính – Hóa đơn" summary:"Danh sách giao dịch thanh toán trực tuyến VNPay"`
	dto.PageReq
	Status string `json:"status" in:"query"`
	Search string `json:"search" in:"query"`
}

type ListOnlineTxnsRes = dto.PageRes[*OnlineTxnItem]
