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
	LockerRow struct {
		entity.Locker
		BranchName string
		RentalID   *uint
		MemberID   *uint
		MemberName string
		MemberCode string
		RentalType string
		StartAt    *time.Time
		EndAt      *time.Time
	}

	LockerRentalRow struct {
		entity.LockerRental
		LockerCode string
		Zone       string
		Size       string
		BranchID   uint
		BranchName string
		MemberCode string
		MemberName string
	}

	LockerFilter struct {
		BranchIDs []uint
		Zone      string
		Size      string
		Status    string
	}

	LockerRentalFilter struct {
		BranchIDs []uint
		LockerID  uint
		MemberID  uint
		Type      string
		Status    string
	}

	LockerRepository interface {
		ListLockers(ctx context.Context, filter LockerFilter) ([]LockerRow, error)
		FindLockerByID(ctx context.Context, id uint) (*LockerRow, error)
		ExistsLockerCode(ctx context.Context, branchID uint, code string, excludeID uint) (bool, error)
		CreateLocker(ctx context.Context, l *entity.Locker) error
		UpdateLocker(ctx context.Context, l *entity.Locker) error
		DeleteLocker(ctx context.Context, id uint) error

		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)

		CreateRental(ctx context.Context, r *entity.LockerRental) error
		EndActiveRentalsByLocker(ctx context.Context, lockerID uint, endedAt time.Time) error
		ListRentals(ctx context.Context, filter LockerRentalFilter) ([]LockerRentalRow, error)
		ListExpiredActiveRentals(ctx context.Context, now time.Time) ([]entity.LockerRental, error)
		ExpireRental(ctx context.Context, rentalID, lockerID uint, now time.Time) error
	}

	lockerRepository struct {
		db *gorm.DB
	}
)

func NewLockerRepository(db *gorm.DB) LockerRepository {
	return &lockerRepository{db: db}
}

func (r *lockerRepository) baseLockers(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).
		Table("lockers l").
		Select(`l.*, COALESCE(b.name, '') AS branch_name,
			lr.id AS rental_id, lr.member_id,
			COALESCE(m.full_name, '') AS member_name,
			COALESCE(m.member_code, '') AS member_code,
			COALESCE(lr.type, '') AS rental_type,
			lr.start_at, lr.end_at`).
		Joins("LEFT JOIN branches b ON b.id = l.branch_id").
		Joins("LEFT JOIN locker_rentals lr ON lr.locker_id = l.id AND lr.status = ?", entity.LockerRentalActive).
		Joins("LEFT JOIN members m ON m.id = lr.member_id")
}

func (r *lockerRepository) ListLockers(ctx context.Context, f LockerFilter) ([]LockerRow, error) {
	q := r.baseLockers(ctx)
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []LockerRow{}, nil
		}
		q = q.Where("l.branch_id IN ?", f.BranchIDs)
	}
	if f.Zone != "" {
		q = q.Where("l.zone = ?", f.Zone)
	}
	if f.Size != "" {
		q = q.Where("l.size = ?", f.Size)
	}
	if f.Status != "" {
		q = q.Where("l.status = ?", f.Status)
	}
	var rows []LockerRow
	if err := q.Order("l.zone ASC, l.code ASC, l.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *lockerRepository) FindLockerByID(ctx context.Context, id uint) (*LockerRow, error) {
	var rows []LockerRow
	if err := r.baseLockers(ctx).Where("l.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *lockerRepository) ExistsLockerCode(ctx context.Context, branchID uint, code string, excludeID uint) (bool, error) {
	q := postgres.Conn(ctx, r.db).Model(&entity.Locker{}).
		Where("branch_id = ? AND UPPER(code) = UPPER(?)", branchID, strings.TrimSpace(code))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *lockerRepository) CreateLocker(ctx context.Context, l *entity.Locker) error {
	return postgres.Conn(ctx, r.db).Create(l).Error
}

func (r *lockerRepository) UpdateLocker(ctx context.Context, l *entity.Locker) error {
	return postgres.Conn(ctx, r.db).Save(l).Error
}

func (r *lockerRepository) DeleteLocker(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Locker{}, id).Error
}

func (r *lockerRepository) FindMemberByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *lockerRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *lockerRepository) CreateRental(ctx context.Context, lr *entity.LockerRental) error {
	return postgres.Conn(ctx, r.db).Create(lr).Error
}

func (r *lockerRepository) EndActiveRentalsByLocker(ctx context.Context, lockerID uint, endedAt time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.LockerRental{}).
		Where("locker_id = ? AND status = ?", lockerID, entity.LockerRentalActive).
		Updates(map[string]any{
			"status":     entity.LockerRentalEnded,
			"end_at":     endedAt,
			"updated_at": endedAt,
		}).Error
}

func (r *lockerRepository) ListRentals(ctx context.Context, f LockerRentalFilter) ([]LockerRentalRow, error) {
	q := postgres.Conn(ctx, r.db).
		Table("locker_rentals lr").
		Select(`lr.*, l.code AS locker_code, l.zone, l.size, l.branch_id,
			COALESCE(b.name, '') AS branch_name,
			COALESCE(m.member_code, '') AS member_code,
			COALESCE(m.full_name, '') AS member_name`).
		Joins("JOIN lockers l ON l.id = lr.locker_id").
		Joins("LEFT JOIN branches b ON b.id = l.branch_id").
		Joins("JOIN members m ON m.id = lr.member_id")
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []LockerRentalRow{}, nil
		}
		q = q.Where("l.branch_id IN ?", f.BranchIDs)
	}
	if f.LockerID > 0 {
		q = q.Where("lr.locker_id = ?", f.LockerID)
	}
	if f.MemberID > 0 {
		q = q.Where("lr.member_id = ?", f.MemberID)
	}
	if f.Type != "" {
		q = q.Where("lr.type = ?", f.Type)
	}
	if f.Status != "" {
		q = q.Where("lr.status = ?", f.Status)
	}
	var rows []LockerRentalRow
	if err := q.Order("lr.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *lockerRepository) ListExpiredActiveRentals(ctx context.Context, now time.Time) ([]entity.LockerRental, error) {
	var rows []entity.LockerRental
	err := postgres.Conn(ctx, r.db).Model(&entity.LockerRental{}).
		Where("status = ? AND end_at IS NOT NULL AND end_at < ?", entity.LockerRentalActive, now).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *lockerRepository) ExpireRental(ctx context.Context, rentalID, lockerID uint, now time.Time) error {
	db := postgres.Conn(ctx, r.db)
	if err := db.Model(&entity.LockerRental{}).
		Where("id = ?", rentalID).
		Updates(map[string]any{"status": entity.LockerRentalExpired, "updated_at": now}).Error; err != nil {
		return err
	}
	return db.Model(&entity.Locker{}).
		Where("id = ? AND status IN ?", lockerID, []string{entity.LockerStatusOccupied, entity.LockerStatusRented}).
		Updates(map[string]any{"status": entity.LockerStatusAvailable, "updated_at": now}).Error
}
