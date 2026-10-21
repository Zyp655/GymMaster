package repository

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	BranchRepository interface {
		List(ctx context.Context, q dto.Query) ([]entity.Branch, int64, error)
		FindByID(ctx context.Context, id uint) (*entity.Branch, error)
		ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error)
		Create(ctx context.Context, b *entity.Branch) error
		Update(ctx context.Context, b *entity.Branch) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		// CountStaff đếm số tài khoản đang được gán role tại chi nhánh.
		CountStaff(ctx context.Context, id uint) (int64, error)
		// Dependencies đếm dữ liệu nghiệp vụ còn gắn với chi nhánh (chặn xóa chi nhánh).
		Dependencies(ctx context.Context, id uint) (BranchDependencies, error)
	}

	// BranchDependencies: hội viên (chi nhánh gốc), gói tập chưa kết thúc, hóa đơn thuộc chi nhánh.
	BranchDependencies struct {
		Members    int64
		ActiveSubs int64
		Invoices   int64
	}

	branchRepository struct {
		db *gorm.DB
	}
)

func NewBranchRepository(db *gorm.DB) BranchRepository {
	return &branchRepository{db: db}
}

func (r *branchRepository) List(ctx context.Context, q dto.Query) ([]entity.Branch, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Branch{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("code ILIKE ? OR name ILIKE ? OR address ILIKE ? OR phone ILIKE ?", kw, kw, kw, kw)
	}
	if status, ok := q.Filters["status"]; ok {
		tx = tx.Where("status = ?", status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.Branch
	if err := tx.Scopes(q.Paginate).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *branchRepository) FindByID(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *branchRepository) ExistsCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Branch{}).
		Where("code = ? AND id <> ?", code, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *branchRepository) Create(ctx context.Context, b *entity.Branch) error {
	return postgres.Conn(ctx, r.db).Create(b).Error
}

func (r *branchRepository) Update(ctx context.Context, b *entity.Branch) error {
	// Select các cột cho phép sửa – tránh mass-assignment ghi đè created_at/created_by.
	return postgres.Conn(ctx, r.db).Model(b).
		Select("code", "name", "address", "phone", "email", "open_time", "close_time", "status", "updated_by").
		Updates(b).Error
}

func (r *branchRepository) Delete(ctx context.Context, id uint, deletedBy *uint) error {
	return postgres.Conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.Branch{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&entity.Branch{}, id).Error
	})
}

func (r *branchRepository) CountStaff(ctx context.Context, id uint) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.UserRole{}).Where("branch_id = ?", id).Count(&n).Error
	return n, err
}

func (r *branchRepository) Dependencies(ctx context.Context, id uint) (BranchDependencies, error) {
	var d BranchDependencies
	err := postgres.Conn(ctx, r.db).Raw(`SELECT
		(SELECT COUNT(*) FROM members WHERE home_branch_id = ? AND deleted_at IS NULL) AS members,
		(SELECT COUNT(*) FROM subscriptions WHERE branch_id = ? AND status IN ('PENDING', 'ACTIVE', 'FROZEN')) AS active_subs,
		(SELECT COUNT(*) FROM invoices WHERE branch_id = ?) AS invoices`, id, id, id).Scan(&d).Error
	return d, err
}
