package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// PortalSubscriptionPlan: hạn mức của gói tập (bảo lưu, lớp nhóm, phạm vi) – cổng hội viên.
type PortalSubscriptionPlan struct {
	SubscriptionID  uint
	BranchID        uint
	Status          string
	StartDate       time.Time
	EndDate         time.Time
	AccessScope     string
	IncludesClasses bool
	MaxFreezeDays   int
	MaxFreezeTimes  int
}

// PortalFreezeRow: yêu cầu bảo lưu thuộc các gói của hội viên.
type PortalFreezeRow struct {
	ID             uint
	SubscriptionID uint
	StartDate      time.Time
	EndDate        time.Time
	Days           int
	Reason         string
	Status         string
	ReviewNote     string
	ReviewedAt     *time.Time
	CreatedAt      time.Time
}

// PortalBusy: một khoảng bận (buổi PT / lớp) dùng để tính khung giờ trống.
type PortalBusy struct {
	StartAt time.Time
	EndAt   time.Time
}

// PortalContract: hợp đồng PT kèm tên HLV / chi nhánh và giờ mở cửa.
type PortalContract struct {
	entity.PTContract
	TrainerName string
	BranchName  string
	OpenTime    string
	CloseTime   string
}

type (
	// PortalRepository: truy vấn riêng của cổng hội viên (mọi hàm nhận memberID đã xác định từ tài khoản đăng nhập).
	PortalRepository interface {
		// FindMemberByUserID: hội viên (chưa xóa) gắn với tài khoản cổng.
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
		// ActiveBranches: chi nhánh ACTIVE (chưa xóa), theo id.
		ActiveBranches(ctx context.Context) ([]entity.Branch, error)
		// SubscriptionPlans: hạn mức gói của mọi gói tập của hội viên.
		SubscriptionPlans(ctx context.Context, memberID uint) ([]PortalSubscriptionPlan, error)
		// Freezes: yêu cầu bảo lưu của mọi gói của hội viên (mới nhất trước).
		Freezes(ctx context.Context, memberID uint) ([]PortalFreezeRow, error)
		// UpdateProfile cập nhật các cột hồ sơ hội viên được phép tự sửa.
		UpdateProfile(ctx context.Context, memberID uint, fields map[string]any) error
		// FindContract: hợp đồng PT (không khóa) kèm HLV, chi nhánh, giờ mở cửa.
		FindContract(ctx context.Context, id uint) (*PortalContract, error)
		// TrainerBusy: buổi PT (SCHEDULED/COMPLETED) + lớp chưa hủy của HLV giao với [from, to).
		TrainerBusy(ctx context.Context, trainerID uint, from, to time.Time) ([]PortalBusy, error)
		// MemberBusy: buổi PT (SCHEDULED/COMPLETED) + lớp đã đặt (BOOKED/WAITLIST/ATTENDED, lớp chưa hủy) của hội viên giao với [from, to).
		MemberBusy(ctx context.Context, memberID uint, from, to time.Time) ([]PortalBusy, error)
	}

	portalRepository struct {
		db *gorm.DB
	}
)

func NewPortalRepository(db *gorm.DB) PortalRepository {
	return &portalRepository{db: db}
}

func (r *portalRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).Order("id").Take(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *portalRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).Take(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *portalRepository) ActiveBranches(ctx context.Context) ([]entity.Branch, error) {
	items := []entity.Branch{}
	err := postgres.Conn(ctx, r.db).Where("status = ?", "ACTIVE").Order("id").Limit(200).Find(&items).Error
	return items, err
}

func (r *portalRepository) SubscriptionPlans(ctx context.Context, memberID uint) ([]PortalSubscriptionPlan, error) {
	items := []PortalSubscriptionPlan{}
	err := postgres.Conn(ctx, r.db).Table("subscriptions s").
		Select(`s.id AS subscription_id, s.branch_id, s.status, s.start_date, s.end_date,
			p.access_scope, p.includes_classes, p.max_freeze_days, p.max_freeze_times`).
		Joins("JOIN membership_plans p ON p.id = s.plan_id").
		Where("s.member_id = ?", memberID).
		Order("s.id DESC").Scan(&items).Error
	return items, err
}

func (r *portalRepository) Freezes(ctx context.Context, memberID uint) ([]PortalFreezeRow, error) {
	items := []PortalFreezeRow{}
	err := postgres.Conn(ctx, r.db).Table("subscription_freezes f").
		Select(`f.id, f.subscription_id, f.start_date, f.end_date, f.days, f.reason, f.status,
			f.review_note, f.reviewed_at, f.created_at`).
		Joins("JOIN subscriptions s ON s.id = f.subscription_id").
		Where("s.member_id = ?", memberID).
		Order("f.id DESC").Scan(&items).Error
	return items, err
}

func (r *portalRepository) UpdateProfile(ctx context.Context, memberID uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Member{}).Where("id = ?", memberID).Updates(fields).Error
}

func (r *portalRepository) FindContract(ctx context.Context, id uint) (*PortalContract, error) {
	var c PortalContract
	err := postgres.Conn(ctx, r.db).Table("pt_contracts c").
		Select(`c.*, COALESCE(u.full_name, '') AS trainer_name, COALESCE(b.name, '') AS branch_name,
			COALESCE(b.open_time, '') AS open_time, COALESCE(b.close_time, '') AS close_time`).
		Joins("LEFT JOIN users u ON u.id = c.trainer_id").
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Where("c.id = ?", id).Take(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *portalRepository) TrainerBusy(ctx context.Context, trainerID uint, from, to time.Time) ([]PortalBusy, error) {
	items := []PortalBusy{}
	err := postgres.Conn(ctx, r.db).Raw(`
		SELECT start_at, end_at FROM pt_sessions
		 WHERE trainer_id = ? AND status IN ('SCHEDULED', 'COMPLETED') AND start_at < ? AND end_at > ?
		UNION ALL
		SELECT start_at, end_at FROM class_schedules
		 WHERE trainer_id = ? AND status <> 'CANCELLED' AND start_at < ? AND end_at > ?
		ORDER BY start_at`, trainerID, to, from, trainerID, to, from).Scan(&items).Error
	return items, err
}

func (r *portalRepository) MemberBusy(ctx context.Context, memberID uint, from, to time.Time) ([]PortalBusy, error) {
	items := []PortalBusy{}
	err := postgres.Conn(ctx, r.db).Raw(`
		SELECT start_at, end_at FROM pt_sessions
		 WHERE member_id = ? AND status IN ('SCHEDULED', 'COMPLETED') AND start_at < ? AND end_at > ?
		UNION ALL
		SELECT cs.start_at, cs.end_at FROM class_bookings cb JOIN class_schedules cs ON cs.id = cb.schedule_id
		 WHERE cb.member_id = ? AND cb.status IN ('BOOKED', 'WAITLIST', 'ATTENDED') AND cs.status <> 'CANCELLED'
		   AND cs.start_at < ? AND cs.end_at > ?
		ORDER BY start_at`, memberID, to, from, memberID, to, from).Scan(&items).Error
	return items, err
}
