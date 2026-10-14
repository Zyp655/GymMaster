package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// ClassScheduleFilter: các buổi lớp bắt đầu trong khoảng [From, To).
type ClassScheduleFilter struct {
	BranchIDs   []uint // nil = không lọc
	From        time.Time
	To          time.Time
	TrainerID   uint
	ClassTypeID uint
	RoomID      uint
	Status      string
}

type (
	ClassScheduleRepository interface {
		List(ctx context.Context, f ClassScheduleFilter) ([]entity.ClassScheduleDetail, error)
		FindByID(ctx context.Context, id uint) (*entity.ClassScheduleDetail, error)
		// FindForUpdate đọc dòng lịch và khóa (SELECT ... FOR UPDATE) – chỉ dùng trong transaction.
		FindForUpdate(ctx context.Context, id uint) (*entity.ClassSchedule, error)
		Create(ctx context.Context, s *entity.ClassSchedule) error
		// CreateIfFree chèn buổi lớp, created=false nếu vi phạm EXCLUDE (trùng phòng/HLV).
		CreateIfFree(ctx context.Context, s *entity.ClassSchedule) (created bool, err error)
		Update(ctx context.Context, s *entity.ClassSchedule) error
		// AddBooked cộng delta vào booked_count.
		AddBooked(ctx context.Context, id uint, delta int) error
		// MarkCancelled chuyển buổi sang CANCELLED, đặt booked_count = 0.
		MarkCancelled(ctx context.Context, id uint, reason string, by *uint) error
		// SeriesFrom trả các buổi SCHEDULED của chuỗi có start_at >= from (theo thứ tự thời gian).
		SeriesFrom(ctx context.Context, seriesID string, from time.Time) ([]entity.ClassSchedule, error)
		// CompleteEnded chuyển các buổi SCHEDULED đã kết thúc trước endedBefore sang COMPLETED.
		CompleteEnded(ctx context.Context, endedBefore time.Time) (int64, error)

		RoomOverlap(ctx context.Context, roomID uint, start, end time.Time, excludeID uint) (bool, error)
		TrainerOverlap(ctx context.Context, trainerID uint, start, end time.Time, excludeID uint) (bool, error)
		// TrainerPTOverlap: HLV có buổi PT (SCHEDULED/COMPLETED) giao với khoảng thời gian.
		TrainerPTOverlap(ctx context.Context, trainerID uint, start, end time.Time) (bool, error)

		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
		FindRoom(ctx context.Context, id uint) (*entity.Room, error)
		FindClassType(ctx context.Context, id uint) (*entity.ClassType, error)
		// TrainerInBranch: user ACTIVE, là HLV (hồ sơ TRAINER hoặc vai trò TRAINER)
		// và có vai trò tại chi nhánh (hoặc vai trò toàn hệ thống).
		TrainerInBranch(ctx context.Context, trainerID, branchID uint) (bool, error)
	}

	classScheduleRepository struct {
		db *gorm.DB
	}
)

func NewClassScheduleRepository(db *gorm.DB) ClassScheduleRepository {
	return &classScheduleRepository{db: db}
}

func (r *classScheduleRepository) detail(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("class_schedules cs").
		Select(`cs.*, ct.code AS class_type_code, ct.name AS class_type_name, ct.color AS class_type_color,
			COALESCE(b.name, '') AS branch_name, COALESCE(rm.name, '') AS room_name, COALESCE(u.full_name, '') AS trainer_name,
			(SELECT count(*) FROM class_bookings cb WHERE cb.schedule_id = cs.id AND cb.status = 'WAITLIST') AS waitlist_count,
			(SELECT count(*) FROM class_bookings cb WHERE cb.schedule_id = cs.id AND cb.status = 'ATTENDED') AS attended_count`).
		Joins("JOIN class_types ct ON ct.id = cs.class_type_id").
		Joins("LEFT JOIN branches b ON b.id = cs.branch_id").
		Joins("LEFT JOIN rooms rm ON rm.id = cs.room_id").
		Joins("LEFT JOIN users u ON u.id = cs.trainer_id")
}

func (r *classScheduleRepository) List(ctx context.Context, f ClassScheduleFilter) ([]entity.ClassScheduleDetail, error) {
	tx := r.detail(ctx).Where("cs.start_at >= ? AND cs.start_at < ?", f.From, f.To)
	if f.BranchIDs != nil {
		tx = tx.Where("cs.branch_id IN ?", f.BranchIDs)
	}
	if f.TrainerID > 0 {
		tx = tx.Where("cs.trainer_id = ?", f.TrainerID)
	}
	if f.ClassTypeID > 0 {
		tx = tx.Where("cs.class_type_id = ?", f.ClassTypeID)
	}
	if f.RoomID > 0 {
		tx = tx.Where("cs.room_id = ?", f.RoomID)
	}
	if f.Status != "" {
		tx = tx.Where("cs.status = ?", f.Status)
	}
	items := []entity.ClassScheduleDetail{}
	err := tx.Order("cs.start_at, cs.id").Limit(3000).Scan(&items).Error
	return items, err
}

func (r *classScheduleRepository) FindByID(ctx context.Context, id uint) (*entity.ClassScheduleDetail, error) {
	var items []entity.ClassScheduleDetail
	if err := r.detail(ctx).Where("cs.id = ?", id).Limit(1).Scan(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &items[0], nil
}

func (r *classScheduleRepository) FindForUpdate(ctx context.Context, id uint) (*entity.ClassSchedule, error) {
	var s entity.ClassSchedule
	err := postgres.Conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, id).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *classScheduleRepository) Create(ctx context.Context, s *entity.ClassSchedule) error {
	return postgres.Conn(ctx, r.db).Create(s).Error
}

func (r *classScheduleRepository) CreateIfFree(ctx context.Context, s *entity.ClassSchedule) (bool, error) {
	// ON CONFLICT DO NOTHING (không chỉ định cột) áp dụng cả cho EXCLUDE constraint.
	res := postgres.Conn(ctx, r.db).Clauses(clause.OnConflict{DoNothing: true}).Create(s)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *classScheduleRepository) Update(ctx context.Context, s *entity.ClassSchedule) error {
	return postgres.Conn(ctx, r.db).Model(s).
		Select("room_id", "trainer_id", "start_at", "end_at", "capacity", "note", "updated_by").Updates(s).Error
}

func (r *classScheduleRepository) AddBooked(ctx context.Context, id uint, delta int) error {
	return postgres.Conn(ctx, r.db).Model(&entity.ClassSchedule{}).Where("id = ?", id).
		UpdateColumn("booked_count", gorm.Expr("booked_count + ?", delta)).Error
}

func (r *classScheduleRepository) MarkCancelled(ctx context.Context, id uint, reason string, by *uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.ClassSchedule{}).Where("id = ?", id).
		Updates(map[string]any{"status": entity.ClassScheduleCancelled, "cancel_reason": reason, "booked_count": 0, "updated_by": by}).Error
}

func (r *classScheduleRepository) SeriesFrom(ctx context.Context, seriesID string, from time.Time) ([]entity.ClassSchedule, error) {
	var items []entity.ClassSchedule
	err := postgres.Conn(ctx, r.db).
		Where("series_id = ? AND start_at >= ? AND status = ?", seriesID, from, entity.ClassScheduleScheduled).
		Order("start_at").Find(&items).Error
	return items, err
}

func (r *classScheduleRepository) CompleteEnded(ctx context.Context, endedBefore time.Time) (int64, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.ClassSchedule{}).
		Where("status = ? AND end_at < ?", entity.ClassScheduleScheduled, endedBefore).
		UpdateColumn("status", entity.ClassScheduleCompleted)
	return res.RowsAffected, res.Error
}

func (r *classScheduleRepository) overlap(ctx context.Context, col string, id uint, start, end time.Time, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.ClassSchedule{}).
		Where(col+" = ? AND status <> ? AND start_at < ? AND end_at > ? AND id <> ?",
			id, entity.ClassScheduleCancelled, end, start, excludeID).
		Count(&n).Error
	return n > 0, err
}

func (r *classScheduleRepository) RoomOverlap(ctx context.Context, roomID uint, start, end time.Time, excludeID uint) (bool, error) {
	return r.overlap(ctx, "room_id", roomID, start, end, excludeID)
}

func (r *classScheduleRepository) TrainerOverlap(ctx context.Context, trainerID uint, start, end time.Time, excludeID uint) (bool, error) {
	return r.overlap(ctx, "trainer_id", trainerID, start, end, excludeID)
}

func (r *classScheduleRepository) TrainerPTOverlap(ctx context.Context, trainerID uint, start, end time.Time) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("pt_sessions").
		Where("trainer_id = ? AND status IN ('SCHEDULED', 'COMPLETED') AND start_at < ? AND end_at > ?", trainerID, end, start).
		Count(&n).Error
	return n > 0, err
}

func (r *classScheduleRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *classScheduleRepository) FindRoom(ctx context.Context, id uint) (*entity.Room, error) {
	var room entity.Room
	if err := postgres.Conn(ctx, r.db).Omit("Branch").First(&room, id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *classScheduleRepository) FindClassType(ctx context.Context, id uint) (*entity.ClassType, error) {
	var t entity.ClassType
	if err := postgres.Conn(ctx, r.db).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *classScheduleRepository) TrainerInBranch(ctx context.Context, trainerID, branchID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("users u").
		Where("u.id = ? AND u.deleted_at IS NULL AND u.status = 'ACTIVE'", trainerID).
		Where(`(EXISTS (SELECT 1 FROM staff_profiles sp WHERE sp.user_id = u.id AND sp.position = 'TRAINER')
			OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = u.id AND r.code = 'TRAINER'))`).
		Where(`EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = u.id AND r.code <> 'MEMBER' AND (ur.branch_id = ? OR ur.branch_id IS NULL))`, branchID).
		Count(&n).Error
	return n > 0, err
}
