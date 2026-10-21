package service

import (
	"context"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"gorm.io/gorm"

	kpiv1 "gymmaster-api/api/kpi/v1"
	schedulev1 "gymmaster-api/api/scheduleext/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// =================================================================================
// Mock KPIRepository
// =================================================================================

type mockKPIRepo struct {
	targets    map[uint]*repository.SalesTargetRow
	nextID     uint
	occupancy  []repository.BranchOccupancyRow
	actuals    repository.KPIActuals
	cohorts    []repository.CohortMemberRow
	activeMap  map[uint]map[string]bool
	lastBranch uint
}

func newMockKPIRepo() *mockKPIRepo {
	return &mockKPIRepo{
		targets:   make(map[uint]*repository.SalesTargetRow),
		nextID:    1,
		activeMap: make(map[uint]map[string]bool),
	}
}

func (m *mockKPIRepo) ListBranchOccupancy(_ context.Context, _ time.Time) ([]repository.BranchOccupancyRow, error) {
	return m.occupancy, nil
}

func (m *mockKPIRepo) ListSalesTargets(_ context.Context, _ dto.Query, branchID uint, period, metric string) ([]repository.SalesTargetRow, int64, error) {
	m.lastBranch = branchID
	var out []repository.SalesTargetRow
	for _, r := range m.targets {
		if branchID > 0 && r.BranchID != branchID {
			continue
		}
		if period != "" && r.Period != period {
			continue
		}
		if metric != "" && r.Metric != metric {
			continue
		}
		out = append(out, *r)
	}
	return out, int64(len(out)), nil
}

func (m *mockKPIRepo) ListTargetsForPeriod(_ context.Context, branchID uint, period string) ([]entity.SalesTarget, error) {
	m.lastBranch = branchID
	var out []entity.SalesTarget
	for _, r := range m.targets {
		if r.Period != period {
			continue
		}
		if branchID > 0 && r.BranchID != branchID {
			continue
		}
		out = append(out, r.SalesTarget)
	}
	return out, nil
}

func (m *mockKPIRepo) FindSalesTargetByID(_ context.Context, id uint) (*repository.SalesTargetRow, error) {
	r, ok := m.targets[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *r
	return &cp, nil
}

func (m *mockKPIRepo) UpsertSalesTarget(_ context.Context, st *entity.SalesTarget) error {
	if st.ID == 0 {
		st.ID = m.nextID
		m.nextID++
	}
	m.targets[st.ID] = &repository.SalesTargetRow{
		SalesTarget: *st,
		BranchName:  "Cầu Giấy",
	}
	return nil
}

func (m *mockKPIRepo) DeleteSalesTarget(_ context.Context, id uint) error {
	delete(m.targets, id)
	return nil
}

func (m *mockKPIRepo) ComputeKPIActuals(_ context.Context, branchID uint, _, _ time.Time) (repository.KPIActuals, error) {
	m.lastBranch = branchID
	return m.actuals, nil
}

func (m *mockKPIRepo) ListCohortMembers(_ context.Context, branchID uint, _ time.Time) ([]repository.CohortMemberRow, error) {
	m.lastBranch = branchID
	return m.cohorts, nil
}

func (m *mockKPIRepo) ListActiveMemberMonths(_ context.Context, _ []uint, _ time.Time) (map[uint]map[string]bool, error) {
	return m.activeMap, nil
}

// =================================================================================
// Mock ScheduleExtRepository
// =================================================================================

type mockScheduleExtRepo struct {
	availabilities []repository.TrainerAvailabilityRow
	leaves         map[uint]*repository.LeaveRequestRow
	classes        map[uint]*repository.ImpactedClassRow
	ptSessions     map[uint]*repository.ImpactedPTSessionRow
	holidays       map[uint]*repository.BranchHolidayRow
	nextID         uint
	extendedCount  int64
	// ineligible: trainer id → không đủ điều kiện (không tồn tại / khóa / không phải HLV / khác chi nhánh).
	ineligible map[uint]bool
	// busy: trainer id → "PT" | "CLASS" (HLV đang bận).
	busy map[uint]string
	// lastReplaceBranch: branch truyền vào ReplaceTrainerAvailability gần nhất.
	lastReplaceBranch uint
}

func newMockScheduleExtRepo() *mockScheduleExtRepo {
	return &mockScheduleExtRepo{
		leaves:     make(map[uint]*repository.LeaveRequestRow),
		classes:    make(map[uint]*repository.ImpactedClassRow),
		ptSessions: make(map[uint]*repository.ImpactedPTSessionRow),
		holidays:   make(map[uint]*repository.BranchHolidayRow),
		nextID:     1,
	}
}

func (m *mockScheduleExtRepo) ListTrainerAvailability(_ context.Context, trainerID, branchID uint) ([]repository.TrainerAvailabilityRow, error) {
	var out []repository.TrainerAvailabilityRow
	for _, a := range m.availabilities {
		if a.TrainerID == trainerID && (branchID == 0 || a.BranchID == branchID) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *mockScheduleExtRepo) ReplaceTrainerAvailability(_ context.Context, trainerID uint, branchID uint, slots []entity.TrainerAvailability) error {
	m.lastReplaceBranch = branchID
	kept := m.availabilities[:0:0]
	for _, a := range m.availabilities {
		if a.TrainerID != trainerID || a.BranchID != branchID {
			kept = append(kept, a)
		}
	}
	m.availabilities = kept
	for i, sl := range slots {
		sl.ID = uint(len(m.availabilities) + i + 1)
		m.availabilities = append(m.availabilities, repository.TrainerAvailabilityRow{
			TrainerAvailability: sl,
			BranchName:          "Cầu Giấy",
		})
	}
	return nil
}

func (m *mockScheduleExtRepo) ListLeaveRequests(_ context.Context, _ dto.Query, staffID, branchID uint, status string) ([]repository.LeaveRequestRow, int64, error) {
	var out []repository.LeaveRequestRow
	for _, lr := range m.leaves {
		if staffID > 0 && lr.StaffID != staffID {
			continue
		}
		if branchID > 0 && (lr.BranchID == nil || *lr.BranchID != branchID) {
			continue
		}
		if status != "" && lr.Status != status {
			continue
		}
		out = append(out, *lr)
	}
	return out, int64(len(out)), nil
}

func (m *mockScheduleExtRepo) FindLeaveRequestByID(_ context.Context, id uint) (*repository.LeaveRequestRow, error) {
	lr, ok := m.leaves[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *lr
	return &cp, nil
}

func (m *mockScheduleExtRepo) CreateLeaveRequest(_ context.Context, lr *entity.StaffLeaveRequest) error {
	lr.ID = m.nextID
	m.nextID++
	m.leaves[lr.ID] = &repository.LeaveRequestRow{
		StaffLeaveRequest: *lr,
		StaffName:         "HLV Minh Tuấn",
	}
	return nil
}

func (m *mockScheduleExtRepo) UpdateLeaveRequest(_ context.Context, lr *entity.StaffLeaveRequest) error {
	row, ok := m.leaves[lr.ID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	row.StaffLeaveRequest = *lr
	return nil
}

func (m *mockScheduleExtRepo) ListImpactedClassSchedules(_ context.Context, trainerID uint, _, _ time.Time) ([]repository.ImpactedClassRow, error) {
	var out []repository.ImpactedClassRow
	for _, c := range m.classes {
		if c.TrainerID == trainerID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (m *mockScheduleExtRepo) ListImpactedPTSessions(_ context.Context, trainerID uint, _, _ time.Time) ([]repository.ImpactedPTSessionRow, error) {
	var out []repository.ImpactedPTSessionRow
	for _, s := range m.ptSessions {
		if s.TrainerID == trainerID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *mockScheduleExtRepo) FindClassScheduleByID(_ context.Context, id uint) (*repository.ImpactedClassRow, error) {
	c, ok := m.classes[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *c
	return &cp, nil
}

func (m *mockScheduleExtRepo) SetClassSubstituteTrainer(_ context.Context, scheduleID uint, subTrainerID *uint, _ string, _ *uint) error {
	c, ok := m.classes[scheduleID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	c.SubstituteTrainerID = subTrainerID
	c.SubstituteTrainerName = "HLV Dạy Thay"
	return nil
}

func (m *mockScheduleExtRepo) FindPTSessionByID(_ context.Context, id uint) (*repository.ImpactedPTSessionRow, error) {
	s, ok := m.ptSessions[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *mockScheduleExtRepo) ReassignPTSessionTrainer(_ context.Context, sessionID, newTrainerID, origTrainerID uint) error {
	s, ok := m.ptSessions[sessionID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	s.TrainerID = newTrainerID
	s.OriginalTrainerID = &origTrainerID
	return nil
}

func (m *mockScheduleExtRepo) TrainerEligible(_ context.Context, trainerID, _ uint) (bool, error) {
	return !m.ineligible[trainerID], nil
}

func (m *mockScheduleExtRepo) TrainerBusy(_ context.Context, trainerID uint, _, _ time.Time, _, _ uint) (string, error) {
	return m.busy[trainerID], nil
}

func (m *mockScheduleExtRepo) ListScheduledClassesBetween(_ context.Context, branchID *uint, fromAt, toAt time.Time) ([]repository.ImpactedClassRow, error) {
	var out []repository.ImpactedClassRow
	for _, c := range m.classes {
		if c.Status == entity.ClassScheduleScheduled && c.StartAt.Before(toAt) && c.EndAt.After(fromAt) &&
			(branchID == nil || c.BranchID == *branchID) {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (m *mockScheduleExtRepo) ListScheduledPTSessionsBetween(_ context.Context, branchID *uint, fromAt, toAt time.Time) ([]repository.ImpactedPTSessionRow, error) {
	var out []repository.ImpactedPTSessionRow
	for _, p := range m.ptSessions {
		if p.Status == entity.PTSessionScheduled && p.StartAt.Before(toAt) && p.EndAt.After(fromAt) &&
			(branchID == nil || p.BranchID == *branchID) {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (m *mockScheduleExtRepo) UpdateBranchHoliday(_ context.Context, h *entity.BranchHoliday) error {
	row, ok := m.holidays[h.ID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	row.BranchHoliday = *h
	return nil
}

func (m *mockScheduleExtRepo) ListBranchHolidays(_ context.Context, _ dto.Query, _ uint, _ int) ([]repository.BranchHolidayRow, int64, error) {
	var out []repository.BranchHolidayRow
	for _, h := range m.holidays {
		out = append(out, *h)
	}
	return out, int64(len(out)), nil
}

func (m *mockScheduleExtRepo) ListUpcomingHolidays(_ context.Context, _ uint, _ time.Time) ([]repository.BranchHolidayRow, error) {
	var out []repository.BranchHolidayRow
	for _, h := range m.holidays {
		out = append(out, *h)
	}
	return out, nil
}

func (m *mockScheduleExtRepo) FindBranchHolidayByID(_ context.Context, id uint) (*repository.BranchHolidayRow, error) {
	h, ok := m.holidays[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *h
	return &cp, nil
}

func (m *mockScheduleExtRepo) CreateBranchHoliday(_ context.Context, h *entity.BranchHoliday) error {
	h.ID = m.nextID
	m.nextID++
	m.holidays[h.ID] = &repository.BranchHolidayRow{
		BranchHoliday: *h,
		BranchName:    "Cầu Giấy",
	}
	return nil
}

func (m *mockScheduleExtRepo) DeleteBranchHoliday(_ context.Context, id uint) error {
	delete(m.holidays, id)
	return nil
}

func (m *mockScheduleExtRepo) ExtendActiveTimeSubscriptions(_ context.Context, _ *uint, _ time.Time) (int64, error) {
	return m.extendedCount, nil
}

// =================================================================================
// Tests for KPIService & ScheduleExtService
// =================================================================================

func TestKPIService_OccupancyTargetsKPIAndCohort(t *testing.T) {
	repo := newMockKPIRepo()
	repo.occupancy = []repository.BranchOccupancyRow{
		{BranchID: 1, BranchCode: "HN-CG", BranchName: "Cầu Giấy", Capacity: 100, PresentCount: 92},
		{BranchID: 2, BranchCode: "HN-TX", BranchName: "Thanh Xuân", Capacity: 100, PresentCount: 20},
	}
	svc := NewKPIService(repo, nil).(*kpiService)
	fixedNow := time.Date(2026, 10, 15, 10, 0, 0, 0, memberVNLoc)
	svc.now = func() time.Time { return fixedNow }

	// 1. BranchOccupancy levels
	occ, err := svc.BranchOccupancy(context.Background())
	if err != nil {
		t.Fatalf("BranchOccupancy failed: %v", err)
	}
	if len(occ) != 2 || occ[0].Level != "FULL" || occ[1].Level != "QUIET" {
		t.Fatalf("unexpected occupancy levels: %+v", occ)
	}

	// 2. Branch-scoped manager cannot create target for another branch
	branch1ManagerCtx := reqctx.WithAuth(context.Background(), &token.Payload{
		UserID:    10,
		Roles:     []string{consts.RoleManager},
		BranchIDs: []uint{1},
	})
	_, err = svc.CreateSalesTarget(branch1ManagerCtx, &kpiv1.CreateSalesTargetReq{
		BranchID:    2,
		Period:      "2026-10",
		Metric:      entity.KPIMetricRevenue,
		TargetValue: 100_000_000,
	})
	if err == nil || gerror.Code(err) != rescode.PermissionDenied {
		t.Fatalf("expected PermissionDenied for cross-branch target creation, got %v", err)
	}

	// 3. Create valid target for branch 1
	created, err := svc.CreateSalesTarget(branch1ManagerCtx, &kpiv1.CreateSalesTargetReq{
		BranchID:    1,
		Period:      "2026-10",
		Metric:      entity.KPIMetricRevenue,
		TargetValue: 200_000_000,
	})
	if err != nil || created.ID == 0 {
		t.Fatalf("CreateSalesTarget failed: %v", err)
	}

	// 4. KPIReport computes achievement percentage accurately
	repo.actuals = repository.KPIActuals{
		Revenue:       150_000_000,
		NewMembers:    25,
		Renewals:      12,
		PTSalesAmount: 40_000_000,
		PTSalesCount:  8,
	}
	report, err := svc.KPIReport(branch1ManagerCtx, &kpiv1.KPIReportReq{Period: "2026-10"})
	if err != nil {
		t.Fatalf("KPIReport failed: %v", err)
	}
	if report.BranchID != 1 || len(report.Metrics) != 4 {
		t.Fatalf("unexpected KPIReport response: %+v", report)
	}
	if report.Metrics[0].Metric != entity.KPIMetricRevenue || report.Metrics[0].AchievementPct != 75.0 {
		t.Fatalf("expected 75%% revenue achievement, got %+v", report.Metrics[0])
	}

	// 5. CohortReport computes retention matrix
	repo.cohorts = []repository.CohortMemberRow{
		{ID: 101, JoinedMonth: "2026-09"},
		{ID: 102, JoinedMonth: "2026-09"},
	}
	repo.activeMap = map[uint]map[string]bool{
		101: {"2026-09": true, "2026-10": true},
		102: {"2026-09": true, "2026-10": false},
	}
	cohortRes, err := svc.CohortReport(branch1ManagerCtx, &kpiv1.CohortReportReq{Months: 2})
	if err != nil {
		t.Fatalf("CohortReport failed: %v", err)
	}
	if len(cohortRes.Cohorts) != 2 || cohortRes.Cohorts[0].Size != 2 || cohortRes.Cohorts[0].RetentionByMonth[1] != 50.0 {
		t.Fatalf("unexpected cohort retention: %+v", cohortRes.Cohorts[0])
	}
}

func TestScheduleExtService_AvailabilityLeaveSubstitutionAndHoliday(t *testing.T) {
	repo := newMockScheduleExtRepo()
	repo.extendedCount = 14
	svc := NewScheduleExtService(repo, nil, nil).(*scheduleExtService)
	fixedNow := time.Date(2026, 10, 15, 9, 0, 0, 0, memberVNLoc)
	svc.now = func() time.Time { return fixedNow }

	ownerCtx := reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 1,
		Roles:  []string{consts.RoleOwner},
	})
	branch1Ctx := reqctx.WithAuth(context.Background(), &token.Payload{
		UserID:    2,
		Roles:     []string{consts.RoleManager},
		BranchIDs: []uint{1},
	})

	// 1. SaveTrainerAvailability validates time order & branch permission
	_, err := svc.SaveTrainerAvailability(branch1Ctx, &schedulev1.SaveTrainerAvailabilityReq{
		ID:       5,
		BranchID: 2,
		Slots: []schedulev1.TrainerAvailabilitySlotInput{
			{DOW: 1, StartTime: "08:00", EndTime: "12:00"},
		},
	})
	if err == nil || gerror.Code(err) != rescode.PermissionDenied {
		t.Fatalf("expected PermissionDenied saving availability for branch 2, got %v", err)
	}

	avail, err := svc.SaveTrainerAvailability(branch1Ctx, &schedulev1.SaveTrainerAvailabilityReq{
		ID:       5,
		BranchID: 1,
		Slots: []schedulev1.TrainerAvailabilitySlotInput{
			{DOW: 1, StartTime: "08:00", EndTime: "12:00"},
		},
	})
	if err != nil || len(avail) != 1 {
		t.Fatalf("SaveTrainerAvailability failed: %v", err)
	}

	// 2. CreateLeaveRequest & ApproveLeaveRequest
	b1 := uint(1)
	lr, err := svc.CreateLeaveRequest(branch1Ctx, &schedulev1.CreateLeaveRequestReq{
		StaffID:  5,
		BranchID: &b1,
		FromAt:   "2026-10-20",
		ToAt:     "2026-10-21",
		Reason:   "Nghỉ phép cá nhân",
	})
	if err != nil || lr.Status != entity.LeaveStatusPending {
		t.Fatalf("CreateLeaveRequest failed: %v", err)
	}

	approved, err := svc.ApproveLeaveRequest(branch1Ctx, &schedulev1.ApproveLeaveRequestReq{
		ID:         lr.ID,
		ReviewNote: "Đã bố trí HLV dạy thay",
	})
	if err != nil || approved.Status != entity.LeaveStatusApproved {
		t.Fatalf("ApproveLeaveRequest failed: %v", err)
	}

	// 3. SubstituteClassTrainer & ReassignPTSession enforce branch scope
	repo.classes[10] = &repository.ImpactedClassRow{
		ClassSchedule: entity.ClassSchedule{ID: 10, BranchID: 2, TrainerID: 5, Status: entity.ClassScheduleScheduled,
			StartAt: fixedNow.Add(24 * time.Hour), EndAt: fixedNow.Add(25 * time.Hour)},
		ClassTypeName: "HIIT",
	}
	_, err = svc.SubstituteClassTrainer(branch1Ctx, &schedulev1.SubstituteClassTrainerReq{
		ID:                  10,
		SubstituteTrainerID: 6,
	})
	if err == nil || gerror.Code(err) != rescode.PermissionDenied {
		t.Fatalf("expected PermissionDenied substituting class on branch 2, got %v", err)
	}

	subClass, err := svc.SubstituteClassTrainer(ownerCtx, &schedulev1.SubstituteClassTrainerReq{
		ID:                  10,
		SubstituteTrainerID: 6,
		Note:                "Dạy thay buổi sáng",
	})
	if err != nil || subClass.SubstituteTrainerID == nil || *subClass.SubstituteTrainerID != 6 {
		t.Fatalf("SubstituteClassTrainer failed: %v", err)
	}

	// 4. CreateBranchHoliday with ExtendSubs = true
	holiday, err := svc.CreateBranchHoliday(ownerCtx, &schedulev1.CreateBranchHolidayReq{
		BranchID:   &b1,
		Date:       "2026-09-02",
		Name:       "Quốc khánh 2/9",
		ExtendSubs: true,
	})
	if err != nil || holiday.ExtendedSubscriptions != 14 {
		t.Fatalf("CreateBranchHoliday failed or did not extend subscriptions: %+v, err=%v", holiday, err)
	}
}
