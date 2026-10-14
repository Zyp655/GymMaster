package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

const (
	LoginMaxFailedAttempts = 5
	LoginLockWindow        = 15 * time.Minute
)

// ---- PasswordResetRepository ----------------------------------------------------

type (
	PasswordResetRepository interface {
		Create(ctx context.Context, pr *entity.PasswordReset) error
		FindActiveForUpdate(ctx context.Context, tokenHash string, now time.Time) (*entity.PasswordReset, error)
		MarkUsed(ctx context.Context, id uint, now time.Time) error
		DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	}

	passwordResetRepository struct {
		db *gorm.DB
	}
)

func NewPasswordResetRepository(db *gorm.DB) PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) Create(ctx context.Context, pr *entity.PasswordReset) error {
	return postgres.Conn(ctx, r.db).Create(pr).Error
}

func (r *passwordResetRepository) FindActiveForUpdate(ctx context.Context, tokenHash string, now time.Time) (*entity.PasswordReset, error) {
	var pr entity.PasswordReset
	err := postgres.Conn(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).
		First(&pr).Error
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, id uint, now time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.PasswordReset{}).
		Where("id = ?", id).Update("used_at", now).Error
}

func (r *passwordResetRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	res := postgres.Conn(ctx, r.db).
		Where("expires_at < ? OR (used_at IS NOT NULL AND used_at < ?)", before, before).
		Delete(&entity.PasswordReset{})
	return res.RowsAffected, res.Error
}

// ---- LoginLimiter ---------------------------------------------------------------

type (
	LoginLimiter interface {
		IsLocked(ctx context.Context, email, ip string, now time.Time) (bool, error)
		RecordAttempt(ctx context.Context, email, ip string, success bool, now time.Time) error
		CleanupBefore(ctx context.Context, before time.Time) (int64, error)
	}

	dbLoginLimiter struct {
		db *gorm.DB
	}
)

func NewLoginLimiter(db *gorm.DB) LoginLimiter {
	return &dbLoginLimiter{db: db}
}

func (l *dbLoginLimiter) IsLocked(ctx context.Context, email, ip string, now time.Time) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	windowStart := now.Add(-LoginLockWindow)

	var count int64
	err := postgres.Conn(ctx, l.db).Model(&entity.LoginAttempt{}).
		Where("lower(email) = ? AND ip = ? AND success = false AND attempted_at >= ?", email, ip, windowStart).
		Where(`attempted_at > COALESCE((
			SELECT MAX(attempted_at) FROM login_attempts
			WHERE lower(email) = ? AND ip = ? AND success = true AND attempted_at >= ?
		), ?)`, email, ip, windowStart, windowStart).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count >= LoginMaxFailedAttempts, nil
}

func (l *dbLoginLimiter) RecordAttempt(ctx context.Context, email, ip string, success bool, now time.Time) error {
	row := &entity.LoginAttempt{
		Email:       strings.ToLower(strings.TrimSpace(email)),
		IP:          ip,
		AttemptedAt: now,
		Success:     success,
	}
	return postgres.Conn(ctx, l.db).Create(row).Error
}

func (l *dbLoginLimiter) CleanupBefore(ctx context.Context, before time.Time) (int64, error) {
	res := postgres.Conn(ctx, l.db).Where("attempted_at < ?", before).Delete(&entity.LoginAttempt{})
	return res.RowsAffected, res.Error
}

// ---- MemberRegisterRepository ---------------------------------------------------

type (
	MemberRegisterRepository interface {
		FindMemberByPhone(ctx context.Context, phone string) (*entity.Member, error)
		DefaultBranchID(ctx context.Context) (uint, error)
		MemberRoleID(ctx context.Context) (uint, error)
		CreateUserAndMember(ctx context.Context, user *entity.User, member *entity.Member, roleID uint) error
		LinkUserToMember(ctx context.Context, memberID uint, user *entity.User, roleID uint) error
		CreateOTP(ctx context.Context, otp *entity.MemberRegisterOTP) error
		FindLatestActiveOTP(ctx context.Context, memberID uint, email string, now time.Time) (*entity.MemberRegisterOTP, error)
		MarkOTPUsed(ctx context.Context, id uint, now time.Time) error
	}

	memberRegisterRepository struct {
		db *gorm.DB
	}
)

func NewMemberRegisterRepository(db *gorm.DB) MemberRegisterRepository {
	return &memberRegisterRepository{db: db}
}

func (r *memberRegisterRepository) FindMemberByPhone(ctx context.Context, phone string) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("phone = ?", phone).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *memberRegisterRepository) DefaultBranchID(ctx context.Context) (uint, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).Select("id").Where("status = ?", consts.StatusActive).Order("id ASC").First(&b).Error; err != nil {
		return 0, err
	}
	return b.ID, nil
}

func (r *memberRegisterRepository) MemberRoleID(ctx context.Context) (uint, error) {
	var role entity.Role
	if err := postgres.Conn(ctx, r.db).Select("id").Where("code = ?", consts.RoleMember).First(&role).Error; err != nil {
		return 0, err
	}
	return role.ID, nil
}

func (r *memberRegisterRepository) CreateUserAndMember(ctx context.Context, user *entity.User, member *entity.Member, roleID uint) error {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Omit("UserRoles").Create(user).Error; err != nil {
		return err
	}
	ur := &entity.UserRole{UserID: user.ID, RoleID: roleID}
	if err := conn.Omit("Role").Create(ur).Error; err != nil {
		return err
	}
	member.UserID = &user.ID
	return conn.Create(member).Error
}

func (r *memberRegisterRepository) LinkUserToMember(ctx context.Context, memberID uint, user *entity.User, roleID uint) error {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Omit("UserRoles").Create(user).Error; err != nil {
		return err
	}
	ur := &entity.UserRole{UserID: user.ID, RoleID: roleID}
	if err := conn.Omit("Role").Create(ur).Error; err != nil {
		return err
	}
	return conn.Model(&entity.Member{}).Where("id = ?", memberID).
		Updates(map[string]any{"user_id": user.ID, "email": user.Email}).Error
}

func (r *memberRegisterRepository) CreateOTP(ctx context.Context, otp *entity.MemberRegisterOTP) error {
	return postgres.Conn(ctx, r.db).Create(otp).Error
}

func (r *memberRegisterRepository) FindLatestActiveOTP(ctx context.Context, memberID uint, email string, now time.Time) (*entity.MemberRegisterOTP, error) {
	var otp entity.MemberRegisterOTP
	err := postgres.Conn(ctx, r.db).
		Where("member_id = ? AND lower(email) = lower(?) AND used_at IS NULL AND expires_at > ?", memberID, email, now).
		Order("id DESC").
		First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *memberRegisterRepository) MarkOTPUsed(ctx context.Context, id uint, now time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.MemberRegisterOTP{}).
		Where("id = ?", id).Update("used_at", now).Error
}
