package service

import (
	"context"
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
	pnl           *repository.ProfitLossData
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

func (f *fakeFinanceRepo) UpdateCashSession(_ context.Context, cs *entity.CashSession) error {
	cp := *cs
	f.sessions[cs.ID] = &cp
	return nil
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

	svc := NewFinanceService(repo, nil, einvoice.NewMockProvider(), nil, nil)
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

	// 2. Open & Close Cash Session -> expected_cash = 2,000,000 + 5,500,000 = 7,500,000
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
	assert.Equal(t, float64(7500000), closed.ExpectedCash)
	assert.Equal(t, float64(-50000), closed.Difference)
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

	// Pay 1st installment
	paidSched, err := svc.PayInstallmentSchedule(ctx, &v1.PayInstallmentScheduleReq{
		ID:     plan.Schedules[0].ID,
		Method: "CASH",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.InstallmentPaid, paidSched.Status)

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
