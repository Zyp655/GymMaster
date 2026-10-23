package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// ClassBookingSubscription là đăng ký gói tập ACTIVE của hội viên kèm thông tin gói,
// dùng để kiểm tra quyền đặt lớp.
type ClassBookingSubscription struct {
	ID                uint
	PlanID            uint
	BranchID          uint
	StartDate         time.Time
	EndDate           time.Time
	RemainingSessions *int
	Status            string
	PlanName          string
	PlanType          string
	AccessScope       string
	IncludesClasses   bool
}

type (
	ClassBookingRepository interface {
		Create(ctx context.Context, b *entity.ClassBooking) error
		FindByID(ctx context.Context, id uint) (*entity.ClassBooking, error)
		FindDetail(ctx context.Context, id uint) (*entity.ClassBookingDetail, error)
		// ListBySchedule trả mọi lượt đặt của một buổi lớp (kèm thông tin hội viên).
		ListBySchedule(ctx context.Context, scheduleID uint) ([]entity.ClassBookingDetail, error)
		// ListByMember trả lượt đặt của hội viên có buổi lớp bắt đầu trong [from, to).
		ListByMember(ctx context.Context, memberID uint, from, to time.Time) ([]entity.ClassBookingDetail, error)
		// ListActiveBySchedule: các lượt BOOKED/WAITLIST của buổi lớp.
		ListActiveBySchedule(ctx context.Context, scheduleID uint) ([]entity.ClassBookingDetail, error)
		// ExistsActive: hội viên đã có lượt BOOKED/WAITLIST/ATTENDED trong buổi lớp.
		ExistsActive(ctx context.Context, scheduleID, memberID uint) (bool, error)
		// MemberOverlap: hội viên có lượt BOOKED/WAITLIST ở buổi lớp khác (chưa hủy) giao giờ.
		MemberOverlap(ctx context.Context, memberID uint, start, end time.Time, excludeScheduleID uint) (bool, error)
		MaxWaitlistPosition(ctx context.Context, scheduleID uint) (int, error)
		// FirstWaitlist trả người đứng đầu danh sách chờ (nil nếu trống).
		FirstWaitlist(ctx context.Context, scheduleID uint) (*entity.ClassBooking, error)
		// UpdateStatus cập nhật trạng thái và các mốc thời gian của lượt đặt.
		UpdateStatus(ctx context.Context, b *entity.ClassBooking) error
		// MarkNoShows chuyển các lượt BOOKED của buổi lớp (chưa hủy) kết thúc trước endedBefore sang NO_SHOW,
		// trả member_id của từng lượt bị chuyển.
		MarkNoShows(ctx context.Context, endedBefore time.Time) ([]uint, error)
		// CountNoShows đếm số NO_SHOW của hội viên với buổi lớp bắt đầu từ since.
		CountNoShows(ctx context.Context, memberID uint, since time.Time) (int64, error)

		FindMember(ctx context.Context, id uint) (*entity.Member, error)
		SetMemberBlockedUntil(ctx context.Context, memberID uint, until time.Time) error
		// ClearMemberBlock gỡ khóa đặt lớp của hội viên (booking_blocked_until = NULL).
		ClearMemberBlock(ctx context.Context, memberID uint) error
		// ActiveSubscriptions trả các đăng ký ACTIVE của hội viên (sắp theo end_date tăng dần).
		ActiveSubscriptions(ctx context.Context, memberID uint) ([]ClassBookingSubscription, error)
	}

	classBookingRepository struct {
		db *gorm.DB
	}
)

func NewClassBookingRepository(db *gorm.DB) ClassBookingRepository {
	return &classBookingRepository{db: db}
}

// detail: lượt đặt kèm hội viên + buổi lớp. trainer_* là HLV thực dạy; waitlist_rank là thứ hạng hiện tại trong danh sách
// chờ (cùng thứ tự đẩy lên của FirstWaitlist: waitlist_position, booked_at, id).
func (r *classBookingRepository) detail(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("class_bookings cb").
		Select(`cb.*, m.member_code, m.full_name AS member_name, m.phone AS member_phone, m.user_id AS member_user_id,
			cs.class_type_id, ct.name AS class_type_name, ct.color AS class_type_color,
			cs.branch_id, COALESCE(b.name, '') AS branch_name, COALESCE(rm.name, '') AS room_name,
			COALESCE(cs.substitute_trainer_id, cs.trainer_id) AS trainer_id, COALESCE(u.full_name, '') AS trainer_name,
			cs.start_at, cs.end_at, cs.status AS schedule_status,
			CASE WHEN cb.status = 'WAITLIST' THEN (SELECT count(*) FROM class_bookings w
				WHERE w.schedule_id = cb.schedule_id AND w.status = 'WAITLIST'
				  AND (COALESCE(w.waitlist_position, 0), w.booked_at, w.id) <= (COALESCE(cb.waitlist_position, 0), cb.booked_at, cb.id))
			END AS waitlist_rank`).
		Joins("JOIN members m ON m.id = cb.member_id").
		Joins("JOIN class_schedules cs ON cs.id = cb.schedule_id").
		Joins("JOIN class_types ct ON ct.id = cs.class_type_id").
		Joins("LEFT JOIN branches b ON b.id = cs.branch_id").
		Joins("LEFT JOIN rooms rm ON rm.id = cs.room_id").
		// HLV thực dạy: HLV dạy thay (nếu có), ngược lại HLV chính.
		Joins("LEFT JOIN users u ON u.id = COALESCE(cs.substitute_trainer_id, cs.trainer_id)")
}

func (r *classBookingRepository) Create(ctx context.Context, b *entity.ClassBooking) error {
	return postgres.Conn(ctx, r.db).Create(b).Error
}

func (r *classBookingRepository) FindByID(ctx context.Context, id uint) (*entity.ClassBooking, error) {
	var b entity.ClassBooking
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *classBookingRepository) FindDetail(ctx context.Context, id uint) (*entity.ClassBookingDetail, error) {
	var items []entity.ClassBookingDetail
	if err := r.detail(ctx).Where("cb.id = ?", id).Limit(1).Scan(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &items[0], nil
}

const classBookingOrder = `CASE cb.status WHEN 'BOOKED' THEN 0 WHEN 'ATTENDED' THEN 0 WHEN 'NO_SHOW' THEN 0
	WHEN 'WAITLIST' THEN 1 ELSE 2 END, cb.waitlist_position NULLS FIRST, cb.booked_at, cb.id`

func (r *classBookingRepository) ListBySchedule(ctx context.Context, scheduleID uint) ([]entity.ClassBookingDetail, error) {
	items := []entity.ClassBookingDetail{}
	err := r.detail(ctx).Where("cb.schedule_id = ?", scheduleID).Order(classBookingOrder).Scan(&items).Error
	return items, err
}

func (r *classBookingRepository) ListActiveBySchedule(ctx context.Context, scheduleID uint) ([]entity.ClassBookingDetail, error) {
	items := []entity.ClassBookingDetail{}
	err := r.detail(ctx).Where("cb.schedule_id = ? AND cb.status IN ('BOOKED', 'WAITLIST')", scheduleID).
		Order(classBookingOrder).Scan(&items).Error
	return items, err
}

func (r *classBookingRepository) ListByMember(ctx context.Context, memberID uint, from, to time.Time) ([]entity.ClassBookingDetail, error) {
	items := []entity.ClassBookingDetail{}
	err := r.detail(ctx).Where("cb.member_id = ? AND cs.start_at >= ? AND cs.start_at < ?", memberID, from, to).
		Order("cs.start_at, cb.id").Limit(1000).Scan(&items).Error
	return items, err
}

func (r *classBookingRepository) ExistsActive(ctx context.Context, scheduleID, memberID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.ClassBooking{}).
		Where("schedule_id = ? AND member_id = ? AND status IN ('BOOKED', 'WAITLIST', 'ATTENDED')", scheduleID, memberID).
		Count(&n).Error
	return n > 0, err
}

func (r *classBookingRepository) MemberOverlap(ctx context.Context, memberID uint, start, end time.Time, excludeScheduleID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("class_bookings cb").
		Joins("JOIN class_schedules cs ON cs.id = cb.schedule_id").
		Where("cb.member_id = ? AND cb.status IN ('BOOKED', 'WAITLIST') AND cs.status <> 'CANCELLED'", memberID).
		Where("cs.id <> ? AND cs.start_at < ? AND cs.end_at > ?", excludeScheduleID, end, start).
		Count(&n).Error
	return n > 0, err
}

func (r *classBookingRepository) MaxWaitlistPosition(ctx context.Context, scheduleID uint) (int, error) {
	var v int
	err := postgres.Conn(ctx, r.db).Model(&entity.ClassBooking{}).
		Where("schedule_id = ? AND status = 'WAITLIST'", scheduleID).
		Select("COALESCE(MAX(waitlist_position), 0)").Scan(&v).Error
	return v, err
}

func (r *classBookingRepository) FirstWaitlist(ctx context.Context, scheduleID uint) (*entity.ClassBooking, error) {
	var items []entity.ClassBooking
	err := postgres.Conn(ctx, r.db).
		Where("schedule_id = ? AND status = 'WAITLIST'", scheduleID).
		Order("waitlist_position, booked_at, id").Limit(1).Find(&items).Error
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}

func (r *classBookingRepository) UpdateStatus(ctx context.Context, b *entity.ClassBooking) error {
	return postgres.Conn(ctx, r.db).Model(&entity.ClassBooking{}).Where("id = ?", b.ID).
		Updates(map[string]any{
			"status":            b.Status,
			"waitlist_position": b.WaitlistPosition,
			"cancelled_at":      b.CancelledAt,
			"attended_at":       b.AttendedAt,
		}).Error
}

func (r *classBookingRepository) MarkNoShows(ctx context.Context, endedBefore time.Time) ([]uint, error) {
	var ids []uint
	err := postgres.Conn(ctx, r.db).Raw(`
		UPDATE class_bookings cb SET status = 'NO_SHOW'
		FROM class_schedules cs
		WHERE cs.id = cb.schedule_id AND cb.status = 'BOOKED'
		  AND cs.status <> 'CANCELLED' AND cs.end_at < ?
		RETURNING cb.member_id`, endedBefore).Scan(&ids).Error
	return ids, err
}

// DeleteClassReview gỡ đánh giá buổi lớp của hội viên (khi điểm danh sửa ATTENDED → NO_SHOW: không còn đủ điều kiện đánh giá).
func (r *classBookingRepository) DeleteClassReview(ctx context.Context, memberID, scheduleID uint) error {
	return postgres.Conn(ctx, r.db).Exec(
		`DELETE FROM reviews WHERE member_id = ? AND target_type = 'CLASS_SCHEDULE' AND target_id = ?`, memberID, scheduleID).Error
}

func (r *classBookingRepository) CountNoShows(ctx context.Context, memberID uint, since time.Time) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("class_bookings cb").
		Joins("JOIN class_schedules cs ON cs.id = cb.schedule_id").
		Where("cb.member_id = ? AND cb.status = 'NO_SHOW' AND cs.start_at >= ?", memberID, since).
		Count(&n).Error
	return n, err
}

func (r *classBookingRepository) FindMember(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *classBookingRepository) SetMemberBlockedUntil(ctx context.Context, memberID uint, until time.Time) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Member{}).Where("id = ?", memberID).
		UpdateColumn("booking_blocked_until", until).Error
}

func (r *classBookingRepository) ClearMemberBlock(ctx context.Context, memberID uint) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Member{}).Where("id = ?", memberID).
		UpdateColumn("booking_blocked_until", nil).Error
}

func (r *classBookingRepository) ActiveSubscriptions(ctx context.Context, memberID uint) ([]ClassBookingSubscription, error) {
	items := []ClassBookingSubscription{}
	err := postgres.Conn(ctx, r.db).Table("subscriptions s").
		Select(`s.id, s.plan_id, s.branch_id, s.start_date, s.end_date, s.remaining_sessions, s.status,
			p.name AS plan_name, p.type AS plan_type, p.access_scope, p.includes_classes`).
		Joins("JOIN membership_plans p ON p.id = s.plan_id").
		Where("s.member_id = ? AND s.status = 'ACTIVE'", memberID).
		Order("s.end_date, s.id").Scan(&items).Error
	return items, err
}
