package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	roomv1 "gymmaster-api/api/room/v1"
	shiftv1 "gymmaster-api/api/shift/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// ---- room -------------------------------------------------------------------

type fakeRoomRepo struct {
	nextID uint
	rooms  map[uint]*entity.Room
	future map[uint]int64
	maxCap map[uint]int
}

func newFakeRoomRepo() *fakeRoomRepo {
	return &fakeRoomRepo{rooms: map[uint]*entity.Room{}, future: map[uint]int64{}, maxCap: map[uint]int{}}
}

func (f *fakeRoomRepo) List(context.Context, dto.Query, []uint) ([]entity.Room, int64, error) {
	return nil, 0, nil
}
func (f *fakeRoomRepo) FindByID(_ context.Context, id uint) (*entity.Room, error) {
	if r, ok := f.rooms[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRoomRepo) ExistsName(_ context.Context, branchID uint, name string, excludeID uint) (bool, error) {
	for _, r := range f.rooms {
		if r.BranchID == branchID && strings.EqualFold(r.Name, name) && r.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeRoomRepo) BranchExists(_ context.Context, id uint) (bool, error) {
	return id == 1 || id == 2, nil
}
func (f *fakeRoomRepo) Create(_ context.Context, r *entity.Room) error {
	f.nextID++
	r.ID = f.nextID
	cp := *r
	f.rooms[r.ID] = &cp
	return nil
}
func (f *fakeRoomRepo) Update(_ context.Context, r *entity.Room) error {
	cp := *r
	f.rooms[r.ID] = &cp
	return nil
}
func (f *fakeRoomRepo) Delete(_ context.Context, id uint, _ *uint) error {
	delete(f.rooms, id)
	return nil
}
func (f *fakeRoomRepo) CountFutureSchedules(_ context.Context, id uint, _ time.Time) (int64, error) {
	return f.future[id], nil
}
func (f *fakeRoomRepo) ActiveOptions(context.Context, []uint) ([]entity.Room, error) { return nil, nil }
func (f *fakeRoomRepo) MaxFutureScheduleCapacity(_ context.Context, id uint, _ time.Time) (int, error) {
	return f.maxCap[id], nil
}

func TestRoom_NameUniquePerBranch(t *testing.T) {
	svc := NewRoomService(newFakeRoomRepo())
	ctx := ownerCtx()

	_, err := svc.Create(ctx, roomv1.RoomInput{BranchID: 1, Name: "Phòng Yoga", Capacity: 20, Status: "ACTIVE"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, roomv1.RoomInput{BranchID: 1, Name: " phòng YOGA ", Capacity: 10, Status: "ACTIVE"})
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "trùng tên (không phân biệt hoa thường) trong cùng chi nhánh")

	_, err = svc.Create(ctx, roomv1.RoomInput{BranchID: 2, Name: "Phòng Yoga", Capacity: 10, Status: "ACTIVE"})
	require.NoError(t, err, "chi nhánh khác được trùng tên")

	_, err = svc.Create(ctx, roomv1.RoomInput{BranchID: 9, Name: "X", Capacity: 10, Status: "ACTIVE"})
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestRoom_BranchScope(t *testing.T) {
	repo := newFakeRoomRepo()
	svc := NewRoomService(repo)
	other, err := svc.Create(ownerCtx(), roomv1.RoomInput{BranchID: 2, Name: "P2", Capacity: 5, Status: "ACTIVE"})
	require.NoError(t, err)

	_, err = svc.Create(managerCtx(), roomv1.RoomInput{BranchID: 2, Name: "P", Capacity: 5, Status: "ACTIVE"})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	_, err = svc.Get(managerCtx(), other.ID)
	require.Error(t, err)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

func TestRoom_DeleteBlockedByFutureSchedules(t *testing.T) {
	repo := newFakeRoomRepo()
	svc := NewRoomService(repo)
	r, err := svc.Create(ownerCtx(), roomv1.RoomInput{BranchID: 1, Name: "P1", Capacity: 5, Status: "ACTIVE"})
	require.NoError(t, err)

	repo.future[r.ID] = 1
	err = svc.Delete(ownerCtx(), r.ID)
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	repo.future[r.ID] = 0
	require.NoError(t, svc.Delete(ownerCtx(), r.ID))
	assert.Empty(t, repo.rooms)
}

func TestRoom_UpdateBlockedByUpcomingSchedules(t *testing.T) {
	repo := newFakeRoomRepo()
	svc := NewRoomService(repo)
	r, err := svc.Create(ownerCtx(), roomv1.RoomInput{BranchID: 1, Name: "P1", Capacity: 20, Status: "ACTIVE"})
	require.NoError(t, err)
	repo.future[r.ID] = 2
	repo.maxCap[r.ID] = 15

	for name, in := range map[string]roomv1.RoomInput{
		"đổi chi nhánh":          {BranchID: 2, Name: "P1", Capacity: 20, Status: "ACTIVE"},
		"ngừng hoạt động":        {BranchID: 1, Name: "P1", Capacity: 20, Status: "INACTIVE"},
		"giảm dưới sức chứa lớp": {BranchID: 1, Name: "P1", Capacity: 10, Status: "ACTIVE"},
	} {
		_, err := svc.Update(ownerCtx(), r.ID, in)
		require.Error(t, err, name)
		assert.Equal(t, rescode.Conflict.Code(), codeOf(err), name)
	}
	assert.Equal(t, 20, repo.rooms[r.ID].Capacity)

	// Giảm nhưng vẫn ≥ sức chứa lớp lớn nhất, đổi tên → được.
	upd, err := svc.Update(ownerCtx(), r.ID, roomv1.RoomInput{BranchID: 1, Name: "P1 mới", Capacity: 15, Status: "ACTIVE"})
	require.NoError(t, err)
	assert.Equal(t, 15, upd.Capacity)

	// Hết lịch sắp tới → chuyển chi nhánh được.
	repo.future[r.ID], repo.maxCap[r.ID] = 0, 0
	_, err = svc.Update(ownerCtx(), r.ID, roomv1.RoomInput{BranchID: 2, Name: "P1 mới", Capacity: 5, Status: "ACTIVE"})
	require.NoError(t, err)
}

// ---- shift ------------------------------------------------------------------

type fakeShiftRepo struct {
	nextID  uint
	shifts  map[uint]*entity.Shift
	members map[uint][]uint // staffID -> branchIDs
}

func newFakeShiftRepo() *fakeShiftRepo {
	return &fakeShiftRepo{shifts: map[uint]*entity.Shift{}, members: map[uint][]uint{10: {1}, 11: {1}, 12: {2}}}
}

func (f *fakeShiftRepo) overlaps(s *entity.Shift) bool {
	for _, x := range f.shifts {
		if x.ID != s.ID && x.StaffID == s.StaffID && x.StartAt.Before(s.EndAt) && s.StartAt.Before(x.EndAt) {
			return true
		}
	}
	return false
}

func (f *fakeShiftRepo) List(_ context.Context, flt repository.ShiftFilter) ([]entity.ShiftDetail, error) {
	out := []entity.ShiftDetail{}
	for _, s := range f.shifts {
		if s.StartAt.Before(flt.To) && s.EndAt.After(flt.From) {
			out = append(out, entity.ShiftDetail{Shift: *s})
		}
	}
	return out, nil
}
func (f *fakeShiftRepo) FindByID(_ context.Context, id uint) (*entity.ShiftDetail, error) {
	if s, ok := f.shifts[id]; ok {
		return &entity.ShiftDetail{Shift: *s}, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeShiftRepo) Create(_ context.Context, s *entity.Shift) error {
	if f.overlaps(s) {
		return &pgconn.PgError{Code: "23P01"}
	}
	f.nextID++
	s.ID = f.nextID
	cp := *s
	f.shifts[s.ID] = &cp
	return nil
}
func (f *fakeShiftRepo) CreateIfFree(ctx context.Context, s *entity.Shift) (bool, error) {
	if f.overlaps(s) {
		return false, nil
	}
	return true, f.Create(ctx, s)
}
func (f *fakeShiftRepo) Update(_ context.Context, s *entity.Shift) error {
	if f.overlaps(s) {
		return &pgconn.PgError{Code: "23P01"}
	}
	cp := *s
	f.shifts[s.ID] = &cp
	return nil
}
func (f *fakeShiftRepo) Delete(_ context.Context, id uint) error {
	delete(f.shifts, id)
	return nil
}
func (f *fakeShiftRepo) StaffInBranch(_ context.Context, staffID, branchID uint) (bool, error) {
	for _, b := range f.members[staffID] {
		if b == branchID {
			return true, nil
		}
	}
	return false, nil
}

func shiftIn(staffID uint, start, end string) shiftv1.ShiftInput {
	return shiftv1.ShiftInput{BranchID: 1, StaffID: staffID, StartAt: start, EndAt: end}
}

func TestShift_CreateRules(t *testing.T) {
	repo := newFakeShiftRepo()
	svc := NewShiftService(repo, noTx{})
	ctx := ownerCtx()

	sh, err := svc.Create(ctx, shiftIn(10, "2026-10-12T06:00:00+07:00", "2026-10-12T14:00:00+07:00"))
	require.NoError(t, err)
	assert.Equal(t, uint(10), sh.StaffID)

	_, err = svc.Create(ctx, shiftIn(10, "2026-10-12T13:00", "2026-10-12T18:00"))
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.Contains(t, err.Error(), "ca trùng giờ")

	_, err = svc.Create(ctx, shiftIn(11, "2026-10-12T13:00", "2026-10-12T18:00"))
	require.NoError(t, err, "nhân viên khác cùng giờ thì được")

	_, err = svc.Create(ctx, shiftIn(12, "2026-10-13T06:00", "2026-10-13T10:00"))
	require.Error(t, err, "nhân viên chi nhánh 2 không xếp được ca chi nhánh 1")
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	_, err = svc.Create(ctx, shiftIn(10, "2026-10-13T10:00", "2026-10-13T09:00"))
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	_, err = svc.Create(ctx, shiftIn(10, "2026-10-13T00:00", "2026-10-14T00:00"))
	require.Error(t, err, "ca dài quá 16 giờ")

	_, err = svc.Create(managerCtx(), shiftv1.ShiftInput{BranchID: 2, StaffID: 12, StartAt: "2026-10-13T06:00", EndAt: "2026-10-13T10:00"})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}

func TestShift_UpdateOverlap(t *testing.T) {
	repo := newFakeShiftRepo()
	svc := NewShiftService(repo, noTx{})
	ctx := ownerCtx()
	_, err := svc.Create(ctx, shiftIn(10, "2026-10-12T06:00", "2026-10-12T10:00"))
	require.NoError(t, err)
	b, err := svc.Create(ctx, shiftIn(10, "2026-10-12T12:00", "2026-10-12T16:00"))
	require.NoError(t, err)

	_, err = svc.Update(ctx, b.ID, shiftIn(10, "2026-10-12T09:00", "2026-10-12T16:00"))
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	_, err = svc.Update(ctx, b.ID, shiftIn(10, "2026-10-12T11:00", "2026-10-12T17:00"))
	require.NoError(t, err, "sửa ca không được tính là trùng với chính nó")
}

func TestShift_CopyWeek(t *testing.T) {
	repo := newFakeShiftRepo()
	svc := NewShiftService(repo, noTx{})
	ctx := ownerCtx()
	_, err := svc.Create(ctx, shiftIn(10, "2026-10-12T06:00", "2026-10-12T14:00")) // thứ Hai
	require.NoError(t, err)
	_, err = svc.Create(ctx, shiftIn(11, "2026-10-18T14:00", "2026-10-18T22:00")) // Chủ nhật
	require.NoError(t, err)
	// Tuần đích đã có ca trùng giờ của NV 10 → bị bỏ qua.
	_, err = svc.Create(ctx, shiftIn(10, "2026-10-19T08:00", "2026-10-19T09:00"))
	require.NoError(t, err)

	res, err := svc.CopyWeek(ctx, &shiftv1.CopyWeekReq{BranchID: 1, FromWeekStart: "2026-10-12", ToWeekStart: "2026-10-19"})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Created)
	assert.Equal(t, 1, res.Skipped)

	found := false
	for _, s := range repo.shifts {
		if s.StaffID == 11 && s.StartAt.Equal(time.Date(2026, 10, 25, 7, 0, 0, 0, time.UTC)) {
			found = true
		}
	}
	assert.True(t, found, "ca Chủ nhật 14:00 (+07) phải được chép sang 25/10")

	_, err = svc.CopyWeek(ctx, &shiftv1.CopyWeekReq{BranchID: 1, FromWeekStart: "2026-10-13", ToWeekStart: "2026-10-19"})
	require.Error(t, err, "ngày đầu tuần phải là thứ Hai")
	_, err = svc.CopyWeek(ctx, &shiftv1.CopyWeekReq{BranchID: 1, FromWeekStart: "2026-10-12", ToWeekStart: "2026-10-12"})
	require.Error(t, err)
	_, err = svc.CopyWeek(managerCtx(), &shiftv1.CopyWeekReq{BranchID: 2, FromWeekStart: "2026-10-12", ToWeekStart: "2026-10-19"})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}

func TestShift_ListValidatesRange(t *testing.T) {
	svc := NewShiftService(newFakeShiftRepo(), noTx{})
	_, err := svc.List(ownerCtx(), &shiftv1.ListReq{From: "2026-10-18", To: "2026-10-12"})
	require.Error(t, err)
	_, err = svc.List(ownerCtx(), &shiftv1.ListReq{From: "2026-01-01", To: "2026-12-31"})
	require.Error(t, err)
	_, err = svc.List(managerCtx(), &shiftv1.ListReq{BranchID: 2, From: "2026-10-12", To: "2026-10-18"})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}
