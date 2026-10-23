package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/finance/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/einvoice"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

func newFinanceQA4Service(t *testing.T) (*financeService, *fakeFinanceRepo, *classFakeAudit, context.Context) {
	t.Helper()
	repo := newFakeFinanceRepo()
	audit := &classFakeAudit{}
	svc := NewFinanceService(repo, &fakeFinanceInvoices{}, einvoice.NewMockProvider(), nil, audit).(*financeService)
	svc.now = func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, memberVNLoc) }
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	return svc, repo, audit, ctx
}

// D1 (FIN-EXP-07…11): mọi thao tác khoản chi đều có audit; khoản đã duyệt không sửa số tiền/không xóa;
// không đổi thẳng APPROVED ↔ REJECTED.
func TestFinanceQA4_ExpenseLifecycleAuditAndGuards(t *testing.T) {
	svc, repo, audit, ctx := newFinanceQA4Service(t)
	cat, err := svc.CreateExpenseCategory(ctx, &v1.CreateExpenseCategoryReq{Name: "Điện", Type: entity.ExpenseTypeVariable})
	require.NoError(t, err)

	exp, err := svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 1_000_000})
	require.NoError(t, err)
	desc := "Sửa mô tả"
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, Description: &desc})
	require.NoError(t, err)
	_, err = svc.ApproveExpense(ctx, &v1.ApproveExpenseReq{ID: exp.ID, Status: entity.ExpenseStatusApproved})
	require.NoError(t, err)

	// Đã duyệt: không sửa số tiền / danh mục / ngày chi, không xóa, không từ chối/duyệt lại.
	amt := 2_000_000.0
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, Amount: &amt})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, SpentAt: "2026-10-01"})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.Equal(t, rescode.Conflict.Code(), codeOf(svc.DeleteExpense(ctx, exp.ID)))
	_, err = svc.ApproveExpense(ctx, &v1.ApproveExpenseReq{ID: exp.ID, Status: entity.ExpenseStatusRejected})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.Equal(t, entity.ExpenseStatusApproved, repo.expenses[exp.ID].Status)
	assert.Equal(t, 1_000_000.0, repo.expenses[exp.ID].Amount)
	// Sửa mô tả khoản đã duyệt vẫn được (không ảnh hưởng số liệu).
	desc2 := "Ghi chú kế toán"
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, Description: &desc2})
	require.NoError(t, err)

	// Khoản bị từ chối → không duyệt thẳng; sửa số tiền → quay về chờ duyệt → duyệt được; xóa khoản chờ duyệt được.
	exp2, err := svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 500_000})
	require.NoError(t, err)
	_, err = svc.ApproveExpense(ctx, &v1.ApproveExpenseReq{ID: exp2.ID, Status: entity.ExpenseStatusRejected})
	require.NoError(t, err)
	_, err = svc.ApproveExpense(ctx, &v1.ApproveExpenseReq{ID: exp2.ID, Status: entity.ExpenseStatusApproved})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	amt2 := 450_000.0
	edited, err := svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp2.ID, Amount: &amt2})
	require.NoError(t, err)
	assert.Equal(t, entity.ExpenseStatusPending, edited.Status)
	require.NoError(t, svc.DeleteExpense(ctx, exp2.ID))
	_, still := repo.expenses[exp2.ID]
	assert.False(t, still)

	assert.Equal(t, []string{
		"CREATE:expense", "UPDATE:expense", "UPDATE:expense", // tạo, sửa mô tả, duyệt
		"UPDATE:expense",                                                       // sửa mô tả khoản đã duyệt
		"CREATE:expense", "UPDATE:expense", "UPDATE:expense", "DELETE:expense", // từ chối, sửa, xóa
	}, audit.actions)
}

// Duyệt và từ chối đồng thời (cả hai đọc thấy PENDING) → chỉ một thắng nhờ UPDATE có điều kiện.
func TestFinanceQA4_ApproveExpense_StaleReadLoses(t *testing.T) {
	svc, repo, _, ctx := newFinanceQA4Service(t)
	repo.expenses[9] = &entity.Expense{ID: 9, BranchID: 1, CategoryID: 1, Amount: 100_000, Status: entity.ExpenseStatusPending}
	repo.categories[1] = &entity.ExpenseCategory{ID: 1, Name: "X"}
	// Yêu cầu khác đã duyệt trước khi UPDATE của yêu cầu này chạy.
	stale := &staleExpenseRepo{fakeFinanceRepo: repo, snapshot: *repo.expenses[9]}
	repo.expenses[9].Status = entity.ExpenseStatusApproved
	svc.repo = stale
	_, err := svc.ApproveExpense(ctx, &v1.ApproveExpenseReq{ID: 9, Status: entity.ExpenseStatusRejected})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.Equal(t, entity.ExpenseStatusApproved, repo.expenses[9].Status)
}

type staleExpenseRepo struct {
	*fakeFinanceRepo
	snapshot entity.Expense
}

func (r *staleExpenseRepo) FindExpenseByID(context.Context, uint) (*repository.ExpenseRow, error) {
	return &repository.ExpenseRow{Expense: r.snapshot}, nil
}

// D5 (FIN-EXP-04/05): ngày chi quá cũ và chi nhánh ngừng hoạt động bị chặn.
func TestFinanceQA4_ExpenseDateAndBranchGuards(t *testing.T) {
	svc, _, _, ctx := newFinanceQA4Service(t)
	cat, err := svc.CreateExpenseCategory(ctx, &v1.CreateExpenseCategoryReq{Name: "Nước"})
	require.NoError(t, err)

	_, err = svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 1000, SpentAt: "1999-01-01"})
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Contains(t, err.Error(), "12 tháng")
	_, err = svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 1000, SpentAt: "2025-10-21"})
	require.Error(t, err, "quá 12 tháng")
	_, err = svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 1000, SpentAt: "2025-10-22"})
	require.NoError(t, err, "đúng 12 tháng vẫn hợp lệ")

	_, err = svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 3, CategoryID: cat.ID, Amount: 1000})
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Contains(t, err.Error(), "ngừng hoạt động")
	_, err = svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 99, CategoryID: cat.ID, Amount: 1000})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

// D4 (FIN-CS-02/03): không mở ca ở chi nhánh ngừng hoạt động; ghi chú > 500 ký tự báo lỗi cụ thể.
func TestFinanceQA4_CashSessionGuards(t *testing.T) {
	svc, _, _, ctx := newFinanceQA4Service(t)
	_, err := svc.OpenCashSession(ctx, &v1.OpenCashSessionReq{BranchID: 3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ngừng hoạt động")

	_, err = svc.OpenCashSession(ctx, &v1.OpenCashSessionReq{BranchID: 1, Note: strings.Repeat("ă", 501)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500 ký tự")
	sess, err := svc.OpenCashSession(ctx, &v1.OpenCashSessionReq{BranchID: 1, Note: strings.Repeat("ă", 500)})
	require.NoError(t, err)
	_, err = svc.CloseCashSession(ctx, &v1.CloseCashSessionReq{ID: sess.ID, Note: strings.Repeat("x", 600)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500 ký tự")
}

// D6 (FIN-RPT-08): P&L với branch_id không tồn tại → 404.
func TestFinanceQA4_ProfitLossUnknownBranch404(t *testing.T) {
	svc, repo, _, ctx := newFinanceQA4Service(t)
	repo.pnl = &repository.ProfitLossData{}
	_, err := svc.ProfitLossReport(ctx, &v1.ProfitLossReportReq{BranchID: 999999})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
	_, err = svc.ProfitLossReport(ctx, &v1.ProfitLossReportReq{BranchID: 3})
	require.NoError(t, err, "chi nhánh ngừng hoạt động vẫn xem được báo cáo lịch sử")
	_, err = svc.ProfitLossReport(ctx, &v1.ProfitLossReportReq{})
	require.NoError(t, err)
}
