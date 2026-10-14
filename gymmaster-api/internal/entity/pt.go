package entity

import "time"

// Trạng thái / nguồn của module huấn luyện cá nhân (migration 000008).
const (
	// Hợp đồng PT (pt_contracts.status).
	PTContractPending   = "PENDING"
	PTContractActive    = "ACTIVE"
	PTContractCompleted = "COMPLETED"
	PTContractExpired   = "EXPIRED"
	PTContractCancelled = "CANCELLED"

	// Buổi PT (pt_sessions.status).
	PTSessionScheduled = "SCHEDULED"
	PTSessionCompleted = "COMPLETED"
	PTSessionCancelled = "CANCELLED"
	PTSessionNoShow    = "NO_SHOW"

	// Nguồn đặt buổi (pt_sessions.source).
	PTSourceCounter = "COUNTER"
	PTSourcePortal  = "PORTAL"
	PTSourceTrainer = "TRAINER"

	// Hoa hồng HLV (trainer_commissions).
	CommissionSourcePTSession = "PT_SESSION"
	CommissionSourceClass     = "CLASS"
	CommissionPending         = "PENDING"
	CommissionPaid            = "PAID"
)

// PTContract là hợp đồng huấn luyện cá nhân: N buổi với một HLV, dùng trong thời hạn [StartDate, EndDate].
// StartDate/EndDate là cột DATE (00:00 UTC của ngày theo lịch Việt Nam).
type PTContract struct {
	ID             uint `gorm:"primaryKey;autoIncrement"`
	MemberID       uint `gorm:"not null"`
	TrainerID      uint `gorm:"not null"`
	PTPackageID    uint `gorm:"column:pt_package_id;not null"`
	BranchID       uint `gorm:"not null"`
	InvoiceID      *uint
	TotalSessions  int       `gorm:"not null"`
	UsedSessions   int       `gorm:"not null;default:0"`
	SessionMinutes int       `gorm:"not null;default:60"`
	StartDate      time.Time `gorm:"type:date;not null"`
	EndDate        time.Time `gorm:"type:date;not null"`
	Price          float64   `gorm:"type:numeric(14,2);not null"`
	Status         string    `gorm:"type:varchar(20);not null"`
	SoldBy         *uint
	Note           string `gorm:"type:varchar(500);not null;default:''"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (PTContract) TableName() string { return "pt_contracts" }

// PTSession là một buổi tập PT.
//
// Buổi "đã trừ vào hợp đồng" (counted) khi Status = COMPLETED, hoặc NO_SHOW có CompletedAt != nil
// (vắng mặt / hủy muộn bị tính buổi). NO_SHOW có CompletedAt = nil là buổi do cron đánh dấu quá giờ
// chưa xác nhận – CHƯA trừ buổi, chờ quản lý xử lý.
type PTSession struct {
	ID                uint `gorm:"primaryKey;autoIncrement"`
	ContractID        uint `gorm:"not null"`
	TrainerID         uint `gorm:"not null"`
	OriginalTrainerID *uint
	MemberID          uint `gorm:"not null"`
	BranchID          uint `gorm:"not null"`
	StartAt           time.Time
	EndAt             time.Time
	Status            string `gorm:"type:varchar(20);not null"`
	WorkoutNote       string `gorm:"type:text;not null;default:''"`
	TrainerNote       string `gorm:"type:text;not null;default:''"`
	CompletedAt       *time.Time
	CancelledAt       *time.Time
	CancelReason      string `gorm:"type:varchar(255);not null;default:''"`
	Source            string `gorm:"type:varchar(10);not null;default:'COUNTER'"`
	CreatedBy         *uint
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (PTSession) TableName() string { return "pt_sessions" }

// Counted cho biết buổi đã được trừ vào số buổi đã dùng của hợp đồng.
func (s *PTSession) Counted() bool {
	return s.Status == PTSessionCompleted || (s.Status == PTSessionNoShow && s.CompletedAt != nil)
}

// TrainerCommission là hoa hồng của HLV cho một nguồn (buổi PT / buổi dạy lớp). UNIQUE(source_type, source_id).
type TrainerCommission struct {
	ID         uint    `gorm:"primaryKey;autoIncrement"`
	TrainerID  uint    `gorm:"not null"`
	BranchID   uint    `gorm:"not null"`
	SourceType string  `gorm:"type:varchar(20);not null"`
	SourceID   uint    `gorm:"not null"`
	BaseAmount float64 `gorm:"type:numeric(14,2);not null"`
	Rate       float64 `gorm:"type:numeric(5,2);not null"`
	Amount     float64 `gorm:"type:numeric(14,2);not null"`
	Period     string  `gorm:"type:char(7);not null"`
	Status     string  `gorm:"type:varchar(20);not null;default:'PENDING'"`
	PaidAt     *time.Time
	PaidBy     *uint
	CreatedAt  time.Time
}

func (TrainerCommission) TableName() string { return "trainer_commissions" }
