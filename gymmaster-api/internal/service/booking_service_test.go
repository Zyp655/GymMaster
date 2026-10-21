package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

type bookingFixture struct {
	store    *classFakeStore
	svc      *bookingService
	notifier *bookingFakeNotifier
	settings bookingFakeSettings
	now      time.Time
}

// Mốc thời gian: Thứ Tư 14/10/2026 09:00 giờ Việt Nam.
func newBookingFixture() *bookingFixture {
	st := newClassFakeStore()
	notifier := &bookingFakeNotifier{}
	settings := bookingFakeSettings{values: map[string]int{
		SettingBookingAdvanceDays:         7,
		SettingBookingCancelBeforeMinutes: 120,
		SettingBookingNoShowLimit:         3,
		SettingBookingBlockDays:           7,
	}}
	svc := NewBookingService(classFakeScheduleRepo{st}, bookingFakeRepo{st}, noTx{}, settings, notifier).(*bookingService)
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, svc.loc)
	svc.now = func() time.Time { return now }
	return &bookingFixture{store: st, svc: svc, notifier: notifier, settings: settings, now: now}
}

// addSchedule tạo buổi lớp 60 phút ở chi nhánh 1, phòng 1, HLV 10.
func (f *bookingFixture) addSchedule(start time.Time, capacity int) *entity.ClassSchedule {
	sc := &entity.ClassSchedule{
		ClassTypeID: 1, BranchID: 1, RoomID: 1, TrainerID: 10,
		StartAt: start, EndAt: start.Add(time.Hour), Capacity: capacity, Status: entity.ClassScheduleScheduled,
	}
	_ = classFakeScheduleRepo{f.store}.Create(context.Background(), sc)
	return f.store.schedules[sc.ID]
}

func bookingSub(id uint, branchID uint) repository.ClassBookingSubscription {
	return repository.ClassBookingSubscription{
		ID: id, BranchID: branchID, Status: "ACTIVE", PlanType: entity.PlanTypeTime, AccessScope: entity.AccessSingleBranch,
		IncludesClasses: true,
		StartDate:       time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:         time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
	}
}

// addMember tạo hội viên ACTIVE có tài khoản cổng (user_id = 1000 + id) và gói hợp lệ ở chi nhánh 1.
func (f *bookingFixture) addMember(id uint) *entity.Member {
	uid := 1000 + id
	m := &entity.Member{BaseEntity: entity.BaseEntity{ID: id}, UserID: &uid, FullName: "HV", Status: entity.MemberStatusActive}
	f.store.members[id] = m
	f.store.subs[id] = []repository.ClassBookingSubscription{bookingSub(id*10, 1)}
	return m
}

func (f *bookingFixture) booking(id uint) *entity.ClassBooking { return f.store.bookings[id] }

func bookingIntPtr(v int) *int { return &v }

// ---- Book ---------------------------------------------------------------------

func TestBookingBook_CapacityThenWaitlist(t *testing.T) {
	f := newBookingFixture()
	sc := f.addSchedule(f.now.Add(26*time.Hour), 2)
	var got []*entity.ClassBooking
	for id := uint(1); id <= 4; id++ {
		f.addMember(id)
		b, err := f.svc.Book(classStaffCtx(), sc.ID, id, entity.BookingSourceCounter)
		require.NoError(t, err)
		got = append(got, b)
	}
	assert.Equal(t, entity.BookingStatusBooked, got[0].Status)
	assert.Equal(t, entity.BookingStatusBooked, got[1].Status)
	assert.Equal(t, entity.BookingStatusWaitlist, got[2].Status)
	assert.Equal(t, 1, *got[2].WaitlistPosition)
	assert.Equal(t, entity.BookingStatusWaitlist, got[3].Status)
	assert.Equal(t, 2, *got[3].WaitlistPosition)
	assert.Equal(t, 2, f.store.schedules[sc.ID].BookedCount, "booked_count chỉ đếm BOOKED")
	assert.Equal(t, uint(10), *got[0].SubscriptionID)
	assert.Equal(t, entity.BookingSourceCounter, got[0].Source)
}

func TestBookingBook_Rules(t *testing.T) {
	type tc struct {
		name  string
		setup func(f *bookingFixture) (scheduleID, memberID uint)
		ctx   context.Context
		src   string
		code  int
		msg   string
	}
	ok := 0
	cases := []tc{
		{name: "hợp lệ", setup: func(f *bookingFixture) (uint, uint) {
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, f.addMember(1).ID
		}, code: ok},
		{name: "lớp đã hủy", setup: func(f *bookingFixture) (uint, uint) {
			sc := f.addSchedule(f.now.Add(3*time.Hour), 5)
			sc.Status = entity.ClassScheduleCancelled
			return sc.ID, f.addMember(1).ID
		}, code: rescode.BadRequest.Code(), msg: "đã bị hủy"},
		{name: "lớp đã bắt đầu", setup: func(f *bookingFixture) (uint, uint) {
			return f.addSchedule(f.now.Add(-10*time.Minute), 5).ID, f.addMember(1).ID
		}, code: rescode.BadRequest.Code(), msg: "đã bắt đầu"},
		{name: "quá số ngày đặt trước", setup: func(f *bookingFixture) (uint, uint) {
			return f.addSchedule(f.now.AddDate(0, 0, 8), 5).ID, f.addMember(1).ID
		}, code: rescode.BadRequest.Code(), msg: "tối đa 7 ngày"},
		{name: "đúng ngày cuối được đặt trước", setup: func(f *bookingFixture) (uint, uint) {
			return f.addSchedule(f.now.AddDate(0, 0, 7).Add(10*time.Hour), 5).ID, f.addMember(1).ID
		}, code: ok},
		{name: "hội viên không hoạt động", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			m.Status = entity.MemberStatusInactive
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "trạng thái hoạt động"},
		{name: "hội viên bị khóa đặt lớp", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			until := f.now.Add(48 * time.Hour)
			m.BookingBlockedUntil = &until
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "bị khóa đặt lớp"},
		{name: "khóa đã hết hạn thì được đặt", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			until := f.now.Add(-time.Hour)
			m.BookingBlockedUntil = &until
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: ok},
		{name: "không có gói", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1] = nil
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "không có gói tập"},
		{name: "gói không gồm lớp nhóm", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].IncludesClasses = false
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "không bao gồm lớp nhóm"},
		{name: "gói hết hạn trước ngày lớp", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].EndDate = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
			return f.addSchedule(f.now.AddDate(0, 0, 2), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "không còn hiệu lực vào ngày"},
		{name: "gói còn hiệu lực đúng ngày cuối", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].EndDate = time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
			return f.addSchedule(f.now.AddDate(0, 0, 2).Add(10*time.Hour), 5).ID, m.ID
		}, code: ok},
		{name: "gói chưa bắt đầu vào ngày lớp", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].StartDate = time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "không còn hiệu lực vào ngày"},
		{name: "gói một chi nhánh khác", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].BranchID = 2
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "không áp dụng tại chi nhánh"},
		{name: "gói mọi chi nhánh", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].BranchID = 2
			f.store.subs[1][0].AccessScope = entity.AccessAllBranches
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: ok},
		{name: "gói theo buổi hết buổi", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].PlanType = entity.PlanTypeSession
			f.store.subs[1][0].RemainingSessions = bookingIntPtr(0)
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: rescode.BadRequest.Code(), msg: "hết buổi"},
		{name: "gói theo buổi còn buổi", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			f.store.subs[1][0].PlanType = entity.PlanTypeSession
			f.store.subs[1][0].RemainingSessions = bookingIntPtr(2)
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: ok},
		{name: "gói thứ hai hợp lệ", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			bad := bookingSub(99, 1)
			bad.IncludesClasses = false
			f.store.subs[1] = append([]repository.ClassBookingSubscription{bad}, f.store.subs[1]...)
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, m.ID
		}, code: ok},
		{name: "trùng giờ lớp khác đã đặt", setup: func(f *bookingFixture) (uint, uint) {
			m := f.addMember(1)
			other := f.addSchedule(f.now.Add(3*time.Hour), 5)
			other.RoomID, other.TrainerID = 2, 11
			_ = bookingFakeRepo{f.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: other.ID, MemberID: 1, Status: entity.BookingStatusBooked})
			return f.addSchedule(f.now.Add(3*time.Hour+30*time.Minute), 5).ID, m.ID
		}, code: rescode.Conflict.Code(), msg: "trùng giờ"},
		{name: "nhân viên khác chi nhánh", ctx: func() context.Context { return managerCtx() }(), setup: func(f *bookingFixture) (uint, uint) {
			sc := f.addSchedule(f.now.Add(3*time.Hour), 5)
			sc.BranchID = 2
			return sc.ID, f.addMember(1).ID
		}, code: rescode.NotFound.Code()},
		{name: "hội viên tự đặt qua cổng", src: entity.BookingSourcePortal, ctx: context.Background(), setup: func(f *bookingFixture) (uint, uint) {
			return f.addSchedule(f.now.Add(3*time.Hour), 5).ID, f.addMember(1).ID
		}, code: ok},
		{name: "không tìm thấy lớp", setup: func(f *bookingFixture) (uint, uint) {
			return 999, f.addMember(1).ID
		}, code: rescode.NotFound.Code()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newBookingFixture()
			sid, mid := c.setup(f)
			ctx := c.ctx
			if ctx == nil {
				ctx = classStaffCtx()
			}
			src := c.src
			if src == "" {
				src = entity.BookingSourceCounter
			}
			b, err := f.svc.Book(ctx, sid, mid, src)
			if c.code == ok {
				require.NoError(t, err)
				assert.Equal(t, entity.BookingStatusBooked, b.Status)
				return
			}
			require.Error(t, err)
			assert.Equal(t, c.code, codeOf(err))
			if c.msg != "" {
				assert.Contains(t, err.Error(), c.msg)
			}
		})
	}
}

func TestBookingBook_Duplicate(t *testing.T) {
	f := newBookingFixture()
	sc := f.addSchedule(f.now.Add(3*time.Hour), 1)
	f.addMember(1)
	_, err := f.svc.Book(classStaffCtx(), sc.ID, 1, "")
	require.NoError(t, err)
	_, err = f.svc.Book(classStaffCtx(), sc.ID, 1, "")
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict.Code(), codeOf(err))
	assert.Equal(t, 1, f.store.schedules[sc.ID].BookedCount)
}

// ---- Cancel ---------------------------------------------------------------------

func TestBookingCancel_PromotesWaitlist(t *testing.T) {
	f := newBookingFixture()
	sc := f.addSchedule(f.now.Add(30*time.Minute), 1)
	for id := uint(1); id <= 3; id++ {
		f.addMember(id)
		_, err := f.svc.Book(classStaffCtx(), sc.ID, id, "")
		require.NoError(t, err)
	}
	require.Equal(t, entity.BookingStatusWaitlist, f.booking(2).Status)

	// Nhân viên hủy hộ không giới hạn thời gian (lớp còn 30 phút).
	require.NoError(t, f.svc.Cancel(classStaffCtx(), 1, false))
	assert.Equal(t, entity.BookingStatusCancelled, f.booking(1).Status)
	assert.NotNil(t, f.booking(1).CancelledAt)
	assert.Equal(t, entity.BookingStatusBooked, f.booking(2).Status, "người đầu waitlist được đẩy lên")
	assert.Nil(t, f.booking(2).WaitlistPosition)
	assert.Equal(t, entity.BookingStatusWaitlist, f.booking(3).Status)
	assert.Equal(t, 1, f.store.schedules[sc.ID].BookedCount)

	require.Len(t, f.notifier.sent, 1)
	assert.Equal(t, NotifyBookingPromoted, f.notifier.sent[0].typ)
	assert.Equal(t, []uint{1002}, f.notifier.sent[0].userIDs)

	// Hủy lại lượt đã hủy → lỗi.
	err := f.svc.Cancel(classStaffCtx(), 1, false)
	require.Error(t, err)
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
}

func TestBookingCancel_WaitlistKeepsCount(t *testing.T) {
	f := newBookingFixture()
	sc := f.addSchedule(f.now.Add(5*time.Hour), 1)
	f.addMember(1)
	f.addMember(2)
	_, _ = f.svc.Book(classStaffCtx(), sc.ID, 1, "")
	_, _ = f.svc.Book(classStaffCtx(), sc.ID, 2, "")
	require.NoError(t, f.svc.Cancel(context.Background(), 2, true))
	assert.Equal(t, entity.BookingStatusCancelled, f.booking(2).Status)
	assert.Equal(t, 1, f.store.schedules[sc.ID].BookedCount)
	assert.Empty(t, f.notifier.sent)
}

func TestBookingCancel_MemberDeadline(t *testing.T) {
	f := newBookingFixture()
	sc := f.addSchedule(f.now.Add(90*time.Minute), 5)
	f.addMember(1)
	_, err := f.svc.Book(context.Background(), sc.ID, 1, entity.BookingSourcePortal)
	require.NoError(t, err)

	err = f.svc.Cancel(context.Background(), 1, true)
	require.Error(t, err, "hội viên phải hủy trước 120 phút")
	assert.Contains(t, err.Error(), "120 phút")

	f.settings.values[SettingBookingCancelBeforeMinutes] = 60
	require.NoError(t, f.svc.Cancel(context.Background(), 1, true))
	assert.Equal(t, 0, f.store.schedules[sc.ID].BookedCount)
}

func TestBookingCancel_StaffScope(t *testing.T) {
	f := newBookingFixture()
	sc := f.addSchedule(f.now.Add(5*time.Hour), 5)
	f.addMember(1)
	_, err := f.svc.Book(classOwnerCtx(), sc.ID, 1, "")
	require.NoError(t, err)
	sc.BranchID = 2
	err = f.svc.Cancel(classStaffCtx(), 1, false)
	require.Error(t, err)
	assert.Equal(t, rescode.NotFound.Code(), codeOf(err))
}

// ---- Attendance ---------------------------------------------------------------

func TestBookingAttendance_Window(t *testing.T) {
	f := newBookingFixture()
	f.addMember(1)
	f.addMember(2)
	later := f.addSchedule(f.now.Add(2*time.Hour), 5)
	_, err := f.svc.Book(classStaffCtx(), later.ID, 1, "")
	require.NoError(t, err)

	_, err = f.svc.MarkAttendance(classStaffCtx(), later.ID, []BookingAttendance{{BookingID: 1, Status: "ATTENDED"}})
	require.Error(t, err, "chưa tới 30 phút trước giờ học")
	assert.Contains(t, err.Error(), "Chưa đến giờ điểm danh")

	// Lùi đồng hồ tới 20 phút trước giờ học.
	f.svc.now = func() time.Time { return later.StartAt.Add(-20 * time.Minute) }
	n, err := f.svc.MarkAttendance(classStaffCtx(), later.ID, []BookingAttendance{{BookingID: 1, Status: "ATTENDED"}})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, entity.BookingStatusAttended, f.booking(1).Status)
	assert.NotNil(t, f.booking(1).AttendedAt)

	// Sửa lại thành NO_SHOW được phép.
	_, err = f.svc.MarkAttendance(classStaffCtx(), later.ID, []BookingAttendance{{BookingID: 1, Status: "NO_SHOW"}})
	require.NoError(t, err)
	assert.Equal(t, entity.BookingStatusNoShow, f.booking(1).Status)
	assert.Nil(t, f.booking(1).AttendedAt)

	// Quá 24h sau khi kết thúc.
	f.svc.now = func() time.Time { return later.EndAt.Add(25 * time.Hour) }
	_, err = f.svc.MarkAttendance(classStaffCtx(), later.ID, []BookingAttendance{{BookingID: 1, Status: "ATTENDED"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quá thời hạn")
}

func TestBookingAttendance_InvalidBookings(t *testing.T) {
	f := newBookingFixture()
	f.addMember(1)
	f.addMember(2)
	a := f.addSchedule(f.now.Add(10*time.Minute), 1)
	b := f.addSchedule(f.now.Add(26*time.Hour), 1)
	b.RoomID, b.TrainerID = 2, 11
	_, _ = f.svc.Book(classStaffCtx(), a.ID, 1, "")    // #1 BOOKED
	_, _ = f.svc.Book(classStaffCtx(), a.ID, 2, "")    // #2 WAITLIST
	_, err := f.svc.Book(classStaffCtx(), b.ID, 1, "") // #3 lớp khác
	require.NoError(t, err)

	_, err = f.svc.MarkAttendance(classStaffCtx(), a.ID, []BookingAttendance{{BookingID: 3, Status: "ATTENDED"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không thuộc buổi lớp")

	_, err = f.svc.MarkAttendance(classStaffCtx(), a.ID, []BookingAttendance{{BookingID: 2, Status: "ATTENDED"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAITLIST")

	a.Status = entity.ClassScheduleCancelled
	_, err = f.svc.MarkAttendance(classStaffCtx(), a.ID, []BookingAttendance{{BookingID: 1, Status: "ATTENDED"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "đã bị hủy")
}

func TestBookingAttendance_ManualNoShowBlocksAndRemovesReview(t *testing.T) {
	f := newBookingFixture()
	f.addMember(1)
	// Hai lần no-show trong 30 ngày.
	for i := 0; i < 2; i++ {
		old := f.addSchedule(f.now.AddDate(0, 0, -(i+3)), 5)
		_ = bookingFakeRepo{f.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: old.ID, MemberID: 1, Status: entity.BookingStatusNoShow})
	}
	sc := f.addSchedule(f.now.Add(10*time.Minute), 5)
	bk, err := f.svc.Book(classStaffCtx(), sc.ID, 1, "")
	require.NoError(t, err)
	_, err = f.svc.MarkAttendance(classStaffCtx(), sc.ID, []BookingAttendance{{BookingID: bk.ID, Status: "ATTENDED"}})
	require.NoError(t, err)
	assert.Nil(t, f.store.members[1].BookingBlockedUntil)
	assert.Empty(t, f.store.removedReviews)

	// Sửa ATTENDED → NO_SHOW: lần vắng thứ 3 → khóa đặt lớp như cron; đánh giá lớp bị gỡ.
	_, err = f.svc.MarkAttendance(classStaffCtx(), sc.ID, []BookingAttendance{{BookingID: bk.ID, Status: "NO_SHOW"}})
	require.NoError(t, err)
	require.NotNil(t, f.store.members[1].BookingBlockedUntil, "NO_SHOW điểm danh tay phải tính vào hạn mức khóa")
	assert.True(t, f.store.members[1].BookingBlockedUntil.Equal(f.now.AddDate(0, 0, 7)))
	assert.Equal(t, [][2]uint{{1, sc.ID}}, f.store.removedReviews)

	// Đánh dấu NO_SHOW lại lần nữa không gỡ/khóa lặp.
	_, err = f.svc.MarkAttendance(classStaffCtx(), sc.ID, []BookingAttendance{{BookingID: bk.ID, Status: "NO_SHOW"}})
	require.NoError(t, err)
	assert.Len(t, f.store.removedReviews, 1)
}

// ---- MarkNoShows ---------------------------------------------------------------

func TestBookingMarkNoShows(t *testing.T) {
	f := newBookingFixture()
	f.addMember(1)
	f.addMember(2)

	// Hai lần no-show cũ của hội viên 1 trong 30 ngày.
	for i := 0; i < 2; i++ {
		old := f.addSchedule(f.now.AddDate(0, 0, -(i+3)), 5)
		_ = bookingFakeRepo{f.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: old.ID, MemberID: 1, Status: entity.BookingStatusNoShow})
	}
	// No-show cũ hơn 30 ngày không tính (hội viên 2).
	for i := 0; i < 2; i++ {
		old := f.addSchedule(f.now.AddDate(0, 0, -(40+i)), 5)
		_ = bookingFakeRepo{f.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: old.ID, MemberID: 2, Status: entity.BookingStatusNoShow})
	}

	ended := f.addSchedule(f.now.Add(-2*time.Hour), 5) // kết thúc 1 giờ trước
	_ = bookingFakeRepo{f.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: ended.ID, MemberID: 1, Status: entity.BookingStatusBooked})
	_ = bookingFakeRepo{f.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: ended.ID, MemberID: 2, Status: entity.BookingStatusBooked})
	justEnded := f.addSchedule(f.now.Add(-70*time.Minute), 5) // kết thúc 10 phút trước
	justEnded.RoomID = 2
	justEnded.TrainerID = 11
	jb := &entity.ClassBooking{ScheduleID: justEnded.ID, MemberID: 2, Status: entity.BookingStatusBooked}
	_ = bookingFakeRepo{f.store}.Create(context.Background(), jb)
	cancelled := f.addSchedule(f.now.Add(-5*time.Hour), 5)
	cancelled.Status = entity.ClassScheduleCancelled
	cb := &entity.ClassBooking{ScheduleID: cancelled.ID, MemberID: 2, Status: entity.BookingStatusBooked}
	_ = bookingFakeRepo{f.store}.Create(context.Background(), cb)

	n, err := f.svc.MarkNoShows(context.Background(), f.now)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, entity.BookingStatusBooked, f.store.bookings[jb.ID].Status, "lớp mới kết thúc 10 phút chưa chuyển NO_SHOW")
	assert.Equal(t, entity.BookingStatusBooked, f.store.bookings[cb.ID].Status, "lớp đã hủy không xét")
	assert.Equal(t, entity.ClassScheduleCompleted, ended.Status)
	assert.Equal(t, entity.ClassScheduleCompleted, justEnded.Status)
	assert.Equal(t, entity.ClassScheduleCancelled, cancelled.Status)

	m1 := f.store.members[1]
	require.NotNil(t, m1.BookingBlockedUntil, "3 no-show trong 30 ngày → khóa")
	assert.True(t, m1.BookingBlockedUntil.Equal(f.now.AddDate(0, 0, 7)))
	assert.Nil(t, f.store.members[2].BookingBlockedUntil, "no-show cũ quá 30 ngày không tính")

	// Bị khóa thì không đặt được lớp.
	sc := f.addSchedule(f.now.Add(26*time.Hour), 5)
	_, err = f.svc.Book(classStaffCtx(), sc.ID, 1, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bị khóa đặt lớp")
}
