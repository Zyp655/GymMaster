package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	ClassTypeRepository interface {
		List(ctx context.Context, q dto.Query) ([]entity.ClassType, int64, error)
		FindByID(ctx context.Context, id uint) (*entity.ClassType, error)
		ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error)
		Create(ctx context.Context, t *entity.ClassType) error
		Update(ctx context.Context, t *entity.ClassType) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		// CountFutureSchedules đếm lịch SCHEDULED chưa diễn ra của loại lớp.
		CountFutureSchedules(ctx context.Context, id uint, now time.Time) (int64, error)
		ActiveOptions(ctx context.Context) ([]entity.ClassType, error)
	}

	classTypeRepository struct {
		db *gorm.DB
	}
)

func NewClassTypeRepository(db *gorm.DB) ClassTypeRepository {
	return &classTypeRepository{db: db}
}

func (r *classTypeRepository) List(ctx context.Context, q dto.Query) ([]entity.ClassType, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.ClassType{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("code ILIKE ? OR name ILIKE ?", kw, kw)
	}
	if v, ok := q.Filters["status"]; ok {
		tx = tx.Where("status = ?", v)
	}
	if v, ok := q.Filters["level"]; ok {
		tx = tx.Where("level = ?", v)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.ClassType
	err := tx.Scopes(q.Paginate).Find(&items).Error
	return items, total, err
}

func (r *classTypeRepository) FindByID(ctx context.Context, id uint) (*entity.ClassType, error) {
	var t entity.ClassType
	if err := postgres.Conn(ctx, r.db).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *classTypeRepository) ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.ClassType{}).
		Where("upper(code) = upper(?) AND id <> ?", code, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *classTypeRepository) Create(ctx context.Context, t *entity.ClassType) error {
	return postgres.Conn(ctx, r.db).Create(t).Error
}

func (r *classTypeRepository) Update(ctx context.Context, t *entity.ClassType) error {
	return postgres.Conn(ctx, r.db).Model(t).
		Select("code", "name", "description", "color", "default_duration_min", "default_capacity", "level", "status", "updated_by").
		Updates(t).Error
}

func (r *classTypeRepository) Delete(ctx context.Context, id uint, deletedBy *uint) error {
	return postgres.Conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.ClassType{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&entity.ClassType{}, id).Error
	})
}

func (r *classTypeRepository) CountFutureSchedules(ctx context.Context, id uint, now time.Time) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.ClassSchedule{}).
		Where("class_type_id = ? AND status = ? AND start_at > ?", id, entity.ClassScheduleScheduled, now).Count(&n).Error
	return n, err
}

func (r *classTypeRepository) ActiveOptions(ctx context.Context) ([]entity.ClassType, error) {
	var items []entity.ClassType
	err := postgres.Conn(ctx, r.db).Where("status = 'ACTIVE'").Order("name").Find(&items).Error
	return items, err
}
