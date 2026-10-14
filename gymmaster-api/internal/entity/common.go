package entity

import (
	"time"

	"gorm.io/gorm"
)

// BaseEntity được nhúng vào các bảng nghiệp vụ: khóa chính, timestamp, xóa mềm và người tạo/sửa.
type BaseEntity struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	CreatedBy *uint
	UpdatedBy *uint
}
