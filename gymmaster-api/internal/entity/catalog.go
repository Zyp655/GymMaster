package entity

import "time"

// Loại gói tập / trạng thái / phạm vi (migration 000004).
const (
	PlanTypeTime    = "TIME"
	PlanTypeSession = "SESSION"

	AccessSingleBranch = "SINGLE_BRANCH"
	AccessAllBranches  = "ALL_BRANCHES"

	PromotionPercent = "PERCENT"
	PromotionFixed   = "FIXED"

	AppliesAll        = "ALL"
	AppliesMembership = "MEMBERSHIP"
	AppliesPT         = "PT"
	AppliesProduct    = "PRODUCT"
)

// MembershipPlan là gói tập.
//   - TIME    : tập không giới hạn số buổi trong DurationDays ngày (SessionCount = nil).
//   - SESSION : được SessionCount lượt check-in, hết hạn sau DurationDays ngày.
type MembershipPlan struct {
	BaseEntity
	Code            string  `gorm:"type:varchar(30);not null"`
	Name            string  `gorm:"type:varchar(150);not null"`
	Description     string  `gorm:"type:text;not null;default:''"`
	Type            string  `gorm:"type:varchar(10);not null"`
	DurationDays    int     `gorm:"not null"`
	SessionCount    *int    // chỉ có với SESSION
	Price           float64 `gorm:"type:numeric(14,2);not null"`
	AccessScope     string  `gorm:"type:varchar(20);not null;default:'SINGLE_BRANCH'"`
	MaxFreezeDays   int     `gorm:"not null;default:0"`
	MaxFreezeTimes  int     `gorm:"not null;default:0"`
	IncludesClasses bool    `gorm:"not null;default:false"`
	TransferFee     float64 `gorm:"type:numeric(14,2);not null;default:0"`
	Status          string  `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	SortOrder       int     `gorm:"not null;default:0"`
}

func (MembershipPlan) TableName() string { return "membership_plans" }

// PTPackage là gói huấn luyện cá nhân (N buổi, dùng trong ValidityDays ngày).
type PTPackage struct {
	BaseEntity
	Code           string  `gorm:"type:varchar(30);not null"`
	Name           string  `gorm:"type:varchar(150);not null"`
	Description    string  `gorm:"type:text;not null;default:''"`
	SessionCount   int     `gorm:"not null"`
	SessionMinutes int     `gorm:"not null;default:60"`
	ValidityDays   int     `gorm:"not null"`
	Price          float64 `gorm:"type:numeric(14,2);not null"`
	Status         string  `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
}

func (PTPackage) TableName() string { return "pt_packages" }

// Promotion là mã khuyến mãi. Code luôn lưu dạng IN HOA.
type Promotion struct {
	BaseEntity
	Code        string   `gorm:"type:varchar(30);not null"`
	Name        string   `gorm:"type:varchar(150);not null"`
	Type        string   `gorm:"type:varchar(10);not null"`
	Value       float64  `gorm:"type:numeric(14,2);not null"`
	MaxDiscount *float64 `gorm:"type:numeric(14,2)"` // trần giảm cho PERCENT
	MinAmount   float64  `gorm:"type:numeric(14,2);not null;default:0"`
	AppliesTo   string   `gorm:"type:varchar(20);not null;default:'ALL'"`
	StartAt     time.Time
	EndAt       time.Time
	UsageLimit  *int   // nil = không giới hạn
	UsedCount   int    `gorm:"not null;default:0"`
	Status      string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
}

func (Promotion) TableName() string { return "promotions" }
