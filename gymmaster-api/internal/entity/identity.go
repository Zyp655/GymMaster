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
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"not null"`
	TokenHash  string    `gorm:"type:char(64);not null;unique"`
	ExpiresAt  time.Time `gorm:"not null"`
	RevokedAt  *time.Time
	UserAgent  string `gorm:"type:varchar(255);not null;default:''"`
	IP         string `gorm:"type:varchar(64);not null;default:''"`
	DeviceName string `gorm:"type:varchar(150);not null;default:''"`
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// PasswordReset là token đặt lại mật khẩu (lưu SHA-256, TTL 30 phút, dùng 1 lần).
type PasswordReset struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	TokenHash string    `gorm:"type:char(64);not null;unique"`
	ExpiresAt time.Time `gorm:"not null"`
	UsedAt    *time.Time
	IP        string `gorm:"type:varchar(64);not null;default:''"`
	CreatedAt time.Time
}

func (PasswordReset) TableName() string { return "password_resets" }

// LoginAttempt ghi lại lần thử đăng nhập để giới hạn brute-force theo email + IP.
type LoginAttempt struct {
	ID          uint      `gorm:"primaryKey"`
	Email       string    `gorm:"type:varchar(150);not null"`
	IP          string    `gorm:"type:varchar(64);not null;default:''"`
	AttemptedAt time.Time `gorm:"not null"`
	Success     bool      `gorm:"not null;default:false"`
}

func (LoginAttempt) TableName() string { return "login_attempts" }

// MemberRegisterOTP lưu mã OTP gửi qua email khi hội viên có sẵn hồ sơ tự tạo tài khoản cổng hội viên.
type MemberRegisterOTP struct {
	ID           uint      `gorm:"primaryKey"`
	MemberID     uint      `gorm:"not null"`
	Email        string    `gorm:"type:varchar(150);not null"`
	PasswordHash string    `gorm:"type:varchar(100);not null"`
	OTPHash      string    `gorm:"column:otp_hash;type:char(64);not null"`
	ExpiresAt    time.Time `gorm:"not null"`
	UsedAt       *time.Time
	CreatedAt    time.Time
}

func (MemberRegisterOTP) TableName() string { return "member_register_otps" }

