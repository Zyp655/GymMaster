package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	AttendanceRow struct {
		entity.StaffAttendance
		StaffCode  string
		StaffName  string
		Position   string
		BranchName string
	}

	AttendanceFilter struct {
		BranchIDs []uint
		StaffID   uint
		From      *time.Time
		To        *time.Time
	}

	PayrollRow struct {
		entity.Payroll
		BranchName  string
		StaffCount  int
		TotalAmount float64
	}

	PayrollItemRow struct {
		entity.PayrollItem
		StaffCode string
		StaffName string
		Position  string
	}

	PayslipRow struct {
		PayrollItemRow
		BranchID   uint
		BranchName string
		Period     string
		Status     string
		PaidAt     *time.Time
	}

	PayrollFilter struct {
		BranchIDs []uint
		Period    string
		Status    string
	}

	StaffPayrollSource struct {
		StaffID         uint
		StaffCode       string
		StaffName       string
		Position        string
		BaseSalary      float64
		WorkDays        float64
		PTCommission    float64
		SalesCommission float64
	}

	PayrollRepository interface {
		FindOpenAttendanceToday(ctx context.Context, staffID uint, dayStart, dayEnd time.Time) (*entity.StaffAttendance, error)
		FindShiftInDay(ctx context.Context, staffID, branchID uint, dayStart, dayEnd time.Time) (*entity.Shift, error)
		FindShiftByID(ctx context.Context, id uint) (*entity.Shift, error)
		CreateAttendance(ctx context.Context, a *entity.StaffAttendance) error
		UpdateAttendance(ctx context.Context, a *entity.StaffAttendance) error
		ListAttendances(ctx context.Context, filter AttendanceFilter) ([]AttendanceRow, error)

		FindPayrollByID(ctx context.Context, id uint) (*PayrollRow, error)
		FindPayrollByBranchPeriod(ctx context.Context, branchID uint, period string) (*PayrollRow, error)
		ListPayrolls(ctx context.Context, filter PayrollFilter) ([]PayrollRow, error)
		CreatePayroll(ctx context.Context, p *entity.Payroll) error
		UpdatePayroll(ctx context.Context, p *entity.Payroll) error

		ListPayrollItems(ctx context.Context, payrollID uint) ([]PayrollItemRow, error)
		UpsertPayrollItem(ctx context.Context, item *entity.PayrollItem) error
		UpdatePayrollItem(ctx context.Context, item *entity.PayrollItem) error

		CollectStaffPayrollSources(ctx context.Context, branchID uint, period string, from, to time.Time) ([]StaffPayrollSource, error)
		MarkTrainerCommissionsPaid(ctx context.Context, branchID uint, period string, paidBy *uint, paidAt time.Time) error
		ListPayslipsByStaff(ctx context.Context, staffID uint) ([]PayslipRow, error)
	}

	payrollRepository struct {
		db *gorm.DB
	}
)

func NewPayrollRepository(db *gorm.DB) PayrollRepository {
	return &payrollRepository{db: db}
}

func (r *payrollRepository) FindOpenAttendanceToday(ctx context.Context, staffID uint, dayStart, dayEnd time.Time) (*entity.StaffAttendance, error) {
	var a entity.StaffAttendance
	err := postgres.Conn(ctx, r.db).
		Where("staff_id = ? AND check_in_at >= ? AND check_in_at < ? AND check_out_at IS NULL", staffID, dayStart, dayEnd).
		Order("check_in_at DESC").
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *payrollRepository) FindShiftInDay(ctx context.Context, staffID, branchID uint, dayStart, dayEnd time.Time) (*entity.Shift, error) {
	var sh entity.Shift
	err := postgres.Conn(ctx, r.db).
		Where("staff_id = ? AND branch_id = ? AND start_at >= ? AND start_at < ?", staffID, branchID, dayStart, dayEnd).
		Order("start_at ASC").
		First(&sh).Error
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (r *payrollRepository) FindShiftByID(ctx context.Context, id uint) (*entity.Shift, error) {
	var sh entity.Shift
	if err := postgres.Conn(ctx, r.db).First(&sh, id).Error; err != nil {
		return nil, err
	}
	return &sh, nil
}

func (r *payrollRepository) CreateAttendance(ctx context.Context, a *entity.StaffAttendance) error {
	return postgres.Conn(ctx, r.db).Create(a).Error
}

func (r *payrollRepository) UpdateAttendance(ctx context.Context, a *entity.StaffAttendance) error {
	return postgres.Conn(ctx, r.db).Save(a).Error
}

func (r *payrollRepository) ListAttendances(ctx context.Context, f AttendanceFilter) ([]AttendanceRow, error) {
	q := postgres.Conn(ctx, r.db).
		Table("staff_attendances sa").
		Select(`sa.*, COALESCE(sp.staff_code, '') AS staff_code,
			COALESCE(u.full_name, '') AS staff_name,
			COALESCE(sp.position, '') AS position,
			COALESCE(b.name, '') AS branch_name`).
		Joins("LEFT JOIN users u ON u.id = sa.staff_id").
		Joins("LEFT JOIN staff_profiles sp ON sp.user_id = sa.staff_id").
		Joins("LEFT JOIN branches b ON b.id = sa.branch_id")
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []AttendanceRow{}, nil
		}
		q = q.Where("sa.branch_id IN ?", f.BranchIDs)
	}
	if f.StaffID > 0 {
		q = q.Where("sa.staff_id = ?", f.StaffID)
	}
	if f.From != nil {
		q = q.Where("sa.check_in_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("sa.check_in_at < ?", *f.To)
	}
	var rows []AttendanceRow
	if err := q.Order("sa.check_in_at DESC, sa.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *payrollRepository) basePayrolls(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).
		Table("payrolls p").
		Select(`p.*, COALESCE(b.name, '') AS branch_name,
			(SELECT COUNT(*) FROM payroll_items pi WHERE pi.payroll_id = p.id) AS staff_count,
			COALESCE((SELECT SUM(pi.total) FROM payroll_items pi WHERE pi.payroll_id = p.id), 0) AS total_amount`).
		Joins("LEFT JOIN branches b ON b.id = p.branch_id")
}

func (r *payrollRepository) FindPayrollByID(ctx context.Context, id uint) (*PayrollRow, error) {
	var rows []PayrollRow
	if err := r.basePayrolls(ctx).Where("p.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *payrollRepository) FindPayrollByBranchPeriod(ctx context.Context, branchID uint, period string) (*PayrollRow, error) {
	var rows []PayrollRow
	if err := r.basePayrolls(ctx).Where("p.branch_id = ? AND p.period = ?", branchID, period).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *payrollRepository) ListPayrolls(ctx context.Context, f PayrollFilter) ([]PayrollRow, error) {
	q := r.basePayrolls(ctx)
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []PayrollRow{}, nil
		}
		q = q.Where("p.branch_id IN ?", f.BranchIDs)
	}
	if f.Period != "" {
		q = q.Where("p.period = ?", f.Period)
	}
	if f.Status != "" {
		q = q.Where("p.status = ?", f.Status)
	}
	var rows []PayrollRow
	if err := q.Order("p.period DESC, p.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *payrollRepository) CreatePayroll(ctx context.Context, p *entity.Payroll) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *payrollRepository) UpdatePayroll(ctx context.Context, p *entity.Payroll) error {
	return postgres.Conn(ctx, r.db).Save(p).Error
}

func (r *payrollRepository) ListPayrollItems(ctx context.Context, payrollID uint) ([]PayrollItemRow, error) {
	var rows []PayrollItemRow
	err := postgres.Conn(ctx, r.db).
		Table("payroll_items pi").
		Select(`pi.*, COALESCE(sp.staff_code, '') AS staff_code,
			COALESCE(u.full_name, '') AS staff_name,
			COALESCE(sp.position, '') AS position`).
		Joins("LEFT JOIN users u ON u.id = pi.staff_id").
		Joins("LEFT JOIN staff_profiles sp ON sp.user_id = pi.staff_id").
		Where("pi.payroll_id = ?", payrollID).
		Order("pi.staff_id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *payrollRepository) UpsertPayrollItem(ctx context.Context, item *entity.PayrollItem) error {
	db := postgres.Conn(ctx, r.db)
	var existing entity.PayrollItem
	err := db.Where("payroll_id = ? AND staff_id = ?", item.PayrollID, item.StaffID).First(&existing).Error
	if err == nil {
		existing.WorkDays = item.WorkDays
		existing.PTCommission = item.PTCommission
		existing.SalesCommission = item.SalesCommission
		if existing.BaseSalary <= 0 {
			existing.BaseSalary = item.BaseSalary
		}
		existing.Total = existing.BaseSalary + existing.PTCommission + existing.SalesCommission + existing.Bonus - existing.Penalty
		existing.UpdatedAt = item.UpdatedAt
		*item = existing
		return db.Save(&existing).Error
	}
	return db.Create(item).Error
}

func (r *payrollRepository) UpdatePayrollItem(ctx context.Context, item *entity.PayrollItem) error {
	return postgres.Conn(ctx, r.db).Save(item).Error
}

func (r *payrollRepository) CollectStaffPayrollSources(ctx context.Context, branchID uint, period string, from, to time.Time) ([]StaffPayrollSource, error) {
	db := postgres.Conn(ctx, r.db)
	var staffRows []struct {
		StaffID    uint
		StaffCode  string
		StaffName  string
		Position   string
		BaseSalary float64
	}
	err := db.Table("staff_profiles sp").
		Select(`sp.user_id AS staff_id, sp.staff_code, COALESCE(u.full_name, '') AS staff_name, sp.position, sp.base_salary`).
		Joins("JOIN users u ON u.id = sp.user_id AND u.deleted_at IS NULL").
		Where(`EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = sp.user_id AND (ur.branch_id = ? OR ur.branch_id IS NULL))
			OR EXISTS (SELECT 1 FROM staff_attendances sa WHERE sa.staff_id = sp.user_id AND sa.branch_id = ? AND sa.check_in_at >= ? AND sa.check_in_at < ?)
			OR EXISTS (SELECT 1 FROM trainer_commissions tc WHERE tc.trainer_id = sp.user_id AND tc.branch_id = ? AND tc.period = ?)`,
			branchID, branchID, from, to, branchID, period).
		Order("sp.user_id ASC").
		Scan(&staffRows).Error
	if err != nil {
		return nil, err
	}

	out := make([]StaffPayrollSource, 0, len(staffRows))
	for _, st := range staffRows {
		var workDays int64
		_ = db.Table("staff_attendances").
			Where("staff_id = ? AND branch_id = ? AND check_in_at >= ? AND check_in_at < ?", st.StaffID, branchID, from, to).
			Count(&workDays).Error

		var ptComm float64
		_ = db.Table("trainer_commissions").
			Select("COALESCE(SUM(amount), 0)").
			Where("trainer_id = ? AND branch_id = ? AND period = ?", st.StaffID, branchID, period).
			Scan(&ptComm).Error

		var subSales float64
		_ = db.Table("subscriptions").
			Select("COALESCE(SUM(price), 0)").
			Where("sold_by = ? AND branch_id = ? AND status <> 'CANCELLED' AND created_at >= ? AND created_at < ?", st.StaffID, branchID, from, to).
			Scan(&subSales).Error

		out = append(out, StaffPayrollSource{
			StaffID:         st.StaffID,
			StaffCode:       st.StaffCode,
			StaffName:       st.StaffName,
			Position:        st.Position,
			BaseSalary:      st.BaseSalary,
			WorkDays:        float64(workDays),
			PTCommission:    ptComm,
			SalesCommission: subSales * 0.02,
		})
	}
	return out, nil
}

func (r *payrollRepository) MarkTrainerCommissionsPaid(ctx context.Context, branchID uint, period string, paidBy *uint, paidAt time.Time) error {
	return postgres.Conn(ctx, r.db).Table("trainer_commissions").
		Where("branch_id = ? AND period = ? AND status = ?", branchID, period, entity.CommissionPending).
		Updates(map[string]any{
			"status":  entity.CommissionPaid,
			"paid_at": paidAt,
			"paid_by": paidBy,
		}).Error
}

func (r *payrollRepository) ListPayslipsByStaff(ctx context.Context, staffID uint) ([]PayslipRow, error) {
	var rows []PayslipRow
	err := postgres.Conn(ctx, r.db).
		Table("payroll_items pi").
		Select(`pi.*, COALESCE(sp.staff_code, '') AS staff_code,
			COALESCE(u.full_name, '') AS staff_name,
			COALESCE(sp.position, '') AS position,
			p.branch_id, COALESCE(b.name, '') AS branch_name,
			p.period, p.status, p.paid_at`).
		Joins("JOIN payrolls p ON p.id = pi.payroll_id").
		Joins("LEFT JOIN branches b ON b.id = p.branch_id").
		Joins("LEFT JOIN users u ON u.id = pi.staff_id").
		Joins("LEFT JOIN staff_profiles sp ON sp.user_id = pi.staff_id").
		Where("pi.staff_id = ?", staffID).
		Order("p.period DESC, pi.id DESC").
		Scan(&rows).Error
	return rows, err
}

// StaffBranchStatus kiểm tra nhân viên chấm công: exists = tài khoản nhân viên (không phải chỉ là hội viên)
// còn hoạt động; inBranch = có vai trò tại chi nhánh branchID hoặc vai trò toàn hệ thống (branch_id NULL).
// (Không nằm trong interface PayrollRepository để các fake cũ vẫn biên dịch; service dùng qua type assertion.)
func (r *payrollRepository) StaffBranchStatus(ctx context.Context, staffID, branchID uint) (exists, inBranch bool, err error) {
	var row struct {
		Total    int64
		InBranch int64
	}
	err = postgres.Conn(ctx, r.db).Table("users u").
		Select("COUNT(*) AS total, COUNT(*) FILTER (WHERE ur.branch_id = ? OR ur.branch_id IS NULL) AS in_branch", branchID).
		Joins("JOIN user_roles ur ON ur.user_id = u.id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("u.id = ? AND u.deleted_at IS NULL AND u.status = 'ACTIVE' AND r.code <> 'MEMBER'", staffID).
		Scan(&row).Error
	return row.Total > 0, row.InBranch > 0, err
}
