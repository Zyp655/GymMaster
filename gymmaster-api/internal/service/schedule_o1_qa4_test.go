package service

import (
	"context"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/class/v1"
	schedulev1 "gymmaster-api/api/scheduleext/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// QA vòng 4 – nhóm O1: lịch lớp, dạy thay, ngày lễ, đặt lớp.

// ---- C1: HLV đang dạy thay phải được tính là bận khi tạo/sửa lớp -----------------

func TestClassSchedule_C1_SubstituteTrainerIsBusy(t *testing.T) {
	f := newClassFixture()
	// Lớp A (HLV 10, phòng 1) 15/10 18:00–19:00, HLV 11 dạy thay.
	a := f.addSchedule(f.classAt(15, 18, 0), 5)
	sub := uint(11)
	a.SubstituteTrainerID = &sub

	// C1-01: tạo lớp mới cho HLV 11 ở phòng khác, 18:15 → trùng giờ dạy thay → 409.
	req := classCreateReq("2026-10-15T18:15:00+07:00")
	req.RoomID, req.TrainerID = 2, 11
	_, err := f.sched.Create(classOwnerCtx(), req)
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict, gerror.Code(err))
	assert.Contains(t, err.Error(), "HLV đã có lớp khác")

	// HLV chính 10 không còn thực dạy lớp A → được xếp lớp khác cùng giờ (phòng khác).
	req.TrainerID = 10
	_, err = f.sched.Create(classOwnerCtx(), req)
	require.NoError(t, err)

	// C1-02: lớp B của HLV 11 lúc 20:00, dời sang 18:40 → 409.
	b := f.addSchedule(f.classAt(15, 20, 0), 5)
	b.RoomID, b.TrainerID = 2, 11
	_, err = f.sched.Update(classOwnerCtx(), &v1.UpdateScheduleReq{ID: b.ID, RoomID: 2, TrainerID: 11,
		StartAt: "2026-10-15T18:40:00+07:00", DurationMin: 15, Capacity: 5})
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict, gerror.Code(err))
}

func TestClassSchedule_C1_UpdateChecksSubstituteAndClearsOnTrainerChange(t *testing.T) {
	f := newClassFixture()
	// Lớp X (HLV 10, phòng 1) 15/10 10:00, HLV 11 dạy thay. Lớp Y (HLV 11, phòng 2) 15/10 12:00.
	x := f.addSchedule(f.classAt(15, 10, 0), 5)
	sub := uint(11)
	x.SubstituteTrainerID = &sub
	y := f.addSchedule(f.classAt(15, 12, 0), 5)
	y.RoomID, y.TrainerID = 2, 11

	// Dời X sang 12:30 giữ HLV chính 10 → HLV thực dạy (11) trùng lớp Y → 409 (trước đây chỉ xét HLV 10).
	upd := &v1.UpdateScheduleReq{ID: x.ID, RoomID: 1, TrainerID: 10, StartAt: "2026-10-15T12:30:00+07:00", DurationMin: 60, Capacity: 5}
	_, err := f.sched.Update(classOwnerCtx(), upd)
	require.Error(t, err)
	assert.Equal(t, rescode.Conflict, gerror.Code(err))

	// Đổi HLV chính (10 → 11) ở giờ cũ: phân công dạy thay cũ phải được bỏ.
	upd = &v1.UpdateScheduleReq{ID: x.ID, RoomID: 1, TrainerID: 11, StartAt: "2026-10-15T10:00:00+07:00", DurationMin: 60, Capacity: 5}
	res, err := f.sched.Update(classOwnerCtx(), upd)
	require.NoError(t, err)
	assert.Equal(t, uint(11), res.TrainerID)
	assert.Nil(t, res.SubstituteTrainerID, "đổi HLV chính → bỏ dạy thay cũ")
	assert.Nil(t, f.store.schedules[x.ID].SubstituteTrainerID)
}

// ---- C3: người dạy thay hiển thị ở API lịch lớp, lọc theo HLV thực dạy ----------

func TestClassSchedule_C3_SubstituteVisibleAndFilter(t *testing.T) {
	f := newClassFixture()
	a := f.addSchedule(f.classAt(15, 18, 0), 5)
	sub := uint(11)
	a.SubstituteTrainerID = &sub
	f.addSchedule(f.classAt(16, 18, 0), 5) // lớp thường của HLV 10

	got, err := f.sched.Get(classOwnerCtx(), a.ID)
	require.NoError(t, err)
	require.NotNil(t, got.SubstituteTrainerID)
	assert.Equal(t, uint(11), *got.SubstituteTrainerID)

	list, err := f.sched.List(classOwnerCtx(), &v1.ListScheduleReq{From: "2026-10-14", To: "2026-10-20", TrainerID: 11})
	require.NoError(t, err)
	require.Len(t, list, 1, "lọc trainer_id = HLV dạy thay phải trả lớp đang dạy thay")
	assert.Equal(t, a.ID, list[0].ID)

	list, err = f.sched.List(classOwnerCtx(), &v1.ListScheduleReq{From: "2026-10-14", To: "2026-10-20", TrainerID: 10})
	require.NoError(t, err)
	require.Len(t, list, 1, "HLV chính không còn thực dạy lớp đã giao dạy thay")
	assert.NotEqual(t, a.ID, list[0].ID)
}

func TestPortal_C3_ClassesShowActualTrainer(t *testing.T) {
	f := newPortalFixture(t)
	sch := f.svc.d.Schedules.(*fakePortalSchedules)
	sub := uint(77)
	sch.rows[0].TrainerID, sch.rows[0].TrainerName = 50, "HLV Chính"
	sch.rows[0].SubstituteTrainerID, sch.rows[0].SubstituteTrainerName = &sub, "HLV Dạy Thay"

	res, err := f.svc.Classes(portalCtx(ptUserA), 0, "", "")
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "HLV Dạy Thay", res.Items[0].TrainerName, "/member/classes trả HLV thực dạy")
	assert.Equal(t, uint(77), res.Items[0].TrainerID)
	assert.Equal(t, "HLV Dạy Thay", res.Items[0].SubstituteTrainerName)
}

// ---- C2: trả lớp về HLV chính cũng phải kiểm tra HLV chính --------------------

func TestScheduleExt_C2_ReturnToMainTrainerIsChecked(t *testing.T) {
	svc, repo, now := newSchedSvc()
	future := now.Add(48 * time.Hour)
	sub := uint(6)
	repo.classes[10] = &repository.ImpactedClassRow{ClassSchedule: entity.ClassSchedule{ID: 10, BranchID: 1, TrainerID: 5,
		SubstituteTrainerID: &sub, Status: entity.ClassScheduleScheduled, StartAt: future, EndAt: future.Add(time.Hour)}}
	// HLV chính 5 đã nhận buổi PT trùng giờ.
	repo.busy = map[uint]string{5: "PT"}
	_, err := svc.SubstituteClassTrainer(schedOwnerCtx(), &schedulev1.SubstituteClassTrainerReq{ID: 10, SubstituteTrainerID: 5})
	if gerror.Code(err) != rescode.Conflict {
		t.Fatalf("trả về HLV chính đang bận phải 409, nhận %v", err)
	}
	if repo.classes[10].SubstituteTrainerID == nil {
		t.Fatalf("bị từ chối nhưng vẫn bỏ dạy thay")
	}
	// HLV chính không còn hợp lệ (khóa/khác chi nhánh) → 400.
	repo.busy = nil
	repo.ineligible = map[uint]bool{5: true}
	if _, err := svc.SubstituteClassTrainer(schedOwnerCtx(), &schedulev1.SubstituteClassTrainerReq{ID: 10, SubstituteTrainerID: 5}); gerror.Code(err) != rescode.BadRequest {
		t.Fatalf("HLV chính không hợp lệ phải 400, nhận %v", err)
	}
	repo.ineligible = nil
	if _, err := svc.SubstituteClassTrainer(schedOwnerCtx(), &schedulev1.SubstituteClassTrainerReq{ID: 10, SubstituteTrainerID: 5}); err != nil || repo.classes[10].SubstituteTrainerID != nil {
		t.Fatalf("HLV chính rảnh → bỏ dạy thay: %v", err)
	}
}

// ---- C4 / HO-05 / HO-06 / C11-01: ngày lễ ----------------------------------------

func TestScheduleExt_C4_HolidayDuplicate(t *testing.T) {
	svc, _, _ := newSchedSvc()
	ctx := schedOwnerCtx()
	b1 := uint(1)
	if _, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: "2026-12-30", Name: "Lễ A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: "2026-12-30", Name: "Lễ B"}); gerror.Code(err) != rescode.Conflict {
		t.Fatalf("2 ngày lễ cùng chi nhánh/cùng ngày phải 409, nhận %v", err)
	}
	// Lễ toàn hệ thống (branch NULL) cùng ngày là phạm vi khác → được; nhưng 2 lễ toàn hệ thống cùng ngày → 409.
	if _, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{Date: "2026-12-30", Name: "Toàn hệ thống"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{Date: "2026-12-30", Name: "Toàn hệ thống 2"}); gerror.Code(err) != rescode.Conflict {
		t.Fatalf("2 lễ toàn hệ thống cùng ngày phải 409, nhận %v", err)
	}
	// Sửa một lễ sang ngày đã có lễ khác cùng chi nhánh → 409.
	other, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: "2026-12-31", Name: "Lễ C"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateBranchHoliday(ctx, &schedulev1.UpdateBranchHolidayReq{ID: other.ID, Date: "2026-12-30", Name: "Lễ C"}); gerror.Code(err) != rescode.Conflict {
		t.Fatalf("sửa trùng ngày phải 409, nhận %v", err)
	}
	// Sửa tên giữ nguyên ngày không bị coi là trùng với chính nó.
	if _, err := svc.UpdateBranchHoliday(ctx, &schedulev1.UpdateBranchHolidayReq{ID: other.ID, Date: "2026-12-31", Name: "Lễ C2"}); err != nil {
		t.Fatalf("sửa tên: %v", err)
	}
}

func TestScheduleExt_HO06_HolidayInPastRejected(t *testing.T) {
	svc, _, now := newSchedSvc()
	ctx := schedOwnerCtx()
	b1 := uint(1)
	yesterday := now.AddDate(0, 0, -1).Format(consts.DateLayout)
	if _, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: yesterday, Name: "Lễ cũ", ExtendSubs: true}); gerror.Code(err) != rescode.BadRequest {
		t.Fatalf("ngày lễ đã qua phải 400, nhận %v", err)
	}
	if _, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: now.Format(consts.DateLayout), Name: "Hôm nay"}); err != nil {
		t.Fatalf("ngày lễ hôm nay vẫn được tạo: %v", err)
	}
}

func TestScheduleExt_C11_01_OpenHolidayRequiresHours(t *testing.T) {
	svc, _, _ := newSchedSvc()
	b1 := uint(1)
	_, err := svc.CreateBranchHoliday(schedOwnerCtx(), &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: "2026-12-30",
		Name: "Mở cửa", Closed: schedBoolPtr(false)})
	if gerror.Code(err) != rescode.BadRequest {
		t.Fatalf("closed=false không giờ phải 400, nhận %v", err)
	}
}

func TestScheduleExt_HO05_DeleteHolidayRevertsExtension(t *testing.T) {
	svc, repo, _ := newSchedSvc()
	ctx := schedOwnerCtx()
	b1 := uint(1)
	repo.extendedCount = 4
	h, err := svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: "2026-12-30", Name: "Lễ", ExtendSubs: true})
	if err != nil || h.ExtendedSubscriptions != 4 {
		t.Fatalf("tạo lễ cộng bù: %+v %v", h, err)
	}
	// Đã cộng bù → không cho đổi ngày / mở cửa lại.
	if _, err := svc.UpdateBranchHoliday(ctx, &schedulev1.UpdateBranchHolidayReq{ID: h.ID, Date: "2026-12-31", Name: "Lễ"}); gerror.Code(err) != rescode.BadRequest {
		t.Fatalf("đổi ngày lễ đã cộng bù phải 400, nhận %v", err)
	}
	if err := svc.DeleteBranchHoliday(ctx, h.ID); err != nil {
		t.Fatal(err)
	}
	if repo.reverted != 4 || len(repo.extensions) != 0 {
		t.Fatalf("xóa lễ phải hoàn tác 4 gói đã cộng bù, nhận reverted=%d ext=%v", repo.reverted, repo.extensions)
	}
	// Tạo – xóa lặp lại: tổng cộng bù ròng = 0.
	for i := 0; i < 3; i++ {
		h, err = svc.CreateBranchHoliday(ctx, &schedulev1.CreateBranchHolidayReq{BranchID: &b1, Date: "2026-12-30", Name: "Lễ", ExtendSubs: true})
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.DeleteBranchHoliday(ctx, h.ID); err != nil {
			t.Fatal(err)
		}
	}
	if repo.reverted != 16 || len(repo.extensions) != 0 {
		t.Fatalf("tạo/xóa lặp phải hoàn tác hết: reverted=%d", repo.reverted)
	}
}

// ---- C11-02: lịch rảnh không nhận ca chồng nhau ------------------------------------

func TestScheduleExt_C11_02_AvailabilityOverlapRejected(t *testing.T) {
	svc, _, _ := newSchedSvc()
	ctx := schedOwnerCtx()
	_, err := svc.SaveTrainerAvailability(ctx, &schedulev1.SaveTrainerAvailabilityReq{ID: 6, BranchID: 1,
		Slots: []schedulev1.TrainerAvailabilitySlotInput{
			{DOW: 1, StartTime: "06:00", EndTime: "12:00"},
			{DOW: 1, StartTime: "10:00", EndTime: "14:00"},
		}})
	if gerror.Code(err) != rescode.BadRequest {
		t.Fatalf("2 ca chồng nhau cùng ngày phải 400, nhận %v", err)
	}
	// Ca nối tiếp (12:00–14:00), khác thứ, hoặc khác khoảng hiệu lực thì hợp lệ.
	_, err = svc.SaveTrainerAvailability(ctx, &schedulev1.SaveTrainerAvailabilityReq{ID: 6, BranchID: 1,
		Slots: []schedulev1.TrainerAvailabilitySlotInput{
			{DOW: 1, StartTime: "06:00", EndTime: "12:00"},
			{DOW: 1, StartTime: "12:00", EndTime: "14:00"},
			{DOW: 2, StartTime: "10:00", EndTime: "14:00"},
			{DOW: 3, StartTime: "06:00", EndTime: "12:00", EffectiveTo: "2026-11-30"},
			{DOW: 3, StartTime: "08:00", EndTime: "12:00", EffectiveFrom: "2026-12-01"},
		}})
	if err != nil {
		t.Fatalf("ca hợp lệ bị từ chối: %v", err)
	}
}

// ---- C9: xem lịch rảnh HLV theo phạm vi chi nhánh ----------------------------------

func TestScheduleExt_C9_AvailabilityBranchScope(t *testing.T) {
	svc, repo, _ := newSchedSvc()
	repo.availabilities = []repository.TrainerAvailabilityRow{
		{TrainerAvailability: entity.TrainerAvailability{ID: 1, TrainerID: 17, BranchID: 2, DOW: 1, StartTime: "06:00", EndTime: "10:00"}},
		{TrainerAvailability: entity.TrainerAvailability{ID: 2, TrainerID: 6, BranchID: 1, DOW: 1, StartTime: "06:00", EndTime: "10:00"}},
		{TrainerAvailability: entity.TrainerAvailability{ID: 3, TrainerID: 6, BranchID: 2, DOW: 2, StartTime: "06:00", EndTime: "10:00"}},
	}
	// HLV 17 chỉ thuộc chi nhánh 2 (không đủ điều kiện ở chi nhánh 1).
	repo.eligibleIn = map[uint][]uint{17: {2}, 6: {1, 2}}
	recep := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 3, Roles: []string{consts.RoleReceptionist},
		BranchIDs: []uint{1}, Permissions: []string{"pt.view"}})

	if _, err := svc.GetTrainerAvailability(recep, &schedulev1.GetTrainerAvailabilityReq{ID: 17}); gerror.Code(err) != rescode.PermissionDenied {
		t.Fatalf("lễ tân CN1 xem HLV CN2 phải 403, nhận %v", err)
	}
	if _, err := svc.GetTrainerAvailability(recep, &schedulev1.GetTrainerAvailabilityReq{ID: 17, BranchID: 2}); gerror.Code(err) != rescode.PermissionDenied {
		t.Fatalf("lễ tân CN1 xem branch_id=2 phải 403, nhận %v", err)
	}
	rows, err := svc.GetTrainerAvailability(recep, &schedulev1.GetTrainerAvailabilityReq{ID: 6})
	if err != nil || len(rows) != 1 || rows[0].BranchID != 1 {
		t.Fatalf("HLV 2 chi nhánh: lễ tân CN1 chỉ thấy lịch CN1: %+v %v", rows, err)
	}
	// HLV tự xem lịch của mình (kể cả chi nhánh khác) và chủ xem tất cả.
	self := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 17, Roles: []string{consts.RoleTrainer}, BranchIDs: []uint{2}})
	if rows, err := svc.GetTrainerAvailability(self, &schedulev1.GetTrainerAvailabilityReq{ID: 17}); err != nil || len(rows) != 1 {
		t.Fatalf("HLV tự xem: %+v %v", rows, err)
	}
	if rows, err := svc.GetTrainerAvailability(schedOwnerCtx(), &schedulev1.GetTrainerAvailabilityReq{ID: 6}); err != nil || len(rows) != 2 {
		t.Fatalf("chủ xem tất cả: %+v %v", rows, err)
	}
}

// ---- C5: sửa NO_SHOW → ATTENDED gỡ khóa đặt lớp ------------------------------------

func TestBookingAttendance_C5_NoShowToAttendedLiftsBlock(t *testing.T) {
	f := newBookingFixture()
	f.addMember(1)
	for i := 0; i < 2; i++ {
		old := f.addSchedule(f.now.AddDate(0, 0, -(i+3)), 5)
		_ = bookingFakeRepo{f.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: old.ID, MemberID: 1, Status: entity.BookingStatusNoShow})
	}
	sc := f.addSchedule(f.now.Add(10*time.Minute), 5)
	bk, err := f.svc.Book(classStaffCtx(), sc.ID, 1, "")
	require.NoError(t, err)
	_, err = f.svc.MarkAttendance(classStaffCtx(), sc.ID, []BookingAttendance{{BookingID: bk.ID, Status: "NO_SHOW"}})
	require.NoError(t, err)
	require.NotNil(t, f.store.members[1].BookingBlockedUntil, "3 lần vắng → khóa")

	// Sửa lại ATTENDED: còn 2 < 3 → gỡ khóa.
	_, err = f.svc.MarkAttendance(classStaffCtx(), sc.ID, []BookingAttendance{{BookingID: bk.ID, Status: "ATTENDED"}})
	require.NoError(t, err)
	assert.Nil(t, f.store.members[1].BookingBlockedUntil, "NO_SHOW → ATTENDED (còn 2 < 3) phải gỡ khóa")

	// Vẫn đủ hạn mức (4 lần vắng, sửa 1) → giữ khóa.
	f2 := newBookingFixture()
	f2.addMember(1)
	for i := 0; i < 3; i++ {
		old := f2.addSchedule(f2.now.AddDate(0, 0, -(i+3)), 5)
		_ = bookingFakeRepo{f2.store}.Create(context.Background(), &entity.ClassBooking{ScheduleID: old.ID, MemberID: 1, Status: entity.BookingStatusNoShow})
	}
	sc2 := f2.addSchedule(f2.now.Add(10*time.Minute), 5)
	until := f2.now.AddDate(0, 0, 7)
	f2.store.members[1].BookingBlockedUntil = &until
	bk2 := &entity.ClassBooking{ScheduleID: sc2.ID, MemberID: 1, Status: entity.BookingStatusNoShow}
	_ = bookingFakeRepo{f2.store}.Create(context.Background(), bk2)
	_, err = f2.svc.MarkAttendance(classStaffCtx(), sc2.ID, []BookingAttendance{{BookingID: bk2.ID, Status: "ATTENDED"}})
	require.NoError(t, err)
	assert.NotNil(t, f2.store.members[1].BookingBlockedUntil, "còn 3 lần vắng → giữ khóa")
}

// ---- not_bookable_reason trên /member/classes --------------------------------------

type fakePortalBookingsWithSubs struct {
	*fakePortalBookings
	subs []repository.ClassBookingSubscription
}

func (f fakePortalBookingsWithSubs) MemberClassSubscriptions(context.Context, uint) ([]repository.ClassBookingSubscription, error) {
	return f.subs, nil
}

func TestPortal_ClassesNotBookableReason(t *testing.T) {
	f := newPortalFixture(t)
	reason := func() (bool, string) {
		t.Helper()
		res, err := f.svc.Classes(portalCtx(ptUserA), 0, "", "")
		require.NoError(t, err)
		require.Len(t, res.Items, 1)
		return res.Items[0].Bookable, res.Items[0].NotBookableReason
	}
	ok, r := reason()
	assert.True(t, ok)
	assert.Equal(t, "", r)

	// Không có gói hợp lệ → NO_SUBSCRIPTION.
	wrapped := fakePortalBookingsWithSubs{fakePortalBookings: f.bookings}
	f.svc.d.Bookings = wrapped
	ok, r = reason()
	assert.False(t, ok)
	assert.Equal(t, "NO_SUBSCRIPTION", r)

	// Có gói bao gồm lớp tại chi nhánh 1 → đặt được.
	wrapped.subs = []repository.ClassBookingSubscription{bookingSub(10, 1)}
	wrapped.subs[0].EndDate = f.now.AddDate(0, 1, 0)
	f.svc.d.Bookings = wrapped
	ok, r = reason()
	assert.True(t, ok)
	assert.Equal(t, "", r)

	// Bị khóa đặt lớp → BLOCKED.
	until := f.now.AddDate(0, 0, 3)
	f.repo.members[ptUserA].BookingBlockedUntil = &until
	ok, r = reason()
	assert.False(t, ok)
	assert.Equal(t, "BLOCKED", r)

	// Đã bắt đầu / ngoài hạn đặt trước.
	row := &entity.ClassScheduleDetail{ClassSchedule: entity.ClassSchedule{BranchID: 1, StartAt: f.now.Add(-time.Minute), EndAt: f.now.Add(time.Hour)}}
	got, err := portalNotBookableReason(context.Background(), row, f.now, false, "2026-10-21", nil, false, f.svc.loc)
	require.NoError(t, err)
	assert.Equal(t, "STARTED", got)
	row.StartAt = f.now.AddDate(0, 0, 10)
	got, err = portalNotBookableReason(context.Background(), row, f.now, false, "2026-10-21", nil, false, f.svc.loc)
	require.NoError(t, err)
	assert.Equal(t, "OUT_OF_RANGE", got)
}
