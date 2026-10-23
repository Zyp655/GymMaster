package entity

import (
	"time"

	"gorm.io/datatypes"
)

type Setting struct {
	Key         string         `gorm:"primaryKey;type:varchar(100)"`
	Value       datatypes.JSON `gorm:"type:jsonb;not null"`
	Description string         `gorm:"type:varchar(255);not null;default:''"`
	UpdatedAt   time.Time
	UpdatedBy   *uint
}

func (Setting) TableName() string { return "settings" }

type Notification struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null"`
	Type      string `gorm:"type:varchar(40);not null"`
	Title     string `gorm:"type:varchar(200);not null"`
	Body      string `gorm:"type:text;not null;default:''"`
	Link      string `gorm:"type:varchar(255);not null;default:''"`
	ReadAt    *time.Time
	CreatedAt time.Time
}

func (Notification) TableName() string { return "notifications" }

type AuditLog struct {
	ID        uint `gorm:"primaryKey"`
	ActorID   *uint
	Action    string `gorm:"type:varchar(50);not null"`
	Entity    string `gorm:"type:varchar(50);not null"`
	EntityID  *uint
	Before    datatypes.JSON `gorm:"type:jsonb"`
	After     datatypes.JSON `gorm:"type:jsonb"`
	IP        string         `gorm:"type:varchar(64);not null;default:''"`
	CreatedAt time.Time
}

func (AuditLog) TableName() string { return "audit_logs" }

type JobRun struct {
	ID         uint   `gorm:"primaryKey"`
	Job        string `gorm:"type:varchar(50);not null"`
	StartedAt  time.Time
	FinishedAt *time.Time
	Status     string `gorm:"type:varchar(10);not null"`
	Affected   int    `gorm:"not null;default:0"`
	Message    string `gorm:"type:text;not null;default:''"`
}

func (JobRun) TableName() string { return "job_runs" }
