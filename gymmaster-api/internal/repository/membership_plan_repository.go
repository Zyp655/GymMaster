package repository

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	MembershipPlanRepository interface {
		List(ctx context.Context, q dto.Query) ([]entity.MembershipPlan, int64, error)
		// ListActive trả các gói đang bán, theo sort_order rồi giá.
		ListActive(ctx context.Context) ([]entity.MembershipPlan, error)
		FindByID(ctx context.Context, id uint) (*entity.MembershipPlan, error)
		ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error)
		Create(ctx context.Context, p *entity.MembershipPlan) error
		Update(ctx context.Context, p *entity.MembershipPlan) error
		UpdateStatus(ctx context.Context, id uint, status string, updatedBy *uint) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		// CountSubscriptions đếm số đăng ký (mọi trạng thái) đang dùng gói.
		CountSubscriptions(ctx context.Context, id uint) (int64, error)
	}

	membershipPlanRepository struct {
		db *gorm.DB
	}
)

func NewMembershipPlanRepository(db *gorm.DB) MembershipPlanRepository {
	return &membershipPlanRepository{db: db}
}

func (r *membershipPlanRepository) List(ctx context.Context, q dto.Query) ([]entity.MembershipPlan, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.MembershipPlan{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("code ILIKE ? OR name ILIKE ?", kw, kw)
	}
	if v, ok := q.Filters["status"]; ok {
		tx = tx.Where("status = ?", v)
	}
	if v, ok := q.Filters["type"]; ok {
		tx = tx.Where("type = ?", v)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.MembershipPlan
	if err := tx.Scopes(q.Paginate).Order("id").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *membershipPlanRepository) ListActive(ctx context.Context) ([]entity.MembershipPlan, error) {
	var items []entity.MembershipPlan
	err := postgres.Conn(ctx, r.db).Where("status = ?", "ACTIVE").
		Order("sort_order ASC, price ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *membershipPlanRepository) FindByID(ctx context.Context, id uint) (*entity.MembershipPlan, error) {
	var p entity.MembershipPlan
	if err := postgres.Conn(ctx, r.db).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *membershipPlanRepository) ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.MembershipPlan{}).
		Where("code = ? AND id <> ?", code, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *membershipPlanRepository) Create(ctx context.Context, p *entity.MembershipPlan) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *membershipPlanRepository) Update(ctx context.Context, p *entity.MembershipPlan) error {
	return postgres.Conn(ctx, r.db).Model(p).
		Select("code", "name", "description", "type", "duration_days", "session_count", "price", "access_scope",
			"max_freeze_days", "max_freeze_times", "includes_classes", "transfer_fee", "status", "sort_order", "updated_by").
		Updates(p).Error
}

func (r *membershipPlanRepository) UpdateStatus(ctx context.Context, id uint, status string, updatedBy *uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.MembershipPlan{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_by": updatedBy}).Error
}

func (r *membershipPlanRepository) Delete(ctx context.Context, id uint, deletedBy *uint) error {
	return postgres.Conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.MembershipPlan{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&entity.MembershipPlan{}, id).Error
	})
}

func (r *membershipPlanRepository) CountSubscriptions(ctx context.Context, id uint) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("subscriptions").Where("plan_id = ?", id).Count(&n).Error
	return n, err
}
