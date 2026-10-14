package entity

import "time"

type User struct {
	BaseEntity
	Email        string `gorm:"type:varchar(150);not null"`
	Phone        string `gorm:"type:varchar(20);not null;default:''"`
	PasswordHash string `gorm:"type:varchar(100);not null"`
	FullName     string `gorm:"type:varchar(150);not null"`
	AvatarURL    string `gorm:"column:avatar_url;type:varchar(500);not null;default:''"`
	Status       string `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	LastLoginAt  *time.Time

	UserRoles []UserRole `gorm:"foreignKey:UserID"`
}

func (User) TableName() string { return "users" }

type Role struct {
	BaseEntity
	Code        string       `gorm:"type:varchar(50);not null"`
	Name        string       `gorm:"type:varchar(100);not null"`
	Description string       `gorm:"type:varchar(255);not null;default:''"`
	IsSystem    bool         `gorm:"not null;default:false"`
	Permissions []Permission `gorm:"many2many:role_permissions"`
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	ID     uint   `gorm:"primaryKey"`
	Code   string `gorm:"type:varchar(100);not null;unique"`
	Module string `gorm:"type:varchar(50);not null"`
	Name   string `gorm:"type:varchar(150);not null"`
}

func (Permission) TableName() string { return "permissions" }

// UserRole gán role cho user, có thể giới hạn trong một chi nhánh (BranchID != nil).
type UserRole struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null"`
	RoleID    uint `gorm:"not null"`
	BranchID  *uint
	CreatedAt time.Time

	Role Role `gorm:"foreignKey:RoleID"`
}

func (UserRole) TableName() string { return "user_roles" }

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	TokenHash string    `gorm:"type:char(64);not null;unique"`
	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
	UserAgent string `gorm:"type:varchar(255);not null;default:''"`
	IP        string `gorm:"type:varchar(64);not null;default:''"`
	CreatedAt time.Time
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
