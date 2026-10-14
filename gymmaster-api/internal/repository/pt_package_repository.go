package repository

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	PTPackageRepository interface {
		List(ctx context.Context, q dto.Query) ([]entity.PTPackage, int64, error)
		// ListActive trả các gói PT đang bán, theo số buổi rồi giá.
		ListActive(ctx context.Context) ([]entity.PTPackage, error)
		FindByID(ctx context.Context, id uint) (*entity.PTPackage, error)
		ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error)
		Create(ctx context.Context, p *entity.PTPackage) error
		Update(ctx context.Context, p *entity.PTPackage) error
		UpdateStatus(ctx context.Context, id uint, status string, updatedBy *uint) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		// CountContracts đếm số hợp đồng PT (mọi trạng thái) dùng gói.
		CountContracts(ctx context.Context, id uint) (int64, error)
	}

	ptPackageRepository struct {
		db *gorm.DB
	}
)

func NewPTPackageRepository(db *gorm.DB) PTPackageRepository {
	return &ptPackageRepository{db: db}
}

func (r *ptPackageRepository) List(ctx context.Context, q dto.Query) ([]entity.PTPackage, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.PTPackage{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("code ILIKE ? OR name ILIKE ?", kw, kw)
	}
	if v, ok := q.Filters["status"]; ok {
		tx = tx.Where("status = ?", v)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.PTPackage
	if err := tx.Scopes(q.Paginate).Order("id").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ptPackageRepository) ListActive(ctx context.Context) ([]entity.PTPackage, error) {
	var items []entity.PTPackage
	err := postgres.Conn(ctx, r.db).Where("status = ?", "ACTIVE").
		Order("session_count ASC, price ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *ptPackageRepository) FindByID(ctx context.Context, id uint) (*entity.PTPackage, error) {
	var p entity.PTPackage
	if err := postgres.Conn(ctx, r.db).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ptPackageRepository) ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.PTPackage{}).
		Where("code = ? AND id <> ?", code, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *ptPackageRepository) Create(ctx context.Context, p *entity.PTPackage) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *ptPackageRepository) Update(ctx context.Context, p *entity.PTPackage) error {
	return postgres.Conn(ctx, r.db).Model(p).
		Select("code", "name", "description", "session_count", "session_minutes", "validity_days", "price", "status", "updated_by").
		Updates(p).Error
}

func (r *ptPackageRepository) UpdateStatus(ctx context.Context, id uint, status string, updatedBy *uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.PTPackage{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_by": updatedBy}).Error
}

func (r *ptPackageRepository) Delete(ctx context.Context, id uint, deletedBy *uint) error {
	return postgres.Conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.PTPackage{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&entity.PTPackage{}, id).Error
	})
}

func (r *ptPackageRepository) CountContracts(ctx context.Context, id uint) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("pt_contracts").Where("pt_package_id = ?", id).Count(&n).Error
	return n, err
}
