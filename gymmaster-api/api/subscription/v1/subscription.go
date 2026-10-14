package v1

import (
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	invoicev1 "gymmaster-api/api/invoice/v1"
	"gymmaster-api/internal/dto"
)

// ---- Đối tượng trả về ---------------------------------------------------------

// Subscription là một dòng trong danh sách đăng ký gói tập.
type Subscription struct {
	ID                uint      `json:"id"`
	MemberID          uint      `json:"member_id"`
	MemberCode        string    `json:"member_code"`
	MemberName        string    `json:"member_name"`
	MemberPhone       string    `json:"member_phone"`
	PlanID            uint      `json:"plan_id"`
	PlanCode          string    `json:"plan_code"`
	PlanName          string    `json:"plan_name"`
	PlanType          string    `json:"plan_type"     dc:"TIME | SESSION"`
	AccessScope       string    `json:"access_scope"  dc:"SINGLE_BRANCH | ALL_BRANCHES"`
	BranchID          uint      `json:"branch_id"`
	BranchName        string    `json:"branch_name"`
	InvoiceID         *uint     `json:"invoice_id"`
	InvoiceNo         string    `json:"invoice_no"`
	InvoiceStatus     string    `json:"invoice_status"`
	StartDate         string    `json:"start_date"    dc:"YYYY-MM-DD"`
	EndDate           string    `json:"end_date"      dc:"YYYY-MM-DD"`
	TotalSessions     *int      `json:"total_sessions"`
	RemainingSessions *int      `json:"remaining_sessions"`
	Price             float64   `json:"price"         dc:"Giá trị gói (số tiền thực bán sau khuyến mãi)"`
	Status            string    `json:"status"        dc:"PENDING | ACTIVE | FROZEN | EXPIRED | CANCELLED | TRANSFERRED"`
	FreezeDaysUsed    int       `json:"freeze_days_used"`
	FreezeTimesUsed   int       `json:"freeze_times_used"`
	PreviousID        *uint     `json:"previous_id"`
	DaysLeft          *int      `json:"days_left"     dc:"Số ngày còn lại đến end_date (chỉ với PENDING/ACTIVE/FROZEN)"`
	SoldBy            *uint     `json:"sold_by"`
	SoldByName        string    `json:"sold_by_name"`
	Note              string    `json:"note"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PlanRef struct {
	ID             uint    `json:"id"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	DurationDays   int     `json:"duration_days"`
	SessionCount   *int    `json:"session_count"`
	Price          float64 `json:"price"`
	AccessScope    string  `json:"access_scope"`
	MaxFreezeDays  int     `json:"max_freeze_days"`
	MaxFreezeTimes int     `json:"max_freeze_times"`
	TransferFee    float64 `json:"transfer_fee"`
	Status         string  `json:"status"`
}

type MemberRef struct {
	ID         uint   `json:"id"`
	MemberCode string `json:"member_code"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Status     string `json:"status"`
}

type Freeze struct {
	ID              uint       `json:"id"`
	SubscriptionID  uint       `json:"subscription_id"`
	MemberID        uint       `json:"member_id"`
	MemberCode      string     `json:"member_code"`
	MemberName      string     `json:"member_name"`
	PlanName        string     `json:"plan_name"`
	BranchID        uint       `json:"branch_id"`
	BranchName      string     `json:"branch_name"`
	StartDate       string     `json:"start_date"`
	EndDate         string     `json:"end_date"`
	Days            int        `json:"days"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status" dc:"PENDING | APPROVED | REJECTED | ENDED | CANCELLED"`
	RequestedBy     *uint      `json:"requested_by"`
	RequestedByName string     `json:"requested_by_name"`
	ReviewedBy      *uint      `json:"reviewed_by"`
	ReviewedByName  string     `json:"reviewed_by_name"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ReviewNote      string     `json:"review_note"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Event struct {
	ID        uint            `json:"id"`
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
	ActorID   *uint           `json:"actor_id"  dc:"null = hệ thống"`
	ActorName string          `json:"actor_name"`
	CreatedAt time.Time       `json:"created_at"`
}

// SubscriptionDetail là chi tiết gói tập kèm gói, hội viên, hóa đơn, bảo lưu và lịch sử.
type SubscriptionDetail struct {
	Subscription
	Plan    PlanRef                  `json:"plan"`
	Member  MemberRef                `json:"member"`
	Invoice *invoicev1.InvoiceDetail `json:"invoice"`
	Freezes []*Freeze                `json:"freezes"`
	Events  []*Event                 `json:"events"`
}

// ---- Input dùng chung ---------------------------------------------------------

// PaymentInput: thu tiền ngay khi bán (tùy chọn).
type PaymentInput struct {
	Amount    float64 `json:"amount"    dc:"Số tiền thu (> 0, ≤ tổng hóa đơn)"`
	Method    string  `json:"method"    dc:"CASH | TRANSFER | CARD"`
	Reference string  `json:"reference" dc:"Mã giao dịch"`
	Note      string  `json:"note"`
}

type RefundInput struct {
	Amount float64 `json:"amount"`
	Method string  `json:"method" dc:"CASH | TRANSFER | CARD"`
}

type SellInput struct {
	MemberID      uint   `json:"member_id"      v:"required|min:1#Vui lòng chọn hội viên|Vui lòng chọn hội viên"`
	PlanID        uint   `json:"plan_id"        v:"required|min:1#Vui lòng chọn gói tập|Vui lòng chọn gói tập"`
	BranchID      uint   `json:"branch_id"      v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	StartDate     string `json:"start_date"     v:"date-format:Y-m-d#Ngày bắt đầu phải có dạng YYYY-MM-DD" dc:"Bỏ trống = hôm nay, hoặc nối tiếp gói đang chạy"`
	PromotionCode string `json:"promotion_code" v:"max-length:30#Mã khuyến mãi tối đa 30 ký tự"`
	Note          string `json:"note"           v:"max-length:500#Ghi chú tối đa 500 ký tự"`
}

// ---- Bán gói ------------------------------------------------------------------

type SellReq struct {
	g.Meta `path:"/subscriptions" method:"post" tags:"Gói tập hội viên" summary:"Bán gói tập" perm:"subscription.sell"`
	SellInput
	Payment *PaymentInput `json:"payment" dc:"Thu tiền ngay (tùy chọn)"`
}
type SellRes = SubscriptionDetail

type QuoteReq struct {
	g.Meta `path:"/subscriptions/quote" method:"post" tags:"Gói tập hội viên" summary:"Xem trước bán gói (không ghi DB)" perm:"subscription.sell"`
	SellInput
}

type PromotionApplied struct {
	ID       uint    `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Discount float64 `json:"discount"`
}

type QuoteRes struct {
	MemberID      uint              `json:"member_id"`
	BranchID      uint              `json:"branch_id"`
	Plan          PlanRef           `json:"plan"`
	StartDate     string            `json:"start_date"`
	EndDate       string            `json:"end_date"`
	Chained       bool              `json:"chained"     dc:"true = nối tiếp gói đang chạy"`
	PreviousID    *uint             `json:"previous_id" dc:"Gói được nối tiếp"`
	TotalSessions *int              `json:"total_sessions"`
	Subtotal      float64           `json:"subtotal"`
	Discount      float64           `json:"discount"`
	Total         float64           `json:"total"`
	Promotion     *PromotionApplied `json:"promotion"`
}

// ---- Danh sách / chi tiết -----------------------------------------------------

type ListReq struct {
	g.Meta `path:"/subscriptions" method:"get" tags:"Gói tập hội viên" summary:"Danh sách đăng ký gói tập" perm:"subscription.view"`
	dto.PageReq
	BranchID       uint   `json:"branch_id"        in:"query"`
	MemberID       uint   `json:"member_id"        in:"query"`
	PlanID         uint   `json:"plan_id"          in:"query"`
	Status         string `json:"status"           in:"query" v:"in:PENDING,ACTIVE,FROZEN,EXPIRED,CANCELLED,TRANSFERRED#Trạng thái không hợp lệ"`
	ExpiringInDays int    `json:"expiring_in_days" in:"query" v:"between:0,365#Số ngày sắp hết hạn từ 0 đến 365" dc:"Gói ACTIVE/FROZEN hết hạn trong N ngày tới (0 = bỏ qua)"`
	From           string `json:"from"             in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"end_date từ"`
	To             string `json:"to"               in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"end_date đến"`
}
type ListRes = dto.PageRes[*Subscription]

type GetReq struct {
	g.Meta `path:"/subscriptions/{id}" method:"get" tags:"Gói tập hội viên" summary:"Chi tiết gói tập" perm:"subscription.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = SubscriptionDetail

type EventsReq struct {
	g.Meta `path:"/subscriptions/{id}/events" method:"get" tags:"Gói tập hội viên" summary:"Lịch sử vòng đời gói tập" perm:"subscription.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type EventsRes []*Event

type MemberSubscriptionsReq struct {
	g.Meta `path:"/members/{id}/subscriptions" method:"get" tags:"Gói tập hội viên" summary:"Lịch sử gói tập của hội viên" perm:"subscription.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type MemberSubscriptionsRes []*Subscription

// ---- Gia hạn / nâng cấp -------------------------------------------------------

type RenewInput struct {
	PlanID        uint          `json:"plan_id"        dc:"Bỏ trống = gói hiện tại"`
	PromotionCode string        `json:"promotion_code" v:"max-length:30#Mã khuyến mãi tối đa 30 ký tự"`
	Note          string        `json:"note"           v:"max-length:500#Ghi chú tối đa 500 ký tự"`
	Payment       *PaymentInput `json:"payment"`
}

type RenewReq struct {
	g.Meta `path:"/subscriptions/{id}/renew" method:"post" tags:"Gói tập hội viên" summary:"Gia hạn (bán gói mới nối tiếp)" perm:"subscription.sell"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	RenewInput
}
type RenewRes = SubscriptionDetail

type RenewQuoteReq struct {
	g.Meta `path:"/subscriptions/{id}/renew/quote" method:"post" tags:"Gói tập hội viên" summary:"Xem trước gia hạn" perm:"subscription.sell"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	RenewInput
}
type RenewQuoteRes = QuoteRes

type UpgradeInput struct {
	PlanID  uint          `json:"plan_id" v:"required|min:1#Vui lòng chọn gói mới|Vui lòng chọn gói mới"`
	Note    string        `json:"note"    v:"max-length:500#Ghi chú tối đa 500 ký tự"`
	Payment *PaymentInput `json:"payment"`
}

type UpgradeReq struct {
	g.Meta `path:"/subscriptions/{id}/upgrade" method:"post" tags:"Gói tập hội viên" summary:"Nâng cấp gói (khấu trừ giá trị còn lại)" perm:"subscription.sell"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	UpgradeInput
}
type UpgradeRes = SubscriptionDetail

type UpgradeQuoteReq struct {
	g.Meta `path:"/subscriptions/{id}/upgrade/quote" method:"post" tags:"Gói tập hội viên" summary:"Xem trước nâng cấp" perm:"subscription.sell"`
	ID     uint `json:"id"      in:"path" v:"required|min:1"`
	PlanID uint `json:"plan_id" v:"required|min:1#Vui lòng chọn gói mới|Vui lòng chọn gói mới"`
}

type UpgradeQuoteRes struct {
	SubscriptionID    uint    `json:"subscription_id"`
	CurrentPlan       PlanRef `json:"current_plan"`
	NewPlan           PlanRef `json:"new_plan"`
	CurrentPrice      float64 `json:"current_price"      dc:"Giá trị gói cũ (subscriptions.price)"`
	TotalDays         int     `json:"total_days"`
	RemainingDays     int     `json:"remaining_days"`
	TotalSessions     *int    `json:"total_sessions"`
	RemainingSessions *int    `json:"remaining_sessions"`
	RemainingValue    float64 `json:"remaining_value"    dc:"Giá trị còn lại được khấu trừ"`
	NewPrice          float64 `json:"new_price"`
	AmountDue         float64 `json:"amount_due"         dc:"= max(0, new_price − remaining_value)"`
	StartDate         string  `json:"start_date"`
	EndDate           string  `json:"end_date"`
}

// ---- Bảo lưu ------------------------------------------------------------------

type FreezeInput struct {
	StartDate string `json:"start_date" v:"required|date-format:Y-m-d#Vui lòng chọn ngày bắt đầu|Ngày bắt đầu phải có dạng YYYY-MM-DD"`
	EndDate   string `json:"end_date"   v:"required|date-format:Y-m-d#Vui lòng chọn ngày kết thúc|Ngày kết thúc phải có dạng YYYY-MM-DD"`
	Reason    string `json:"reason"     v:"max-length:255#Lý do tối đa 255 ký tự"`
}

type FreezeReq struct {
	g.Meta `path:"/subscriptions/{id}/freezes" method:"post" tags:"Gói tập hội viên" summary:"Tạo yêu cầu bảo lưu" perm:"subscription.freeze"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	FreezeInput
}
type FreezeRes = Freeze

type FreezeListReq struct {
	g.Meta `path:"/subscription-freezes" method:"get" tags:"Gói tập hội viên" summary:"Danh sách yêu cầu bảo lưu" perm:"subscription.view,subscription.approve_freeze"`
	dto.PageReq
	Status         string `json:"status"          in:"query" v:"in:PENDING,APPROVED,REJECTED,ENDED,CANCELLED#Trạng thái không hợp lệ"`
	BranchID       uint   `json:"branch_id"       in:"query"`
	SubscriptionID uint   `json:"subscription_id" in:"query"`
}
type FreezeListRes = dto.PageRes[*Freeze]

type ReviewInput struct {
	Note string `json:"note" v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}

type ApproveFreezeReq struct {
	g.Meta `path:"/subscription-freezes/{id}/approve" method:"post" tags:"Gói tập hội viên" summary:"Duyệt bảo lưu" perm:"subscription.approve_freeze"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ReviewInput
}
type ApproveFreezeRes = Freeze

type RejectFreezeReq struct {
	g.Meta `path:"/subscription-freezes/{id}/reject" method:"post" tags:"Gói tập hội viên" summary:"Từ chối bảo lưu" perm:"subscription.approve_freeze"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ReviewInput
}
type RejectFreezeRes = Freeze

type EndFreezeReq struct {
	g.Meta `path:"/subscription-freezes/{id}/end-early" method:"post" tags:"Gói tập hội viên" summary:"Kết thúc bảo lưu sớm / hủy yêu cầu chưa áp dụng" perm:"subscription.freeze,subscription.approve_freeze"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ReviewInput
}
type EndFreezeRes = Freeze

// ---- Chuyển nhượng / hủy ------------------------------------------------------

type TransferInput struct {
	ToMemberID uint          `json:"to_member_id" v:"required|min:1#Vui lòng chọn hội viên nhận|Vui lòng chọn hội viên nhận"`
	Note       string        `json:"note"         v:"max-length:500#Ghi chú tối đa 500 ký tự"`
	Payment    *PaymentInput `json:"payment"      dc:"Thu phí chuyển nhượng ngay"`
}

type TransferReq struct {
	g.Meta `path:"/subscriptions/{id}/transfer" method:"post" tags:"Gói tập hội viên" summary:"Chuyển nhượng gói" perm:"subscription.transfer"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	TransferInput
}
type TransferRes = SubscriptionDetail

type CancelInput struct {
	Reason string       `json:"reason" v:"required|max-length:255#Vui lòng nhập lý do hủy|Lý do tối đa 255 ký tự"`
	Refund *RefundInput `json:"refund" dc:"Hoàn tiền kèm theo (tùy chọn)"`
}

type CancelReq struct {
	g.Meta `path:"/subscriptions/{id}/cancel" method:"post" tags:"Gói tập hội viên" summary:"Hủy gói tập" perm:"subscription.cancel"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	CancelInput
}
type CancelRes = SubscriptionDetail
