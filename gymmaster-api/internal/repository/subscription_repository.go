package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// SubscriptionRow là gói tập kèm thông tin hiển thị.
	SubscriptionRow struct {
		entity.Subscription
		MemberCode    string
		MemberName    string
		MemberPhone   string
		PlanCode      string
		PlanName      string
		PlanType      string
		AccessScope   string
		BranchName    string
		InvoiceNo     string
		InvoiceStatus string
		SoldByName    string
	}

	// SubscriptionListFilter là bộ lọc danh sách gói đã chuẩn hóa ở service. Ngày là cột DATE (00:00 UTC).
	SubscriptionListFilter struct {
		BranchIDs []uint
		MemberID  uint
		PlanID    uint
		Status    string
		// ExpiringFrom/To: chỉ gói ACTIVE/FROZEN có end_date trong [From, To].
		ExpiringFrom *time.Time
		ExpiringTo   *time.Time
		EndFrom      *time.Time
		EndTo        *time.Time
	}

	FreezeRow struct {
		entity.SubscriptionFreeze
		MemberID        uint
		MemberCode      string
		MemberName      string
		PlanName        string
		BranchID        uint
		BranchName      string
		RequestedByName string
		ReviewedByName  string
	}

	FreezeListFilter struct {
		BranchIDs      []uint
		Status         string
		SubscriptionID uint
	}

	SubscriptionEventRow struct {
		entity.SubscriptionEvent
		ActorName string
	}

	// ExpiringSubscription là gói sắp hết hạn (cho nhắc gia hạn).
	ExpiringSubscription struct {
		SubscriptionID uint
		MemberID       uint
		MemberUserID   *uint
		MemberCode     string
		MemberName     string
		MemberPhone    string
		MemberEmail    string
		PlanID         uint
		PlanName       string
		BranchID       uint
		BranchName     string
		EndDate        time.Time
		DaysLeft       int
	}

	SubscriptionRepository interface {
		Create(ctx context.Context, s *entity.Subscription) error
		// FindByID đọc gói; lock = true → SELECT ... FOR UPDATE.
		FindByID(ctx context.Context, id uint, lock bool) (*entity.Subscription, error)
		UpdateFields(ctx context.Context, id uint, fields map[string]any) error
		FindRow(ctx context.Context, id uint) (*SubscriptionRow, error)
		List(ctx context.Context, q dto.Query, f SubscriptionListFilter) ([]SubscriptionRow, int64, error)
		// ListByMember trả mọi gói của hội viên (mới nhất trước); branchIDs nil = không giới hạn.
		ListByMember(ctx context.Context, memberID uint, branchIDs []uint) ([]SubscriptionRow, error)
		// LatestOpen trả gói ACTIVE/FROZEN chưa hết hạn (end_date >= today) có end_date muộn nhất của hội viên
		// cùng phạm vi: cùng access_scope, và cùng chi nhánh nếu SINGLE_BRANCH. nil nếu không có.
		LatestOpen(ctx context.Context, memberID uint, accessScope string, branchID uint, today time.Time) (*entity.Subscription, error)

		CreateEvent(ctx context.Context, e *entity.SubscriptionEvent) error
		ListEvents(ctx context.Context, subscriptionID uint) ([]SubscriptionEventRow, error)
		// LastEvent trả sự kiện mới nhất thuộc một trong các loại types (nil nếu không có).
		LastEvent(ctx context.Context, subscriptionID uint, types ...string) (*entity.SubscriptionEvent, error)

		CreateFreeze(ctx context.Context, f *entity.SubscriptionFreeze) error
		FindFreeze(ctx context.Context, id uint, lock bool) (*entity.SubscriptionFreeze, error)
		UpdateFreezeFields(ctx context.Context, id uint, fields map[string]any) error
		FreezesBySubscription(ctx context.Context, subscriptionID uint) ([]FreezeRow, error)
		FindFreezeRow(ctx context.Context, id uint) (*FreezeRow, error)
		ListFreezes(ctx context.Context, q dto.Query, f FreezeListFilter) ([]FreezeRow, int64, error)
		// CancelOpenFreezes chuyển các yêu cầu PENDING/APPROVED của gói sang trạng thái status.
		CancelOpenFreezes(ctx context.Context, subscriptionID uint, status string) error

		// Truy vấn cho cron.
		IDsToExpire(ctx context.Context, today time.Time) ([]uint, error)
		// StalePending: gói PENDING tạo trước createdBefore mà hóa đơn chưa thu đồng nào.
		StalePending(ctx context.Context, createdBefore time.Time) ([]entity.Subscription, error)
		FreezeIDsToStart(ctx context.Context, today time.Time) ([]uint, error)
		FreezeIDsToEnd(ctx context.Context, today time.Time) ([]uint, error)
		// ListExpiring: gói ACTIVE có end_date thuộc endDates và chưa có gói nối tiếp.
		ListExpiring(ctx context.Context, endDates []time.Time) ([]ExpiringSubscription, error)

		FindPlan(ctx context.Context, id uint) (*entity.MembershipPlan, error)
		FindMember(ctx context.Context, id uint) (*entity.Member, error)
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
	}

	subscriptionRepository struct {
		db *gorm.DB
	}
)

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

const subscriptionRowSelect = `s.*, m.member_code, m.full_name AS member_name, m.phone AS member_phone,
	p.code AS plan_code, p.name AS plan_name, p.type AS plan_type, p.access_scope,
	b.name AS branch_name, COALESCE(i.invoice_no, '') AS invoice_no, COALESCE(i.status, '') AS invoice_status,
	COALESCE(u.full_name, '') AS sold_by_name`

func (r *subscriptionRepository) rowQuery(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("subscriptions s").
		Joins("JOIN members m ON m.id = s.member_id").
		Joins("JOIN membership_plans p ON p.id = s.plan_id").
		Joins("JOIN branches b ON b.id = s.branch_id").
		Joins("LEFT JOIN invoices i ON i.id = s.invoice_id").
		Joins("LEFT JOIN users u ON u.id = s.sold_by")
}

func (r *subscriptionRepository) Create(ctx context.Context, s *entity.Subscription) error {
	return postgres.Conn(ctx, r.db).Create(s).Error
}

func (r *subscriptionRepository) FindByID(ctx context.Context, id uint, lock bool) (*entity.Subscription, error) {
	db := postgres.Conn(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var s entity.Subscription
	if err := db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *subscriptionRepository) UpdateFields(ctx context.Context, id uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Subscription{}).Where("id = ?", id).Updates(fields).Error
}

func (r *subscriptionRepository) FindRow(ctx context.Context, id uint) (*SubscriptionRow, error) {
	var rows []SubscriptionRow
	if err := r.rowQuery(ctx).Select(subscriptionRowSelect).Where("s.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *subscriptionRepository) List(ctx context.Context, q dto.Query, f SubscriptionListFilter) ([]SubscriptionRow, int64, error) {
	tx := r.rowQuery(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("s.branch_id IN ?", f.BranchIDs)
	}
	if f.MemberID > 0 {
		tx = tx.Where("s.member_id = ?", f.MemberID)
	}
	if f.PlanID > 0 {
		tx = tx.Where("s.plan_id = ?", f.PlanID)
	}
	if f.Status != "" {
		tx = tx.Where("s.status = ?", f.Status)
	}
	if f.ExpiringFrom != nil && f.ExpiringTo != nil {
		tx = tx.Where("s.status IN ('ACTIVE','FROZEN') AND s.end_date BETWEEN ? AND ?", *f.ExpiringFrom, *f.ExpiringTo)
	}
	if f.EndFrom != nil {
		tx = tx.Where("s.end_date >= ?", *f.EndFrom)
	}
	if f.EndTo != nil {
		tx = tx.Where("s.end_date <= ?", *f.EndTo)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("m.full_name ILIKE ? OR m.phone LIKE ? OR m.member_code ILIKE ? OR p.name ILIKE ? OR i.invoice_no ILIKE ?", kw, kw, kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []SubscriptionRow
	if err := tx.Select(subscriptionRowSelect).Scopes(q.Paginate).Order("s.id DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *subscriptionRepository) ListByMember(ctx context.Context, memberID uint, branchIDs []uint) ([]SubscriptionRow, error) {
	tx := r.rowQuery(ctx).Select(subscriptionRowSelect).Where("s.member_id = ?", memberID)
	if branchIDs != nil {
		tx = tx.Where("s.branch_id IN ?", branchIDs)
	}
	var rows []SubscriptionRow
	err := tx.Order("s.start_date DESC, s.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *subscriptionRepository) LatestOpen(ctx context.Context, memberID uint, accessScope string, branchID uint, today time.Time) (*entity.Subscription, error) {
	var rows []entity.Subscription
	err := postgres.Conn(ctx, r.db).Table("subscriptions s").Select("s.*").
		Joins("JOIN membership_plans p ON p.id = s.plan_id").
		Where("s.member_id = ? AND s.status IN ('ACTIVE','FROZEN') AND s.end_date >= ?", memberID, today).
		Where("p.access_scope = ?", accessScope).
		Where("p.access_scope = 'ALL_BRANCHES' OR s.branch_id = ?", branchID).
		Order("s.end_date DESC, s.id DESC").Limit(1).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (r *subscriptionRepository) CreateEvent(ctx context.Context, e *entity.SubscriptionEvent) error {
	return postgres.Conn(ctx, r.db).Create(e).Error
}

func (r *subscriptionRepository) ListEvents(ctx context.Context, subscriptionID uint) ([]SubscriptionEventRow, error) {
	var rows []SubscriptionEventRow
	err := postgres.Conn(ctx, r.db).Table("subscription_events e").
		Select("e.*, COALESCE(u.full_name, '') AS actor_name").
		Joins("LEFT JOIN users u ON u.id = e.actor_id").
		Where("e.subscription_id = ?", subscriptionID).Order("e.created_at, e.id").Scan(&rows).Error
	return rows, err
}

func (r *subscriptionRepository) LastEvent(ctx context.Context, subscriptionID uint, types ...string) (*entity.SubscriptionEvent, error) {
	var rows []entity.SubscriptionEvent
	err := postgres.Conn(ctx, r.db).Where("subscription_id = ? AND event_type IN ?", subscriptionID, types).
		Order("id DESC").Limit(1).Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (r *subscriptionRepository) CreateFreeze(ctx context.Context, f *entity.SubscriptionFreeze) error {
	return postgres.Conn(ctx, r.db).Create(f).Error
}

func (r *subscriptionRepository) FindFreeze(ctx context.Context, id uint, lock bool) (*entity.SubscriptionFreeze, error) {
	db := postgres.Conn(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var f entity.SubscriptionFreeze
	if err := db.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *subscriptionRepository) UpdateFreezeFields(ctx context.Context, id uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.SubscriptionFreeze{}).Where("id = ?", id).Updates(fields).Error
}

const freezeRowSelect = `f.*, s.member_id, m.member_code, m.full_name AS member_name, p.name AS plan_name,
	s.branch_id, b.name AS branch_name, COALESCE(ru.full_name, '') AS requested_by_name, COALESCE(vu.full_name, '') AS reviewed_by_name`

func (r *subscriptionRepository) freezeQuery(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("subscription_freezes f").
		Joins("JOIN subscriptions s ON s.id = f.subscription_id").
		Joins("JOIN members m ON m.id = s.member_id").
		Joins("JOIN membership_plans p ON p.id = s.plan_id").
		Joins("JOIN branches b ON b.id = s.branch_id").
		Joins("LEFT JOIN users ru ON ru.id = f.requested_by").
		Joins("LEFT JOIN users vu ON vu.id = f.reviewed_by")
}

func (r *subscriptionRepository) FreezesBySubscription(ctx context.Context, subscriptionID uint) ([]FreezeRow, error) {
	var rows []FreezeRow
	err := r.freezeQuery(ctx).Select(freezeRowSelect).Where("f.subscription_id = ?", subscriptionID).
		Order("f.start_date, f.id").Scan(&rows).Error
	return rows, err
}

func (r *subscriptionRepository) FindFreezeRow(ctx context.Context, id uint) (*FreezeRow, error) {
	var rows []FreezeRow
	if err := r.freezeQuery(ctx).Select(freezeRowSelect).Where("f.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *subscriptionRepository) ListFreezes(ctx context.Context, q dto.Query, f FreezeListFilter) ([]FreezeRow, int64, error) {
	tx := r.freezeQuery(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("s.branch_id IN ?", f.BranchIDs)
	}
	if f.Status != "" {
		tx = tx.Where("f.status = ?", f.Status)
	}
	if f.SubscriptionID > 0 {
		tx = tx.Where("f.subscription_id = ?", f.SubscriptionID)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("m.full_name ILIKE ? OR m.phone LIKE ? OR m.member_code ILIKE ?", kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []FreezeRow
	if err := tx.Select(freezeRowSelect).Scopes(q.Paginate).Order("f.id DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *subscriptionRepository) CancelOpenFreezes(ctx context.Context, subscriptionID uint, status string) error {
	return postgres.Conn(ctx, r.db).Model(&entity.SubscriptionFreeze{}).
		Where("subscription_id = ? AND status IN ('PENDING','APPROVED')", subscriptionID).
		Update("status", status).Error
}

func (r *subscriptionRepository) IDsToExpire(ctx context.Context, today time.Time) ([]uint, error) {
	var ids []uint
	err := postgres.Conn(ctx, r.db).Model(&entity.Subscription{}).
		Where("status IN ('ACTIVE','FROZEN') AND end_date < ?", today).Order("id").Pluck("id", &ids).Error
	return ids, err
}

func (r *subscriptionRepository) StalePending(ctx context.Context, createdBefore time.Time) ([]entity.Subscription, error) {
	var rows []entity.Subscription
	err := postgres.Conn(ctx, r.db).Table("subscriptions s").Select("s.*").
		Joins("LEFT JOIN invoices i ON i.id = s.invoice_id").
		Where("s.status = 'PENDING' AND s.created_at < ?", createdBefore).
		Where("i.id IS NULL OR (i.paid_amount = 0 AND i.status = 'UNPAID')").
		Order("s.id").Scan(&rows).Error
	return rows, err
}

func (r *subscriptionRepository) FreezeIDsToStart(ctx context.Context, today time.Time) ([]uint, error) {
	var ids []uint
	err := postgres.Conn(ctx, r.db).Table("subscription_freezes f").
		Joins("JOIN subscriptions s ON s.id = f.subscription_id").
		Where("f.status = 'APPROVED' AND f.start_date <= ? AND f.end_date >= ? AND s.status = 'ACTIVE'", today, today).
		Order("f.id").Pluck("f.id", &ids).Error
	return ids, err
}

func (r *subscriptionRepository) FreezeIDsToEnd(ctx context.Context, today time.Time) ([]uint, error) {
	var ids []uint
	err := postgres.Conn(ctx, r.db).Table("subscription_freezes f").
		Joins("JOIN subscriptions s ON s.id = f.subscription_id").
		Where("f.status = 'APPROVED' AND f.end_date < ? AND s.status = 'FROZEN'", today).
		Order("f.id").Pluck("f.id", &ids).Error
	return ids, err
}

func (r *subscriptionRepository) ListExpiring(ctx context.Context, endDates []time.Time) ([]ExpiringSubscription, error) {
	if len(endDates) == 0 {
		return nil, nil
	}
	var rows []ExpiringSubscription
	err := postgres.Conn(ctx, r.db).Table("subscriptions s").
		Select(`s.id AS subscription_id, s.member_id, m.user_id AS member_user_id, m.member_code, m.full_name AS member_name,
			m.phone AS member_phone, m.email AS member_email, s.plan_id, p.name AS plan_name, s.branch_id, b.name AS branch_name, s.end_date`).
		Joins("JOIN members m ON m.id = s.member_id AND m.deleted_at IS NULL").
		Joins("JOIN membership_plans p ON p.id = s.plan_id").
		Joins("JOIN branches b ON b.id = s.branch_id").
		Where("s.status = 'ACTIVE' AND s.end_date IN ?", endDates).
		Where(`NOT EXISTS (SELECT 1 FROM subscriptions n WHERE n.member_id = s.member_id AND n.id <> s.id
			AND n.status IN ('PENDING','ACTIVE','FROZEN') AND n.end_date > s.end_date)`).
		Order("s.end_date, s.id").Scan(&rows).Error
	return rows, err
}

func (r *subscriptionRepository) FindPlan(ctx context.Context, id uint) (*entity.MembershipPlan, error) {
	var p entity.MembershipPlan
	if err := postgres.Conn(ctx, r.db).Unscoped().First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *subscriptionRepository) FindMember(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *subscriptionRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}
