package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/payroll/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

func payrollQA4Ctx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
}

func newPayrollQA4(sources []repository.StaffPayrollSource) (*payrollService, *fakePayrollRepo, *classFakeAudit) {
	repo := newFakePayrollRepo()
	repo.sources = sources
	audit := &classFakeAudit{}
	svc := NewPayrollService(repo, nil, audit).(*payrollService)
	svc.now = func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, memberVNLoc) }
	return svc, repo, audit
}

// PR-01 / FIN-PAY-02: lương cơ bản theo ngày công (công chuẩn 26/tháng, không vượt lương hợp đồng).
func TestPayrollItem_BasePayByWorkDays(t *testing.T) {
	cases := []struct {
		base, days, want float64
	}{
		{8_000_000, 0, 0},
		{8_000_000, 13, 4_000_000},
		{8_000_000, 26, 8_000_000},
		{8_000_000, 30, 8_000_000}, // làm quá công chuẩn không vượt lương CB (tăng ca tính riêng)
		{7_000_000, 1, 269_231},
		{0, 26, 0},
	}
	for _, c := range cases {
		it := entity.PayrollItem{BaseSalary: c.base, WorkDays: c.days, PTCommission: 100, Bonus: 50, Penalty: 20}
		assert.Equal(t, c.want, it.BasePay(), c)
		it.RecalcTotal()
		assert.Equal(t, c.want+130, it.Total)
	}
}

// PR-01 + FIN-PAY-03: tổng hợp lương – 0 ngày công → lương CB 0; NV lương CB 0đ không bị gán 10.000.000.
func TestPayrollGenerate_BaseSalaryNoDefaultAndWorkDays(t *testing.T) {
	svc, _, _ := newPayrollQA4([]repository.StaffPayrollSource{
		{StaffID: 1, BaseSalary: 8_000_000, WorkDays: 0, SalesCommission: 50_000},
		{StaffID: 2, BaseSalary: 0, WorkDays: 20},
		{StaffID: 3, BaseSalary: 5_200_000, WorkDays: 10},
	})
	detail, err := svc.GeneratePayroll(payrollQA4Ctx(), &v1.GeneratePayrollReq{BranchID: 1, Period: "2026-10"})
	require.NoError(t, err)
	got := map[uint]*v1.PayrollLineItem{}
	for _, it := range detail.Items {
		got[it.StaffID] = it
	}
	assert.Equal(t, 50_000.0, got[1].Total, "0 ngày công → chỉ còn hoa hồng")
	assert.Equal(t, 8_000_000.0, got[1].BaseSalary, "lương CB hợp đồng giữ nguyên để hiển thị")
	assert.Equal(t, 0.0, got[2].BaseSalary, "không gán mặc định 10.000.000")
	assert.Equal(t, 0.0, got[2].Total)
	assert.Equal(t, 2_000_000.0, got[3].BasePay)
	assert.Equal(t, 26.0, got[3].StandardWorkDays)
}

// PR-02 / FIN-PAY-05: hoa hồng PT đã chi riêng sau khi tổng hợp → không cho khóa/chi (tránh trả trùng);
// tổng hợp lại thì khớp và chi được, đánh dấu PAID đúng một lần.
func TestPayrollLockPay_RejectsCommissionDrift(t *testing.T) {
	svc, repo, audit := newPayrollQA4([]repository.StaffPayrollSource{
		{StaffID: 5, BaseSalary: 2_600_000, WorkDays: 26, PTCommission: 15_000},
	})
	ctx := payrollQA4Ctx()
	detail, err := svc.GeneratePayroll(ctx, &v1.GeneratePayrollReq{BranchID: 1, Period: "2026-10"})
	require.NoError(t, err)
	assert.Equal(t, 2_615_000.0, detail.Items[0].Total)

	// Hoa hồng bị chi riêng qua /commissions/pay sau khi tổng hợp → không còn PENDING.
	repo.pendingComm = map[uint]float64{}
	_, err = svc.LockPayroll(ctx, detail.ID)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	_, err = svc.PayPayroll(ctx, detail.ID)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.False(t, repo.paidComm)

	// Tổng hợp lại: nguồn không còn hoa hồng PT (đã chi) → khóa + chi được.
	repo.sources[0].PTCommission = 0
	detail, err = svc.GeneratePayroll(ctx, &v1.GeneratePayrollReq{BranchID: 1, Period: "2026-10"})
	require.NoError(t, err)
	assert.Equal(t, 2_600_000.0, detail.Items[0].Total)
	_, err = svc.LockPayroll(ctx, detail.ID)
	require.NoError(t, err)
	_, err = svc.PayPayroll(ctx, detail.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, repo.paidCommCalls)
	assert.Contains(t, audit.actions, "LOCK:payroll")
	assert.Contains(t, audit.actions, "PAY:payroll")
}

// stalePayrollRepo: mô phỏng hai yêu cầu chi lương đồng thời – cả hai đọc thấy bảng lương LOCKED.
type stalePayrollRepo struct {
	*fakePayrollRepo
	snapshot repository.PayrollRow
}

func (r *stalePayrollRepo) FindPayrollByID(context.Context, uint) (*repository.PayrollRow, error) {
	cp := r.snapshot
	return &cp, nil
}

// FIN-PAY-10: chi lương song song 2 lần → chỉ 1 thành công, hoa hồng chỉ đánh dấu một lần.
func TestPayrollPay_ConcurrentOnlyOneWins(t *testing.T) {
	svc, repo, _ := newPayrollQA4([]repository.StaffPayrollSource{{StaffID: 5, BaseSalary: 1_000_000, WorkDays: 26}})
	ctx := payrollQA4Ctx()
	detail, err := svc.GeneratePayroll(ctx, &v1.GeneratePayrollReq{BranchID: 1, Period: "2026-10"})
	require.NoError(t, err)
	_, err = svc.LockPayroll(ctx, detail.ID)
	require.NoError(t, err)

	stale := &stalePayrollRepo{fakePayrollRepo: repo, snapshot: repository.PayrollRow{Payroll: *repo.payrolls[detail.ID]}}
	svc.repo = stale
	_, err1 := svc.PayPayroll(ctx, detail.ID)
	_, err2 := svc.PayPayroll(ctx, detail.ID)
	require.NoError(t, err1)
	require.Error(t, err2)
	assert.Equal(t, rescode.PayrollLocked.Code(), codeOf(err2))
	assert.Equal(t, 1, repo.paidCommCalls)
	assert.Equal(t, entity.PayrollStatusPaid, repo.payrolls[detail.ID].Status)
}
