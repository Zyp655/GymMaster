package service

import (
	"context"
	"sync"
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

// staleFindFinanceRepo mô phỏng hai yêu cầu đóng ca đồng thời: cả hai đều đọc thấy ca OPEN
// (FindCashSessionByID trả bản chụp lúc mở), nhưng chỉ một UPDATE có điều kiện thành công.
type staleFindFinanceRepo struct {
	*fakeFinanceRepo
	mu       sync.Mutex
	snapshot map[uint]entity.CashSession
}

func (r *staleFindFinanceRepo) FindCashSessionByID(_ context.Context, id uint) (*repository.CashSessionRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &repository.CashSessionRow{CashSession: r.snapshot[id]}, nil
}

func (r *staleFindFinanceRepo) CloseCashSession(ctx context.Context, cs *entity.CashSession) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fakeFinanceRepo.CloseCashSession(ctx, cs)
}

func TestFinanceService_CloseCashSession_ConcurrentOnlyOneWins(t *testing.T) {
	base := newFakeFinanceRepo()
	opened := entity.CashSession{ID: 7, BranchID: 1, StaffID: 1, OpeningCash: 1_000_000, Status: entity.CashSessionOpen,
		OpenedAt: time.Date(2026, 10, 21, 7, 0, 0, 0, memberVNLoc)}
	cp := opened
	base.sessions[7] = &cp
	repo := &staleFindFinanceRepo{fakeFinanceRepo: base, snapshot: map[uint]entity.CashSession{7: opened}}
	svc := NewFinanceService(repo, &fakeFinanceInvoices{}, einvoice.NewMockProvider(), nil, nil)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	var wg sync.WaitGroup
	results := make([]error, 4)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, results[i] = svc.CloseCashSession(ctx, &v1.CloseCashSessionReq{ID: 7, CountedCash: float64(1_000_000 + i*1000)})
		}(i)
	}
	wg.Wait()

	ok, conflicts := 0, 0
	for _, err := range results {
		if err == nil {
			ok++
			continue
		}
		assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
		conflicts++
	}
	assert.Equal(t, 1, ok, "chỉ một yêu cầu đóng ca thành công")
	assert.Equal(t, 3, conflicts, "các yêu cầu còn lại nhận 409")
	assert.Equal(t, entity.CashSessionClosed, base.sessions[7].Status)
}

func TestFinanceService_ExpenseSpentAtNotInFuture(t *testing.T) {
	repo := newFakeFinanceRepo()
	svc := NewFinanceService(repo, &fakeFinanceInvoices{}, einvoice.NewMockProvider(), nil, nil)
	svc.(*financeService).now = func() time.Time { return time.Date(2026, 10, 21, 23, 30, 0, 0, memberVNLoc) }
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	cat, err := svc.CreateExpenseCategory(ctx, &v1.CreateExpenseCategoryReq{Name: "Điện", Type: entity.ExpenseTypeVariable})
	require.NoError(t, err)

	_, err = svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 100_000, SpentAt: "2030-01-01"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sau ngày hôm nay")
	_, err = svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 100_000, SpentAt: "2026-10-22"})
	require.Error(t, err, "ngày mai (giờ VN) bị từ chối")

	exp, err := svc.CreateExpense(ctx, &v1.CreateExpenseReq{BranchID: 1, CategoryID: cat.ID, Amount: 100_000, SpentAt: "2026-10-21"})
	require.NoError(t, err, "hôm nay hợp lệ")
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, SpentAt: "2030-01-01"})
	require.Error(t, err)
	_, err = svc.UpdateExpense(ctx, &v1.UpdateExpenseReq{ID: exp.ID, SpentAt: "2026-09-30"})
	require.NoError(t, err)
}
