package entity

import "time"

// Trạng thái / cấp độ / nguồn đặt chỗ của lớp nhóm (migration 000007).
const (
	ClassLevelAll          = "ALL"
	ClassLevelBeginner     = "BEGINNER"
	ClassLevelIntermediate = "INTERMEDIATE"
	ClassLevelAdvanced     = "ADVANCED"

	ClassScheduleScheduled = "SCHEDULED"
	ClassScheduleCancelled = "CANCELLED"
	ClassScheduleCompleted = "COMPLETED"

	BookingStatusBooked    = "BOOKED"
	BookingStatusWaitlist  = "WAITLIST"
	BookingStatusCancelled = "CANCELLED"
	BookingStatusAttended  = "ATTENDED"
	BookingStatusNoShow    = "NO_SHOW"

	BookingSourceCounter = "COUNTER"
	BookingSourcePortal  = "PORTAL"
)

// ClassType là loại lớp nhóm (Yoga, Spinning, ...).
type ClassType struct {
	BaseEntity
	Code               string `gorm:"type:varchar(30);not null"`
	Name               string `gorm:"type:varchar(100);not null"`
	Description        string `gorm:"type:text;not null;default:''"`
	Color              string `gorm:"type:varchar(7);not null;default:'#F97316'"`
	DefaultDurationMin int    `gorm:"not null;default:60"`
	DefaultCapacity    int    `gorm:"not null;default:20"`
	Level              string `gorm:"type:varchar(20);not null;default:'ALL'"`
	Status             string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
}

func (ClassType) TableName() string { return "class_types" }

// ClassSchedule là một buổi lớp cụ thể. SeriesID gom các buổi tạo cùng lúc theo lịch lặp tuần.
type ClassSchedule struct {
	ID                  uint `gorm:"primaryKey;autoIncrement"`
	ClassTypeID         uint `gorm:"not null"`
	BranchID            uint `gorm:"not null"`
	RoomID              uint `gorm:"not null"`
	TrainerID           uint `gorm:"not null"`
	SubstituteTrainerID *uint
	StartAt             time.Time
	EndAt               time.Time
	Capacity            int     `gorm:"not null"`
	BookedCount         int     `gorm:"not null;default:0"`
	Status              string  `gorm:"type:varchar(20);not null;default:'SCHEDULED'"`
	SeriesID            *string `gorm:"type:uuid"`
	CancelReason        string  `gorm:"type:varchar(255);not null;default:''"`
	Note                string  `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedBy           *uint
	UpdatedBy           *uint
}

func (ClassSchedule) TableName() string { return "class_schedules" }

// ClassScheduleDetail là buổi lớp kèm thông tin hiển thị lấy bằng JOIN.
type ClassScheduleDetail struct {
	ClassSchedule
	ClassTypeCode         string
	ClassTypeName         string
	ClassTypeColor        string
	BranchName            string
	RoomName              string
	TrainerName           string
	SubstituteTrainerName string // tên HLV dạy thay ("" nếu không có)
	WaitlistCount         int
	AttendedCount         int
}

// ClassBooking là một lượt đặt chỗ của hội viên.
type ClassBooking struct {
	ID               uint `gorm:"primaryKey;autoIncrement"`
	ScheduleID       uint `gorm:"not null"`
	MemberID         uint `gorm:"not null"`
	SubscriptionID   *uint
	Status           string `gorm:"type:varchar(20);not null"`
	WaitlistPosition *int
	BookedAt         time.Time
	CancelledAt      *time.Time
	AttendedAt       *time.Time
	Source           string `gorm:"type:varchar(10);not null;default:'COUNTER'"`
	CreatedBy        *uint
	UpdatedAt        time.Time
}

func (ClassBooking) TableName() string { return "class_bookings" }

// ClassBookingDetail là lượt đặt chỗ kèm thông tin hội viên và buổi lớp.
type ClassBookingDetail struct {
	ClassBooking
	MemberCode     string
	MemberName     string
	MemberPhone    string
	MemberUserID   *uint
	ClassTypeID    uint
	ClassTypeName  string
	ClassTypeColor string
	BranchID       uint
	BranchName     string
	RoomName       string
	TrainerID      uint // HLV thực dạy (HLV dạy thay nếu có)
	TrainerName    string
	StartAt        time.Time
	EndAt          time.Time
	ScheduleStatus string
	WaitlistRank   *int // thứ hạng hiện tại trong danh sách chờ (1, 2, …); nil nếu không WAITLIST
}
