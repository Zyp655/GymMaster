package entity

import "time"

const (
	ReviewTargetPTSession     = "PT_SESSION"
	ReviewTargetClassSchedule = "CLASS_SCHEDULE"
)

// Review là đánh giá của hội viên cho buổi PT hoặc buổi lớp học đã tham gia.
type Review struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	MemberID   uint   `gorm:"not null"`
	TargetType string `gorm:"type:varchar(20);not null"`
	TargetID   uint   `gorm:"not null"`
	TrainerID  uint   `gorm:"not null"`
	BranchID   *uint
	Rating     int    `gorm:"not null"`
	Comment    string `gorm:"type:text;not null;default:''"`
	CreatedAt  time.Time
}

func (Review) TableName() string { return "reviews" }
