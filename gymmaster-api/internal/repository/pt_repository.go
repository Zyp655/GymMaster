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
	// PTContractRow là hợp đồng PT kèm thông tin hiển thị.
	PTContractRow struct {
		entity.PTContract
		MemberCode        string
		MemberName        string
		MemberPhone       string
		TrainerName       string
		PackageCode       string
		PackageName       string
		BranchName        string
		InvoiceNo         string
		InvoiceStatus     string
		SoldByName        string
		ScheduledSessions int
	}

	PTContractFilter struct {
		BranchIDs []uint // nil = không giới hạn
		TrainerID uint
		MemberID  uint
		PackageID uint
		Status    string
	}

	// PTSessionRow là buổi PT kèm thông tin hiển thị.
	PTSessionRow struct {
		entity.PTSession
		TrainerName   string
		MemberCode    string
		MemberName    string
		MemberPhone   string
		MemberUserID  *uint
		BranchName    string
		PackageName   string
		CreatedByName string
	}

	// PTSessionFilter: buổi có start_at trong [From, To).
	PTSessionFilter struct {
		BranchIDs  []uint
		TrainerID  uint
		MemberID   uint
		ContractID uint
		Status     string
		From       time.Time
		To         time.Time
	}

	// PTClassSlot là buổi lớp nhóm trong lịch dạy của HLV.
	PTClassSlot struct {
		ID            uint
		BranchID      uint
		BranchName    string
		TrainerID     uint
		TrainerName   string
		ClassTypeName string
		Color         string
		RoomName      string
		Status        string
		StartAt       time.Time
		EndAt         time.Time
		BookedCount   int
		Capacity      int
	}

	PTCommissionRow struct {
		entity.TrainerCommission
		TrainerName    string
		BranchName     string
		PaidByName     string
		MemberName     string
		SessionStartAt *time.Time
		ContractID     *uint
	}

	PTCommissionFilter struct {
		BranchIDs  []uint
		TrainerID  uint
		Period     string
		Status     string
		SourceType string
	}

	PTCommissionSummaryRow struct {
		TrainerID     uint
		TrainerName   string
		StaffCode     string
		Sessions      int
		BaseTotal     float64
		TotalAmount   float64
		PaidAmount    float64
		PendingAmount float64
	}

	// PTRepository: hợp đồng PT, buổi PT, hoa hồng HLV và các tra cứu liên quan.
	PTRepository interface {
		// ---- Tra cứu ----
		FindMember(ctx context.Context, id uint) (*entity.Member, error)
		// FindPackage đọc cả gói đã xóa mềm (service tự kiểm tra DeletedAt khi bán).
		FindPackage(ctx context.Context, id uint) (*entity.PTPackage, error)
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)
		// TrainerInBranch: user ACTIVE, là HLV (hồ sơ TRAINER hoặc vai trò TRAINER),
		// có vai trò tại chi nhánh (hoặc vai trò toàn hệ thống).
		TrainerInBranch(ctx context.Context, trainerID, branchID uint) (bool, error)
		// CommissionRate: staff_profiles.commission_rate của HLV (0 nếu chưa có hồ sơ).
		CommissionRate(ctx context.Context, trainerID uint) (float64, error)
		UserName(ctx context.Context, id uint) (string, error)

		// ---- Hợp đồng ----
		CreateContract(ctx context.Context, c *entity.PTContract) error
		// FindContract: lock = true → SELECT ... FOR UPDATE.
		FindContract(ctx context.Context, id uint, lock bool) (*entity.PTContract, error)
		UpdateContractFields(ctx context.Context, id uint, fields map[string]any) error
		FindContractRow(ctx context.Context, id uint) (*PTContractRow, error)
		ListContracts(ctx context.Context, q dto.Query, f PTContractFilter) ([]PTContractRow, int64, error)
		// ListContractsByMember: mới nhất trước; branchIDs nil = không giới hạn.
		ListContractsByMember(ctx context.Context, memberID uint, branchIDs []uint) ([]PTContractRow, error)
		CountScheduled(ctx context.Context, contractID uint) (int, error)
		// ContractsToExpire: hợp đồng ACTIVE có end_date < today.
		ContractsToExpire(ctx context.Context, today time.Time) ([]entity.PTContract, error)

		// ---- Buổi tập ----
		CreateSession(ctx context.Context, s *entity.PTSession) error
		FindSession(ctx context.Context, id uint, lock bool) (*entity.PTSession, error)
		UpdateSessionFields(ctx context.Context, id uint, fields map[string]any) error
		FindSessionRow(ctx context.Context, id uint) (*PTSessionRow, error)
		ListSessions(ctx context.Context, f PTSessionFilter) ([]PTSessionRow, error)
		SessionsByContract(ctx context.Context, contractID uint) ([]PTSessionRow, error)
		// ScheduledSessions: buổi SCHEDULED của hợp đồng có start_at > after (nil = mọi buổi), sớm nhất trước.
		ScheduledSessions(ctx context.Context, contractID uint, after *time.Time) ([]entity.PTSession, error)
		// OverdueSessions: buổi SCHEDULED có end_at < before.
		OverdueSessions(ctx context.Context, before time.Time) ([]PTSessionRow, error)

		// Kiểm tra trùng lịch (khoảng [start, end), buổi đã hủy không tính).
		TrainerPTOverlap(ctx context.Context, trainerID uint, start, end time.Time, excludeID uint) (bool, error)
		TrainerClassOverlap(ctx context.Context, trainerID uint, start, end time.Time) (bool, error)
		MemberPTOverlap(ctx context.Context, memberID uint, start, end time.Time, excludeID uint) (bool, error)
		MemberClassOverlap(ctx context.Context, memberID uint, start, end time.Time) (bool, error)
		// TrainerClasses: buổi lớp nhóm (chưa hủy) của HLV có start_at trong [from, to).
		TrainerClasses(ctx context.Context, trainerID uint, from, to time.Time) ([]PTClassSlot, error)

		// ---- Hoa hồng ----
		CreateCommission(ctx context.Context, c *entity.TrainerCommission) error
		ListCommissions(ctx context.Context, q dto.Query, f PTCommissionFilter) ([]PTCommissionRow, int64, error)
		CommissionSummary(ctx context.Context, f PTCommissionFilter) ([]PTCommissionSummaryRow, error)
		// PayCommissions chuyển mọi dòng PENDING của HLV trong kỳ (giới hạn chi nhánh nếu branchIDs != nil) sang PAID.
		// Trả số dòng và tổng tiền.
		PayCommissions(ctx context.Context, trainerID uint, period string, branchIDs []uint, paidBy *uint, at time.Time) (int, float64, error)
	}

	ptRepository struct {
		db *gorm.DB
	}
)

func NewPTRepository(db *gorm.DB) PTRepository {
	return &ptRepository{db: db}
}

// ---- Tra cứu ----------------------------------------------------------------------

func (r *ptRepository) FindMember(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ptRepository) FindPackage(ctx context.Context, id uint) (*entity.PTPackage, error) {
	var p entity.PTPackage
	if err := postgres.Conn(ctx, r.db).Unscoped().First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ptRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *ptRepository) TrainerInBranch(ctx context.Context, trainerID, branchID uint) (bool, error) {
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

func (r *ptRepository) CommissionRate(ctx context.Context, trainerID uint) (float64, error) {
	var rates []float64
	err := postgres.Conn(ctx, r.db).Table("staff_profiles").Where("user_id = ?", trainerID).Limit(1).Pluck("commission_rate", &rates).Error
	if err != nil || len(rates) == 0 {
		return 0, err
	}
	return rates[0], nil
}

func (r *ptRepository) UserName(ctx context.Context, id uint) (string, error) {
	var names []string
	err := postgres.Conn(ctx, r.db).Table("users").Where("id = ?", id).Limit(1).Pluck("full_name", &names).Error
	if err != nil || len(names) == 0 {
		return "", err
	}
	return names[0], nil
}

// ---- Hợp đồng ---------------------------------------------------------------------

const ptContractRowSelect = `c.*, m.member_code, m.full_name AS member_name, m.phone AS member_phone,
	COALESCE(t.full_name, '') AS trainer_name, p.code AS package_code, p.name AS package_name,
	b.name AS branch_name, COALESCE(i.invoice_no, '') AS invoice_no, COALESCE(i.status, '') AS invoice_status,
	COALESCE(u.full_name, '') AS sold_by_name,
	(SELECT COUNT(*) FROM pt_sessions ps WHERE ps.contract_id = c.id AND ps.status = 'SCHEDULED') AS scheduled_sessions`

func (r *ptRepository) contractQuery(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("pt_contracts c").
		Joins("JOIN members m ON m.id = c.member_id").
		Joins("JOIN pt_packages p ON p.id = c.pt_package_id").
		Joins("JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN users t ON t.id = c.trainer_id").
		Joins("LEFT JOIN invoices i ON i.id = c.invoice_id").
		Joins("LEFT JOIN users u ON u.id = c.sold_by")
}

func (r *ptRepository) CreateContract(ctx context.Context, c *entity.PTContract) error {
	return postgres.Conn(ctx, r.db).Create(c).Error
}

func (r *ptRepository) FindContract(ctx context.Context, id uint, lock bool) (*entity.PTContract, error) {
	db := postgres.Conn(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var c entity.PTContract
	if err := db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ptRepository) UpdateContractFields(ctx context.Context, id uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.PTContract{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ptRepository) FindContractRow(ctx context.Context, id uint) (*PTContractRow, error) {
	var rows []PTContractRow
	if err := r.contractQuery(ctx).Select(ptContractRowSelect).Where("c.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *ptRepository) ListContracts(ctx context.Context, q dto.Query, f PTContractFilter) ([]PTContractRow, int64, error) {
	tx := r.contractQuery(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("c.branch_id IN ?", f.BranchIDs)
	}
	if f.TrainerID > 0 {
		tx = tx.Where("c.trainer_id = ?", f.TrainerID)
	}
	if f.MemberID > 0 {
		tx = tx.Where("c.member_id = ?", f.MemberID)
	}
	if f.PackageID > 0 {
		tx = tx.Where("c.pt_package_id = ?", f.PackageID)
	}
	if f.Status != "" {
		tx = tx.Where("c.status = ?", f.Status)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("m.full_name ILIKE ? OR m.phone LIKE ? OR m.member_code ILIKE ? OR t.full_name ILIKE ? OR p.name ILIKE ? OR p.code ILIKE ? OR i.invoice_no ILIKE ?",
			kw, kw, kw, kw, kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PTContractRow
	if err := tx.Select(ptContractRowSelect).Scopes(q.Paginate).Order("c.id DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *ptRepository) ListContractsByMember(ctx context.Context, memberID uint, branchIDs []uint) ([]PTContractRow, error) {
	tx := r.contractQuery(ctx).Select(ptContractRowSelect).Where("c.member_id = ?", memberID)
	if branchIDs != nil {
		tx = tx.Where("c.branch_id IN ?", branchIDs)
	}
	var rows []PTContractRow
	err := tx.Order("c.start_date DESC, c.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *ptRepository) CountScheduled(ctx context.Context, contractID uint) (int, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.PTSession{}).
		Where("contract_id = ? AND status = ?", contractID, entity.PTSessionScheduled).Count(&n).Error
	return int(n), err
}

func (r *ptRepository) ContractsToExpire(ctx context.Context, today time.Time) ([]entity.PTContract, error) {
	var rows []entity.PTContract
	err := postgres.Conn(ctx, r.db).Where("status = ? AND end_date < ?", entity.PTContractActive, today).
		Order("id").Find(&rows).Error
	return rows, err
}

// ---- Buổi tập ---------------------------------------------------------------------

const ptSessionRowSelect = `s.*, COALESCE(t.full_name, '') AS trainer_name,
	m.member_code, m.full_name AS member_name, m.phone AS member_phone, m.user_id AS member_user_id,
	b.name AS branch_name, COALESCE(p.name, '') AS package_name, COALESCE(cu.full_name, '') AS created_by_name`

func (r *ptRepository) sessionQuery(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("pt_sessions s").
		Joins("JOIN members m ON m.id = s.member_id").
		Joins("JOIN branches b ON b.id = s.branch_id").
		Joins("JOIN pt_contracts c ON c.id = s.contract_id").
		Joins("LEFT JOIN pt_packages p ON p.id = c.pt_package_id").
		Joins("LEFT JOIN users t ON t.id = s.trainer_id").
		Joins("LEFT JOIN users cu ON cu.id = s.created_by").
		Select(ptSessionRowSelect)
}

func (r *ptRepository) CreateSession(ctx context.Context, s *entity.PTSession) error {
	return postgres.Conn(ctx, r.db).Create(s).Error
}

func (r *ptRepository) FindSession(ctx context.Context, id uint, lock bool) (*entity.PTSession, error) {
	db := postgres.Conn(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var s entity.PTSession
	if err := db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ptRepository) UpdateSessionFields(ctx context.Context, id uint, fields map[string]any) error {
	return postgres.Conn(ctx, r.db).Model(&entity.PTSession{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ptRepository) FindSessionRow(ctx context.Context, id uint) (*PTSessionRow, error) {
	var rows []PTSessionRow
	if err := r.sessionQuery(ctx).Where("s.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *ptRepository) ListSessions(ctx context.Context, f PTSessionFilter) ([]PTSessionRow, error) {
	tx := r.sessionQuery(ctx).Where("s.start_at >= ? AND s.start_at < ?", f.From, f.To)
	if f.BranchIDs != nil {
		tx = tx.Where("s.branch_id IN ?", f.BranchIDs)
	}
	if f.TrainerID > 0 {
		tx = tx.Where("s.trainer_id = ?", f.TrainerID)
	}
	if f.MemberID > 0 {
		tx = tx.Where("s.member_id = ?", f.MemberID)
	}
	if f.ContractID > 0 {
		tx = tx.Where("s.contract_id = ?", f.ContractID)
	}
	if f.Status != "" {
		tx = tx.Where("s.status = ?", f.Status)
	}
	var rows []PTSessionRow
	err := tx.Order("s.start_at, s.id").Scan(&rows).Error
	return rows, err
}

func (r *ptRepository) SessionsByContract(ctx context.Context, contractID uint) ([]PTSessionRow, error) {
	var rows []PTSessionRow
	err := r.sessionQuery(ctx).Where("s.contract_id = ?", contractID).Order("s.start_at, s.id").Scan(&rows).Error
	return rows, err
}

func (r *ptRepository) ScheduledSessions(ctx context.Context, contractID uint, after *time.Time) ([]entity.PTSession, error) {
	tx := postgres.Conn(ctx, r.db).Where("contract_id = ? AND status = ?", contractID, entity.PTSessionScheduled)
	if after != nil {
		tx = tx.Where("start_at > ?", *after)
	}
	var rows []entity.PTSession
	err := tx.Order("start_at, id").Find(&rows).Error
	return rows, err
}

func (r *ptRepository) OverdueSessions(ctx context.Context, before time.Time) ([]PTSessionRow, error) {
	var rows []PTSessionRow
	err := r.sessionQuery(ctx).Where("s.status = ? AND s.end_at < ?", entity.PTSessionScheduled, before).
		Order("s.end_at, s.id").Scan(&rows).Error
	return rows, err
}

func (r *ptRepository) TrainerPTOverlap(ctx context.Context, trainerID uint, start, end time.Time, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("pt_sessions").
		Where("trainer_id = ? AND status IN ('SCHEDULED', 'COMPLETED') AND start_at < ? AND end_at > ? AND id <> ?",
			trainerID, end, start, excludeID).
		Count(&n).Error
	return n > 0, err
}

func (r *ptRepository) TrainerClassOverlap(ctx context.Context, trainerID uint, start, end time.Time) (bool, error) {
	var n int64
	// HLV thực dạy = HLV dạy thay (nếu có), ngược lại là HLV chính.
	err := postgres.Conn(ctx, r.db).Table("class_schedules").
		Where("COALESCE(substitute_trainer_id, trainer_id) = ? AND status <> 'CANCELLED' AND start_at < ? AND end_at > ?",
			trainerID, end, start).
		Count(&n).Error
	return n > 0, err
}

func (r *ptRepository) MemberPTOverlap(ctx context.Context, memberID uint, start, end time.Time, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("pt_sessions").
		Where("member_id = ? AND status IN ('SCHEDULED', 'COMPLETED') AND start_at < ? AND end_at > ? AND id <> ?",
			memberID, end, start, excludeID).
		Count(&n).Error
	return n > 0, err
}

func (r *ptRepository) MemberClassOverlap(ctx context.Context, memberID uint, start, end time.Time) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Table("class_bookings cb").
		Joins("JOIN class_schedules cs ON cs.id = cb.schedule_id").
		Where("cb.member_id = ? AND cb.status IN ('BOOKED', 'ATTENDED') AND cs.status <> 'CANCELLED'", memberID).
		Where("cs.start_at < ? AND cs.end_at > ?", end, start).
		Count(&n).Error
	return n > 0, err
}

func (r *ptRepository) TrainerClasses(ctx context.Context, trainerID uint, from, to time.Time) ([]PTClassSlot, error) {
	var rows []PTClassSlot
	err := postgres.Conn(ctx, r.db).Table("class_schedules cs").
		Select(`cs.id, cs.branch_id, b.name AS branch_name, cs.trainer_id, COALESCE(u.full_name, '') AS trainer_name,
			ct.name AS class_type_name, ct.color, COALESCE(rm.name, '') AS room_name, cs.status,
			cs.start_at, cs.end_at, cs.booked_count, cs.capacity`).
		Joins("JOIN class_types ct ON ct.id = cs.class_type_id").
		Joins("JOIN branches b ON b.id = cs.branch_id").
		Joins("LEFT JOIN rooms rm ON rm.id = cs.room_id").
		Joins("LEFT JOIN users u ON u.id = cs.trainer_id").
		Where("cs.trainer_id = ? AND cs.status <> 'CANCELLED' AND cs.start_at >= ? AND cs.start_at < ?", trainerID, from, to).
		Order("cs.start_at, cs.id").Scan(&rows).Error
	return rows, err
}

// ---- Hoa hồng ---------------------------------------------------------------------

func (r *ptRepository) CreateCommission(ctx context.Context, c *entity.TrainerCommission) error {
	return postgres.Conn(ctx, r.db).Create(c).Error
}

func (r *ptRepository) commissionQuery(ctx context.Context, f PTCommissionFilter) *gorm.DB {
	tx := postgres.Conn(ctx, r.db).Table("trainer_commissions tc")
	if f.BranchIDs != nil {
		tx = tx.Where("tc.branch_id IN ?", f.BranchIDs)
	}
	if f.TrainerID > 0 {
		tx = tx.Where("tc.trainer_id = ?", f.TrainerID)
	}
	if f.Period != "" {
		tx = tx.Where("tc.period = ?", f.Period)
	}
	if f.Status != "" {
		tx = tx.Where("tc.status = ?", f.Status)
	}
	if f.SourceType != "" {
		tx = tx.Where("tc.source_type = ?", f.SourceType)
	}
	return tx
}

func (r *ptRepository) ListCommissions(ctx context.Context, q dto.Query, f PTCommissionFilter) ([]PTCommissionRow, int64, error) {
	tx := r.commissionQuery(ctx, f).
		Joins("LEFT JOIN users t ON t.id = tc.trainer_id").
		Joins("JOIN branches b ON b.id = tc.branch_id").
		Joins("LEFT JOIN users pu ON pu.id = tc.paid_by").
		Joins("LEFT JOIN pt_sessions s ON tc.source_type = 'PT_SESSION' AND s.id = tc.source_id").
		Joins("LEFT JOIN members m ON m.id = s.member_id")
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("t.full_name ILIKE ? OR m.full_name ILIKE ?", kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PTCommissionRow
	err := tx.Select(`tc.*, COALESCE(t.full_name, '') AS trainer_name, b.name AS branch_name,
			COALESCE(pu.full_name, '') AS paid_by_name, COALESCE(m.full_name, '') AS member_name,
			s.start_at AS session_start_at, s.contract_id`).
		Scopes(q.Paginate).Order("tc.id DESC").Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *ptRepository) CommissionSummary(ctx context.Context, f PTCommissionFilter) ([]PTCommissionSummaryRow, error) {
	var rows []PTCommissionSummaryRow
	err := r.commissionQuery(ctx, f).
		Joins("LEFT JOIN users t ON t.id = tc.trainer_id").
		Joins("LEFT JOIN staff_profiles sp ON sp.user_id = tc.trainer_id").
		Select(`tc.trainer_id, COALESCE(t.full_name, '') AS trainer_name, COALESCE(sp.staff_code, '') AS staff_code,
			COUNT(*) AS sessions, COALESCE(SUM(tc.base_amount), 0) AS base_total, COALESCE(SUM(tc.amount), 0) AS total_amount,
			COALESCE(SUM(tc.amount) FILTER (WHERE tc.status = 'PAID'), 0) AS paid_amount,
			COALESCE(SUM(tc.amount) FILTER (WHERE tc.status = 'PENDING'), 0) AS pending_amount`).
		Group("tc.trainer_id, t.full_name, sp.staff_code").
		Order("total_amount DESC, tc.trainer_id").Scan(&rows).Error
	return rows, err
}

func (r *ptRepository) PayCommissions(ctx context.Context, trainerID uint, period string, branchIDs []uint, paidBy *uint, at time.Time) (int, float64, error) {
	var rows []entity.TrainerCommission
	tx := postgres.Conn(ctx, r.db).Model(&rows).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}, {Name: "amount"}}}).
		Where("trainer_id = ? AND period = ? AND status = ?", trainerID, period, entity.CommissionPending)
	if branchIDs != nil {
		tx = tx.Where("branch_id IN ?", branchIDs)
	}
	if err := tx.Updates(map[string]any{"status": entity.CommissionPaid, "paid_at": at, "paid_by": paidBy}).Error; err != nil {
		return 0, 0, err
	}
	var sum float64
	for _, c := range rows {
		sum += c.Amount
	}
	return len(rows), sum, nil
}
