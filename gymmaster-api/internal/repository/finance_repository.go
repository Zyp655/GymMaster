package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	ExpenseRow struct {
		entity.Expense
		BranchName   string
		CategoryName string
		CategoryType string
		CreatorName  string
		ApproverName string
	}

	ExpenseFilter struct {
		BranchIDs  []uint
		CategoryID uint
		Status     string
		From       *time.Time
		To         *time.Time
	}

	CashSessionRow struct {
		entity.CashSession
		BranchName string
		StaffName  string
	}

	CashSessionFilter struct {
		BranchIDs []uint
		Status    string
	}

	ProfitLossData struct {
		MembershipRevenue float64
		PTRevenue         float64
		POSRevenue        float64
		LockerRevenue     float64
		OtherRevenue      float64
		Refunds           float64
		POSCost           float64
		OperatingExpenses float64
		PayrollExpenses   float64
		// ExpensesByCategory: chi phí vận hành đã duyệt theo danh mục (giảm dần theo số tiền).
		ExpensesByCategory []ExpenseCategoryTotal
	}

	ExpenseCategoryTotal struct {
		CategoryID   uint
		CategoryName string
		CategoryType string
		Amount       float64
	}

	InstallmentPlanRow struct {
		entity.InstallmentPlan
		InvoiceNo  string
		BranchID   uint
		MemberCode string
		MemberName string
	}

	InstallmentScheduleRow struct {
		entity.InstallmentSchedule
		InvoiceID  uint
		InvoiceNo  string
		BranchID   uint
		MemberID   uint
		MemberCode string
		MemberName string
	}

	InstallmentFilter struct {
		BranchIDs []uint
		Status    string
		MemberID  uint
		InvoiceID uint
	}

	FinanceRepository interface {
		ListExpenseCategories(ctx context.Context) ([]entity.ExpenseCategory, error)
		ExistsExpenseCategoryName(ctx context.Context, name string) (bool, error)
		CreateExpenseCategory(ctx context.Context, c *entity.ExpenseCategory) error
		FindExpenseCategoryByID(ctx context.Context, id uint) (*entity.ExpenseCategory, error)

		ListExpenses(ctx context.Context, filter ExpenseFilter) ([]ExpenseRow, error)
		FindExpenseByID(ctx context.Context, id uint) (*ExpenseRow, error)
		CreateExpense(ctx context.Context, e *entity.Expense) error
		UpdateExpense(ctx context.Context, e *entity.Expense) error
		DeleteExpense(ctx context.Context, id uint) error

		FindOpenCashSession(ctx context.Context, branchID uint) (*entity.CashSession, error)
		FindCashSessionByID(ctx context.Context, id uint) (*CashSessionRow, error)
		CreateCashSession(ctx context.Context, cs *entity.CashSession) error
		// CloseCashSession đóng ca có điều kiện (UPDATE ... WHERE status = 'OPEN'); false = ca đã bị đóng
		// bởi yêu cầu khác (hai người đóng cùng lúc) → không ghi đè.
		CloseCashSession(ctx context.Context, cs *entity.CashSession) (bool, error)
		ListCashSessions(ctx context.Context, filter CashSessionFilter) ([]CashSessionRow, error)
		SumCashPaymentsBetween(ctx context.Context, branchID uint, from, to time.Time) (float64, error)
		// SumCashRefundsBetween: tổng tiền mặt đã hoàn (refunds.method = CASH) của chi nhánh trong khoảng thời gian.
		SumCashRefundsBetween(ctx context.Context, branchID uint, from, to time.Time) (float64, error)

		CalcProfitLoss(ctx context.Context, branchIDs []uint, from, to time.Time) (*ProfitLossData, error)

		FindInvoiceByID(ctx context.Context, id uint) (*entity.Invoice, error)
		FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error)
		FindInstallmentPlanByInvoice(ctx context.Context, invoiceID uint) (*InstallmentPlanRow, error)
		CreateInstallmentPlan(ctx context.Context, p *entity.InstallmentPlan, schedules []entity.InstallmentSchedule) error
		ListInstallmentPlansByMember(ctx context.Context, memberID uint) ([]InstallmentPlanRow, error)
		ListSchedulesByPlanIDs(ctx context.Context, planIDs []uint) ([]InstallmentScheduleRow, error)
		ListInstallmentSchedules(ctx context.Context, filter InstallmentFilter) ([]InstallmentScheduleRow, error)
		FindInstallmentScheduleByID(ctx context.Context, id uint) (*InstallmentScheduleRow, error)
		UpdateInstallmentSchedule(ctx context.Context, s *entity.InstallmentSchedule) error
		// LockInstallmentSchedule khóa (FOR UPDATE) một kỳ trả góp – gọi trong transaction.
		LockInstallmentSchedule(ctx context.Context, id uint) (*entity.InstallmentSchedule, error)
	}

	financeRepository struct {
		db *gorm.DB
	}
)

func NewFinanceRepository(db *gorm.DB) FinanceRepository {
	return &financeRepository{db: db}
}

func (r *financeRepository) ListExpenseCategories(ctx context.Context) ([]entity.ExpenseCategory, error) {
	var rows []entity.ExpenseCategory
	err := postgres.Conn(ctx, r.db).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *financeRepository) ExistsExpenseCategoryName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := postgres.Conn(ctx, r.db).Model(&entity.ExpenseCategory{}).
		Where("UPPER(name) = UPPER(?)", strings.TrimSpace(name)).
		Count(&count).Error
	return count > 0, err
}

func (r *financeRepository) CreateExpenseCategory(ctx context.Context, c *entity.ExpenseCategory) error {
	return postgres.Conn(ctx, r.db).Create(c).Error
}

func (r *financeRepository) FindExpenseCategoryByID(ctx context.Context, id uint) (*entity.ExpenseCategory, error) {
	var c entity.ExpenseCategory
	if err := postgres.Conn(ctx, r.db).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *financeRepository) baseExpenses(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).
		Table("expenses e").
		Select(`e.*, COALESCE(b.name, '') AS branch_name,
			COALESCE(ec.name, '') AS category_name,
			COALESCE(ec.type, '') AS category_type,
			COALESCE(cu.full_name, '') AS creator_name,
			COALESCE(au.full_name, '') AS approver_name`).
		Joins("LEFT JOIN branches b ON b.id = e.branch_id").
		Joins("LEFT JOIN expense_categories ec ON ec.id = e.category_id").
		Joins("LEFT JOIN users cu ON cu.id = e.created_by").
		Joins("LEFT JOIN users au ON au.id = e.approved_by")
}

func (r *financeRepository) ListExpenses(ctx context.Context, f ExpenseFilter) ([]ExpenseRow, error) {
	q := r.baseExpenses(ctx)
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []ExpenseRow{}, nil
		}
		q = q.Where("e.branch_id IN ?", f.BranchIDs)
	}
	if f.CategoryID > 0 {
		q = q.Where("e.category_id = ?", f.CategoryID)
	}
	if f.Status != "" {
		q = q.Where("e.status = ?", f.Status)
	}
	if f.From != nil {
		q = q.Where("e.spent_at >= ?", f.From.Format("2006-01-02"))
	}
	if f.To != nil {
		q = q.Where("e.spent_at <= ?", f.To.Format("2006-01-02"))
	}
	var rows []ExpenseRow
	if err := q.Order("e.spent_at DESC, e.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *financeRepository) FindExpenseByID(ctx context.Context, id uint) (*ExpenseRow, error) {
	var rows []ExpenseRow
	if err := r.baseExpenses(ctx).Where("e.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *financeRepository) CreateExpense(ctx context.Context, e *entity.Expense) error {
	return postgres.Conn(ctx, r.db).Create(e).Error
}

func (r *financeRepository) UpdateExpense(ctx context.Context, e *entity.Expense) error {
	return postgres.Conn(ctx, r.db).Save(e).Error
}

func (r *financeRepository) DeleteExpense(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Expense{}, id).Error
}

func (r *financeRepository) FindOpenCashSession(ctx context.Context, branchID uint) (*entity.CashSession, error) {
	var cs entity.CashSession
	err := postgres.Conn(ctx, r.db).
		Where("branch_id = ? AND status = ?", branchID, entity.CashSessionOpen).
		Order("opened_at DESC").
		First(&cs).Error
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

func (r *financeRepository) baseCashSessions(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).
		Table("cash_sessions cs").
		Select(`cs.*, COALESCE(b.name, '') AS branch_name, COALESCE(u.full_name, '') AS staff_name`).
		Joins("LEFT JOIN branches b ON b.id = cs.branch_id").
		Joins("LEFT JOIN users u ON u.id = cs.staff_id")
}

func (r *financeRepository) FindCashSessionByID(ctx context.Context, id uint) (*CashSessionRow, error) {
	var rows []CashSessionRow
	if err := r.baseCashSessions(ctx).Where("cs.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *financeRepository) CreateCashSession(ctx context.Context, cs *entity.CashSession) error {
	return postgres.Conn(ctx, r.db).Create(cs).Error
}

func (r *financeRepository) CloseCashSession(ctx context.Context, cs *entity.CashSession) (bool, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.CashSession{}).
		Where("id = ? AND status = ?", cs.ID, entity.CashSessionOpen).
		Updates(map[string]any{
			"closed_at":     cs.ClosedAt,
			"expected_cash": cs.ExpectedCash,
			"counted_cash":  cs.CountedCash,
			"difference":    cs.Difference,
			"note":          cs.Note,
			"status":        entity.CashSessionClosed,
			"updated_at":    cs.UpdatedAt,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *financeRepository) ListCashSessions(ctx context.Context, f CashSessionFilter) ([]CashSessionRow, error) {
	q := r.baseCashSessions(ctx)
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []CashSessionRow{}, nil
		}
		q = q.Where("cs.branch_id IN ?", f.BranchIDs)
	}
	if f.Status != "" {
		q = q.Where("cs.status = ?", f.Status)
	}
	var rows []CashSessionRow
	if err := q.Order("cs.opened_at DESC, cs.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *financeRepository) SumCashPaymentsBetween(ctx context.Context, branchID uint, from, to time.Time) (float64, error) {
	var sum float64
	err := postgres.Conn(ctx, r.db).
		Table("payments p").
		Select("COALESCE(SUM(p.amount), 0)").
		Joins("JOIN invoices i ON i.id = p.invoice_id").
		Where("i.branch_id = ? AND p.method = ? AND p.paid_at >= ? AND p.paid_at <= ?",
			branchID, entity.PaymentCash, from, to).
		Scan(&sum).Error
	return sum, err
}

func (r *financeRepository) SumCashRefundsBetween(ctx context.Context, branchID uint, from, to time.Time) (float64, error) {
	var sum float64
	err := postgres.Conn(ctx, r.db).
		Table("refunds rf").
		Select("COALESCE(SUM(rf.amount), 0)").
		Joins("JOIN invoices i ON i.id = rf.invoice_id").
		Where("i.branch_id = ? AND rf.method = ? AND rf.created_at >= ? AND rf.created_at <= ?",
			branchID, entity.PaymentCash, from, to).
		Scan(&sum).Error
	return sum, err
}

func (r *financeRepository) CalcProfitLoss(ctx context.Context, branchIDs []uint, from, to time.Time) (*ProfitLossData, error) {
	db := postgres.Conn(ctx, r.db)
	endExclusive := to.AddDate(0, 0, 1)
	out := &ProfitLossData{}

	var itemRevs []struct {
		ItemType string
		Amount   float64
	}
	iq := db.Table("invoice_items ii").
		Select("ii.item_type, COALESCE(SUM(ii.amount), 0) AS amount").
		Joins("JOIN invoices i ON i.id = ii.invoice_id").
		Where("i.status IN ? AND i.issued_at >= ? AND i.issued_at < ?",
			[]string{entity.InvoiceStatusPaid, entity.InvoiceStatusPartial, entity.InvoiceStatusRefunded}, from, endExclusive)
	if branchIDs != nil {
		iq = iq.Where("i.branch_id IN ?", branchIDs)
	}
	if err := iq.Group("ii.item_type").Scan(&itemRevs).Error; err != nil {
		return nil, err
	}
	for _, ir := range itemRevs {
		switch ir.ItemType {
		case entity.InvoiceItemMembership:
			out.MembershipRevenue += ir.Amount
		case entity.InvoiceItemPT:
			out.PTRevenue += ir.Amount
		case entity.InvoiceItemProduct:
			out.POSRevenue += ir.Amount
		case entity.InvoiceItemLocker:
			out.LockerRevenue += ir.Amount
		default:
			out.OtherRevenue += ir.Amount
		}
	}

	// Hoàn tiền
	rq := db.Table("refunds rf").
		Select("COALESCE(SUM(rf.amount), 0)").
		Joins("JOIN invoices i ON i.id = rf.invoice_id").
		Where("rf.created_at >= ? AND rf.created_at < ?", from, endExclusive)
	if branchIDs != nil {
		rq = rq.Where("i.branch_id IN ?", branchIDs)
	}
	if err := rq.Scan(&out.Refunds).Error; err != nil {
		return nil, err
	}

	// Giá vốn POS = giá vốn hàng xuất bán (SALE, quantity âm) − giá vốn hàng khách trả lại (SALE_RETURN, quantity dương).
	cq := db.Table("stock_transactions st").
		Select("COALESCE(-SUM(st.quantity * st.unit_cost), 0)").
		Where("st.type IN ? AND st.created_at >= ? AND st.created_at < ?",
			[]string{entity.StockSale, entity.StockSaleReturn}, from, endExclusive)
	if branchIDs != nil {
		cq = cq.Where("st.branch_id IN ?", branchIDs)
	}
	if err := cq.Scan(&out.POSCost).Error; err != nil {
		return nil, err
	}

	// Chi phí vận hành (expenses APPROVED)
	eq := db.Table("expenses e").
		Select("COALESCE(SUM(e.amount), 0)").
		Where("e.status = ? AND e.spent_at >= ? AND e.spent_at <= ?",
			entity.ExpenseStatusApproved, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if branchIDs != nil {
		eq = eq.Where("e.branch_id IN ?", branchIDs)
	}
	if err := eq.Scan(&out.OperatingExpenses).Error; err != nil {
		return nil, err
	}

	// Chi phí vận hành theo danh mục
	ecq := db.Table("expenses e").
		Select(`e.category_id, COALESCE(ec.name, '') AS category_name, COALESCE(ec.type, '') AS category_type,
			COALESCE(SUM(e.amount), 0) AS amount`).
		Joins("LEFT JOIN expense_categories ec ON ec.id = e.category_id").
		Where("e.status = ? AND e.spent_at >= ? AND e.spent_at <= ?",
			entity.ExpenseStatusApproved, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if branchIDs != nil {
		ecq = ecq.Where("e.branch_id IN ?", branchIDs)
	}
	if err := ecq.Group("e.category_id, ec.name, ec.type").Order("amount DESC, e.category_id ASC").
		Scan(&out.ExpensesByCategory).Error; err != nil {
		return nil, err
	}

	// Chi phí lương (payroll_items của payrolls LOCKED/PAID trong khoảng kỳ YYYY-MM)
	fromPeriod := from.Format("2006-01")
	toPeriod := to.Format("2006-01")
	pq := db.Table("payroll_items pi").
		Select("COALESCE(SUM(pi.total), 0)").
		Joins("JOIN payrolls p ON p.id = pi.payroll_id").
		Where("p.status IN ? AND p.period >= ? AND p.period <= ?",
			[]string{entity.PayrollStatusLocked, entity.PayrollStatusPaid}, fromPeriod, toPeriod)
	if branchIDs != nil {
		pq = pq.Where("p.branch_id IN ?", branchIDs)
	}
	if err := pq.Scan(&out.PayrollExpenses).Error; err != nil {
		return nil, err
	}

	return out, nil
}

func (r *financeRepository) FindInvoiceByID(ctx context.Context, id uint) (*entity.Invoice, error) {
	var inv entity.Invoice
	if err := postgres.Conn(ctx, r.db).First(&inv, id).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *financeRepository) FindMemberByUserID(ctx context.Context, userID uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *financeRepository) baseInstallmentPlans(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).
		Table("installment_plans ip").
		Select(`ip.*, COALESCE(i.invoice_no, '') AS invoice_no, COALESCE(i.branch_id, 0) AS branch_id,
			COALESCE(m.member_code, '') AS member_code, COALESCE(m.full_name, '') AS member_name`).
		Joins("LEFT JOIN invoices i ON i.id = ip.invoice_id").
		Joins("LEFT JOIN members m ON m.id = ip.member_id")
}

func (r *financeRepository) FindInstallmentPlanByInvoice(ctx context.Context, invoiceID uint) (*InstallmentPlanRow, error) {
	var rows []InstallmentPlanRow
	if err := r.baseInstallmentPlans(ctx).Where("ip.invoice_id = ?", invoiceID).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *financeRepository) CreateInstallmentPlan(ctx context.Context, p *entity.InstallmentPlan, schedules []entity.InstallmentSchedule) error {
	db := postgres.Conn(ctx, r.db)
	if err := db.Create(p).Error; err != nil {
		return err
	}
	for i := range schedules {
		schedules[i].PlanID = p.ID
	}
	if len(schedules) > 0 {
		return db.Create(&schedules).Error
	}
	return nil
}

func (r *financeRepository) ListInstallmentPlansByMember(ctx context.Context, memberID uint) ([]InstallmentPlanRow, error) {
	var rows []InstallmentPlanRow
	err := r.baseInstallmentPlans(ctx).Where("ip.member_id = ?", memberID).Order("ip.id DESC").Scan(&rows).Error
	return rows, err
}

func (r *financeRepository) baseSchedules(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).
		Table("installment_schedules s").
		Select(`s.*, ip.invoice_id, COALESCE(i.invoice_no, '') AS invoice_no, COALESCE(i.branch_id, 0) AS branch_id,
			ip.member_id, COALESCE(m.member_code, '') AS member_code, COALESCE(m.full_name, '') AS member_name`).
		Joins("JOIN installment_plans ip ON ip.id = s.plan_id").
		Joins("LEFT JOIN invoices i ON i.id = ip.invoice_id").
		Joins("LEFT JOIN members m ON m.id = ip.member_id")
}

func (r *financeRepository) ListSchedulesByPlanIDs(ctx context.Context, planIDs []uint) ([]InstallmentScheduleRow, error) {
	if len(planIDs) == 0 {
		return []InstallmentScheduleRow{}, nil
	}
	var rows []InstallmentScheduleRow
	err := r.baseSchedules(ctx).Where("s.plan_id IN ?", planIDs).Order("s.plan_id ASC, s.seq ASC").Scan(&rows).Error
	return rows, err
}

func (r *financeRepository) ListInstallmentSchedules(ctx context.Context, f InstallmentFilter) ([]InstallmentScheduleRow, error) {
	q := r.baseSchedules(ctx)
	if f.BranchIDs != nil {
		if len(f.BranchIDs) == 0 {
			return []InstallmentScheduleRow{}, nil
		}
		q = q.Where("i.branch_id IN ?", f.BranchIDs)
	}
	if f.Status != "" {
		q = q.Where("s.status = ?", f.Status)
	}
	if f.MemberID > 0 {
		q = q.Where("ip.member_id = ?", f.MemberID)
	}
	if f.InvoiceID > 0 {
		q = q.Where("ip.invoice_id = ?", f.InvoiceID)
	}
	var rows []InstallmentScheduleRow
	if err := q.Order("s.due_date ASC, s.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *financeRepository) FindInstallmentScheduleByID(ctx context.Context, id uint) (*InstallmentScheduleRow, error) {
	var rows []InstallmentScheduleRow
	if err := r.baseSchedules(ctx).Where("s.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *financeRepository) UpdateInstallmentSchedule(ctx context.Context, s *entity.InstallmentSchedule) error {
	return postgres.Conn(ctx, r.db).Save(s).Error
}

func (r *financeRepository) LockInstallmentSchedule(ctx context.Context, id uint) (*entity.InstallmentSchedule, error) {
	var sc entity.InstallmentSchedule
	if err := postgres.Conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).First(&sc, id).Error; err != nil {
		return nil, err
	}
	return &sc, nil
}
