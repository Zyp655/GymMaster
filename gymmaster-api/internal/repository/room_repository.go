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
	RoomRepository interface {
		List(ctx context.Context, q dto.Query, branchIDs []uint) ([]entity.Room, int64, error)
		FindByID(ctx context.Context, id uint) (*entity.Room, error)
		ExistsName(ctx context.Context, branchID uint, name string, excludeID uint) (bool, error)
		BranchExists(ctx context.Context, branchID uint) (bool, error)
		Create(ctx context.Context, room *entity.Room) error
		Update(ctx context.Context, room *entity.Room) error
		Delete(ctx context.Context, id uint, deletedBy *uint) error
		// CountFutureSchedules đếm lịch lớp SCHEDULED chưa diễn ra đang dùng phòng.
		CountFutureSchedules(ctx context.Context, id uint, now time.Time) (int64, error)
		// MaxFutureScheduleCapacity: sức chứa lớn nhất của các lịch lớp SCHEDULED chưa kết thúc dùng phòng (0 nếu không có).
		MaxFutureScheduleCapacity(ctx context.Context, id uint, now time.Time) (int, error)
		ActiveOptions(ctx context.Context, branchIDs []uint) ([]entity.Room, error)
	}

	roomRepository struct {
		db *gorm.DB
	}
)

func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) List(ctx context.Context, q dto.Query, branchIDs []uint) ([]entity.Room, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Room{})
	if branchIDs != nil {
		tx = tx.Where("branch_id IN ?", branchIDs)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR description ILIKE ?", kw, kw)
	}
	if status, ok := q.Filters["status"]; ok {
		tx = tx.Where("status = ?", status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.Room
	err := tx.Preload("Branch").Scopes(q.Paginate).Find(&items).Error
	return items, total, err
}

func (r *roomRepository) FindByID(ctx context.Context, id uint) (*entity.Room, error) {
	var room entity.Room
	if err := postgres.Conn(ctx, r.db).Preload("Branch").First(&room, id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) ExistsName(ctx context.Context, branchID uint, name string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Room{}).
		Where("branch_id = ? AND lower(name) = lower(?) AND id <> ?", branchID, name, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *roomRepository) BranchExists(ctx context.Context, branchID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Branch{}).Where("id = ?", branchID).Count(&n).Error
	return n > 0, err
}

func (r *roomRepository) Create(ctx context.Context, room *entity.Room) error {
	return postgres.Conn(ctx, r.db).Omit("Branch").Create(room).Error
}

func (r *roomRepository) Update(ctx context.Context, room *entity.Room) error {
	return postgres.Conn(ctx, r.db).Model(room).Omit("Branch").
		Select("branch_id", "name", "capacity", "description", "status", "updated_by").Updates(room).Error
}

func (r *roomRepository) Delete(ctx context.Context, id uint, deletedBy *uint) error {
	return postgres.Conn(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.Room{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
			return err
		}
		return tx.Delete(&entity.Room{}, id).Error
	})
}

func (r *roomRepository) CountFutureSchedules(ctx context.Context, id uint, now time.Time) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("class_schedules").
		Where("room_id = ? AND status = 'SCHEDULED' AND start_at > ?", id, now).Count(&n).Error
	return n, err
}

func (r *roomRepository) MaxFutureScheduleCapacity(ctx context.Context, id uint, now time.Time) (int, error) {
	var n int
	err := postgres.Conn(ctx, r.db).Table("class_schedules").
		Where("room_id = ? AND status = 'SCHEDULED' AND end_at > ?", id, now).
		Select("COALESCE(MAX(capacity), 0)").Scan(&n).Error
	return n, err
}

func (r *roomRepository) ActiveOptions(ctx context.Context, branchIDs []uint) ([]entity.Room, error) {
	tx := postgres.Conn(ctx, r.db).Where("status = 'ACTIVE'")
	if branchIDs != nil {
		tx = tx.Where("branch_id IN ?", branchIDs)
	}
	var items []entity.Room
	err := tx.Order("branch_id, name").Find(&items).Error
	return items, err
}
