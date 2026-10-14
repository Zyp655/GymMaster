package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	UserRepository interface {
		FindByEmail(ctx context.Context, email string) (*entity.User, error)
		FindByID(ctx context.Context, id uint) (*entity.User, error)
		Create(ctx context.Context, u *entity.User) error
		AssignRole(ctx context.Context, ur *entity.UserRole) error
		UpdateLastLogin(ctx context.Context, id uint, at time.Time) error
		UpdatePassword(ctx context.Context, id uint, passwordHash string) error
		// PermissionCodes trả các mã quyền (không trùng) của tập role.
		PermissionCodes(ctx context.Context, roleIDs []uint) ([]string, error)
	}

	userRepository struct {
		db *gorm.DB
	}
)

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) withRoles(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Preload("UserRoles.Role")
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	if err := r.withRoles(ctx).Where("lower(email) = lower(?)", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var u entity.User
	if err := r.withRoles(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) Create(ctx context.Context, u *entity.User) error {
	return postgres.Conn(ctx, r.db).Omit("UserRoles").Create(u).Error
}

func (r *userRepository) AssignRole(ctx context.Context, ur *entity.UserRole) error {
	return postgres.Conn(ctx, r.db).Omit("Role").Create(ur).Error
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id uint, at time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.User{}).Where("id = ?", id).
		UpdateColumn("last_login_at", at).Error
}

func (r *userRepository) UpdatePassword(ctx context.Context, id uint, passwordHash string) error {
	return postgres.Conn(ctx, r.db).Model(&entity.User{}).Where("id = ?", id).
		Updates(map[string]any{"password_hash": passwordHash, "updated_by": id}).Error
}

func (r *userRepository) PermissionCodes(ctx context.Context, roleIDs []uint) ([]string, error) {
	codes := []string{}
	if len(roleIDs) == 0 {
		return codes, nil
	}
	err := postgres.Conn(ctx, r.db).
		Table("permissions p").
		Distinct("p.code").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Where("rp.role_id IN ?", roleIDs).
		Order("p.code").
		Pluck("p.code", &codes).Error
	return codes, err
}
