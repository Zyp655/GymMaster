package entity

import "time"

// Phương thức, kết quả và lý do từ chối check-in (CHECK trong migration 000006).
const (
	CheckinMethodQR     = "QR"
	CheckinMethodManual = "MANUAL"

	CheckinAllowed = "ALLOWED"
	CheckinDenied  = "DENIED"

	CheckinDenyMemberInactive       = "MEMBER_INACTIVE"
	CheckinDenyNoActiveSubscription = "NO_ACTIVE_SUBSCRIPTION"
	CheckinDenyWrongBranch          = "WRONG_BRANCH"
	CheckinDenyNoSessionsLeft       = "NO_SESSIONS_LEFT"
	CheckinDenySubscriptionFrozen   = "SUBSCRIPTION_FROZEN"
	CheckinDenyDuplicate            = "DUPLICATE"
	CheckinDenyQRExpired            = "QR_EXPIRED"
	CheckinDenyQRInvalid            = "QR_INVALID"
	CheckinDenyBranchClosed         = "BRANCH_CLOSED"
)

// Checkin là một lần quét/ra vào. Mọi lần quét đều được ghi, kể cả bị từ chối.
type Checkin struct {
	ID             uint `gorm:"primaryKey;autoIncrement"`
	MemberID       uint `gorm:"not null"`
	BranchID       uint `gorm:"not null"`
	SubscriptionID *uint
	CheckedInAt    time.Time `gorm:"not null"`
	CheckedOutAt   *time.Time
	Method         string `gorm:"type:varchar(10);not null"`
	Result         string `gorm:"type:varchar(10);not null"`
	DenyReason     string `gorm:"type:varchar(40);not null;default:''"`
	StaffID        *uint
	Note           string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt      time.Time
}

func (Checkin) TableName() string { return "checkins" }

// CheckinSubscriptionView là dữ liệu CHỈ ĐỌC của một gói tập (subscriptions JOIN membership_plans)
// mà module check-in cần để quyết định cho vào. Module bán gói sở hữu entity Subscription chính thức.
type CheckinSubscriptionView struct {
	ID                uint
	MemberID          uint
	PlanID            uint
	BranchID          uint
	BranchName        string
	StartDate         time.Time
	EndDate           time.Time
	TotalSessions     *int
	RemainingSessions *int
	Status            string
	PlanName          string
	PlanType          string // TIME | SESSION
	AccessScope       string // SINGLE_BRANCH | ALL_BRANCHES
}
