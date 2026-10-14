package entity

import "time"

// Nhóm cơ và trạng thái giáo án (CHECK trong migration 000013).
const (
	MuscleChest     = "CHEST"
	MuscleBack      = "BACK"
	MuscleShoulders = "SHOULDERS"
	MuscleLegs      = "LEGS"
	MuscleArms      = "ARMS"
	MuscleCore      = "CORE"
	MuscleCardio    = "CARDIO"
	MuscleFullBody  = "FULL_BODY"

	WorkoutPlanDraft     = "DRAFT"
	WorkoutPlanActive    = "ACTIVE"
	WorkoutPlanCompleted = "COMPLETED"
	WorkoutPlanArchived  = "ARCHIVED"
)

// Exercise là một bài tập trong thư viện.
type Exercise struct {
	BaseEntity
	Name        string `gorm:"type:varchar(150);not null"`
	MuscleGroup string `gorm:"type:varchar(50);not null"`
	Equipment   string `gorm:"type:varchar(100);not null;default:''"`
	VideoURL    string `gorm:"column:video_url;type:varchar(500);not null;default:''"`
	Description string `gorm:"type:text;not null;default:''"`
}

func (Exercise) TableName() string { return "exercises" }

// WorkoutPlan là giáo án tập luyện (MemberID nil = giáo án mẫu dùng chung để clone).
type WorkoutPlan struct {
	BaseEntity
	MemberID     *uint
	TrainerID    uint `gorm:"not null"`
	PTContractID *uint
	Name         string     `gorm:"type:varchar(150);not null"`
	Goal         string     `gorm:"type:varchar(255);not null;default:''"`
	StartDate    *time.Time `gorm:"type:date"`
	EndDate      *time.Time `gorm:"type:date"`
	Status       string     `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
}

func (WorkoutPlan) TableName() string { return "workout_plans" }

// WorkoutPlanDay là một buổi/ngày trong giáo án (day_index 1..14).
type WorkoutPlanDay struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	PlanID   uint   `gorm:"not null"`
	DayIndex int    `gorm:"not null"`
	Title    string `gorm:"type:varchar(150);not null"`
}

func (WorkoutPlanDay) TableName() string { return "workout_plan_days" }

// WorkoutPlanItem là một bài tập cụ thể trong ngày của giáo án.
type WorkoutPlanItem struct {
	ID         uint    `gorm:"primaryKey;autoIncrement"`
	DayID      uint    `gorm:"not null"`
	ExerciseID uint    `gorm:"not null"`
	Sets       int     `gorm:"not null;default:3"`
	Reps       int     `gorm:"not null;default:10"`
	Weight     float64 `gorm:"type:numeric(6,2);not null;default:0"`
	RestSec    int     `gorm:"not null;default:60"`
	Note       string  `gorm:"type:varchar(255);not null;default:''"`
	Sort       int     `gorm:"not null;default:0"`
}

func (WorkoutPlanItem) TableName() string { return "workout_plan_items" }

// WorkoutLog là nhật ký buổi tập do hội viên tự ghi nhận.
type WorkoutLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	MemberID    uint      `gorm:"not null"`
	PlanItemID  *uint
	ExerciseID  uint      `gorm:"not null"`
	PerformedAt time.Time `gorm:"not null"`
	Sets        int       `gorm:"not null"`
	Reps        int       `gorm:"not null"`
	Weight      float64   `gorm:"type:numeric(6,2);not null;default:0"`
	Note        string    `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt   time.Time
}

func (WorkoutLog) TableName() string { return "workout_logs" }
