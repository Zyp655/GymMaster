package entity

import (
	"time"

	"gorm.io/datatypes"
)

// Sprint 27: 2FA, quản lý phiên & tuân thủ Nghị định 13/2023.
const (
	ConsentTypeTerms     = "TERMS"
	ConsentTypeMarketing = "MARKETING"
	ConsentTypePhoto     = "PHOTO"

	DataRequestExport = "EXPORT"
	DataRequestDelete = "DELETE"

	DataRequestPending    = "PENDING"
	DataRequestProcessing = "PROCESSING"
	DataRequestDone       = "DONE"
	DataRequestRejected   = "REJECTED"
)

type UserMFA struct {
	UserID            uint   `gorm:"primaryKey"`
	TOTPSecretEnc     string `gorm:"column:totp_secret_enc;type:varchar(255);not null"`
	EnabledAt         *time.Time
	RecoveryCodesHash datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (UserMFA) TableName() string { return "user_mfa" }

type Consent struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	MemberID  uint      `gorm:"not null"`
	Type      string    `gorm:"type:varchar(30);not null"`
	Granted   bool      `gorm:"not null;default:true"`
	GrantedAt time.Time `gorm:"not null"`
	IP        string    `gorm:"type:varchar(64);not null;default:''"`
}

func (Consent) TableName() string { return "consents" }

type DataRequest struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	MemberID  uint   `gorm:"not null"`
	Type      string `gorm:"type:varchar(20);not null"`
	Status    string `gorm:"type:varchar(20);not null;default:'PENDING'"`
	Reason    string `gorm:"type:varchar(500);not null;default:''"`
	FileURL   string `gorm:"column:file_url;type:varchar(500);not null;default:''"`
	HandledBy *uint
	Note      string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt time.Time
	DoneAt    *time.Time
}

func (DataRequest) TableName() string { return "data_requests" }
