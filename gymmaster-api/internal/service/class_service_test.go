package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/class/v1"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

type classFixture struct {
	*bookingFixture
	sched *classScheduleService
	audit *classFakeAudit
}

func newClassFixture() *classFixture {
	bf := newBookingFixture()
	audit := &classFakeAudit{}
	sched := NewClassScheduleService(classFakeScheduleRepo{bf.store}, bookingFakeRepo{bf.store}, noTx{}, audit, bf.notifier).(*classScheduleService)
	sched.now = bf.svc.now
	return &classFixture{bookingFixture: bf, sched: sched, audit: audit}
}

// classAt: giờ Việt Nam ngày d (tháng 10/2026), hh:mm.
func (f *classFixture) classAt(day, hh, mm int) time.Time {
	return time.Date(2026, 10, day, hh, mm, 0, 0, f.sched.loc)
}

func classCreateReq(start string) *v1.CreateScheduleReq {
	return &v1.CreateScheduleReq{ClassTypeID: 1, BranchID: 1, RoomID: 1, TrainerID: 10, StartAt: start}
}

// ---- Create -------------------------------------------------------------------

func TestClassScheduleCreate_Defaults(t *testing.T) {
	f := newClassFixture()
	res, err := f.sched.Create(classOwnerCtx(), classCreateReq("2026-10-15T18:00:00+07:00"))
	require.NoError(t, err)
	assert.Equal(t, 60, res.DurationMin, "thời lượng mặc định theo loại lớp")
	assert.Equal(t, 15, res.Capacity, "sức chứa mặc định = min(20 của loại lớp, 15 của phòng)")
	assert.Equal(t, entity.ClassScheduleScheduled, res.Status)
	assert.Equal(t, 15, res.AvailableSlots)
	assert.Contains(t, f.audit.actions, "CREATE:class_schedule")
}

func TestClassScheduleCreate_Rules(t *testing.T) {
	cases := []struct {
		name string
		mod  func(f *classFixture, r *v1.CreateScheduleReq)
		ctx  context.Context
		code int
		msg  string
	}{
		{"vượt sức chứa phòng", func(_ *classFixture, r *v1.CreateScheduleReq) { r.Capacity = 16 }, nil, rescode.BadRequest.Code(), "sức chứa phòng"},
		{"trước giờ mở cửa", func(_ *classFixture, r *v1.CreateScheduleReq) { r.StartAt = "2026-10-15T04:30:00+07:00" }, nil, rescode.BadRequest.Code(), "giờ mở cửa"},
		{"kết thúc sau giờ đóng cửa", func(_ *classFixture, r *v1.CreateScheduleReq) { r.StartAt = "2026-10-15T21:30:00+07:00" }, nil, rescode.BadRequest.Code(), "giờ mở cửa"},
		{"HLV không thuộc chi nhánh", func(_ *classFixture, r *v1.CreateScheduleReq) { r.TrainerID = 12 }, nil, rescode.BadRequest.Code(), "HLV"},
		{"loại lớp ngừng hoạt động", func(_ *classFixture, r *v1.CreateScheduleReq) { r.ClassTypeID = 2 }, nil, rescode.BadRequest.Code(), "ngừng hoạt động"},
		{"phòng khác chi nhánh", func(_ *classFixture, r *v1.CreateScheduleReq) { r.RoomID = 3 }, nil, rescode.BadRequest.Code(), "Phòng không thuộc"},
		{"lịch trong quá khứ", func(_ *classFixture, r *v1.CreateScheduleReq) { r.StartAt = "2026-10-14T08:00:00+07:00" }, nil, rescode.BadRequest.Code(), "quá khứ"},
		{"nhân viên khác chi nhánh", func(_ *classFixture, r *v1.CreateScheduleReq) { r.BranchID, r.RoomID, r.TrainerID = 2, 3, 12 }, managerCtx(), rescode.PermissionDenied.Code(), ""},
		{"HLV trùng buổi PT", func(f *classFixture, r *v1.CreateScheduleReq) {
			f.store.pts = append(f.store.pts, classFakePT{trainerID: 10, start: f.classAt(15, 18, 30), end: f.classAt(15, 19, 30)})
		}, nil, rescode.Conflict.Code(), "buổi PT"},
		{"trùng phòng", func(f *classFixture, r *v1.CreateScheduleReq) {
			sc := f.addSchedule(f.classAt(15, 17, 30), 5)
			sc.TrainerID = 11
		}, nil, rescode.Conflict.Code(), "Phòng đã có lớp"},
		{"trùng HLV", func(f *classFixture, r *v1.CreateScheduleReq) {
			sc := f.addSchedule(f.classAt(15, 18, 30), 5)
			sc.RoomID = 2
		}, nil, rescode.Conflict.Code(), "HLV đã có lớp"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newClassFixture()
			r := classCreateReq("2026-10-15T18:00:00+07:00")
			c.mod(f, r)
			ctx := c.ctx
			if ctx == nil {
				ctx = classOwnerCtx()
			}
			_, err := f.sched.Create(ctx, r)
			require.Error(t, err)
			assert.Equal(t, c.code, codeOf(err))
			if c.msg != "" {
				assert.Contains(t, err.Error(), c.msg)
			}
		})
	}
}

// ---- Recurring ------------------------------------------------------------------

func TestClassScheduleRecurring(t *testing.T) {
	f := newClassFixture()
	// Lớp đã có ở phòng 1 vào thứ Hai 19/10 18:00 → buổi ngày đó bị bỏ qua.
	busy := f.addSchedule(f.classAt(19, 18, 0), 5)
	busy.TrainerID = 11
	// HLV 10 có buổi PT thứ Tư 21/10.
	f.store.pts = append(f.store.pts, classFakePT{trainerID: 10, start: f.classAt(21, 18, 0), end: f.classAt(21, 19, 0)})

	req := &v1.CreateRecurringReq{
		ClassTypeID: 1, BranchID: 1, RoomID: 1, TrainerID: 10,
		StartDate: "2026-10-12", EndDate: "2026-10-25", Weekdays: []int{1, 3}, StartTime: "18:00", DurationMin: 60, Capacity: 10,
	}
	res, err := f.sched.CreateRecurring(classOwnerCtx(), req)
	require.NoError(t, err)
	// Thứ Hai/Tư: 12 (đã qua), 14 (18:00 hôm nay – chưa qua), 19 (trùng phòng), 21 (trùng PT) → tạo 14/10.
	assert.Equal(t, 1, res.Created)
	require.Len(t, res.Skipped, 3)
	assert.Equal(t, "2026-10-12", res.Skipped[0].Date)
	assert.Equal(t, classPastMsg, res.Skipped[0].Reason)
	assert.Equal(t, "2026-10-19", res.Skipped[1].Date)
	assert.Equal(t, classRoomOverlapMsg, res.Skipped[1].Reason)
	assert.Equal(t, "2026-10-21", res.Skipped[2].Date)
	assert.Equal(t, classTrainerPTMsg, res.Skipped[2].Reason)
	assert.NotEmpty(t, res.SeriesID)
	created := f.store.schedules[res.ScheduleIDs[0]]
	assert.Equal(t, res.SeriesID, *created.SeriesID)
	assert.Equal(t, 10, created.Capacity)
	assert.True(t, created.StartAt.Equal(f.classAt(14, 18, 0)))
	assert.True(t, created.EndAt.Equal(f.classAt(14, 19, 0)))
}

func TestClassScheduleRecurring_Validation(t *testing.T) {
	f := newClassFixture()
	base := v1.CreateRecurringReq{
		ClassTypeID: 1, BranchID: 1, RoomID: 1, TrainerID: 10,
		StartDate: "2026-10-15", EndDate: "2027-01-06", Weekdays: []int{1}, StartTime: "18:00",
	}
	r := base
	_, err := f.sched.CreateRecurring(classOwnerCtx(), &r)
	require.NoError(t, err, "đúng 12 tuần (84 ngày) được phép")

	r = base
	r.EndDate = "2027-01-07"
	_, err = f.sched.CreateRecurring(classOwnerCtx(), &r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "12 tuần")

	r = base
	r.Weekdays = []int{0}
	_, err = f.sched.CreateRecurring(classOwnerCtx(), &r)
	require.Error(t, err)

	r = base
	r.StartTime = "21:30"
	_, err = f.sched.CreateRecurring(classOwnerCtx(), &r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "giờ mở cửa")
}

// ---- Update -------------------------------------------------------------------

func TestClassScheduleUpdate_Capacity(t *testing.T) {
	f := newClassFixture()
	sc := f.addSchedule(f.classAt(15, 18, 0), 1)
	for id := uint(1); id <= 3; id++ {
		f.addMember(id)
		_, err := f.svc.Book(classStaffCtx(), sc.ID, id, "")
		require.NoError(t, err)
	}
	upd := &v1.UpdateScheduleReq{ID: sc.ID, RoomID: 1, TrainerID: 10, StartAt: "2026-10-15T18:00:00+07:00", DurationMin: 60, Capacity: 3}
	res, err := f.sched.Update(classOwnerCtx(), upd)
	require.NoError(t, err)
	assert.Equal(t, 3, res.BookedCount, "tăng sức chứa → đẩy 2 người từ waitlist lên")
	assert.Equal(t, 0, res.WaitlistCount)
	assert.Len(t, f.notifier.sent, 2)

	upd.Capacity = 2
	_, err = f.sched.Update(classOwnerCtx(), upd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không được nhỏ hơn số chỗ đã đặt")

	sc.StartAt = f.now.Add(-time.Minute)
	upd.Capacity = 5
	_, err = f.sched.Update(classOwnerCtx(), upd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "đã bắt đầu")
}

func TestClassScheduleUpdate_ExcludesSelfFromOverlap(t *testing.T) {
	f := newClassFixture()
	sc := f.addSchedule(f.classAt(15, 18, 0), 5)
	upd := &v1.UpdateScheduleReq{ID: sc.ID, RoomID: 1, TrainerID: 10, StartAt: "2026-10-15T18:30:00+07:00", DurationMin: 45, Capacity: 5}
	res, err := f.sched.Update(classOwnerCtx(), upd)
	require.NoError(t, err)
	assert.Equal(t, 45, res.DurationMin)
	assert.Contains(t, f.audit.actions, "UPDATE:class_schedule")
}

// ---- Cancel -------------------------------------------------------------------

func TestClassScheduleCancel_Following(t *testing.T) {
	f := newClassFixture()
	res, err := f.sched.CreateRecurring(classOwnerCtx(), &v1.CreateRecurringReq{
		ClassTypeID: 1, BranchID: 1, RoomID: 1, TrainerID: 10,
		StartDate: "2026-10-15", EndDate: "2026-11-05", Weekdays: []int{4}, StartTime: "18:00",
	})
	require.NoError(t, err)
	require.Equal(t, 4, res.Created) // 15, 22, 29/10, 5/11
	ids := res.ScheduleIDs

	f.settings.values[SettingBookingAdvanceDays] = 14
	f.addMember(1)
	f.addMember(2)
	f.store.members[2].UserID = nil // không có tài khoản cổng → không nhận thông báo
	_, err = f.svc.Book(classStaffCtx(), ids[1], 1, "")
	require.NoError(t, err)
	_, err = f.svc.Book(classStaffCtx(), ids[1], 2, "")
	require.NoError(t, err)

	out, err := f.sched.Cancel(classOwnerCtx(), ids[1], "HLV nghỉ", "following")
	require.NoError(t, err)
	assert.Equal(t, 3, out.CancelledSchedules)
	assert.Equal(t, 2, out.CancelledBookings)
	assert.Equal(t, entity.ClassScheduleScheduled, f.store.schedules[ids[0]].Status, "buổi trước không bị hủy")
	for _, id := range ids[1:] {
		assert.Equal(t, entity.ClassScheduleCancelled, f.store.schedules[id].Status)
		assert.Equal(t, "HLV nghỉ", f.store.schedules[id].CancelReason)
	}
	assert.Equal(t, 0, f.store.schedules[ids[1]].BookedCount)
	for _, b := range f.store.bookings {
		assert.Equal(t, entity.BookingStatusCancelled, b.Status)
	}
	require.Len(t, f.notifier.sent, 1)
	assert.Equal(t, NotifyClassCancelled, f.notifier.sent[0].typ)
	assert.Equal(t, []uint{1001}, f.notifier.sent[0].userIDs)

	// Hủy lại buổi đã hủy → lỗi.
	_, err = f.sched.Cancel(classOwnerCtx(), ids[2], "x", "one")
	require.Error(t, err)
}

func TestClassScheduleCancel_OneAndStarted(t *testing.T) {
	f := newClassFixture()
	a := f.addSchedule(f.classAt(15, 18, 0), 5)
	out, err := f.sched.Cancel(classOwnerCtx(), a.ID, "Mất điện", "")
	require.NoError(t, err)
	assert.Equal(t, 1, out.CancelledSchedules)
	assert.Contains(t, f.audit.actions, "CANCEL:class_schedule")

	started := f.addSchedule(f.now.Add(-10*time.Minute), 5)
	started.RoomID = 2
	_, err = f.sched.Cancel(classOwnerCtx(), started.ID, "x", "one")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "đã diễn ra")

	_, err = f.sched.Cancel(classStaffCtx(), 999, "x", "one")
	require.Error(t, err)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

// ---- List ---------------------------------------------------------------------

func TestClassScheduleList_TrainerOnlySeesOwn(t *testing.T) {
	f := newClassFixture()
	f.addSchedule(f.classAt(15, 8, 0), 5) // HLV 10
	other := f.addSchedule(f.classAt(15, 10, 0), 5)
	other.TrainerID = 11

	req := &v1.ListScheduleReq{From: "2026-10-12", To: "2026-10-18"}
	items, err := f.sched.List(classTrainerCtx(), req)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, uint(10), items[0].TrainerID)

	items, err = f.sched.List(classStaffCtx(), req)
	require.NoError(t, err)
	assert.Len(t, items, 2)

	_, err = f.sched.List(classStaffCtx(), &v1.ListScheduleReq{BranchID: 2, From: "2026-10-12", To: "2026-10-18"})
	require.Error(t, err)
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	_, err = f.sched.List(classOwnerCtx(), &v1.ListScheduleReq{From: "2026-10-01", To: "2026-12-31"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "62 ngày")
}

// ---- Class type -----------------------------------------------------------------

func TestClassType_CodeAndDelete(t *testing.T) {
	st := newClassFakeStore()
	repo := classFakeTypeRepo{classFakeStore: st, future: map[uint]int64{}}
	audit := &classFakeAudit{}
	svc := NewClassTypeService(repo, audit)

	_, err := svc.Create(classOwnerCtx(), v1.ClassTypeInput{Code: " yoga ", Name: "Yoga 2"})
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err), "mã chuẩn hóa IN HOA và trùng YOGA")

	ct, err := svc.Create(classOwnerCtx(), v1.ClassTypeInput{Code: "spin", Name: "Spinning"})
	require.NoError(t, err)
	assert.Equal(t, "SPIN", ct.Code)
	assert.Equal(t, "#F97316", ct.Color)
	assert.Equal(t, 60, ct.DefaultDurationMin)
	assert.Equal(t, 20, ct.DefaultCapacity)
	assert.Equal(t, "ALL", ct.Level)

	repo.future[ct.ID] = 2
	err = svc.Delete(classOwnerCtx(), ct.ID)
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))

	repo.future[ct.ID] = 0
	require.NoError(t, svc.Delete(classOwnerCtx(), ct.ID))
	assert.Contains(t, audit.actions, "DELETE:class_type")
}
