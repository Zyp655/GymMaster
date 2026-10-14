package entity

import "time"

const (
	LeadSourceWebsite  = "WEBSITE"
	LeadSourceWalkIn   = "WALK_IN"
	LeadSourceReferral = "REFERRAL"
	LeadSourceFacebook = "FACEBOOK"
	LeadSourceOther    = "OTHER"

	LeadStatusNew       = "NEW"
	LeadStatusContacted = "CONTACTED"
	LeadStatusTrial     = "TRIAL"
	LeadStatusConverted = "CONVERTED"
	LeadStatusLost      = "LOST"

	LeadActivityCall  = "CALL"
	LeadActivitySMS   = "SMS"
	LeadActivityVisit = "VISIT"
	LeadActivityNote  = "NOTE"
)

// Lead là khách tiềm năng trong CRM.
type Lead struct {
	BaseEntity
	BranchID       uint   `gorm:"not null"`
	FullName       string `gorm:"type:varchar(150);not null"`
	Phone          string `gorm:"type:varchar(20);not null"`
	Email          string `gorm:"type:varchar(150);not null;default:''"`
	Source         string `gorm:"type:varchar(20);not null;default:'WEBSITE'"`
	InterestPlanID *uint
	Status         string `gorm:"type:varchar(20);not null;default:'NEW'"`
	AssignedTo     *uint
	TrialDate      *time.Time `gorm:"type:date"`
	MemberID       *uint
	Note           string `gorm:"type:text;not null;default:''"`
}

func (Lead) TableName() string { return "leads" }

// LeadActivity là nhật ký chăm sóc khách tiềm năng.
type LeadActivity struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	LeadID    uint   `gorm:"not null"`
	Type      string `gorm:"type:varchar(20);not null"`
	Content   string `gorm:"type:text;not null"`
	CreatedBy *uint
	CreatedAt time.Time
}

func (LeadActivity) TableName() string { return "lead_activities" }

// TrialPass là vé tập thử cấp cho khách tiềm năng.
type TrialPass struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	LeadID    uint      `gorm:"not null"`
	BranchID  uint      `gorm:"not null"`
	ValidDate time.Time `gorm:"type:date;not null"`
	QRToken   string    `gorm:"column:qr_token;type:varchar(100);not null;unique"`
	UsedAt    *time.Time
	CreatedBy *uint
	CreatedAt time.Time
}

func (TrialPass) TableName() string { return "trial_passes" }
