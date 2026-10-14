package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// ShiftFilter: các ca giao với khoảng [From, To).
type ShiftFilter struct {
	BranchIDs []uint // nil = không lọc
	StaffID   uint
	From      time.Time
	To        time.Time
}

type (
	ShiftRepository interface {
		List(ctx context.Context, f ShiftFilter) ([]entity.ShiftDetail, error)
		FindByID(ctx context.Context, id uint) (*entity.ShiftDetail, error)
		Create(ctx context.Context, s *entity.Shift) error
		// CreateIfFree chèn ca, bỏ qua (created=false) nếu trùng giờ với ca khác của nhân viên.
		CreateIfFree(ctx context.Context, s *entity.Shift) (created bool, err error)
		Update(ctx context.Context, s *entity.Shift) error
		Delete(ctx context.Context, id uint) error
		// StaffInBranch: nhân viên còn hoạt động, chưa xóa và có vai trò tại chi nhánh.
		StaffInBranch(ctx context.Context, staffID, branchID uint) (bool, error)
	}

	shiftRepository struct {
		db *gorm.DB
	}
)

func NewShiftRepository(db *gorm.DB) ShiftRepository {
	return &shiftRepository{db: db}
}

func (r *shiftRepository) detail(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("shifts").
		Select(`shifts.*, u.full_name AS staff_name, COALESCE(sp.staff_code, '') AS staff_code,
			COALESCE(sp.position, '') AS position, COALESCE(b.name, '') AS branch_name`).
		Joins("JOIN users u ON u.id = shifts.staff_id").
		Joins("LEFT JOIN staff_profiles sp ON sp.user_id = shifts.staff_id").
		Joins("LEFT JOIN branches b ON b.id = shifts.branch_id")
}

func (r *shiftRepository) List(ctx context.Context, f ShiftFilter) ([]entity.ShiftDetail, error) {
	tx := r.detail(ctx).Where("shifts.start_at < ? AND shifts.end_at > ?", f.To, f.From)
	if f.BranchIDs != nil {
		tx = tx.Where("shifts.branch_id IN ?", f.BranchIDs)
	}
	if f.StaffID > 0 {
		tx = tx.Where("shifts.staff_id = ?", f.StaffID)
	}
	items := []entity.ShiftDetail{}
	err := tx.Order("shifts.start_at, u.full_name").Limit(2000).Scan(&items).Error
	return items, err
}

func (r *shiftRepository) FindByID(ctx context.Context, id uint) (*entity.ShiftDetail, error) {
	var items []entity.ShiftDetail
	if err := r.detail(ctx).Where("shifts.id = ?", id).Limit(1).Scan(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &items[0], nil
}

func (r *shiftRepository) Create(ctx context.Context, s *entity.Shift) error {
	return postgres.Conn(ctx, r.db).Create(s).Error
}

func (r *shiftRepository) CreateIfFree(ctx context.Context, s *entity.Shift) (bool, error) {
	// ON CONFLICT DO NOTHING (không chỉ định cột) cũng áp dụng cho EXCLUDE constraint,
	// nên ca trùng giờ bị bỏ qua mà không làm hỏng transaction.
	res := postgres.Conn(ctx, r.db).Clauses(clause.OnConflict{DoNothing: true}).Create(s)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *shiftRepository) Update(ctx context.Context, s *entity.Shift) error {
	return postgres.Conn(ctx, r.db).Model(s).
		Select("branch_id", "staff_id", "start_at", "end_at", "note", "updated_by").Updates(s).Error
}

func (r *shiftRepository) Delete(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Shift{}, id).Error
}

func (r *shiftRepository) StaffInBranch(ctx context.Context, staffID, branchID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("users u").
		Joins("JOIN user_roles ur ON ur.user_id = u.id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("u.id = ? AND u.deleted_at IS NULL AND u.status = 'ACTIVE' AND ur.branch_id = ? AND r.code <> 'MEMBER'", staffID, branchID).
		Count(&n).Error
	return n > 0, err
}
