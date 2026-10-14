package entity

import "time"

// Giới tính, nguồn, trạng thái hội viên (CHECK trong migration 000003).
const (
	MemberGenderMale   = "MALE"
	MemberGenderFemale = "FEMALE"
	MemberGenderOther  = "OTHER"

	MemberSourceWalkIn   = "WALK_IN"
	MemberSourceReferral = "REFERRAL"
	MemberSourceFacebook = "FACEBOOK"
	MemberSourceWebsite  = "WEBSITE"
	MemberSourceOther    = "OTHER"

	MemberStatusActive   = "ACTIVE"
	MemberStatusInactive = "INACTIVE"
	MemberStatusBanned   = "BANNED"
)

type Member struct {
	BaseEntity
	// Tài khoản cổng hội viên (role MEMBER), nil = chưa cấp.
	UserID *uint
	// Mã GMxxxxxx do DB sinh (sequence) – chỉ đọc.
	MemberCode            string     `gorm:"->;type:varchar(20)"`
	FullName              string     `gorm:"type:varchar(150);not null"`
	Gender                string     `gorm:"type:varchar(10);not null"`
	Dob                   *time.Time `gorm:"type:date"`
	Phone                 string     `gorm:"type:varchar(20);not null"`
	Email                 string     `gorm:"type:varchar(150);not null;default:''"`
	IDCard                string     `gorm:"column:id_card;type:varchar(20);not null;default:''"`
	Address               string     `gorm:"type:varchar(255);not null;default:''"`
	AvatarURL             string     `gorm:"column:avatar_url;type:varchar(500);not null;default:''"`
	EmergencyContactName  string     `gorm:"type:varchar(150);not null;default:''"`
	EmergencyContactPhone string     `gorm:"type:varchar(20);not null;default:''"`
	HomeBranchID          uint       `gorm:"not null"`
	Source                string     `gorm:"type:varchar(20);not null"`
	Status                string     `gorm:"type:varchar(20);not null"`
	// Bí mật ký QR động (pkg/qrtoken). DB sinh khi tạo – chỉ đọc qua GORM, đổi bằng UpdateColumn.
	QRSecret            string `gorm:"->;column:qr_secret;type:varchar(64)"`
	BookingBlockedUntil *time.Time
	Note                string `gorm:"type:text;not null;default:''"`
}

func (Member) TableName() string { return "members" }

// BodyMetric là một lần đo chỉ số cơ thể. BMI là cột GENERATED trong DB (chỉ đọc).
type BodyMetric struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	MemberID     uint      `gorm:"not null"`
	MeasuredAt   time.Time `gorm:"not null"`
	WeightKg     float64   `gorm:"type:numeric(5,2);not null"`
	HeightCm     float64   `gorm:"type:numeric(5,2);not null"`
	BodyFatPct   *float64  `gorm:"type:numeric(5,2)"`
	MuscleMassKg *float64  `gorm:"type:numeric(5,2)"`
	WaistCm      *float64  `gorm:"type:numeric(5,2)"`
	ChestCm      *float64  `gorm:"type:numeric(5,2)"`
	HipCm        *float64  `gorm:"type:numeric(5,2)"`
	BMI          *float64  `gorm:"->;column:bmi;type:numeric(5,2)"`
	Note         string    `gorm:"type:varchar(255);not null;default:''"`
	MeasuredBy   *uint
	CreatedAt    time.Time
}

func (BodyMetric) TableName() string { return "body_metrics" }
