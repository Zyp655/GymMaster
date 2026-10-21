package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	MemberContractRow struct {
		entity.MemberContract
		MemberCode   string
		MemberName   string
		MemberPhone  string
		TemplateCode string
		TemplateName string
	}

	DayPassRow struct {
		entity.DayPass
		BranchName  string
		SponsorName string
	}

	// ActiveSubPlan: gói tập đang hiệu lực (ACTIVE, trong khoảng ngày) kèm gói định nghĩa.
	ActiveSubPlan struct {
		Sub  entity.Subscription
		Plan entity.MembershipPlan
	}

	ContractPassRepository interface {
		ListTemplates(ctx context.Context, q dto.Query, status string) ([]entity.ContractTemplate, int64, error)
		FindTemplateByID(ctx context.Context, id uint) (*entity.ContractTemplate, error)
		FindFirstActiveTemplate(ctx context.Context) (*entity.ContractTemplate, error)
		ExistsTemplateCode(ctx context.Context, code string, excludeID uint) (bool, error)
		CreateTemplate(ctx context.Context, t *entity.ContractTemplate) error
		UpdateTemplate(ctx context.Context, t *entity.ContractTemplate) error
		DeleteTemplate(ctx context.Context, id uint) error

		ListMemberContracts(ctx context.Context, q dto.Query, memberID uint, status string) ([]MemberContractRow, int64, error)
		ListContractsByMember(ctx context.Context, memberID uint) ([]MemberContractRow, error)
		FindMemberContractByID(ctx context.Context, id uint) (*MemberContractRow, error)
		CreateMemberContract(ctx context.Context, mc *entity.MemberContract) error
		UpdateMemberContract(ctx context.Context, mc *entity.MemberContract) error

		ListDayPasses(ctx context.Context, q dto.Query, branchIDs []uint, passType string) ([]DayPassRow, int64, error)
		ListSponsorGuestPasses(ctx context.Context, sponsorMemberID uint) ([]DayPassRow, error)
		CountSponsorGuestPassesInMonth(ctx context.Context, sponsorMemberID uint, monthStart, monthEnd time.Time) (int, error)
		FindDayPassByID(ctx context.Context, id uint) (*DayPassRow, error)
		CreateDayPass(ctx context.Context, dp *entity.DayPass) error

		FindMemberByID(ctx context.Context, id uint) (*entity.Member, error)
		// LockMember khóa dòng hội viên (FOR UPDATE) – tuần tự hóa việc dùng hạn mức vé mời; gọi trong transaction.
		LockMember(ctx context.Context, id uint) error
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		FindActiveMemberPlan(ctx context.Context, memberID uint) (*entity.MembershipPlan, *uint, error)
		// ListActiveSubPlans: các gói ACTIVE đang hiệu lực tại ngày today (start_date ≤ today ≤ end_date).
		ListActiveSubPlans(ctx context.Context, memberID uint, today time.Time) ([]ActiveSubPlan, error)
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
	}

	contractPassRepository struct {
		db *gorm.DB
	}
)

func NewContractPassRepository(db *gorm.DB) ContractPassRepository {
	return &contractPassRepository{db: db}
}

// ---- Contract Templates ---------------------------------------------------------

func (r *contractPassRepository) ListTemplates(ctx context.Context, q dto.Query, status string) ([]entity.ContractTemplate, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.ContractTemplate{})
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("code ILIKE ? OR name ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.ContractTemplate
	if err := q.Paginate(tx).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *contractPassRepository) FindTemplateByID(ctx context.Context, id uint) (*entity.ContractTemplate, error) {
	var t entity.ContractTemplate
	if err := postgres.Conn(ctx, r.db).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *contractPassRepository) FindFirstActiveTemplate(ctx context.Context) (*entity.ContractTemplate, error) {
	var t entity.ContractTemplate
	if err := postgres.Conn(ctx, r.db).Where("status = 'ACTIVE'").Order("id ASC").First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *contractPassRepository) ExistsTemplateCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.ContractTemplate{}).Where("UPPER(code) = UPPER(?)", strings.TrimSpace(code))
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var count int64
	err := tx.Count(&count).Error
	return count > 0, err
}

func (r *contractPassRepository) CreateTemplate(ctx context.Context, t *entity.ContractTemplate) error {
	return postgres.Conn(ctx, r.db).Create(t).Error
}

func (r *contractPassRepository) UpdateTemplate(ctx context.Context, t *entity.ContractTemplate) error {
	return postgres.Conn(ctx, r.db).Save(t).Error
}

func (r *contractPassRepository) DeleteTemplate(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.ContractTemplate{}, id).Error
}

// ---- Member Contracts -----------------------------------------------------------

const memberContractSelect = `mc.*, COALESCE(m.member_code, '') AS member_code,
	COALESCE(m.full_name, '') AS member_name, COALESCE(m.phone, '') AS member_phone,
	COALESCE(ct.code, '') AS template_code, COALESCE(ct.name, '') AS template_name`

func (r *contractPassRepository) baseMemberContract(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("member_contracts mc").
		Select(memberContractSelect).
		Joins("LEFT JOIN members m ON m.id = mc.member_id").
		Joins("LEFT JOIN contract_templates ct ON ct.id = mc.template_id")
}

func (r *contractPassRepository) ListMemberContracts(ctx context.Context, q dto.Query, memberID uint, status string) ([]MemberContractRow, int64, error) {
	tx := r.baseMemberContract(ctx)
	if memberID > 0 {
		tx = tx.Where("mc.member_id = ?", memberID)
	}
	if status != "" {
		tx = tx.Where("mc.status = ?", status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("m.full_name ILIKE ? OR m.member_code ILIKE ? OR ct.name ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []MemberContractRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *contractPassRepository) ListContractsByMember(ctx context.Context, memberID uint) ([]MemberContractRow, error) {
	var rows []MemberContractRow
	err := r.baseMemberContract(ctx).Where("mc.member_id = ?", memberID).Order("mc.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *contractPassRepository) FindMemberContractByID(ctx context.Context, id uint) (*MemberContractRow, error) {
	var row MemberContractRow
	res := r.baseMemberContract(ctx).Where("mc.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *contractPassRepository) CreateMemberContract(ctx context.Context, mc *entity.MemberContract) error {
	return postgres.Conn(ctx, r.db).Create(mc).Error
}

func (r *contractPassRepository) UpdateMemberContract(ctx context.Context, mc *entity.MemberContract) error {
	return postgres.Conn(ctx, r.db).Save(mc).Error
}

// ---- Day Passes & Guest Passes --------------------------------------------------

const dayPassSelect = `dp.*, COALESCE(b.name, '') AS branch_name, COALESCE(sm.full_name, '') AS sponsor_name`

func (r *contractPassRepository) baseDayPass(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("day_passes dp").
		Select(dayPassSelect).
		Joins("LEFT JOIN branches b ON b.id = dp.branch_id").
		Joins("LEFT JOIN members sm ON sm.id = dp.sponsor_member_id")
}

func (r *contractPassRepository) ListDayPasses(ctx context.Context, q dto.Query, branchIDs []uint, passType string) ([]DayPassRow, int64, error) {
	tx := r.baseDayPass(ctx)
	if len(branchIDs) > 0 {
		tx = tx.Where("dp.branch_id IN ?", branchIDs)
	}
	if passType != "" {
		tx = tx.Where("dp.type = ?", passType)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		tx = tx.Where("dp.buyer_name ILIKE ? OR dp.phone ILIKE ? OR dp.qr_token ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DayPassRow
	if err := q.Paginate(tx).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *contractPassRepository) ListSponsorGuestPasses(ctx context.Context, sponsorMemberID uint) ([]DayPassRow, error) {
	var rows []DayPassRow
	err := r.baseDayPass(ctx).
		Where("dp.sponsor_member_id = ?", sponsorMemberID).
		Order("dp.created_at DESC, dp.id DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *contractPassRepository) CountSponsorGuestPassesInMonth(ctx context.Context, sponsorMemberID uint, monthStart, monthEnd time.Time) (int, error) {
	var count int64
	err := postgres.Conn(ctx, r.db).Model(&entity.DayPass{}).
		Where("sponsor_member_id = ? AND created_at >= ? AND created_at < ?", sponsorMemberID, monthStart, monthEnd).
		Count(&count).Error
	return int(count), err
}

func (r *contractPassRepository) FindDayPassByID(ctx context.Context, id uint) (*DayPassRow, error) {
	var row DayPassRow
	res := r.baseDayPass(ctx).Where("dp.id = ?", id).Limit(1).Find(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (r *contractPassRepository) CreateDayPass(ctx context.Context, dp *entity.DayPass) error {
	return postgres.Conn(ctx, r.db).Create(dp).Error
}

// ---- Members & Active Plan ------------------------------------------------------

func (r *contractPassRepository) FindMemberByID(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *contractPassRepository) LockMember(ctx context.Context, id uint) error {
	var ids []uint
	return postgres.Conn(ctx, r.db).Raw("SELECT id FROM members WHERE id = ? FOR UPDATE", id).Scan(&ids).Error
}

func (r *contractPassRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *contractPassRepository) FindActiveMemberPlan(ctx context.Context, memberID uint) (*entity.MembershipPlan, *uint, error) {
	conn := postgres.Conn(ctx, r.db)
	var sub entity.Subscription
	if err := conn.Where("member_id = ? AND status IN ('ACTIVE', 'PENDING')", memberID).
		Order("CASE WHEN status = 'ACTIVE' THEN 0 ELSE 1 END, id DESC").
		First(&sub).Error; err != nil {
		return nil, nil, err
	}
	var plan entity.MembershipPlan
	if err := conn.First(&plan, sub.PlanID).Error; err != nil {
		return nil, &sub.ID, err
	}
	return &plan, &sub.ID, nil
}

func (r *contractPassRepository) ListActiveSubPlans(ctx context.Context, memberID uint, today time.Time) ([]ActiveSubPlan, error) {
	conn := postgres.Conn(ctx, r.db)
	var subs []entity.Subscription
	if err := conn.Where("member_id = ? AND status = 'ACTIVE' AND start_date <= ? AND end_date >= ?", memberID, today, today).
		Order("id DESC").Find(&subs).Error; err != nil {
		return nil, err
	}
	out := make([]ActiveSubPlan, 0, len(subs))
	for _, sub := range subs {
		var plan entity.MembershipPlan
		if err := conn.Unscoped().First(&plan, sub.PlanID).Error; err != nil {
			return nil, err
		}
		out = append(out, ActiveSubPlan{Sub: sub, Plan: plan})
	}
	return out, nil
}

func (r *contractPassRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}
