package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	LeadRow struct {
		entity.Lead
		BranchName       string
		InterestPlanName string
		AssignedName     string
		MemberCode       string
	}

	LeadFilter struct {
		BranchIDs  []uint
		Status     string
		Source     string
		AssignedTo uint
		Search     string
	}

	LeadActivityRow struct {
		entity.LeadActivity
		CreatorName string
	}

	TrialPassRow struct {
		entity.TrialPass
		BranchName string
	}

	PublicScheduleRow struct {
		entity.ClassSchedule
		ClassTypeCode  string
		ClassTypeName  string
		ClassTypeColor string
		Level          string
		BranchName     string
		RoomName       string
		TrainerName    string
	}

	AtRiskMemberRow struct {
		MemberID      uint
		MemberCode    string
		FullName      string
		Phone         string
		Email         string
		BranchID      uint
		BranchName    string
		PlanName      string
		EndDate       time.Time
		LastCheckinAt *time.Time
		DaysInactive  int
	}

	LeadCountRow struct {
		Key   string
		Count int64
	}

	LeadsReportData struct {
		Total     int64
		Converted int64
		ByStatus  []LeadCountRow
		BySource  []LeadCountRow
	}

	LeadRepository interface {
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
		ListLeads(ctx context.Context, q dto.Query, f LeadFilter) ([]LeadRow, int64, error)
		FindLeadByID(ctx context.Context, id uint) (*LeadRow, error)
		CreateLead(ctx context.Context, l *entity.Lead) error
		UpdateLead(ctx context.Context, l *entity.Lead) error
		DeleteLead(ctx context.Context, id uint, deletedBy *uint) error

		ListLeadActivities(ctx context.Context, leadID uint) ([]LeadActivityRow, error)
		CreateLeadActivity(ctx context.Context, a *entity.LeadActivity) error

		ListTrialPasses(ctx context.Context, leadID uint) ([]TrialPassRow, error)
		CreateTrialPass(ctx context.Context, tp *entity.TrialPass) error

		FindMemberByPhone(ctx context.Context, phone string) (*entity.Member, error)
		CreateMember(ctx context.Context, m *entity.Member) error

		ListPublicSchedules(ctx context.Context, branchID uint, from, to time.Time) ([]PublicScheduleRow, error)
		ListAtRiskMembers(ctx context.Context, branchIDs []uint, today time.Time, daysInactive int) ([]AtRiskMemberRow, error)
		LeadsReport(ctx context.Context, branchIDs []uint, from, to *time.Time) (*LeadsReportData, error)

		ListBirthdayMembers(ctx context.Context, today time.Time) ([]entity.Member, error)
		EnsureBirthdayPromotion(ctx context.Context, p *entity.Promotion) error
	}

	leadRepository struct {
		db *gorm.DB
	}
)

func NewLeadRepository(db *gorm.DB) LeadRepository {
	return &leadRepository{db: db}
}

func (r *leadRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

const leadRowSelect = `l.*, COALESCE(b.name, '') AS branch_name, COALESCE(mp.name, '') AS interest_plan_name,
	COALESCE(u.full_name, '') AS assigned_name, COALESCE(m.member_code, '') AS member_code`

const leadRowJoins = `LEFT JOIN branches b ON b.id = l.branch_id
	LEFT JOIN membership_plans mp ON mp.id = l.interest_plan_id
	LEFT JOIN users u ON u.id = l.assigned_to
	LEFT JOIN members m ON m.id = l.member_id`

func (r *leadRepository) baseLead(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("leads l").
		Select(leadRowSelect).
		Joins(leadRowJoins).
		Where("l.deleted_at IS NULL")
}

func (r *leadRepository) ListLeads(ctx context.Context, q dto.Query, f LeadFilter) ([]LeadRow, int64, error) {
	tx := r.baseLead(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("l.branch_id IN ?", f.BranchIDs)
	}
	if f.Status != "" {
		tx = tx.Where("l.status = ?", f.Status)
	}
	if f.Source != "" {
		tx = tx.Where("l.source = ?", f.Source)
	}
	if f.AssignedTo > 0 {
		tx = tx.Where("l.assigned_to = ?", f.AssignedTo)
	}
	kw := strings.TrimSpace(f.Search)
	if kw == "" {
		kw = q.Keyword
	}
	if kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("l.full_name ILIKE ? OR l.phone ILIKE ? OR l.email ILIKE ? OR l.note ILIKE ?", like, like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []LeadRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *leadRepository) FindLeadByID(ctx context.Context, id uint) (*LeadRow, error) {
	var row LeadRow
	res := r.baseLead(ctx).Where("l.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *leadRepository) CreateLead(ctx context.Context, l *entity.Lead) error {
	return postgres.Conn(ctx, r.db).Create(l).Error
}

func (r *leadRepository) UpdateLead(ctx context.Context, l *entity.Lead) error {
	return postgres.Conn(ctx, r.db).Save(l).Error
}

func (r *leadRepository) DeleteLead(ctx context.Context, id uint, deletedBy *uint) error {
	now := time.Now()
	return postgres.Conn(ctx, r.db).Model(&entity.Lead{}).Where("id = ?", id).
		Updates(map[string]any{"deleted_at": now, "updated_by": deletedBy, "updated_at": now}).Error
}

func (r *leadRepository) ListLeadActivities(ctx context.Context, leadID uint) ([]LeadActivityRow, error) {
	var rows []LeadActivityRow
	err := postgres.Conn(ctx, r.db).Table("lead_activities la").
		Select("la.*, COALESCE(u.full_name, '') AS creator_name").
		Joins("LEFT JOIN users u ON u.id = la.created_by").
		Where("la.lead_id = ?", leadID).
		Order("la.created_at DESC, la.id DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *leadRepository) CreateLeadActivity(ctx context.Context, a *entity.LeadActivity) error {
	return postgres.Conn(ctx, r.db).Create(a).Error
}

func (r *leadRepository) ListTrialPasses(ctx context.Context, leadID uint) ([]TrialPassRow, error) {
	var rows []TrialPassRow
	err := postgres.Conn(ctx, r.db).Table("trial_passes tp").
		Select("tp.*, COALESCE(b.name, '') AS branch_name").
		Joins("LEFT JOIN branches b ON b.id = tp.branch_id").
		Where("tp.lead_id = ?", leadID).
		Order("tp.created_at DESC, tp.id DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *leadRepository) CreateTrialPass(ctx context.Context, tp *entity.TrialPass) error {
	return postgres.Conn(ctx, r.db).Create(tp).Error
}

func (r *leadRepository) FindMemberByPhone(ctx context.Context, phone string) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("phone = ?", phone).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *leadRepository) CreateMember(ctx context.Context, m *entity.Member) error {
	return postgres.Conn(ctx, r.db).Create(m).Error
}

func (r *leadRepository) ListPublicSchedules(ctx context.Context, branchID uint, from, to time.Time) ([]PublicScheduleRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("class_schedules cs").
		Select(`cs.*, ct.code AS class_type_code, ct.name AS class_type_name, ct.color AS class_type_color,
			ct.level AS level, b.name AS branch_name, rm.name AS room_name, u.full_name AS trainer_name`).
		Joins("JOIN class_types ct ON ct.id = cs.class_type_id AND ct.deleted_at IS NULL").
		Joins("JOIN branches b ON b.id = cs.branch_id AND b.deleted_at IS NULL").
		Joins("JOIN rooms rm ON rm.id = cs.room_id AND rm.deleted_at IS NULL").
		Joins("JOIN users u ON u.id = cs.trainer_id").
		Where("cs.status = ? AND cs.start_at >= ? AND cs.start_at < ?", entity.ClassScheduleScheduled, from, to)
	if branchID > 0 {
		tx = tx.Where("cs.branch_id = ?", branchID)
	}
	var rows []PublicScheduleRow
	err := tx.Order("cs.start_at ASC, cs.id ASC").Scan(&rows).Error
	return rows, err
}

func (r *leadRepository) ListAtRiskMembers(ctx context.Context, branchIDs []uint, today time.Time, daysInactive int) ([]AtRiskMemberRow, error) {
	todayStr := today.Format(consts.DateLayout)
	cutoff := today.AddDate(0, 0, -daysInactive)
	tx := postgres.Conn(ctx, r.db).Table("members m").
		Select(`m.id AS member_id, m.member_code, m.full_name, m.phone, m.email,
			sub.branch_id, COALESCE(b.name, '') AS branch_name, COALESCE(mp.name, '') AS plan_name,
			sub.end_date, lc.last_checkin_at,
			EXTRACT(DAY FROM (?::timestamptz - COALESCE(lc.last_checkin_at, sub.start_date::timestamptz)))::int AS days_inactive`, today).
		Joins(`JOIN LATERAL (
			SELECT s.id, s.plan_id, s.branch_id, s.start_date, s.end_date
			FROM subscriptions s
			WHERE s.member_id = m.id AND s.status = 'ACTIVE' AND s.start_date <= ? AND s.end_date >= ?
			ORDER BY s.end_date DESC, s.id DESC
			LIMIT 1
		) sub ON true`, todayStr, todayStr).
		Joins("LEFT JOIN branches b ON b.id = sub.branch_id").
		Joins("LEFT JOIN membership_plans mp ON mp.id = sub.plan_id").
		Joins(`LEFT JOIN LATERAL (
			SELECT MAX(c.checked_in_at) AS last_checkin_at
			FROM checkins c
			WHERE c.member_id = m.id AND c.result = 'ALLOWED'
		) lc ON true`).
		Where("m.deleted_at IS NULL AND m.status = ?", entity.MemberStatusActive).
		Where("(lc.last_checkin_at IS NULL AND sub.start_date <= ?) OR (lc.last_checkin_at IS NOT NULL AND lc.last_checkin_at <= ?)",
			cutoff.Format(consts.DateLayout), cutoff)

	if branchIDs != nil {
		tx = tx.Where("sub.branch_id IN ?", branchIDs)
	}
	var rows []AtRiskMemberRow
	err := tx.Order("days_inactive DESC, m.id ASC").Scan(&rows).Error
	return rows, err
}

func (r *leadRepository) LeadsReport(ctx context.Context, branchIDs []uint, from, to *time.Time) (*LeadsReportData, error) {
	base := func() *gorm.DB {
		tx := postgres.Conn(ctx, r.db).Table("leads l").Where("l.deleted_at IS NULL")
		if branchIDs != nil {
			tx = tx.Where("l.branch_id IN ?", branchIDs)
		}
		if from != nil {
			tx = tx.Where("l.created_at >= ?", *from)
		}
		if to != nil {
			tx = tx.Where("l.created_at < ?", *to)
		}
		return tx
	}

	out := &LeadsReportData{}
	if err := base().Count(&out.Total).Error; err != nil {
		return nil, err
	}
	if err := base().Where("l.status = ?", entity.LeadStatusConverted).Count(&out.Converted).Error; err != nil {
		return nil, err
	}
	if err := base().Select("l.status AS key, COUNT(*) AS count").Group("l.status").Order("l.status ASC").Scan(&out.ByStatus).Error; err != nil {
		return nil, err
	}
	if err := base().Select("l.source AS key, COUNT(*) AS count").Group("l.source").Order("l.source ASC").Scan(&out.BySource).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *leadRepository) ListBirthdayMembers(ctx context.Context, today time.Time) ([]entity.Member, error) {
	var rows []entity.Member
	err := postgres.Conn(ctx, r.db).Model(&entity.Member{}).
		Where("status = ? AND dob IS NOT NULL AND EXTRACT(MONTH FROM dob) = ? AND EXTRACT(DAY FROM dob) = ?",
			entity.MemberStatusActive, int(today.Month()), today.Day()).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *leadRepository) EnsureBirthdayPromotion(ctx context.Context, p *entity.Promotion) error {
	conn := postgres.Conn(ctx, r.db)
	var count int64
	if err := conn.Model(&entity.Promotion{}).Where("UPPER(code) = UPPER(?)", p.Code).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return conn.Create(p).Error
}
