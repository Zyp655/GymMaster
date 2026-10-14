package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// MemberRef là thông tin rút gọn của hội viên trên hóa đơn.
type MemberRef struct {
	ID         uint   `json:"id"`
	MemberCode string `json:"member_code"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
}

// Invoice là một dòng trong danh sách hóa đơn.
type Invoice struct {
	ID             uint       `json:"id"`
	InvoiceNo      string     `json:"invoice_no"`
	MemberID       *uint      `json:"member_id"`
	Member         *MemberRef `json:"member"      dc:"null = khách vãng lai"`
	BranchID       uint       `json:"branch_id"`
	BranchName     string     `json:"branch_name"`
	Subtotal       float64    `json:"subtotal"    dc:"Tổng tiền hàng trước giảm giá"`
	Discount       float64    `json:"discount"    dc:"Tổng giảm giá (giảm theo dòng + khuyến mãi)"`
	Total          float64    `json:"total"       dc:"= subtotal - discount"`
	PaidAmount     float64    `json:"paid_amount"`
	RefundedAmount float64    `json:"refunded_amount"`
	Due            float64    `json:"due"         dc:"Còn phải thu = total - paid_amount"`
	Status         string     `json:"status"      dc:"UNPAID | PARTIAL | PAID | VOID | REFUNDED"`
	Source         string     `json:"source"      dc:"COUNTER | POS | PORTAL"`
	PromotionID    *uint      `json:"promotion_id"`
	PromotionCode  string     `json:"promotion_code"`
	Note           string     `json:"note"`
	IssuedBy       *uint      `json:"issued_by"`
	IssuedByName   string     `json:"issued_by_name"`
	IssuedAt       time.Time  `json:"issued_at"`
	VoidedAt       *time.Time `json:"voided_at"`
	VoidReason     string     `json:"void_reason"`
	CreatedAt      time.Time  `json:"created_at"`
}

type InvoiceItem struct {
	ID          uint    `json:"id"`
	ItemType    string  `json:"item_type" dc:"MEMBERSHIP | PT | PRODUCT | FEE | OTHER"`
	RefID       *uint   `json:"ref_id"    dc:"MEMBERSHIP/FEE: subscriptions.id · PT: pt_contracts.id · PRODUCT: products.id"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Discount    float64 `json:"discount"  dc:"Giảm giá của dòng (gồm phần khuyến mãi được phân bổ)"`
	Amount      float64 `json:"amount"    dc:"= quantity × unit_price − discount"`
}

type Payment struct {
	ID             uint      `json:"id"`
	InvoiceID      uint      `json:"invoice_id"`
	Amount         float64   `json:"amount"`
	Method         string    `json:"method" dc:"CASH | TRANSFER | CARD"`
	Reference      string    `json:"reference"`
	PaidAt         time.Time `json:"paid_at"`
	ReceivedBy     *uint     `json:"received_by"`
	ReceivedByName string    `json:"received_by_name"`
	Note           string    `json:"note"`
}

type Refund struct {
	ID             uint      `json:"id"`
	InvoiceID      uint      `json:"invoice_id"`
	Amount         float64   `json:"amount"`
	Method         string    `json:"method"`
	Reason         string    `json:"reason"`
	RefundedBy     *uint     `json:"refunded_by"`
	RefundedByName string    `json:"refunded_by_name"`
	CreatedAt      time.Time `json:"created_at"`
}

// InvoiceDetail là hóa đơn kèm các dòng, lần thu và lần hoàn tiền.
type InvoiceDetail struct {
	Invoice
	VoidedByName string         `json:"voided_by_name"`
	Items        []*InvoiceItem `json:"items"`
	Payments     []*Payment     `json:"payments"`
	Refunds      []*Refund      `json:"refunds"`
}

// PaymentInput là body thu tiền.
type PaymentInput struct {
	Amount    float64 `json:"amount"    v:"required|min:1#Vui lòng nhập số tiền|Số tiền phải lớn hơn 0"`
	Method    string  `json:"method"    v:"required|in:CASH,TRANSFER,CARD#Vui lòng chọn phương thức thanh toán|Phương thức thanh toán không hợp lệ"`
	Reference string  `json:"reference" v:"max-length:100#Mã giao dịch tối đa 100 ký tự" dc:"Mã giao dịch chuyển khoản / POS thẻ"`
	Note      string  `json:"note"      v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}

// RefundInput là body hoàn tiền.
type RefundInput struct {
	Amount float64 `json:"amount" v:"required|min:1#Vui lòng nhập số tiền hoàn|Số tiền hoàn phải lớn hơn 0"`
	Method string  `json:"method" v:"required|in:CASH,TRANSFER,CARD#Vui lòng chọn phương thức hoàn tiền|Phương thức hoàn tiền không hợp lệ"`
	Reason string  `json:"reason" v:"required|max-length:255#Vui lòng nhập lý do hoàn tiền|Lý do tối đa 255 ký tự"`
}

type ListReq struct {
	g.Meta `path:"/invoices" method:"get" tags:"Hóa đơn" summary:"Danh sách hóa đơn" perm:"invoice.view"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query" dc:"Lọc chi nhánh (ngoài phạm vi → 403)"`
	MemberID uint   `json:"member_id" in:"query"`
	Status   string `json:"status"    in:"query" v:"in:UNPAID,PARTIAL,PAID,VOID,REFUNDED#Trạng thái không hợp lệ"`
	Source   string `json:"source"    in:"query" v:"in:COUNTER,POS,PORTAL#Kênh bán không hợp lệ"`
	From     string `json:"from"      in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"Ngày lập từ (YYYY-MM-DD)"`
	To       string `json:"to"        in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"Ngày lập đến (YYYY-MM-DD, tính cả ngày)"`
}
type ListRes = dto.PageRes[*Invoice]

type GetReq struct {
	g.Meta `path:"/invoices/{id}" method:"get" tags:"Hóa đơn" summary:"Chi tiết hóa đơn" perm:"invoice.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = InvoiceDetail

type CollectReq struct {
	g.Meta `path:"/invoices/{id}/payments" method:"post" tags:"Hóa đơn" summary:"Thu tiền hóa đơn" perm:"invoice.collect"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	PaymentInput
}
type CollectRes = InvoiceDetail

type VoidReq struct {
	g.Meta `path:"/invoices/{id}/void" method:"post" tags:"Hóa đơn" summary:"Hủy hóa đơn (chỉ khi chưa thu tiền)" perm:"invoice.void"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Reason string `json:"reason" v:"required|max-length:255#Vui lòng nhập lý do hủy|Lý do tối đa 255 ký tự"`
}
type VoidRes = InvoiceDetail

type RefundReq struct {
	g.Meta `path:"/invoices/{id}/refunds" method:"post" tags:"Hóa đơn" summary:"Hoàn tiền hóa đơn" perm:"invoice.refund"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	RefundInput
}
type RefundRes = InvoiceDetail

// PrintReq trả dữ liệu đủ để in hóa đơn.
type PrintReq struct {
	g.Meta `path:"/invoices/{id}/print" method:"get" tags:"Hóa đơn" summary:"Dữ liệu in hóa đơn" perm:"invoice.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type PrintBranch struct {
	ID      uint   `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
}

type PrintRes struct {
	Invoice       *InvoiceDetail `json:"invoice"`
	Branch        PrintBranch    `json:"branch"`
	Member        *MemberRef     `json:"member"`
	TotalInWords  string         `json:"total_in_words"  dc:"Tổng tiền bằng chữ, ví dụ 'Một triệu hai trăm mười lăm nghìn đồng'"`
	DueInWords    string         `json:"due_in_words"`
	PrintedAt     time.Time      `json:"printed_at"`
	PrintedByName string         `json:"printed_by_name"`
}

// PaymentListReq là sổ thu tiền.
type PaymentListReq struct {
	g.Meta `path:"/payments" method:"get" tags:"Hóa đơn" summary:"Sổ thu tiền" perm:"invoice.view"`
	dto.PageReq
	BranchID   uint   `json:"branch_id"   in:"query"`
	Method     string `json:"method"      in:"query" v:"in:CASH,TRANSFER,CARD#Phương thức không hợp lệ"`
	From       string `json:"from"        in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD"`
	To         string `json:"to"          in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD"`
	ReceivedBy uint   `json:"received_by" in:"query" dc:"Người thu (users.id)"`
}

type PaymentItem struct {
	Payment
	InvoiceNo  string `json:"invoice_no"`
	BranchID   uint   `json:"branch_id"`
	BranchName string `json:"branch_name"`
	MemberID   *uint  `json:"member_id"`
	MemberCode string `json:"member_code"`
	MemberName string `json:"member_name"`
}

type MethodTotal struct {
	Method string  `json:"method"`
	Amount float64 `json:"amount"`
	Count  int64   `json:"count"`
}

type PaymentSummary struct {
	TotalAmount float64        `json:"total_amount"`
	Count       int64          `json:"count"`
	ByMethod    []*MethodTotal `json:"by_method" dc:"Luôn đủ 3 phương thức CASH, TRANSFER, CARD"`
}

type PaymentListRes struct {
	dto.PageRes[*PaymentItem]
	Summary PaymentSummary `json:"summary" dc:"Tổng theo toàn bộ bộ lọc (không chỉ trang hiện tại)"`
}
