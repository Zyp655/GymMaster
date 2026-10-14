package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	ChallengeRow struct {
		entity.Challenge
		BranchName       string
		ParticipantCount int64
	}

	LeaderboardRow struct {
		MemberID      uint
		MemberCode    string
		FullName      string
		AvatarURL     string
		LoyaltyPoints int
		CheckinCount  int64
	}

	LoyaltyRepository interface {
		ListRewards(ctx context.Context, q dto.Query, status, rewardType string) ([]entity.LoyaltyReward, int64, error)
		ListActiveRewards(ctx context.Context) ([]entity.LoyaltyReward, error)
		FindRewardByID(ctx context.Context, id uint) (*entity.LoyaltyReward, error)
		CreateReward(ctx context.Context, r *entity.LoyaltyReward) error
		UpdateReward(ctx context.Context, r *entity.LoyaltyReward) error
		DeleteReward(ctx context.Context, id uint) error
		DecrementRewardStock(ctx context.Context, id uint) (bool, error)

		ListBadges(ctx context.Context, q dto.Query) ([]entity.Badge, int64, error)
		ListAllBadges(ctx context.Context) ([]entity.Badge, error)
		FindBadgeByID(ctx context.Context, id uint) (*entity.Badge, error)
		ExistsBadgeCode(ctx context.Context, code string) (bool, error)
		CreateBadge(ctx context.Context, b *entity.Badge) error
		DeleteBadge(ctx context.Context, id uint) error
		ListMemberBadges(ctx context.Context, memberID uint) ([]entity.MemberBadge, error)
		AwardBadge(ctx context.Context, mb *entity.MemberBadge) error

		ListChallenges(ctx context.Context, q dto.Query, branchID uint, status string) ([]ChallengeRow, int64, error)
		ListActiveChallenges(ctx context.Context, branchID uint) ([]ChallengeRow, error)
		FindChallengeByID(ctx context.Context, id uint) (*ChallengeRow, error)
		CreateChallenge(ctx context.Context, c *entity.Challenge) error
		UpdateChallenge(ctx context.Context, c *entity.Challenge) error
		DeleteChallenge(ctx context.Context, id uint) error

		FindChallengeParticipant(ctx context.Context, challengeID, memberID uint) (*entity.ChallengeParticipant, error)
		ListMemberParticipants(ctx context.Context, memberID uint) ([]entity.ChallengeParticipant, error)
		SaveChallengeParticipant(ctx context.Context, cp *entity.ChallengeParticipant) error
		CountMemberMetric(ctx context.Context, memberID uint, metric string, from, to *time.Time) (int, error)

		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		AddMemberPoints(ctx context.Context, memberID uint, delta int) (int, error)
		UpdateMemberReferralCode(ctx context.Context, memberID uint, code string) error

		CreateLedger(ctx context.Context, l *entity.LoyaltyLedger) error
		ListMemberLedgers(ctx context.Context, memberID uint, limit int) ([]entity.LoyaltyLedger, error)
		CreatePromotion(ctx context.Context, p *entity.Promotion) error

		TopMembersByPoints(ctx context.Context, limit int) ([]LeaderboardRow, error)
		TopMembersByCheckins(ctx context.Context, limit int) ([]LeaderboardRow, error)
		ListReferredMembers(ctx context.Context, memberID uint) ([]entity.Member, error)
		SumReferralPoints(ctx context.Context, memberID uint) (int, error)
	}

	loyaltyRepository struct {
		db *gorm.DB
	}
)

func NewLoyaltyRepository(db *gorm.DB) LoyaltyRepository {
	return &loyaltyRepository{db: db}
}

func (r *loyaltyRepository) ListRewards(ctx context.Context, q dto.Query, status, rewardType string) ([]entity.LoyaltyReward, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.LoyaltyReward{})
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if rewardType != "" {
		tx = tx.Where("reward_type = ?", rewardType)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.LoyaltyReward
	if err := q.Paginate(tx).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *loyaltyRepository) ListActiveRewards(ctx context.Context) ([]entity.LoyaltyReward, error) {
	var items []entity.LoyaltyReward
	err := postgres.Conn(ctx, r.db).Where("status = 'ACTIVE'").Order("cost_points ASC, id ASC").Find(&items).Error
	return items, err
}

func (r *loyaltyRepository) FindRewardByID(ctx context.Context, id uint) (*entity.LoyaltyReward, error) {
	var item entity.LoyaltyReward
	if err := postgres.Conn(ctx, r.db).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *loyaltyRepository) CreateReward(ctx context.Context, item *entity.LoyaltyReward) error {
	return postgres.Conn(ctx, r.db).Create(item).Error
}

func (r *loyaltyRepository) UpdateReward(ctx context.Context, item *entity.LoyaltyReward) error {
	return postgres.Conn(ctx, r.db).Save(item).Error
}

func (r *loyaltyRepository) DeleteReward(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.LoyaltyReward{}, id).Error
}

func (r *loyaltyRepository) DecrementRewardStock(ctx context.Context, id uint) (bool, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.LoyaltyReward{}).
		Where("id = ? AND stock > 0", id).
		UpdateColumns(map[string]any{
			"stock":      gorm.Expr("stock - 1"),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *loyaltyRepository) ListBadges(ctx context.Context, q dto.Query) ([]entity.Badge, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Badge{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("code ILIKE ? OR name ILIKE ? OR description ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.Badge
	if err := q.Paginate(tx).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *loyaltyRepository) ListAllBadges(ctx context.Context) ([]entity.Badge, error) {
	var items []entity.Badge
	err := postgres.Conn(ctx, r.db).Order("id ASC").Find(&items).Error
	return items, err
}

func (r *loyaltyRepository) FindBadgeByID(ctx context.Context, id uint) (*entity.Badge, error) {
	var b entity.Badge
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *loyaltyRepository) ExistsBadgeCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Badge{}).
		Where("UPPER(code) = UPPER(?)", strings.TrimSpace(code)).Count(&count).Error
	return count > 0, err
}

func (r *loyaltyRepository) CreateBadge(ctx context.Context, b *entity.Badge) error {
	return postgres.Conn(ctx, r.db).Create(b).Error
}

func (r *loyaltyRepository) DeleteBadge(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Badge{}, id).Error
}

func (r *loyaltyRepository) ListMemberBadges(ctx context.Context, memberID uint) ([]entity.MemberBadge, error) {
	var items []entity.MemberBadge
	err := postgres.Conn(ctx, r.db).Where("member_id = ?", memberID).Find(&items).Error
	return items, err
}

func (r *loyaltyRepository) AwardBadge(ctx context.Context, mb *entity.MemberBadge) error {
	return postgres.Conn(ctx, r.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(mb).Error
}

const challengeSelect = `c.*, COALESCE(b.name, '') AS branch_name,
	(SELECT COUNT(*) FROM challenge_participants cp WHERE cp.challenge_id = c.id) AS participant_count`

func (r *loyaltyRepository) baseChallenge(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("challenges c").
		Select(challengeSelect).
		Joins("LEFT JOIN branches b ON b.id = c.branch_id")
}

func (r *loyaltyRepository) ListChallenges(ctx context.Context, q dto.Query, branchID uint, status string) ([]ChallengeRow, int64, error) {
	tx := r.baseChallenge(ctx)
	if branchID > 0 {
		tx = tx.Where("(c.branch_id IS NULL OR c.branch_id = ?)", branchID)
	}
	if status != "" {
		tx = tx.Where("c.status = ?", status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("c.name ILIKE ? OR c.description ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ChallengeRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *loyaltyRepository) ListActiveChallenges(ctx context.Context, branchID uint) ([]ChallengeRow, error) {
	tx := r.baseChallenge(ctx).Where("c.status = 'ACTIVE'")
	if branchID > 0 {
		tx = tx.Where("(c.branch_id IS NULL OR c.branch_id = ?)", branchID)
	}
	var rows []ChallengeRow
	err := tx.Order("c.end_date ASC, c.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *loyaltyRepository) FindChallengeByID(ctx context.Context, id uint) (*ChallengeRow, error) {
	var row ChallengeRow
	res := r.baseChallenge(ctx).Where("c.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *loyaltyRepository) CreateChallenge(ctx context.Context, c *entity.Challenge) error {
	return postgres.Conn(ctx, r.db).Create(c).Error
}

func (r *loyaltyRepository) UpdateChallenge(ctx context.Context, c *entity.Challenge) error {
	return postgres.Conn(ctx, r.db).Save(c).Error
}

func (r *loyaltyRepository) DeleteChallenge(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Challenge{}, id).Error
}

func (r *loyaltyRepository) FindChallengeParticipant(ctx context.Context, challengeID, memberID uint) (*entity.ChallengeParticipant, error) {
	var cp entity.ChallengeParticipant
	if err := postgres.Conn(ctx, r.db).
		Where("challenge_id = ? AND member_id = ?", challengeID, memberID).
		First(&cp).Error; err != nil {
		return nil, err
	}
	return &cp, nil
}

func (r *loyaltyRepository) ListMemberParticipants(ctx context.Context, memberID uint) ([]entity.ChallengeParticipant, error) {
	var items []entity.ChallengeParticipant
	err := postgres.Conn(ctx, r.db).Where("member_id = ?", memberID).Find(&items).Error
	return items, err
}

func (r *loyaltyRepository) SaveChallengeParticipant(ctx context.Context, cp *entity.ChallengeParticipant) error {
	return postgres.Conn(ctx, r.db).Save(cp).Error
}

func (r *loyaltyRepository) CountMemberMetric(ctx context.Context, memberID uint, metric string, from, to *time.Time) (int, error) {
	conn := postgres.Conn(ctx, r.db)
	var count int64
	switch strings.ToUpper(metric) {
	case entity.ChallengeMetricClasses:
		tx := conn.Table("class_bookings cb").
			Joins("JOIN class_schedules cs ON cs.id = cb.schedule_id").
			Where("cb.member_id = ? AND cb.status = 'ATTENDED'", memberID)
		if from != nil {
			tx = tx.Where("cs.start_at >= ?", *from)
		}
		if to != nil {
			tx = tx.Where("cs.start_at < ?", to.AddDate(0, 0, 1))
		}
		if err := tx.Count(&count).Error; err != nil {
			return 0, err
		}
	case entity.ChallengeMetricPTSessions:
		tx := conn.Table("pt_sessions").
			Where("member_id = ? AND status = 'COMPLETED'", memberID)
		if from != nil {
			tx = tx.Where("start_at >= ?", *from)
		}
		if to != nil {
			tx = tx.Where("start_at < ?", to.AddDate(0, 0, 1))
		}
		if err := tx.Count(&count).Error; err != nil {
			return 0, err
		}
	default: // CHECKINS
		tx := conn.Table("checkins").
			Where("member_id = ? AND result = 'ALLOWED'", memberID)
		if from != nil {
			tx = tx.Where("checked_in_at >= ?", *from)
		}
		if to != nil {
			tx = tx.Where("checked_in_at < ?", to.AddDate(0, 0, 1))
		}
		if err := tx.Count(&count).Error; err != nil {
			return 0, err
		}
	}
	return int(count), nil
}

func (r *loyaltyRepository) FindMemberByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *loyaltyRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *loyaltyRepository) AddMemberPoints(ctx context.Context, memberID uint, delta int) (int, error) {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Model(&entity.Member{}).
		Where("id = ?", memberID).
		UpdateColumns(map[string]any{
			"loyalty_points": gorm.Expr("loyalty_points + ?", delta),
			"updated_at":     time.Now(),
		}).Error; err != nil {
		return 0, err
	}
	var m entity.Member
	if err := conn.Select("id, loyalty_points").First(&m, memberID).Error; err != nil {
		return 0, err
	}
	return m.LoyaltyPoints, nil
}

func (r *loyaltyRepository) UpdateMemberReferralCode(ctx context.Context, memberID uint, code string) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Member{}).
		Where("id = ?", memberID).
		UpdateColumn("referral_code", code).Error
}

func (r *loyaltyRepository) CreateLedger(ctx context.Context, l *entity.LoyaltyLedger) error {
	return postgres.Conn(ctx, r.db).Create(l).Error
}

func (r *loyaltyRepository) ListMemberLedgers(ctx context.Context, memberID uint, limit int) ([]entity.LoyaltyLedger, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []entity.LoyaltyLedger
	err := postgres.Conn(ctx, r.db).
		Where("member_id = ?", memberID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (r *loyaltyRepository) CreatePromotion(ctx context.Context, p *entity.Promotion) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *loyaltyRepository) TopMembersByPoints(ctx context.Context, limit int) ([]LeaderboardRow, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows []LeaderboardRow
	err := postgres.Conn(ctx, r.db).Table("members m").
		Select(`m.id AS member_id, m.member_code, m.full_name, m.avatar_url, m.loyalty_points,
			(SELECT COUNT(*) FROM checkins ck WHERE ck.member_id = m.id AND ck.result = 'ALLOWED') AS checkin_count`).
		Where("m.deleted_at IS NULL AND m.status = 'ACTIVE'").
		Order("m.loyalty_points DESC, checkin_count DESC, m.id ASC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *loyaltyRepository) TopMembersByCheckins(ctx context.Context, limit int) ([]LeaderboardRow, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows []LeaderboardRow
	err := postgres.Conn(ctx, r.db).Table("members m").
		Select(`m.id AS member_id, m.member_code, m.full_name, m.avatar_url, m.loyalty_points,
			(SELECT COUNT(*) FROM checkins ck WHERE ck.member_id = m.id AND ck.result = 'ALLOWED') AS checkin_count`).
		Where("m.deleted_at IS NULL AND m.status = 'ACTIVE'").
		Order("checkin_count DESC, m.loyalty_points DESC, m.id ASC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *loyaltyRepository) ListReferredMembers(ctx context.Context, memberID uint) ([]entity.Member, error) {
	var items []entity.Member
	err := postgres.Conn(ctx, r.db).
		Where("referred_by = ? AND deleted_at IS NULL", memberID).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

func (r *loyaltyRepository) SumReferralPoints(ctx context.Context, memberID uint) (int, error) {
	var sum int
	err := postgres.Conn(ctx, r.db).Table("loyalty_ledgers").
		Select("COALESCE(SUM(points), 0)").
		Where("member_id = ? AND reason = ?", memberID, entity.LoyaltyReasonReferral).
		Scan(&sum).Error
	return sum, err
}
