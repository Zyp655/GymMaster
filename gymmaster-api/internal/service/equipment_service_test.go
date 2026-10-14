package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/equipment/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeEquipmentRepo struct {
	nextEqID  uint
	nextLogID uint
	eqs       map[uint]*entity.Equipment
	logs      map[uint]*entity.MaintenanceLog
}

func newFakeEquipmentRepo() *fakeEquipmentRepo {
	return &fakeEquipmentRepo{
		eqs:  map[uint]*entity.Equipment{},
		logs: map[uint]*entity.MaintenanceLog{},
	}
}

func (f *fakeEquipmentRepo) ListEquipments(_ context.Context, _ dto.Query, filter repository.EquipmentFilter) ([]repository.EquipmentRow, int64, error) {
	var out []repository.EquipmentRow
	for _, eq := range f.eqs {
		if filter.Status != "" && eq.Status != filter.Status {
			continue
		}
		out = append(out, repository.EquipmentRow{Equipment: *eq})
	}
	return out, int64(len(out)), nil
}

func (f *fakeEquipmentRepo) FindEquipmentByID(_ context.Context, id uint) (*repository.EquipmentRow, error) {
	if eq, ok := f.eqs[id]; ok {
		return &repository.EquipmentRow{Equipment: *eq, BranchName: "CN Quận 1"}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeEquipmentRepo) ExistsEquipmentCode(_ context.Context, code string, excludeID uint) (bool, error) {
	for _, eq := range f.eqs {
		if strings.EqualFold(eq.Code, strings.TrimSpace(code)) && eq.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeEquipmentRepo) CreateEquipment(_ context.Context, e *entity.Equipment) error {
	f.nextEqID++
	e.ID = f.nextEqID
	cp := *e
	f.eqs[e.ID] = &cp
	return nil
}

func (f *fakeEquipmentRepo) UpdateEquipment(_ context.Context, e *entity.Equipment) error {
	cp := *e
	f.eqs[e.ID] = &cp
	return nil
}

func (f *fakeEquipmentRepo) DeleteEquipment(_ context.Context, id uint, _ *uint) error {
	delete(f.eqs, id)
	return nil
}

func (f *fakeEquipmentRepo) ListMaintenanceLogs(_ context.Context, _ dto.Query, filter repository.MaintenanceLogFilter) ([]repository.MaintenanceLogRow, int64, error) {
	var out []repository.MaintenanceLogRow
	for _, l := range f.logs {
		if filter.EquipmentID > 0 && l.EquipmentID != filter.EquipmentID {
			continue
		}
		if filter.Status != "" && l.Status != filter.Status {
			continue
		}
		out = append(out, repository.MaintenanceLogRow{MaintenanceLog: *l})
	}
	return out, int64(len(out)), nil
}

func (f *fakeEquipmentRepo) ListEquipmentLogs(_ context.Context, equipmentID uint) ([]repository.MaintenanceLogRow, error) {
	var out []repository.MaintenanceLogRow
	for _, l := range f.logs {
		if l.EquipmentID == equipmentID {
			out = append(out, repository.MaintenanceLogRow{MaintenanceLog: *l})
		}
	}
	return out, nil
}

func (f *fakeEquipmentRepo) FindMaintenanceLogByID(_ context.Context, id uint) (*repository.MaintenanceLogRow, error) {
	if l, ok := f.logs[id]; ok {
		var bid uint
		if eq, ok := f.eqs[l.EquipmentID]; ok {
			bid = eq.BranchID
		}
		return &repository.MaintenanceLogRow{MaintenanceLog: *l, BranchID: bid}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeEquipmentRepo) CreateMaintenanceLog(_ context.Context, l *entity.MaintenanceLog) error {
	f.nextLogID++
	l.ID = f.nextLogID
	l.CreatedAt = time.Now()
	cp := *l
	f.logs[l.ID] = &cp
	return nil
}

func (f *fakeEquipmentRepo) UpdateMaintenanceLog(_ context.Context, l *entity.MaintenanceLog) error {
	cp := *l
	f.logs[l.ID] = &cp
	return nil
}

func (f *fakeEquipmentRepo) DeleteMaintenanceLog(_ context.Context, id uint) error {
	delete(f.logs, id)
	return nil
}

func (f *fakeEquipmentRepo) CountOpenLogsExcluding(_ context.Context, equipmentID, excludeLogID uint) (int64, error) {
	var count int64
	for _, l := range f.logs {
		if l.EquipmentID == equipmentID && l.ID != excludeLogID &&
			(l.Status == entity.MaintenanceStatusOpen || l.Status == entity.MaintenanceStatusInProgress) {
			count++
		}
	}
	return count, nil
}

func (f *fakeEquipmentRepo) MaintenanceSummary(_ context.Context, _ []uint, _, _ *time.Time) (*repository.MaintenanceSummaryData, error) {
	var total float64
	var count int64
	for _, l := range f.logs {
		total += l.Cost
		count++
	}
	return &repository.MaintenanceSummaryData{
		TotalCost: total,
		TotalLogs: count,
	}, nil
}

func (f *fakeEquipmentRepo) ListDueMaintenance(_ context.Context, today time.Time) ([]repository.EquipmentRow, error) {
	todayStr := today.Format(consts.DateLayout)
	var out []repository.EquipmentRow
	for _, eq := range f.eqs {
		if eq.Status == entity.EquipmentStatusActive && eq.NextMaintenanceDate != nil &&
			eq.NextMaintenanceDate.Format(consts.DateLayout) <= todayStr {
			out = append(out, repository.EquipmentRow{Equipment: *eq, BranchName: "CN Quận 1"})
		}
	}
	return out, nil
}

type fakeEquipmentNotifier struct {
	permCalls []struct {
		Perm     string
		BranchID *uint
		Type     string
		Title    string
		Body     string
	}
}

func (n *fakeEquipmentNotifier) Notify(context.Context, []uint, string, string, string, string) error {
	return nil
}

func (n *fakeEquipmentNotifier) NotifyPermission(_ context.Context, perm string, branchID *uint, typ, title, body, _ string) error {
	n.permCalls = append(n.permCalls, struct {
		Perm     string
		BranchID *uint
		Type     string
		Title    string
		Body     string
	}{Perm: perm, BranchID: branchID, Type: typ, Title: title, Body: body})
	return nil
}

func TestEquipmentService_ReportIssueAndCompleteMaintenance(t *testing.T) {
	repo := newFakeEquipmentRepo()
	notif := &fakeEquipmentNotifier{}
	svc := NewEquipmentService(repo, nil, nil, notif)
	fixedNow := time.Date(2026, 10, 14, 10, 0, 0, 0, memberVNLoc)
	svc.(*equipmentService).now = func() time.Time { return fixedNow }

	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	eq, err := svc.CreateEquipment(ctx, v1.EquipmentInput{
		BranchID:                1,
		Code:                    "TM-001",
		Name:                    "Máy chạy bộ LifeFitness",
		Category:                "CARDIO",
		MaintenanceIntervalDays: 30,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EquipmentStatusActive, eq.Status)

	// Duplicate code -> Conflict
	_, err = svc.CreateEquipment(ctx, v1.EquipmentInput{
		BranchID: 1,
		Code:     "tm-001",
		Name:     "Duplicate",
	})
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	// Report issue -> creates REPAIR / OPEN log and sets equipment to BROKEN
	log1, err := svc.ReportIssue(ctx, eq.ID, "Băng tải bị trượt")
	require.NoError(t, err)
	assert.Equal(t, entity.MaintenanceTypeRepair, log1.Type)
	assert.Equal(t, entity.MaintenanceStatusOpen, log1.Status)
	assert.Equal(t, entity.EquipmentStatusBroken, repo.eqs[eq.ID].Status)

	// Add second open log
	log2, err := svc.CreateMaintenanceLog(ctx, v1.MaintenanceLogInput{
		EquipmentID: eq.ID,
		Type:        entity.MaintenanceTypePeriodic,
		Description: "Tra dầu định kỳ",
		Status:      entity.MaintenanceStatusInProgress,
	})
	require.NoError(t, err)

	// Complete log1 while log2 is still IN_PROGRESS -> equipment stays BROKEN
	_, err = svc.UpdateMaintenanceLog(ctx, log1.ID, v1.MaintenanceLogInput{
		Status: entity.MaintenanceStatusDone,
		Cost:   500000,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EquipmentStatusBroken, repo.eqs[eq.ID].Status)

	// Complete log2 (no more open logs) -> equipment transitions back to ACTIVE and NextMaintenanceDate = today + 30 days
	_, err = svc.UpdateMaintenanceLog(ctx, log2.ID, v1.MaintenanceLogInput{
		Status: entity.MaintenanceStatusDone,
		Cost:   200000,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EquipmentStatusActive, repo.eqs[eq.ID].Status)
	require.NotNil(t, repo.eqs[eq.ID].NextMaintenanceDate)
	assert.Equal(t, "2026-11-13", repo.eqs[eq.ID].NextMaintenanceDate.Format(consts.DateLayout))
}

func TestEquipmentService_RemindDueMaintenance(t *testing.T) {
	repo := newFakeEquipmentRepo()
	notif := &fakeEquipmentNotifier{}
	svc := NewEquipmentService(repo, nil, nil, notif)

	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
	_, err := svc.CreateEquipment(ctx, v1.EquipmentInput{
		BranchID:                1,
		Code:                    "TM-002",
		Name:                    "Xe đạp tập",
		NextMaintenanceDate:     "2026-10-10",
		MaintenanceIntervalDays: 60,
	})
	require.NoError(t, err)

	now := time.Date(2026, 10, 14, 7, 0, 0, 0, memberVNLoc)
	n, err := svc.RemindDueMaintenance(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	require.Len(t, notif.permCalls, 1)
	assert.Equal(t, consts.PermEquipmentManage, notif.permCalls[0].Perm)
	assert.Equal(t, NotifyMaintenanceDue, notif.permCalls[0].Type)
}
