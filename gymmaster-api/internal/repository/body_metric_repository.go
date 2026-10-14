package repository

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// BodyMetricRow là một lần đo kèm tên người ghi.
	BodyMetricRow struct {
		entity.BodyMetric
		MeasuredByName string
	}

	BodyMetricRepository interface {
		// ListByMember trả các lần đo theo thứ tự thời gian tăng dần (cho biểu đồ).
		ListByMember(ctx context.Context, memberID uint) ([]BodyMetricRow, error)
		Latest(ctx context.Context, memberID uint) (*BodyMetricRow, error)
		FindByID(ctx context.Context, id uint) (*BodyMetricRow, error)
		Create(ctx context.Context, m *entity.BodyMetric) error
		Delete(ctx context.Context, id uint) error
	}

	bodyMetricRepository struct {
		db *gorm.DB
	}
)

func NewBodyMetricRepository(db *gorm.DB) BodyMetricRepository {
	return &bodyMetricRepository{db: db}
}

func (r *bodyMetricRepository) base(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("body_metrics bm").
		Select("bm.*, COALESCE(u.full_name, '') AS measured_by_name").
		Joins("LEFT JOIN users u ON u.id = bm.measured_by")
}

func (r *bodyMetricRepository) ListByMember(ctx context.Context, memberID uint) ([]BodyMetricRow, error) {
	rows := []BodyMetricRow{}
	err := r.base(ctx).Where("bm.member_id = ?", memberID).
		Order("bm.measured_at ASC, bm.id ASC").Scan(&rows).Error
	return rows, err
}

func (r *bodyMetricRepository) Latest(ctx context.Context, memberID uint) (*BodyMetricRow, error) {
	var rows []BodyMetricRow
	if err := r.base(ctx).Where("bm.member_id = ?", memberID).
		Order("bm.measured_at DESC, bm.id DESC").Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (r *bodyMetricRepository) FindByID(ctx context.Context, id uint) (*BodyMetricRow, error) {
	var rows []BodyMetricRow
	if err := r.base(ctx).Where("bm.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *bodyMetricRepository) Create(ctx context.Context, m *entity.BodyMetric) error {
	return postgres.Conn(ctx, r.db).Create(m).Error
}

func (r *bodyMetricRepository) Delete(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.BodyMetric{}, id).Error
}
