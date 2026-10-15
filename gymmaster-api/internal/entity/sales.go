package entity

import (
	"time"

	"gorm.io/datatypes"
)

// Trạng thái / loại dùng trong module bán hàng (CHECK trong migration 000005).
const (
	// Trạng thái gói tập (subscriptions.status).
	SubStatusPending     = "PENDING"
	SubStatusActive      = "ACTIVE"
	SubStatusFrozen      = "FROZEN"
	SubStatusExpired     = "EXPIRED"
	SubStatusCancelled   = "CANCELLED"
	SubStatusTransferred = "TRANSFERRED"

	// Loại sự kiện vòng đời gói tập (subscription_events.event_type).
	SubEventCreated         = "CREATED"
	SubEventActivated       = "ACTIVATED"
	SubEventRenewed         = "RENEWED"
	SubEventUpgraded        = "UPGRADED"
	SubEventFreezeRequested = "FREEZE_REQUESTED"
	SubEventFrozen          = "FROZEN"
	SubEventUnfrozen        = "UNFROZEN"
	SubEventFreezeRejected  = "FREEZE_REJECTED"
	SubEventTransferred     = "TRANSFERRED"
	SubEventCancelled       = "CANCELLED"
	SubEventExpired         = "EXPIRED"
	SubEventSessionUsed     = "SESSION_USED"

	// Trạng thái yêu cầu bảo lưu (subscription_freezes.status).
	FreezeStatusPending   = "PENDING"
	FreezeStatusApproved  = "APPROVED"
	FreezeStatusRejected  = "REJECTED"
	FreezeStatusEnded     = "ENDED"
	FreezeStatusCancelled = "CANCELLED"

	// Trạng thái hóa đơn (invoices.status).
	InvoiceStatusUnpaid   = "UNPAID"
	InvoiceStatusPartial  = "PARTIAL"
	InvoiceStatusPaid     = "PAID"
	InvoiceStatusVoid     = "VOID"
	InvoiceStatusRefunded = "REFUNDED"

	// Kênh lập hóa đơn (invoices.source).
	InvoiceSourceCounter = "COUNTER"
	InvoiceSourcePOS     = "POS"
	InvoiceSourcePortal  = "PORTAL"

	// Loại dòng hóa đơn (invoice_items.item_type).
	InvoiceItemMembership = "MEMBERSHIP" // ref_id = subscriptions.id
	InvoiceItemPT         = "PT"         // ref_id = pt_contracts.id
	InvoiceItemProduct    = "PRODUCT"    // ref_id = products.id
	InvoiceItemFee        = "FEE"        // phí chuyển nhượng gói: ref_id = subscriptions.id (gói của người nhận)
	InvoiceItemLocker     = "LOCKER"     // ref_id = locker_rentals.id
	InvoiceItemOther      = "OTHER"

	// Phương thức thanh toán / hoàn tiền.
	PaymentCash     = "CASH"
	PaymentTransfer = "TRANSFER"
	PaymentCard     = "CARD"
	PaymentVNPay    = "VNPAY"

	// Trạng thái giao dịch thanh toán trực tuyến (online_payment_txns.status).
	OnlineTxnPending = "PENDING"
	OnlineTxnSuccess = "SUCCESS"
	OnlineTxnFailed  = "FAILED"
	OnlineTxnExpired = "EXPIRED"
)

// Subscription là một lần đăng ký gói tập của hội viên.
// StartDate/EndDate là cột DATE: luôn lưu dạng 00:00 UTC của ngày (theo lịch Việt Nam).
type Subscription struct {
	ID                uint `gorm:"primaryKey;autoIncrement"`
	MemberID          uint `gorm:"not null"`
	PlanID            uint `gorm:"not null"`
	BranchID          uint `gorm:"not null"`
	InvoiceID         *uint
	StartDate         time.Time `gorm:"type:date;not null"`
	EndDate           time.Time `gorm:"type:date;not null"`
	TotalSessions     *int
	RemainingSessions *int
	Price             float64 `gorm:"type:numeric(14,2);not null"`
	Status            string  `gorm:"type:varchar(20);not null"`
	FreezeDaysUsed    int     `gorm:"not null;default:0"`
	FreezeTimesUsed   int     `gorm:"not null;default:0"`
	PreviousID        *uint
	SoldBy            *uint
	Note              string `gorm:"type:varchar(500);not null;default:''"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (Subscription) TableName() string { return "subscriptions" }

// SubscriptionFreeze là một yêu cầu bảo lưu. Days là cột GENERATED (end_date - start_date + 1).
type SubscriptionFreeze struct {
	ID             uint      `gorm:"primaryKey;autoIncrement"`
	SubscriptionID uint      `gorm:"not null"`
	StartDate      time.Time `gorm:"type:date;not null"`
	EndDate        time.Time `gorm:"type:date;not null"`
	Days           int       `gorm:"->"`
	Reason         string    `gorm:"type:varchar(255);not null;default:''"`
	Status         string    `gorm:"type:varchar(20);not null"`
	RequestedBy    *uint
	ReviewedBy     *uint
	ReviewedAt     *time.Time
	ReviewNote     string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt      time.Time
}

func (SubscriptionFreeze) TableName() string { return "subscription_freezes" }

// SubscriptionEvent là một mốc trong vòng đời gói tập (timeline). ActorID nil = hệ thống (cron).
type SubscriptionEvent struct {
	ID             uint           `gorm:"primaryKey;autoIncrement"`
	SubscriptionID uint           `gorm:"not null"`
	EventType      string         `gorm:"type:varchar(30);not null"`
	Data           datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	ActorID        *uint
	CreatedAt      time.Time
}

func (SubscriptionEvent) TableName() string { return "subscription_events" }

// Invoice là hóa đơn dùng chung cho mọi loại bán (gói tập, PT, POS).
// Bất biến: Total = Subtotal - Discount; RefundedAmount <= PaidAmount <= Total.
type Invoice struct {
	ID             uint    `gorm:"primaryKey;autoIncrement"`
	InvoiceNo      string  `gorm:"->;type:varchar(30)"` // DB sinh: HDyyMMdd00001
	MemberID       *uint   // nil = khách vãng lai (POS)
	BranchID       uint    `gorm:"not null"`
	Subtotal       float64 `gorm:"type:numeric(14,2);not null"`
	Discount       float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Total          float64 `gorm:"type:numeric(14,2);not null"`
	PaidAmount     float64 `gorm:"type:numeric(14,2);not null;default:0"`
	RefundedAmount float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Status         string  `gorm:"type:varchar(20);not null"`
	Source         string  `gorm:"type:varchar(20);not null"`
	PromotionID    *uint
	Note           string `gorm:"type:varchar(500);not null;default:''"`
	IssuedBy       *uint
	IssuedAt       time.Time `gorm:"not null"`
	VoidedAt       *time.Time
	VoidedBy       *uint
	VoidReason     string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Items []InvoiceItem `gorm:"-"`
}

func (Invoice) TableName() string { return "invoices" }

// Due là số tiền còn phải thu.
func (i *Invoice) Due() float64 { return i.Total - i.PaidAmount }

// InvoiceItem là một dòng hóa đơn. Amount = Quantity × UnitPrice − Discount.
type InvoiceItem struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	InvoiceID   uint   `gorm:"not null"`
	ItemType    string `gorm:"type:varchar(20);not null"`
	RefID       *uint
	Description string  `gorm:"type:varchar(255);not null"`
	Quantity    int     `gorm:"not null;default:1"`
	UnitPrice   float64 `gorm:"type:numeric(14,2);not null"`
	Discount    float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Amount      float64 `gorm:"type:numeric(14,2);not null"`
}

func (InvoiceItem) TableName() string { return "invoice_items" }

// Payment là một lần thu tiền cho hóa đơn.
type Payment struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	InvoiceID  uint      `gorm:"not null"`
	Amount     float64   `gorm:"type:numeric(14,2);not null"`
	Method     string    `gorm:"type:varchar(20);not null"`
	Reference  string    `gorm:"type:varchar(100);not null;default:''"`
	PaidAt     time.Time `gorm:"not null"`
	ReceivedBy *uint
	Note       string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt  time.Time
}

func (Payment) TableName() string { return "payments" }

// Refund là một lần hoàn tiền cho hóa đơn.
type Refund struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"`
	InvoiceID   uint    `gorm:"not null"`
	OnlineTxnID *uint   `gorm:"column:online_txn_id"`
	Amount      float64 `gorm:"type:numeric(14,2);not null"`
	Method      string  `gorm:"type:varchar(20);not null"`
	Status      string  `gorm:"type:varchar(20);not null;default:'SUCCESS'"`
	GatewayRef  string  `gorm:"column:gateway_ref;type:varchar(100);not null;default:''"`
	Reason      string  `gorm:"type:varchar(255);not null"`
	RefundedBy  *uint
	CreatedAt   time.Time
}

func (Refund) TableName() string { return "refunds" }

// OnlinePaymentTxn là một giao dịch thanh toán trực tuyến qua cổng VNPay.
type OnlinePaymentTxn struct {
	ID            uint           `gorm:"primaryKey;autoIncrement"`
	InvoiceID     uint           `gorm:"not null"`
	MemberID      uint           `gorm:"not null"`
	TxnRef        string         `gorm:"type:varchar(64);not null;unique"`
	Amount        float64        `gorm:"type:numeric(14,2);not null"`
	Status        string         `gorm:"type:varchar(20);not null;default:'PENDING'"`
	ProviderTxnNo string         `gorm:"type:varchar(100);not null;default:''"`
	BankCode      string         `gorm:"type:varchar(50);not null;default:''"`
	RawIPN        datatypes.JSON `gorm:"column:raw_ipn;type:jsonb"`
	CreatedAt     time.Time
	PaidAt        *time.Time
}

func (OnlinePaymentTxn) TableName() string { return "online_payment_txns" }

