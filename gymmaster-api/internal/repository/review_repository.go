package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	ReviewRow struct {
		entity.Review
		MemberCode  string
		MemberName  string
		TrainerName string
		BranchName  string
	}

	ReviewFilter struct {
		BranchIDs  []uint
		TrainerID  uint
		TargetType string
		Rating     int
	}

	ReviewRepository interface {
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		FindPTSession(ctx context.Context, id uint) (*entity.PTSession, error)
		FindClassSchedule(ctx context.Context, id uint) (*entity.ClassSchedule, error)
		FindMemberClassBooking(ctx context.Context, memberID, scheduleID uint) (*entity.ClassBooking, error)
		ExistsByMemberTarget(ctx context.Context, memberID uint, targetType string, targetID uint) (bool, error)
		Create(ctx context.Context, rev *entity.Review) error
		UpdateTrainerRating(ctx context.Context, trainerID uint) error
		ListByMember(ctx context.Context, memberID uint) ([]ReviewRow, error)
		List(ctx context.Context, q dto.Query, f ReviewFilter) ([]ReviewRow, int64, error)
	}

	reviewRepository struct {
		db *gorm.DB
	}
)

func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

const reviewRowSelect = `r.*, COALESCE(m.member_code, '') AS member_code,
	COALESCE(m.full_name, '') AS member_name, COALESCE(u.full_name, '') AS trainer_name,
	COALESCE(b.name, '') AS branch_name`

const reviewRowJoins = `LEFT JOIN members m ON m.id = r.member_id
	LEFT JOIN users u ON u.id = r.trainer_id
	LEFT JOIN branches b ON b.id = r.branch_id`

func (r *reviewRepository) base(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("reviews r").Select(reviewRowSelect).Joins(reviewRowJoins)
}

func (r *reviewRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *reviewRepository) FindPTSession(ctx context.Context, id uint) (*entity.PTSession, error) {
	var s entity.PTSession
	if err := postgres.Conn(ctx, r.db).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *reviewRepository) FindClassSchedule(ctx context.Context, id uint) (*entity.ClassSchedule, error) {
	var s entity.ClassSchedule
	if err := postgres.Conn(ctx, r.db).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *reviewRepository) FindMemberClassBooking(ctx context.Context, memberID, scheduleID uint) (*entity.ClassBooking, error) {
	var b entity.ClassBooking
	if err := postgres.Conn(ctx, r.db).
		Where("member_id = ? AND schedule_id = ? AND status = ?", memberID, scheduleID, entity.BookingStatusAttended).
		Order("id DESC").
		First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *reviewRepository) ExistsByMemberTarget(ctx context.Context, memberID uint, targetType string, targetID uint) (bool, error) {
	var count int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Review{}).
		Where("member_id = ? AND target_type = ? AND target_id = ?", memberID, targetType, targetID).
		Count(&count).Error
	return count > 0, err
}

func (r *reviewRepository) Create(ctx context.Context, rev *entity.Review) error {
	return postgres.Conn(ctx, r.db).Create(rev).Error
}

func (r *reviewRepository) UpdateTrainerRating(ctx context.Context, trainerID uint) error {
	var stats struct {
		Avg   float64
		Count int
	}
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Table("reviews").
		Select("COALESCE(ROUND(AVG(rating)::numeric, 2), 0) AS avg, COUNT(*) AS count").
		Where("trainer_id = ?", trainerID).
		Scan(&stats).Error; err != nil {
		return err
	}
	return conn.Model(&entity.TrainerProfile{}).
		Where("user_id = ?", trainerID).
		Updates(map[string]any{
			"rating_avg":   stats.Avg,
			"rating_count": stats.Count,
		}).Error
}

func (r *reviewRepository) ListByMember(ctx context.Context, memberID uint) ([]ReviewRow, error) {
	var rows []ReviewRow
	err := r.base(ctx).Where("r.member_id = ?", memberID).Order("r.created_at DESC, r.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *reviewRepository) List(ctx context.Context, q dto.Query, f ReviewFilter) ([]ReviewRow, int64, error) {
	tx := r.base(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("r.branch_id IN ?", f.BranchIDs)
	}
	if f.TrainerID > 0 {
		tx = tx.Where("r.trainer_id = ?", f.TrainerID)
	}
	if f.TargetType != "" {
		tx = tx.Where("r.target_type = ?", f.TargetType)
	}
	if f.Rating > 0 {
		tx = tx.Where("r.rating = ?", f.Rating)
	}
	if q.Keyword != "" {
		like := "%" + strings.TrimSpace(q.Keyword) + "%"
		tx = tx.Where("r.comment ILIKE ? OR m.full_name ILIKE ? OR u.full_name ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ReviewRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
