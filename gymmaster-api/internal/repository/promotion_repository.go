package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	PromotionRepository interface {
		// List hỗ trợ Filters: status, applies_to, validity (active|upcoming|expired|inactive) tính tại now.
		List(ctx context.Context, q dto.Query, now time.Time) ([]entity.Promotion, int64, error)
		FindByID(ctx context.Context, id uint) (*entity.Promotion, error)
		// FindByCode tìm theo mã, không phân biệt hoa thường.
		FindByCode(ctx context.Context, code string) (*entity.Promotion, error)
		ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error)
		Create(ctx context.Context, p *entity.Promotion) error
		Update(ctx context.Context, p *entity.Promotion) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		// IncrementUsage tăng used_count nguyên tử nếu còn lượt; trả số dòng bị ảnh hưởng (0 = hết lượt).
		IncrementUsage(ctx context.Context, id uint) (int64, error)
	}

	promotionRepository struct {
		db *gorm.DB
	}
)

func NewPromotionRepository(db *gorm.DB) PromotionRepository {
	return &promotionRepository{db: db}
}

const promoExhausted = "(usage_limit IS NOT NULL AND used_count >= usage_limit)"

func (r *promotionRepository) List(ctx context.Context, q dto.Query, now time.Time) ([]entity.Promotion, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Promotion{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("code ILIKE ? OR name ILIKE ?", kw, kw)
	}
	if v, ok := q.Filters["status"]; ok {
		tx = tx.Where("status = ?", v)
	}
	if v, ok := q.Filters["applies_to"]; ok {
		tx = tx.Where("applies_to = ?", v)
	}
	switch q.Filters["validity"] {
	case "active":
		tx = tx.Where("status = 'ACTIVE' AND start_at <= ? AND end_at >= ? AND NOT "+promoExhausted, now, now)
	case "upcoming":
		tx = tx.Where("status = 'ACTIVE' AND start_at > ?", now)
	case "expired":
		tx = tx.Where("end_at < ? OR "+promoExhausted, now)
	case "inactive":
		tx = tx.Where("status = 'INACTIVE'")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.Promotion
	if err := tx.Scopes(q.Paginate).Order("id").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *promotionRepository) FindByID(ctx context.Context, id uint) (*entity.Promotion, error) {
	var p entity.Promotion
	if err := postgres.Conn(ctx, r.db).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *promotionRepository) FindByCode(ctx context.Context, code string) (*entity.Promotion, error) {
	var p entity.Promotion
	err := postgres.Conn(ctx, r.db).Where("upper(code) = ?", strings.ToUpper(strings.TrimSpace(code))).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *promotionRepository) ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Promotion{}).
		Where("upper(code) = ? AND id <> ?", strings.ToUpper(code), excludeID).Count(&n).Error
	return n > 0, err
}

func (r *promotionRepository) Create(ctx context.Context, p *entity.Promotion) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *promotionRepository) Update(ctx context.Context, p *entity.Promotion) error {
	return postgres.Conn(ctx, r.db).Model(p).
		Select("code", "name", "type", "value", "max_discount", "min_amount", "applies_to",
			"start_at", "end_at", "usage_limit", "status", "updated_by").
		Updates(p).Error
}

func (r *promotionRepository) Delete(ctx context.Context, id uint, deletedBy *uint) error {
	return postgres.Conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.Promotion{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&entity.Promotion{}, id).Error
	})
}

func (r *promotionRepository) IncrementUsage(ctx context.Context, id uint) (int64, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.Promotion{}).
		Where("id = ? AND (usage_limit IS NULL OR used_count < usage_limit)", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	return res.RowsAffected, res.Error
}
