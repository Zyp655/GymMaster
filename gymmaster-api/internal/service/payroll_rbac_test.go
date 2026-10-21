package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/payroll/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

// fakePayrollRepoWithStaff bổ sung năng lực kiểm tra nhân viên – chi nhánh.
type fakePayrollRepoWithStaff struct {
	*fakePayrollRepo
	staffBranches map[uint][]uint // staffID -> chi nhánh
}

func (f *fakePayrollRepoWithStaff) StaffBranchStatus(_ context.Context, staffID, branchID uint) (bool, bool, error) {
	bs, ok := f.staffBranches[staffID]
	if !ok {
		return false, false, nil
	}
	for _, b := range bs {
		if b == branchID {
			return true, true, nil
		}
	}
	return true, false, nil
}

func TestPayrollScanAttendance_ValidatesStaff(t *testing.T) {
	repo := &fakePayrollRepoWithStaff{fakePayrollRepo: newFakePayrollRepo(), staffBranches: map[uint][]uint{7: {1}, 8: {2}}}
	svc := NewPayrollService(repo, nil, nil)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleManager}, BranchIDs: []uint{1}})

	_, err := svc.ScanAttendance(ctx, &v1.ScanAttendanceReq{BranchID: 1, StaffID: 999999})
	require.Error(t, err)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err), "nhân viên không tồn tại → 404, không phải 500")

	_, err = svc.ScanAttendance(ctx, &v1.ScanAttendanceReq{BranchID: 1, StaffID: 8})
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), "nhân viên chi nhánh khác")

	_, err = svc.ScanAttendance(ctx, &v1.ScanAttendanceReq{BranchID: 2, StaffID: 8})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err), "quản lý không được chấm công ở chi nhánh khác")

	res, err := svc.ScanAttendance(ctx, &v1.ScanAttendanceReq{BranchID: 1, StaffID: 7})
	require.NoError(t, err)
	assert.Equal(t, "CHECK_IN", res.Action)
}

func TestPayrollGenerate_RejectsFuturePeriod(t *testing.T) {
	svc := NewPayrollService(newFakePayrollRepo(), nil, nil)
	svc.(*payrollService).now = func() time.Time { return time.Date(2026, 10, 21, 10, 0, 0, 0, memberVNLoc) }
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})

	for _, p := range []string{"2099-01", "2026-11"} {
		_, err := svc.GeneratePayroll(ctx, &v1.GeneratePayrollReq{BranchID: 1, Period: p})
		require.Error(t, err, p)
		assert.Equal(t, rescode.BadRequest.Code(), codeOf(err), p)
	}
	_, err := svc.GeneratePayroll(ctx, &v1.GeneratePayrollReq{BranchID: 1, Period: "2026-10"})
	require.NoError(t, err)
}
