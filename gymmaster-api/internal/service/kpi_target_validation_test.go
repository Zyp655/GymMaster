package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kpiv1 "gymmaster-api/api/kpi/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// mockKPIRepo (kpi_scheduleext_service_test.go): mọi chi nhánh/nhân viên đều hợp lệ.
func (m *mockKPIRepo) BranchExists(context.Context, uint) (bool, error) { return true, nil }
func (m *mockKPIRepo) BranchActive(context.Context, uint) (bool, error) { return true, nil }
func (m *mockKPIRepo) StaffBranchStatus(context.Context, uint, uint) (bool, bool, error) {
	return true, true, nil
}

// kpiScopedRepo: chi nhánh/nhân viên theo bảng + thực tế KPI riêng từng chi nhánh.
type kpiScopedRepo struct {
	*mockKPIRepo
	branches map[uint]bool
	staff    map[uint]map[uint]bool // staffID → branchID → true; nil map = không phải nhân viên
	actualBy map[uint]repository.KPIActuals
	inactive map[uint]bool // chi nhánh tồn tại nhưng ngừng hoạt động
}

func (r *kpiScopedRepo) BranchExists(_ context.Context, id uint) (bool, error) {
	return r.branches[id], nil
}
func (r *kpiScopedRepo) BranchActive(_ context.Context, id uint) (bool, error) {
	return r.branches[id] && !r.inactive[id], nil
}
func (r *kpiScopedRepo) StaffBranchStatus(_ context.Context, staffID, branchID uint) (bool, bool, error) {
	b, ok := r.staff[staffID]
	return ok, ok && b[branchID], nil
}
func (r *kpiScopedRepo) ComputeKPIActuals(_ context.Context, branchID uint, _, _ time.Time) (repository.KPIActuals, error) {
	if branchID == 0 {
		var sum repository.KPIActuals
		for _, a := range r.actualBy {
			sum.Revenue += a.Revenue
			sum.NewMembers += a.NewMembers
			sum.Renewals += a.Renewals
			sum.PTSalesAmount += a.PTSalesAmount
			sum.PTSalesCount += a.PTSalesCount
		}
		return sum, nil
	}
	return r.actualBy[branchID], nil
}

func newKPIScopedService() (*kpiService, *kpiScopedRepo) {
	repo := &kpiScopedRepo{
		mockKPIRepo: newMockKPIRepo(),
		branches:    map[uint]bool{1: true, 2: true},
		staff:       map[uint]map[uint]bool{20: {1: true}}, // nhân viên 20 thuộc chi nhánh 1
		actualBy: map[uint]repository.KPIActuals{
			1: {Revenue: 100_000_000, NewMembers: 10},
			2: {Revenue: 300_000_000, NewMembers: 30},
		},
	}
	svc := NewKPIService(repo, nil).(*kpiService)
	svc.now = func() time.Time { return time.Date(2026, 10, 15, 10, 0, 0, 0, memberVNLoc) }
	return svc, repo
}

func TestKPIService_CreateSalesTargetValidatesBranchAndStaff(t *testing.T) {
	svc, _ := newKPIScopedService()
	owner := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	req := func(branch uint, staff *uint) *kpiv1.CreateSalesTargetReq {
		return &kpiv1.CreateSalesTargetReq{BranchID: branch, StaffID: staff, Period: "2026-10",
			Metric: entity.KPIMetricRevenue, TargetValue: 1_000_000}
	}
	ptr := func(v uint) *uint { return &v }

	_, err := svc.CreateSalesTarget(owner, req(99999, nil))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "chi nhánh không tồn tại/đã xóa")
	_, err = svc.CreateSalesTarget(owner, req(1, ptr(555)))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "staff_id không phải nhân viên (hội viên / không tồn tại)")
	_, err = svc.CreateSalesTarget(owner, req(2, ptr(20)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không thuộc chi nhánh")
	_, err = svc.CreateSalesTarget(owner, req(1, ptr(20)))
	require.NoError(t, err, "nhân viên thuộc đúng chi nhánh")
	_, err = svc.CreateSalesTarget(owner, req(1, ptr(0)))
	require.NoError(t, err, "staff_id = 0 → chỉ tiêu chi nhánh")
}

func TestKPIService_ReportUnknownBranch404(t *testing.T) {
	svc, _ := newKPIScopedService()
	owner := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	_, err := svc.KPIReport(owner, &kpiv1.KPIReportReq{BranchID: 99999})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
	_, err = svc.CohortReport(owner, &kpiv1.CohortReportReq{BranchID: 99999})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
	_, err = svc.ListSalesTargets(owner, &kpiv1.ListSalesTargetsReq{BranchID: 99999})
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

// Toàn hệ thống: chỉ so thực tế của các chi nhánh CÓ chỉ tiêu; chỉ số chưa đặt chỉ tiêu → has_target = false.
func TestKPIService_SystemWideComparesOnlyBranchesWithTargets(t *testing.T) {
	svc, repo := newKPIScopedService()
	owner := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	repo.targets[1] = &repository.SalesTargetRow{SalesTarget: entity.SalesTarget{ID: 1, BranchID: 1, Period: "2026-10",
		Metric: entity.KPIMetricRevenue, TargetValue: 200_000_000}}
	// chỉ tiêu cá nhân không cộng vào chỉ tiêu chi nhánh
	staff := uint(20)
	repo.targets[2] = &repository.SalesTargetRow{SalesTarget: entity.SalesTarget{ID: 2, BranchID: 1, StaffID: &staff,
		Period: "2026-10", Metric: entity.KPIMetricRevenue, TargetValue: 50_000_000}}

	res, err := svc.KPIReport(owner, &kpiv1.KPIReportReq{Period: "2026-10"})
	require.NoError(t, err)
	byMetric := map[string]*kpiv1.KPIMetricComparison{}
	for _, m := range res.Metrics {
		byMetric[m.Metric] = m
	}
	rev := byMetric[entity.KPIMetricRevenue]
	assert.True(t, rev.HasTarget)
	assert.Equal(t, 1, rev.TargetBranchCount)
	assert.Equal(t, float64(100_000_000), rev.Actual, "chỉ cộng chi nhánh 1 (có chỉ tiêu), không cộng chi nhánh 2")
	assert.Equal(t, float64(200_000_000), rev.TargetValue)
	assert.Equal(t, 50.0, rev.AchievementPct)

	nm := byMetric[entity.KPIMetricNewMembers]
	assert.False(t, nm.HasTarget)
	assert.Equal(t, float64(40), nm.Actual, "chưa đặt chỉ tiêu → thực tế toàn hệ thống")
	assert.Equal(t, 0.0, nm.AchievementPct)
}

// D5 (FIN-RPT-12): đặt chỉ tiêu cho chi nhánh ngừng hoạt động → 400.
func TestKPIService_CreateSalesTargetRejectsInactiveBranch(t *testing.T) {
	svc, repo := newKPIScopedService()
	repo.branches[3] = true
	repo.inactive = map[uint]bool{3: true}
	owner := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	_, err := svc.CreateSalesTarget(owner, &kpiv1.CreateSalesTargetReq{BranchID: 3, Period: "2026-10",
		Metric: entity.KPIMetricRevenue, TargetValue: 1000})
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	assert.Contains(t, err.Error(), "ngừng hoạt động")
}
