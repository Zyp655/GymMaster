package service

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/payroll/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakePayrollRepo struct {
	nextAttID     uint
	nextPayrollID uint
	nextItemID    uint
	attendances   map[uint]*entity.StaffAttendance
	shifts        map[uint]*entity.Shift
	payrolls      map[uint]*entity.Payroll
	items         map[uint]*entity.PayrollItem
	sources       []repository.StaffPayrollSource
	paidComm      bool
	paidCommCalls int
	// pendingComm: hoa hồng PT PENDING đang gắn vào bảng lương (theo HLV) – mô phỏng trainer_commissions.payroll_id.
	pendingComm map[uint]float64
}

func newFakePayrollRepo() *fakePayrollRepo {
	return &fakePayrollRepo{
		attendances: map[uint]*entity.StaffAttendance{},
		shifts:      map[uint]*entity.Shift{},
		payrolls:    map[uint]*entity.Payroll{},
		items:       map[uint]*entity.PayrollItem{},
	}
}

func (f *fakePayrollRepo) FindOpenAttendanceToday(_ context.Context, staffID uint, dayStart, dayEnd time.Time) (*entity.StaffAttendance, error) {
	for _, a := range f.attendances {
		if a.StaffID == staffID && !a.CheckInAt.Before(dayStart) && a.CheckInAt.Before(dayEnd) && a.CheckOutAt == nil {
			cp := *a
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePayrollRepo) FindShiftInDay(_ context.Context, staffID, branchID uint, dayStart, dayEnd time.Time) (*entity.Shift, error) {
	for _, sh := range f.shifts {
		if sh.StaffID == staffID && sh.BranchID == branchID && !sh.StartAt.Before(dayStart) && sh.StartAt.Before(dayEnd) {
			cp := *sh
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePayrollRepo) FindShiftByID(_ context.Context, id uint) (*entity.Shift, error) {
	if sh, ok := f.shifts[id]; ok {
		cp := *sh
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePayrollRepo) CreateAttendance(_ context.Context, a *entity.StaffAttendance) error {
	f.nextAttID++
	a.ID = f.nextAttID
	cp := *a
	f.attendances[a.ID] = &cp
	return nil
}

func (f *fakePayrollRepo) UpdateAttendance(_ context.Context, a *entity.StaffAttendance) error {
	cp := *a
	f.attendances[a.ID] = &cp
	return nil
}

func (f *fakePayrollRepo) ListAttendances(_ context.Context, _ repository.AttendanceFilter) ([]repository.AttendanceRow, error) {
	var out []repository.AttendanceRow
	for _, a := range f.attendances {
		out = append(out, repository.AttendanceRow{StaffAttendance: *a})
	}
	return out, nil
}

func (f *fakePayrollRepo) FindPayrollByID(_ context.Context, id uint) (*repository.PayrollRow, error) {
	if p, ok := f.payrolls[id]; ok {
		return &repository.PayrollRow{Payroll: *p, BranchName: "CN Quận 1"}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePayrollRepo) FindPayrollByBranchPeriod(_ context.Context, branchID uint, period string) (*repository.PayrollRow, error) {
	for _, p := range f.payrolls {
		if p.BranchID == branchID && p.Period == period {
			return &repository.PayrollRow{Payroll: *p, BranchName: "CN Quận 1"}, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePayrollRepo) ListPayrolls(_ context.Context, _ repository.PayrollFilter) ([]repository.PayrollRow, error) {
	var out []repository.PayrollRow
	for _, p := range f.payrolls {
		out = append(out, repository.PayrollRow{Payroll: *p, BranchName: "CN Quận 1"})
	}
	return out, nil
}

func (f *fakePayrollRepo) CreatePayroll(_ context.Context, p *entity.Payroll) error {
	f.nextPayrollID++
	p.ID = f.nextPayrollID
	cp := *p
	f.payrolls[p.ID] = &cp
	return nil
}

func (f *fakePayrollRepo) UpdatePayroll(_ context.Context, p *entity.Payroll) error {
	cp := *p
	f.payrolls[p.ID] = &cp
	return nil
}

func (f *fakePayrollRepo) ListPayrollItems(_ context.Context, payrollID uint) ([]repository.PayrollItemRow, error) {
	var out []repository.PayrollItemRow
	for _, it := range f.items {
		if it.PayrollID == payrollID {
			out = append(out, repository.PayrollItemRow{
				PayrollItem: *it,
				StaffCode:   "NV001",
				StaffName:   "Lê Văn C",
				Position:    "TRAINER",
			})
		}
	}
	return out, nil
}

func (f *fakePayrollRepo) UpsertPayrollItem(_ context.Context, item *entity.PayrollItem) error {
	for _, existing := range f.items {
		if existing.PayrollID == item.PayrollID && existing.StaffID == item.StaffID {
			existing.WorkDays = item.WorkDays
			existing.PTCommission = item.PTCommission
			existing.SalesCommission = item.SalesCommission
			existing.RecalcTotal()
			*item = *existing
			return nil
		}
	}
	f.nextItemID++
	item.ID = f.nextItemID
	cp := *item
	f.items[item.ID] = &cp
	return nil
}

func (f *fakePayrollRepo) UpdatePayrollItem(_ context.Context, item *entity.PayrollItem) error {
	cp := *item
	f.items[item.ID] = &cp
	return nil
}

func (f *fakePayrollRepo) CollectStaffPayrollSources(context.Context, uint, uint, string, time.Time, time.Time) ([]repository.StaffPayrollSource, error) {
	return f.sources, nil
}

func (f *fakePayrollRepo) MarkTrainerCommissionsPaid(context.Context, uint, *uint, time.Time) error {
	f.paidComm = true
	f.paidCommCalls++
	f.pendingComm = nil
	return nil
}

func (f *fakePayrollRepo) LockPayrollRow(_ context.Context, id uint) (*entity.Payroll, error) {
	if p, ok := f.payrolls[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePayrollRepo) TransitionPayroll(_ context.Context, p *entity.Payroll, from []string) (bool, error) {
	cur, ok := f.payrolls[p.ID]
	if !ok || !slices.Contains(from, cur.Status) {
		return false, nil
	}
	cp := *p
	f.payrolls[p.ID] = &cp
	return true, nil
}

// AttachTrainerCommissions: mặc định gắn đúng PT commission của nguồn (khớp, không lệch).
func (f *fakePayrollRepo) AttachTrainerCommissions(context.Context, uint, uint, string) error {
	f.pendingComm = map[uint]float64{}
	for _, src := range f.sources {
		if src.PTCommission != 0 {
			f.pendingComm[src.StaffID] = src.PTCommission
		}
	}
	return nil
}

func (f *fakePayrollRepo) SumPayrollCommissions(context.Context, uint) (map[uint]float64, error) {
	out := map[uint]float64{}
	for k, v := range f.pendingComm {
		out[k] = v
	}
	return out, nil
}

func (f *fakePayrollRepo) ListPayslipsByStaff(_ context.Context, staffID uint) ([]repository.PayslipRow, error) {
	var out []repository.PayslipRow
	for _, it := range f.items {
		if it.StaffID == staffID {
			p := f.payrolls[it.PayrollID]
			out = append(out, repository.PayslipRow{
				PayrollItemRow: repository.PayrollItemRow{PayrollItem: *it, StaffCode: "NV001", StaffName: "Lê Văn C"},
				BranchID:       p.BranchID,
				Period:         p.Period,
				Status:         p.Status,
				PaidAt:         p.PaidAt,
			})
		}
	}
	return out, nil
}

func TestPayrollService_AttendanceAndPayrollLifecycle(t *testing.T) {
	repo := newFakePayrollRepo()
	shiftStart := time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)
	shiftEnd := time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC)
	repo.shifts[1] = &entity.Shift{
		ID:       1,
		BranchID: 1,
		StaffID:  5,
		StartAt:  shiftStart,
		EndAt:    shiftEnd,
	}
	repo.sources = []repository.StaffPayrollSource{
		{
			StaffID:         5,
			StaffCode:       "NV005",
			StaffName:       "Lê Văn C",
			Position:        "TRAINER",
			BaseSalary:      12000000,
			WorkDays:        24,
			PTCommission:    3000000,
			SalesCommission: 500000,
		},
	}

	svc := NewPayrollService(repo, nil, nil)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 5, Roles: []string{consts.RoleOwner}})

	// 1. Scan check-in at 08:15 -> 15 minutes late
	svc.(*payrollService).now = func() time.Time { return shiftStart.Add(15 * time.Minute) }
	attIn, err := svc.ScanAttendance(ctx, &v1.ScanAttendanceReq{BranchID: 1, Method: "QR"})
	require.NoError(t, err)
	assert.Equal(t, "CHECK_IN", attIn.Action)
	assert.Equal(t, 15, attIn.LateMinutes)
	assert.Nil(t, attIn.CheckOutAt)

	// 2. Second scan in same day at 16:30 -> checkout with 30 early minutes
	svc.(*payrollService).now = func() time.Time { return shiftEnd.Add(-30 * time.Minute) }
	attOut, err := svc.ScanAttendance(ctx, &v1.ScanAttendanceReq{BranchID: 1, Method: "QR"})
	require.NoError(t, err)
	assert.Equal(t, "CHECK_OUT", attOut.Action)
	require.NotNil(t, attOut.CheckOutAt)
	assert.Equal(t, 30, attOut.EarlyMinutes)

	// 3. Generate payroll for 2026-10
	detail, err := svc.GeneratePayroll(ctx, &v1.GeneratePayrollReq{BranchID: 1, Period: "2026-10"})
	require.NoError(t, err)
	assert.Equal(t, entity.PayrollStatusDraft, detail.Status)
	require.Len(t, detail.Items, 1)
	// Lương CB theo ngày công: 12.000.000 × 24/26 = 11.076.923 (+3.000.000 PT +500.000 sales).
	assert.Equal(t, float64(11076923), detail.Items[0].BasePay)
	assert.Equal(t, float64(14576923), detail.Items[0].Total)

	// 4. Update bonus and penalty while DRAFT
	bonus := 1000000.0
	penalty := 200000.0
	updated, err := svc.UpdatePayroll(ctx, &v1.UpdatePayrollReq{
		ID: detail.ID,
		Items: []v1.PayrollItemUpdateInput{
			{StaffID: 5, Bonus: &bonus, Penalty: &penalty},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, float64(15376923), updated.Items[0].Total)

	// 5. Lock payroll -> further UpdatePayroll returns PayrollLocked
	locked, err := svc.LockPayroll(ctx, detail.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.PayrollStatusLocked, locked.Status)

	_, err = svc.UpdatePayroll(ctx, &v1.UpdatePayrollReq{ID: detail.ID})
	assert.Equal(t, rescode.PayrollLocked.Code(), codeOf(err))

	// 6. Pay payroll -> marks PAID and trainer_commissions paid
	paid, err := svc.PayPayroll(ctx, detail.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.PayrollStatusPaid, paid.Status)
	assert.True(t, repo.paidComm)

	// 7. Export CSV & MyPayslips
	exp, err := svc.ExportPayroll(ctx, detail.ID)
	require.NoError(t, err)
	assert.Contains(t, exp.CSV, "Lê Văn C")

	slips, err := svc.MyPayslips(ctx)
	require.NoError(t, err)
	require.Len(t, slips, 1)
	assert.Equal(t, "2026-10", slips[0].Period)
}
