package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	RefreshTokenRepository interface {
		Create(ctx context.Context, t *entity.RefreshToken) error
		// FindActiveForUpdate tìm token còn hiệu lực và khóa dòng (SELECT ... FOR UPDATE)
		// để hai request refresh đồng thời không cùng dùng được một token.
		FindActiveForUpdate(ctx context.Context, tokenHash string, now time.Time) (*entity.RefreshToken, error)
		// FindByHash tìm token bất kể trạng thái (dùng để phát hiện token đã thu hồi bị dùng lại).
		FindByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
		Revoke(ctx context.Context, id uint, at time.Time) error
		RevokeByHash(ctx context.Context, userID uint, tokenHash string, at time.Time) error
		RevokeAllByUser(ctx context.Context, userID uint, at time.Time) error
	}

	refreshTokenRepository struct {
		db *gorm.DB
	}
)

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, t *entity.RefreshToken) error {
	return postgres.Conn(ctx, r.db).Create(t).Error
}

func (r *refreshTokenRepository) FindActiveForUpdate(ctx context.Context, tokenHash string, now time.Time) (*entity.RefreshToken, error) {
	var t entity.RefreshToken
	err := postgres.Conn(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *refreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var t entity.RefreshToken
	if err := postgres.Conn(ctx, r.db).Where("token_hash = ?", tokenHash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uint, at time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("id = ?", id).Update("revoked_at", at).Error
}

func (r *refreshTokenRepository) RevokeByHash(ctx context.Context, userID uint, tokenHash string, at time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("user_id = ? AND token_hash = ? AND revoked_at IS NULL", userID, tokenHash).
		Update("revoked_at", at).Error
}

func (r *refreshTokenRepository) RevokeAllByUser(ctx context.Context, userID uint, at time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", at).Error
}
