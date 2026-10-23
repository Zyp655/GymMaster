package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/finance/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/einvoice"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeFinanceRepo struct {
	nextCatID   uint
	nextExpID   uint
	nextSessID  uint
	nextPlanID  uint
	nextSchedID uint

	categories map[uint]*entity.ExpenseCategory
	expenses   map[uint]*entity.Expense
	sessions   map[uint]*entity.CashSession
	invoices   map[uint]*entity.Invoice
	members    map[uint]*entity.Member
	plans      map[uint]*entity.InstallmentPlan
	schedules  map[uint]*entity.InstallmentSchedule

	cashCollected float64
	cashRefunded  float64
	pnl           *repository.ProfitLossData
}

// fakeFinanceInvoices: InvoiceService giả chỉ cài AddPayment (ghi nhận lời gọi, có thể trả lỗi).
type fakeFinanceInvoices struct {
	InvoiceService
	payments []PaymentInput
	failWith error
	invoice  *entity.Invoice
}

// LockInvoice: hóa đơn giả còn nợ (f.invoice nếu được gán).
func (f *fakeFinanceInvoices) LockInvoice(_ context.Context, id uint) (*entity.Invoice, error) {
	if f.invoice != nil {
		cp := *f.invoice
		return &cp, nil
	}
	return &entity.Invoice{ID: id, Status: entity.InvoiceStatusPartial, Total: 12000000, PaidAmount: 3000000}, nil
}

func (f *fakeFinanceInvoices) AddPayment(_ context.Context, _ uint, in PaymentInput) (*entity.Invoice, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.payments = append(f.payments, in)
	return &entity.Invoice{}, nil
}

func newFakeFinanceRepo() *fakeFinanceRepo {
	uid := uint(101)
	mid := uint(10)
	return &fakeFinanceRepo{
		categories: map[uint]*entity.ExpenseCategory{},
		expenses:   map[uint]*entity.Expense{},
		sessions:   map[uint]*entity.CashSession{},
		invoices: map[uint]*entity.Invoice{
			1: {ID: 1, InvoiceNo: "HD26101400001", BranchID: 1, MemberID: &mid, Total: 12000000, PaidAmount: 3000000, Status: entity.InvoiceStatusPartial},
		},
		members: map[uint]*entity.Member{
			10: {BaseEntity: entity.BaseEntity{ID: 10}, UserID: &uid, MemberCode: "HV00010", FullName: "Phạm Thị D"},
		},
		plans:     map[uint]*entity.InstallmentPlan{},
		schedules: map[uint]*entity.InstallmentSchedule{},
	}
}

func (f *fakeFinanceRepo) ListExpenseCategories(context.Context) ([]entity.ExpenseCategory, error) {
	var out []entity.ExpenseCategory
	for _, c := range f.categories {
		out = append(out, *c)
	}
	return out, nil
}

func (f *fakeFinanceRepo) ExistsExpenseCategoryName(_ context.Context, name string) (bool, error) {
	for _, c := range f.categories {
		if strings.EqualFold(c.Name, strings.TrimSpace(name)) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeFinanceRepo) CreateExpenseCategory(_ context.Context, c *entity.ExpenseCategory) error {
	f.nextCatID++
	c.ID = f.nextCatID
	cp := *c
	f.categories[c.ID] = &cp
	return nil
}

func (f *fakeFinanceRepo) FindExpenseCategoryByID(_ context.Context, id uint) (*entity.ExpenseCategory, error) {
	if c, ok := f.categories[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) ListExpenses(_ context.Context, _ repository.ExpenseFilter) ([]repository.ExpenseRow, error) {
	var out []repository.ExpenseRow
	for _, e := range f.expenses {
		out = append(out, repository.ExpenseRow{Expense: *e})
	}
	return out, nil
}

func (f *fakeFinanceRepo) FindExpenseByID(_ context.Context, id uint) (*repository.ExpenseRow, error) {
	if e, ok := f.expenses[id]; ok {
		return &repository.ExpenseRow{Expense: *e}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) CreateExpense(_ context.Context, e *entity.Expense) error {
	f.nextExpID++
	e.ID = f.nextExpID
	cp := *e
	f.expenses[e.ID] = &cp
	return nil
}

func (f *fakeFinanceRepo) UpdateExpense(_ context.Context, e *entity.Expense) error {
	cp := *e
	f.expenses[e.ID] = &cp
	return nil
}

func (f *fakeFinanceRepo) DeleteExpense(_ context.Context, id uint) error {
	delete(f.expenses, id)
	return nil
}

func (f *fakeFinanceRepo) SetExpenseStatus(_ context.Context, id uint, from, to string, approvedBy *uint, at time.Time) (bool, error) {
	e, ok := f.expenses[id]
	if !ok || e.Status != from {
		return false, nil
	}
	e.Status, e.ApprovedBy, e.UpdatedAt = to, approvedBy, at
	return true, nil
}

// FindBranchStatus: chi nhánh 1, 2 hoạt động; 3 ngừng hoạt động; còn lại không tồn tại.
func (f *fakeFinanceRepo) FindBranchStatus(_ context.Context, id uint) (string, error) {
	switch id {
	case 1, 2:
		return consts.StatusActive, nil
	case 3:
		return consts.StatusInactive, nil
	}
	return "", gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) FindOpenCashSession(_ context.Context, branchID uint) (*entity.CashSession, error) {
	for _, cs := range f.sessions {
		if cs.BranchID == branchID && cs.Status == entity.CashSessionOpen {
			return cs, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) FindCashSessionByID(_ context.Context, id uint) (*repository.CashSessionRow, error) {
	if cs, ok := f.sessions[id]; ok {
		return &repository.CashSessionRow{CashSession: *cs}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) CreateCashSession(_ context.Context, cs *entity.CashSession) error {
	f.nextSessID++
	cs.ID = f.nextSessID
	cp := *cs
	f.sessions[cs.ID] = &cp
	return nil
}

func (f *fakeFinanceRepo) CloseCashSession(_ context.Context, cs *entity.CashSession) (bool, error) {
	if cur, ok := f.sessions[cs.ID]; !ok || cur.Status != entity.CashSessionOpen {
		return false, nil
	}
	cp := *cs
	cp.Status = entity.CashSessionClosed
	f.sessions[cs.ID] = &cp
	return true, nil
}

func (f *fakeFinanceRepo) ListCashSessions(_ context.Context, _ repository.CashSessionFilter) ([]repository.CashSessionRow, error) {
	var out []repository.CashSessionRow
	for _, cs := range f.sessions {
		out = append(out, repository.CashSessionRow{CashSession: *cs})
	}
	return out, nil
}

func (f *fakeFinanceRepo) SumCashPaymentsBetween(context.Context, uint, time.Time, time.Time) (float64, error) {
	return f.cashCollected, nil
}

func (f *fakeFinanceRepo) SumCashRefundsBetween(context.Context, uint, time.Time, time.Time) (float64, error) {
	return f.cashRefunded, nil
}

func (f *fakeFinanceRepo) LockInstallmentSchedule(_ context.Context, id uint) (*entity.InstallmentSchedule, error) {
	if sc, ok := f.schedules[id]; ok {
		cp := *sc
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) CalcProfitLoss(context.Context, []uint, time.Time, time.Time) (*repository.ProfitLossData, error) {
	return f.pnl, nil
}

func (f *fakeFinanceRepo) FindInvoiceByID(_ context.Context, id uint) (*entity.Invoice, error) {
	if inv, ok := f.invoices[id]; ok {
		return inv, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			return m, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) FindInstallmentPlanByInvoice(_ context.Context, invoiceID uint) (*repository.InstallmentPlanRow, error) {
	for _, p := range f.plans {
		if p.InvoiceID == invoiceID {
			return &repository.InstallmentPlanRow{InstallmentPlan: *p}, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) CreateInstallmentPlan(_ context.Context, p *entity.InstallmentPlan, schedules []entity.InstallmentSchedule) error {
	f.nextPlanID++
	p.ID = f.nextPlanID
	cp := *p
	f.plans[p.ID] = &cp
	for i := range schedules {
		f.nextSchedID++
		schedules[i].ID = f.nextSchedID
		schedules[i].PlanID = p.ID
		sc := schedules[i]
		f.schedules[sc.ID] = &sc
	}
	return nil
}

func (f *fakeFinanceRepo) ListInstallmentPlansByMember(_ context.Context, memberID uint) ([]repository.InstallmentPlanRow, error) {
	var out []repository.InstallmentPlanRow
	for _, p := range f.plans {
		if p.MemberID == memberID {
			out = append(out, repository.InstallmentPlanRow{InstallmentPlan: *p, InvoiceNo: "HD26101400001"})
		}
	}
	return out, nil
}

func (f *fakeFinanceRepo) ListSchedulesByPlanIDs(_ context.Context, planIDs []uint) ([]repository.InstallmentScheduleRow, error) {
	var out []repository.InstallmentScheduleRow
	for _, sc := range f.schedules {
		out = append(out, repository.InstallmentScheduleRow{InstallmentSchedule: *sc})
	}
	return out, nil
}

func (f *fakeFinanceRepo) ListInstallmentSchedules(_ context.Context, _ repository.InstallmentFilter) ([]repository.InstallmentScheduleRow, error) {
	var out []repository.InstallmentScheduleRow
	for _, sc := range f.schedules {
		out = append(out, repository.InstallmentScheduleRow{InstallmentSchedule: *sc})
	}
	return out, nil
}

func (f *fakeFinanceRepo) FindInstallmentScheduleByID(_ context.Context, id uint) (*repository.InstallmentScheduleRow, error) {
	if sc, ok := f.schedules[id]; ok {
		return &repository.InstallmentScheduleRow{InstallmentSchedule: *sc, BranchID: 1, InvoiceID: 1}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFinanceRepo) UpdateInstallmentSchedule(_ context.Context, s *entity.InstallmentSchedule) error {
	cp := *s
	f.schedules[s.ID] = &cp
	return nil
}

func TestFinanceService_CashSessionPnLInstallmentsAndEInvoice(t *testing.T) {
	repo := newFakeFinanceRepo()
	repo.cashCollected = 5500000
	repo.cashRefunded = 300000
	repo.pnl = &repository.ProfitLossData{
		MembershipRevenue: 50000000,
		PTRevenue:         20000000,
		POSRevenue:        5000000,
		LockerRevenue:     1500000,
		Refunds:           1000000,
		POSCost:           2000000,
		OperatingExpenses: 15000000,
		PayrollExpenses:   25000000,
	}

	invoices := &fakeFinanceInvoices{}
	svc := NewFinanceService(repo, invoices, einvoice.NewMockProvider(), nil, nil)
	svc.(*financeService).now = func() time.Time { return time.Date(2026, 10, 21, 9, 0, 0, 0, memberVNLoc) }
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 101, Roles: []string{consts.RoleMember}})

	// 1. Expense Category + Expense + Approve
	cat, err := svc.CreateExpenseCategory(ctx, &v1.CreateExpenseCategoryReq{
		Name: "Tiền điện tháng 10",
		Type: entity.ExpenseTypeVariable,
	})
	require.NoError(t, err)

	exp, err := svc.CreateExpense(ctx, &v1.CreateExpenseReq{
		BranchID:    1,
		CategoryID:  cat.ID,
		Amount:      4500000,
		Description: "Hóa đơn điện lực Q1",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ExpenseStatusPending, exp.Status)

	approved, err := svc.ApproveExpense(ctx, &v1.ApproveExpenseReq{
		ID:     exp.ID,
		Status: entity.ExpenseStatusApproved,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.ExpenseStatusApproved, approved.Status)

	// PUT không được tự duyệt chi phí
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, Status: entity.ExpenseStatusRejected})
	require.Error(t, err)
	// Danh mục không tồn tại → 404 (không phải 500)
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, CategoryID: 999})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "danh mục")
	// Số tiền 0 bị từ chối
	zero := 0.0
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, Amount: &zero})
	require.Error(t, err)
	// Khoản đã duyệt không sửa được số tiền (D1)
	newAmt := 4600000.0
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, Amount: &newAmt})
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// 2. Open & Close Cash Session -> expected_cash = 2,000,000 + 5,500,000 − 300,000 (hoàn tiền mặt) = 7,200,000
	sess, err := svc.OpenCashSession(ctx, &v1.OpenCashSessionReq{
		BranchID:    1,
		OpeningCash: 2000000,
	})
	require.NoError(t, err)

	closed, err := svc.CloseCashSession(ctx, &v1.CloseCashSessionReq{
		ID:          sess.ID,
		CountedCash: 7450000,
	})
	require.NoError(t, err)
	assert.Equal(t, float64(7200000), closed.ExpectedCash)
	assert.Equal(t, float64(250000), closed.Difference)
	assert.Equal(t, entity.CashSessionClosed, closed.Status)

	// 3. P&L Report
	pnl, err := svc.ProfitLossReport(ctx, &v1.ProfitLossReportReq{BranchID: 1})
	require.NoError(t, err)
	assert.Equal(t, float64(76500000), pnl.GrossRevenue)
	assert.Equal(t, float64(75500000), pnl.NetRevenue)
	assert.Equal(t, float64(42000000), pnl.TotalExpenses)
	assert.Equal(t, float64(33500000), pnl.NetProfit)

	// 4. Create Installment Plan (12m - 3m paid = 9m split into 3 installments of 3m each)
	plan, err := svc.CreateInstallmentPlan(ctx, &v1.CreateInstallmentPlanReq{
		ID:           1,
		Installments: 3,
		FirstDueDate: "2026-11-01",
	})
	require.NoError(t, err)
	assert.Equal(t, float64(9000000), plan.Total)
	require.Len(t, plan.Schedules, 3)
	assert.Equal(t, float64(3000000), plan.Schedules[0].Amount)

	assert.Equal(t, "2026-12-01", plan.Schedules[1].DueDate)

	// Đã có kế hoạch → 409
	_, err = svc.CreateInstallmentPlan(ctx, &v1.CreateInstallmentPlanReq{ID: 1, Installments: 3})
	require.Error(t, err)

	// Phương thức không hợp lệ bị từ chối, kỳ vẫn PENDING
	_, err = svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{ID: plan.Schedules[0].ID, Method: "BITCOIN"})
	require.Error(t, err)
	assert.Equal(t, entity.InstallmentPending, repo.schedules[plan.Schedules[0].ID].Status)

	// Thu tiền thất bại (vd. vượt số còn nợ) → kỳ KHÔNG bị đánh dấu PAID
	invoices.failWith = errors.New("Số tiền thu vượt quá số còn nợ")
	_, err = svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{ID: plan.Schedules[0].ID, Method: "CASH"})
	require.Error(t, err)
	assert.Equal(t, entity.InstallmentPending, repo.schedules[plan.Schedules[0].ID].Status)
	invoices.failWith = nil

	// Pay 1st installment
	paidSched, err := svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{
		ID:     plan.Schedules[0].ID,
		Method: "transfer",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.InstallmentPaid, paidSched.Status)
	require.Len(t, invoices.payments, 1)
	assert.Equal(t, float64(3000000), invoices.payments[0].Amount)
	assert.Equal(t, entity.PaymentTransfer, invoices.payments[0].Method)

	// Thu lại kỳ đã thu → 409, không ghi thêm phiếu thu
	_, err = svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{ID: plan.Schedules[0].ID, Method: "CASH"})
	require.Error(t, err)
	require.Len(t, invoices.payments, 1)

	// Hóa đơn chỉ còn nợ 1tr (< 3tr của kỳ 2, do đã thu trực tiếp) → chỉ thu 1tr và tất toán kỳ.
	invoices.invoice = &entity.Invoice{ID: 1, Status: entity.InvoiceStatusPartial, Total: 12000000, PaidAmount: 11000000}
	sc2, err := svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{ID: plan.Schedules[1].ID, Method: "cash"})
	require.NoError(t, err)
	require.Len(t, invoices.payments, 2)
	assert.Equal(t, float64(1000000), invoices.payments[1].Amount)
	assert.Equal(t, entity.PaymentCash, invoices.payments[1].Method)
	assert.Equal(t, entity.InstallmentPaid, sc2.Status)
	assert.Equal(t, float64(1000000), sc2.PaidAmount)
	assert.Equal(t, float64(1000000), repo.schedules[plan.Schedules[1].ID].Amount)

	// Hóa đơn đã thu đủ / kỳ đã tất toán → không thu, không ghi phiếu thu.
	invoices.invoice = &entity.Invoice{ID: 1, Status: entity.InvoiceStatusPaid, Total: 12000000, PaidAmount: 12000000}
	_, err = svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{ID: plan.Schedules[2].ID, Method: "CASH"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	repo.schedules[plan.Schedules[2].ID].Status = installmentSettled
	_, err = svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{ID: plan.Schedules[2].ID, Method: "CASH"})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	require.Len(t, invoices.payments, 2)
	invoices.invoice = nil

	// Member installments
	memPlans, err := svc.MemberInstallments(memberCtx)
	require.NoError(t, err)
	require.Len(t, memPlans, 1)

	// 5. Issue E-Invoice
	einv, err := svc.IssueEInvoice(ctx, 1)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(einv.EInvoiceNo, "EINV-"))
	assert.NotEmpty(t, einv.LookupCode)
	assert.Contains(t, einv.PDFURL, ".pdf")
}

func TestInstallmentDueDate_ClampsToMonthEnd(t *testing.T) {
	d := func(s string) time.Time {
		v, err := time.Parse(consts.DateLayout, s)
		require.NoError(t, err)
		return v
	}
	cases := []struct {
		first string
		i     int
		want  string
	}{
		{"2026-01-31", 0, "2026-01-31"},
		{"2026-01-31", 1, "2026-02-28"},
		{"2026-01-31", 2, "2026-03-31"},
		{"2026-01-31", 3, "2026-04-30"},
		{"2028-01-31", 1, "2028-02-29"}, // năm nhuận
		{"2026-08-31", 1, "2026-09-30"},
		{"2026-11-30", 3, "2027-02-28"}, // qua năm
		{"2026-10-15", 11, "2027-09-15"},
	}
	for _, c := range cases {
		got := installmentDueDate(d(c.first), c.i).Format(consts.DateLayout)
		assert.Equal(t, c.want, got, "first=%s i=%d", c.first, c.i)
	}
}

func TestInstallmentSplitAmounts_SumsToTotal(t *testing.T) {
	amts := installmentSplitAmounts(1000000, 3)
	assert.Equal(t, []float64{333333, 333333, 333334}, amts)
	var sum float64
	for _, a := range installmentSplitAmounts(12345679, 7) {
		sum += a
	}
	assert.Equal(t, float64(12345679), sum)
}

func TestFinanceService_CreateInstallmentPlanRejects(t *testing.T) {
	repo := newFakeFinanceRepo()
	mid := uint(10)
	repo.invoices[2] = &entity.Invoice{ID: 2, BranchID: 1, MemberID: &mid, Total: 5000000, PaidAmount: 5000000, Status: entity.InvoiceStatusPaid}
	repo.invoices[3] = &entity.Invoice{ID: 3, BranchID: 1, MemberID: &mid, Total: 5000000, Status: entity.InvoiceStatusVoid}
	repo.invoices[4] = &entity.Invoice{ID: 4, BranchID: 1, MemberID: &mid, Total: 5000000, PaidAmount: 5000000, RefundedAmount: 5000000, Status: entity.InvoiceStatusRefunded}
	svc := NewFinanceService(repo, &fakeFinanceInvoices{}, einvoice.NewMockProvider(), nil, nil)
	svc.(*financeService).now = func() time.Time { return time.Date(2026, 10, 21, 9, 0, 0, 0, memberVNLoc) }
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	for _, id := range []uint{2, 3, 4} {
		_, err := svc.CreateInstallmentPlan(ctx, &v1.CreateInstallmentPlanReq{ID: id, Installments: 3})
		require.Error(t, err, "invoice %d", id)
	}
	// Ngày đến hạn kỳ đầu ở quá khứ
	_, err := svc.CreateInstallmentPlan(ctx, &v1.CreateInstallmentPlanReq{ID: 1, Installments: 3, FirstDueDate: "2026-10-20"})
	require.Error(t, err)
	assert.Empty(t, repo.plans)
	// Hôm nay thì hợp lệ, tổng kế hoạch = số còn nợ
	plan, err := svc.CreateInstallmentPlan(ctx, &v1.CreateInstallmentPlanReq{ID: 1, Installments: 4, FirstDueDate: "2026-10-21"})
	require.NoError(t, err)
	assert.Equal(t, float64(9000000), plan.Total)
}
