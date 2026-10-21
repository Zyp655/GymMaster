package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	AccessDeviceRow struct {
		entity.AccessDevice
		BranchName string
	}

	AccessDeviceFilter struct {
		BranchIDs []uint
		Status    string
	}

	DeviceRepository interface {
		List(ctx context.Context, filter AccessDeviceFilter) ([]AccessDeviceRow, error)
		FindByID(ctx context.Context, id uint) (*AccessDeviceRow, error)
		FindByAPIKeyHash(ctx context.Context, keyHash string) (*AccessDeviceRow, error)
		ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error)
		Create(ctx context.Context, d *entity.AccessDevice) error
		Update(ctx context.Context, d *entity.AccessDevice) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		TouchSeen(ctx context.Context, id uint, seenAt time.Time) error
		ListStaleActiveDevices(ctx context.Context, cutoff time.Time) ([]AccessDeviceRow, error)
		MarkDevicesOffline(ctx context.Context, ids []uint) error
	}

	deviceRepository struct {
		db *gorm.DB
	}
)

func NewDeviceRepository(db *gorm.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) base(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).
		Table("access_devices ad").
		Select("ad.*, COALESCE(b.name, '') AS branch_name").
		Joins("LEFT JOIN branches b ON b.id = ad.branch_id").
		Where("ad.deleted_at IS NULL")
}

func (r *deviceRepository) List(ctx context.Context, f AccessDeviceFilter) ([]AccessDeviceRow, error) {
	q := r.base(ctx)
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []AccessDeviceRow{}, nil
		}
		q = q.Where("ad.branch_id IN ?", f.BranchIDs)
	}
	if f.Status != "" {
		q = q.Where("ad.status = ?", f.Status)
	}
	var rows []AccessDeviceRow
	if err := q.Order("ad.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *deviceRepository) FindByID(ctx context.Context, id uint) (*AccessDeviceRow, error) {
	var rows []AccessDeviceRow
	if err := r.base(ctx).Where("ad.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *deviceRepository) FindByAPIKeyHash(ctx context.Context, keyHash string) (*AccessDeviceRow, error) {
	var rows []AccessDeviceRow
	if err := r.base(ctx).Where("ad.api_key_hash = ?", keyHash).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *deviceRepository) ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	q := postgres.Conn(ctx, r.db).Model(&entity.AccessDevice{}).
		Where("UPPER(code) = UPPER(?)", strings.TrimSpace(code))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *deviceRepository) Create(ctx context.Context, d *entity.AccessDevice) error {
	return postgres.Conn(ctx, r.db).Omit("CreatedBy", "UpdatedBy").Create(d).Error
}

func (r *deviceRepository) Update(ctx context.Context, d *entity.AccessDevice) error {
	return postgres.Conn(ctx, r.db).Omit("CreatedBy", "UpdatedBy").Save(d).Error
}

func (r *deviceRepository) Delete(ctx context.Context, id uint, _ *uint) error {
	now := time.Now()
	return postgres.Conn(ctx, r.db).Model(&entity.AccessDevice{}).
		Where("id = ?", id).
		Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error
}

func (r *deviceRepository) TouchSeen(ctx context.Context, id uint, seenAt time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.AccessDevice{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_seen_at": seenAt,
			"status":       entity.DeviceStatusActive,
			"updated_at":   seenAt,
		}).Error
}

func (r *deviceRepository) ListStaleActiveDevices(ctx context.Context, cutoff time.Time) ([]AccessDeviceRow, error) {
	var rows []AccessDeviceRow
	err := r.base(ctx).
		Where("ad.status = ? AND ad.last_seen_at IS NOT NULL AND ad.last_seen_at < ?", entity.DeviceStatusActive, cutoff).
		Order("ad.id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *deviceRepository) MarkDevicesOffline(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return postgres.Conn(ctx, r.db).Model(&entity.AccessDevice{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":     entity.DeviceStatusOffline,
			"updated_at": time.Now(),
		}).Error
}
