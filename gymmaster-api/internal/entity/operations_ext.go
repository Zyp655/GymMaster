package entity

import "time"

// Sprint 18: Thiết bị kiểm soát ra vào & Kiosk.
const (
	DeviceTypeTurnstile = "TURNSTILE"
	DeviceTypeDoor      = "DOOR"
	DeviceTypeKiosk     = "KIOSK"

	DeviceStatusActive   = "ACTIVE"
	DeviceStatusOffline  = "OFFLINE"
	DeviceStatusDisabled = "DISABLED"
)

type AccessDevice struct {
	BaseEntity
	BranchID   uint   `gorm:"not null"`
	Code       string `gorm:"type:varchar(50);not null"` // duy nhất trong thiết bị chưa xóa (uq_access_devices_code_active)
	Name       string `gorm:"type:varchar(150);not null"`
	Type       string `gorm:"type:varchar(20);not null;default:'KIOSK'"`
	APIKeyHash string `gorm:"column:api_key_hash;type:char(64);not null"`
	LastSeenAt *time.Time
	Status     string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
}

func (AccessDevice) TableName() string { return "access_devices" }

// Sprint 19: Tủ đồ & dịch vụ thuê tủ.
const (
	LockerSizeSmall    = "SMALL"
	LockerSizeStandard = "STANDARD"
	LockerSizeLarge    = "LARGE"

	LockerStatusAvailable = "AVAILABLE"
	LockerStatusOccupied  = "OCCUPIED"
	LockerStatusRented    = "RENTED"
	LockerStatusBroken    = "BROKEN"

	LockerRentalDaily   = "DAILY"
	LockerRentalMonthly = "MONTHLY"

	LockerRentalActive  = "ACTIVE"
	LockerRentalEnded   = "ENDED"
	LockerRentalExpired = "EXPIRED"
)

type Locker struct {
	ID         uint    `gorm:"primaryKey;autoIncrement"`
	BranchID   uint    `gorm:"not null"`
	Code       string  `gorm:"type:varchar(30);not null"`
	Zone       string  `gorm:"type:varchar(50);not null;default:'Khu A'"`
	Size       string  `gorm:"type:varchar(20);not null;default:'STANDARD'"`
	MonthlyFee float64 `gorm:"type:numeric(14,2);not null"` // không default ở tag: GORM bỏ qua 0 khi Create → tủ miễn phí thành 150000
	Status     string  `gorm:"type:varchar(20);not null;default:'AVAILABLE'"`
	Note       string  `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Locker) TableName() string { return "lockers" }

type LockerRental struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	LockerID  uint      `gorm:"not null"`
	MemberID  uint      `gorm:"not null"`
	Type      string    `gorm:"type:varchar(20);not null"`
	StartAt   time.Time `gorm:"not null"`
	EndAt     *time.Time
	InvoiceID *uint
	Price     float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Status    string  `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	Note      string  `gorm:"type:varchar(255);not null;default:''"`
	CreatedBy *uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (LockerRental) TableName() string { return "locker_rentals" }

// Sprint 24: Hợp đồng điện tử, vé ngày & vé khách.
const (
	ContractStatusPending = "PENDING"
	ContractStatusSigned  = "SIGNED"

	DayPassTypeDay   = "DAY"
	DayPassTypeGuest = "GUEST"
)

type ContractTemplate struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Code      string `gorm:"type:varchar(50);not null;unique"`
	Name      string `gorm:"type:varchar(150);not null"`
	BodyHTML  string `gorm:"column:body_html;type:text;not null"`
	Version   int    `gorm:"not null;default:1"`
	Status    string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ContractTemplate) TableName() string { return "contract_templates" }

type MemberContract struct {
	ID              uint `gorm:"primaryKey;autoIncrement"`
	MemberID        uint `gorm:"not null"`
	TemplateID      uint `gorm:"not null"`
	TemplateVersion int  `gorm:"not null;default:1"`
	SubscriptionID  *uint
	RenderedHTML    string `gorm:"column:rendered_html;type:text;not null;default:''"`
	Status          string `gorm:"type:varchar(20);not null;default:'PENDING'"`
	SignedAt        *time.Time
	SignatureURL    string `gorm:"column:signature_url;type:text;not null;default:''"`
	PDFURL          string `gorm:"column:pdf_url;type:varchar(500);not null;default:''"`
	IP              string `gorm:"type:varchar(64);not null;default:''"`
	UserAgent       string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (MemberContract) TableName() string { return "member_contracts" }

type DayPass struct {
	ID              uint   `gorm:"primaryKey;autoIncrement"`
	BranchID        uint   `gorm:"not null"`
	BuyerName       string `gorm:"type:varchar(150);not null"`
	Phone           string `gorm:"type:varchar(20);not null"`
	MemberID        *uint
	SponsorMemberID *uint
	Type            string    `gorm:"type:varchar(20);not null;default:'DAY'"`
	ValidDate       time.Time `gorm:"type:date;not null"`
	QRToken         string    `gorm:"column:qr_token;type:varchar(64);not null;unique"`
	Price           float64   `gorm:"type:numeric(14,2);not null;default:0"`
	InvoiceID       *uint
	UsedAt          *time.Time
	CreatedBy       *uint
	CreatedAt       time.Time
}

func (DayPass) TableName() string { return "day_passes" }

// Sprint 25: Lịch rảnh HLV, đơn nghỉ & ngày lễ.
const (
	LeaveStatusPending  = "PENDING"
	LeaveStatusApproved = "APPROVED"
	LeaveStatusRejected = "REJECTED"
)

type TrainerAvailability struct {
	ID            uint       `gorm:"primaryKey;autoIncrement"`
	TrainerID     uint       `gorm:"not null"`
	BranchID      uint       `gorm:"not null"`
	DOW           int        `gorm:"column:dow;not null"`
	StartTime     string     `gorm:"type:varchar(5);not null"`
	EndTime       string     `gorm:"type:varchar(5);not null"`
	EffectiveFrom *time.Time `gorm:"type:date"`
	EffectiveTo   *time.Time `gorm:"type:date"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (TrainerAvailability) TableName() string { return "trainer_availabilities" }

type StaffLeaveRequest struct {
	ID         uint `gorm:"primaryKey;autoIncrement"`
	StaffID    uint `gorm:"not null"`
	BranchID   *uint
	FromAt     time.Time `gorm:"not null"`
	ToAt       time.Time `gorm:"not null"`
	Reason     string    `gorm:"type:varchar(500);not null;default:''"`
	Status     string    `gorm:"type:varchar(20);not null;default:'PENDING'"`
	ApprovedBy *uint
	ApprovedAt *time.Time
	ReviewNote string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (StaffLeaveRequest) TableName() string { return "staff_leave_requests" }

// BranchHoliday: ngày lễ của chi nhánh (closed = đóng cả ngày; closed=false + open/close_time = giờ mở cửa đặc biệt).
// Closed KHÔNG có default:true ở tag GORM: GORM bỏ qua giá trị zero (false) khi Create nếu tag có default → ngày lễ
// "giờ đặc biệt" bị lưu thành đóng cửa. Cột DB vẫn DEFAULT TRUE cho dữ liệu chèn tay.
type BranchHoliday struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	BranchID   *uint     // nil = toàn hệ thống
	Date       time.Time `gorm:"type:date;not null"`
	Name       string    `gorm:"type:varchar(150);not null"`
	Closed     bool      `gorm:"not null"`
	OpenTime   *string   `gorm:"type:varchar(5)"`
	CloseTime  *string   `gorm:"type:varchar(5)"`
	ExtendSubs bool      `gorm:"not null;default:false"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (BranchHoliday) TableName() string { return "branch_holidays" }
