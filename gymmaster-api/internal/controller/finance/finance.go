package finance

import (
	"context"

	v1 "gymmaster-api/api/finance/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.FinanceService
}

type MemberControllerV1 struct {
	svc service.FinanceService
}

func NewAdminV1(svc service.FinanceService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.FinanceService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

func (c *AdminControllerV1) ListExpenseCategories(ctx context.Context, _ *v1.ListExpenseCategoriesReq) (res v1.ListExpenseCategoriesRes, err error) {
	return c.svc.ListExpenseCategories(ctx)
}

func (c *AdminControllerV1) CreateExpenseCategory(ctx context.Context, req *v1.CreateExpenseCategoryReq) (res *v1.CreateExpenseCategoryRes, err error) {
	return c.svc.CreateExpenseCategory(ctx, req)
}

func (c *AdminControllerV1) ListExpenses(ctx context.Context, req *v1.ListExpensesReq) (res v1.ListExpensesRes, err error) {
	return c.svc.ListExpenses(ctx, req)
}

func (c *AdminControllerV1) CreateExpense(ctx context.Context, req *v1.CreateExpenseReq) (res *v1.CreateExpenseRes, err error) {
	return c.svc.CreateExpense(ctx, req)
}

func (c *AdminControllerV1) UpdateExpense(ctx context.Context, req *v1.UpdateExpenseReq) (res *v1.UpdateExpenseRes, err error) {
	return c.svc.UpdateExpense(ctx, req)
}

func (c *AdminControllerV1) DeleteExpense(ctx context.Context, req *v1.DeleteExpenseReq) (res *v1.DeleteExpenseRes, err error) {
	if err := c.svc.DeleteExpense(ctx, req.ID); err != nil {
		return nil, err
	}
	return &v1.DeleteExpenseRes{}, nil
}

func (c *AdminControllerV1) ApproveExpense(ctx context.Context, req *v1.ApproveExpenseReq) (res *v1.ApproveExpenseRes, err error) {
	return c.svc.ApproveExpense(ctx, req)
}

func (c *AdminControllerV1) OpenCashSession(ctx context.Context, req *v1.OpenCashSessionReq) (res *v1.OpenCashSessionRes, err error) {
	return c.svc.OpenCashSession(ctx, req)
}

func (c *AdminControllerV1) CloseCashSession(ctx context.Context, req *v1.CloseCashSessionReq) (res *v1.CloseCashSessionRes, err error) {
	return c.svc.CloseCashSession(ctx, req)
}

func (c *AdminControllerV1) ListCashSessions(ctx context.Context, req *v1.ListCashSessionsReq) (res v1.ListCashSessionsRes, err error) {
	return c.svc.ListCashSessions(ctx, req)
}

func (c *AdminControllerV1) ProfitLossReport(ctx context.Context, req *v1.ProfitLossReportReq) (res *v1.ProfitLossReportRes, err error) {
	return c.svc.ProfitLossReport(ctx, req)
}

func (c *AdminControllerV1) CreateInstallmentPlan(ctx context.Context, req *v1.CreateInstallmentPlanReq) (res *v1.CreateInstallmentPlanRes, err error) {
	return c.svc.CreateInstallmentPlan(ctx, req)
}

func (c *AdminControllerV1) ListInstallments(ctx context.Context, req *v1.ListInstallmentsReq) (res v1.ListInstallmentsRes, err error) {
	return c.svc.ListInstallments(ctx, req)
}

func (c *AdminControllerV1) PayInstallmentSchedule(ctx context.Context, req *v1.PayInstallmentScheduleReq) (res *v1.PayInstallmentScheduleRes, err error) {
	return c.svc.PayInstallmentSchedule(ctx, req)
}

func (c *AdminControllerV1) IssueEInvoice(ctx context.Context, req *v1.IssueEInvoiceReq) (res *v1.IssueEInvoiceRes, err error) {
	return c.svc.IssueEInvoice(ctx, req.ID)
}

func (c *MemberControllerV1) MemberInstallments(ctx context.Context, _ *v1.MemberInstallmentsReq) (res v1.MemberInstallmentsRes, err error) {
	return c.svc.MemberInstallments(ctx)
}
