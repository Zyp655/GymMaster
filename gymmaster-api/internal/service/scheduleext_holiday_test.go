package service

import (
	"context"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"

	schedulev1 "gymmaster-api/api/scheduleext/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

func schedOwnerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
}

func schedStrPtr(s string) *string { return &s }
func schedBoolPtr(b bool) *bool    { return &b }

func TestScheduleNormalizeHHMM(t *testing.T) {
	cases := map[string]string{"7:00": "07:00", "07:00": "07:00", " 23:59 ": "23:59", "0:05": "00:05"}
	for in, want := range cases {
		got, ok := scheduleNormalizeHHMM(in)
		if !ok || got != want {
			t.Errorf("%q → %q,%v; muốn %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "7h sáng", "24:00", "7:5", "07:60", "0700", "07:00:00", "-1:00"} {
		if _, ok := scheduleNormalizeHHMM(bad); ok {
			t.Errorf("%q phải bị từ chối", bad)
		}
	}
}

func TestBuildBranchHoliday(t *testing.T) {
	b1 := uint(1)
	// Giờ đặc biệt: closed=false phải được giữ (lỗi cũ: GORM default:true biến thành đóng cửa).
	h, err := buildBranchHoliday(&b1, "2026-12-30", " Giờ đặc biệt ", schedBoolPtr(false), schedStrPtr("7:00"), schedStrPtr("12:30"))
	if err != nil || h.Closed || *h.OpenTime != "07:00" || *h.CloseTime != "12:30" || h.Name != "Giờ đặc biệt" {
		t.Fatalf("giờ đặc biệt sai: %+v err=%v", h, err)
	}
	// Mặc định đóng cửa, bỏ giờ.
	h, err = buildBranchHoliday(nil, "2026-12-31", "Tết", nil, schedStrPtr("07:00"), schedStrPtr("12:00"))
	if err != nil || !h.Closed || h.OpenTime != nil || h.BranchID != nil {
		t.Fatalf("mặc định đóng cửa sai: %+v err=%v", h, err)
	}
	// closed=false không có giờ → ngày lễ mở theo giờ thường.
	h, err = buildBranchHoliday(&b1, "2026-12-31", "Lễ", schedBoolPtr(false), nil, schedStrPtr(" "))
	if err != nil || h.Closed || h.OpenTime != nil {
		t.Fatalf("ngày lễ không giờ sai: %+v err=%v", h, err)
	}
	bad := []struct {
		name        string
		date, label string
		open, close *string
	}{
		{"giờ sai định dạng", "2026-12-30", "x", schedStrPtr("7h sáng"), schedStrPtr("12:00")},
		{"mở >= đóng", "2026-12-30", "x", schedStrPtr("12:00"), schedStrPtr("08:00")},
		{"thiếu giờ đóng", "2026-12-30", "x", schedStrPtr("08:00"), nil},
		{"ngày sai", "30/12/2026", "x", nil, nil},
		{"tên rỗng", "2026-12-30", "  ", nil, nil},
	}
	for _, tc := range bad {
		if _, err := buildBranchHoliday(&b1, tc.date, tc.label, schedBoolPtr(false), tc.open, tc.close); gerror.Code(err) != rescode.BadRequest {
			t.Errorf("%s: muốn BadRequest, nhận %v", tc.name, err)
		}
	}
}

func TestHolidaySlotViolation(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 12, 30, h, m, 0, 0, subVNLoc) }
	special := &entity.BranchHoliday{Name: "Giờ đặc biệt", OpenTime: schedStrPtr("08:00"), CloseTime: schedStrPtr("12:00")}
	if msg := holidaySlotViolation(special, at(9, 0), at(10, 0)); msg != "" {
		t.Errorf("trong giờ đặc biệt phải được: %s", msg)
	}
	if msg := holidaySlotViolation(special, at(8, 0), at(12, 0)); msg != "" {
		t.Errorf("trọn khung giờ đặc biệt phải được: %s", msg)
	}
	if msg := holidaySlotViolation(special, at(11, 30), at(12, 30)); msg == "" {
		t.Errorf("vượt giờ đóng cửa đặc biệt phải bị chặn")
	}
	if msg := holidaySlotViolation(special, at(7, 0), at(8, 0)); msg == "" {
		t.Errorf("trước giờ mở cửa đặc biệt phải bị chặn")
	}
	closed := &entity.BranchHoliday{Name: "Tết", Closed: true}
	if msg := holidaySlotViolation(closed, at(9, 0), at(10, 0)); msg == "" {
		t.Errorf("ngày đóng cửa phải bị chặn")
	}
	if msg := holidaySlotViolation(&entity.BranchHoliday{Name: "Lễ"}, at(5, 0), at(6, 0)); msg != "" {
		t.Errorf("ngày lễ không giờ đặc biệt không chặn: %s", msg)
	}
	if holidaySlotViolation(nil, at(5, 0), at(6, 0)) != "" {
		t.Errorf("không có ngày lễ không chặn")
	}
}

func newSchedSvc() (*scheduleExtService, *mockScheduleExtRepo, time.Time) {
	repo := newMockScheduleExtRepo()
	svc := NewScheduleExtService(repo, nil, nil).(*scheduleExtService)
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, memberVNLoc)
	svc.now = func() time.Time { return now }
	return svc, repo, now
}

func TestScheduleExt_HolidayCreateUpdateImpacts(t *testing.T) {
	svc, repo, _ := newSchedSvc()
	b1 := uint(1)
	day := func(h int) time.Time { return time.Date(2026, 12, 30, h, 0, 0, 0, subVNLoc) }
	repo.classes[1] = &repository.ImpactedClassRow{ClassSchedule: entity.ClassSchedule{ID: 1, BranchID: 1, TrainerID: 6,
		Status: entity.ClassScheduleScheduled, StartAt: day(9), EndAt: day(10)}}
	repo.classes[2] = &repository.ImpactedClassRow{ClassSchedule: entity.ClassSchedule{ID: 2, BranchID: 1, TrainerID: 6,
		Status: entity.ClassScheduleScheduled, StartAt: day(18), EndAt: day(19)}}
	repo.ptSessions[3] = &repository.ImpactedPTSessionRow{PTSession: entity.PTSession{ID: 3, BranchID: 1, TrainerID: 6,
		Status: entity.PTSessionScheduled, StartAt: day(19), EndAt: day(20)}}
	repo.ptSessions[4] = &repository.ImpactedPTSessionRow{PTSession: entity.PTSession{ID: 4, BranchID: 2, TrainerID: 7,
		Status: entity.PTSessionScheduled, StartAt: day(19), EndAt: day(20)}}

	// Giờ đặc biệt 08:00-12:00: lớp 18h và PT 19h ở chi nhánh 1 bị ảnh hưởng; PT chi nhánh 2 không.
	res, err := svc.CreateBranchHoliday(schedOwnerCtx(), &schedulev1.CreateBranchHolidayReq{
		BranchID: &b1, Date: "2026-12-30", Name: "FIX2 giờ đặc biệt", Closed: schedBoolPtr(false),
		OpenTime: schedStrPtr("8:00"), CloseTime: schedStrPtr("12:00"), ExtendSubs: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Closed || res.OpenTime == nil || *res.OpenTime != "08:00" || res.ExtendSubs {
		t.Fatalf("giờ đặc biệt lưu sai (closed phải false, không cộng bù): %+v", res)
	}
	if len(res.AffectedClassSchedules) != 1 || res.AffectedClassSchedules[0].ID != 2 ||
		len(res.AffectedPTSessions) != 1 || res.AffectedPTSessions[0].ID != 3 {
		t.Fatalf("ảnh hưởng sai: %+v / %+v", res.AffectedClassSchedules, res.AffectedPTSessions)
	}

	// Sửa thành đóng cửa cả ngày → cả 2 lớp + PT chi nhánh 1.
	upd, err := svc.UpdateBranchHoliday(schedOwnerCtx(), &schedulev1.UpdateBranchHolidayReq{
		ID: res.ID, Date: "2026-12-30", Name: "FIX2 đóng cửa", Closed: schedBoolPtr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !upd.Closed || upd.OpenTime != nil || len(upd.AffectedClassSchedules) != 2 || len(upd.AffectedPTSessions) != 1 {
		t.Fatalf("sửa đóng cửa sai: %+v", upd)
	}
	if !repo.holidays[res.ID].Closed || repo.holidays[res.ID].Name != "FIX2 đóng cửa" {
		t.Fatalf("repo chưa lưu bản sửa: %+v", repo.holidays[res.ID])
	}

	// Validate giờ khi sửa; id không tồn tại → 404; quản lý chi nhánh 1 không sửa được lịch toàn hệ thống.
	if _, err := svc.UpdateBranchHoliday(schedOwnerCtx(), &schedulev1.UpdateBranchHolidayReq{
		ID: res.ID, Date: "2026-12-30", Name: "x", Closed: schedBoolPtr(false), OpenTime: schedStrPtr("7h sáng"), CloseTime: schedStrPtr("12:00"),
	}); gerror.Code(err) != rescode.BadRequest {
		t.Errorf("giờ sai phải 400, nhận %v", err)
	}
	if _, err := svc.UpdateBranchHoliday(schedOwnerCtx(), &schedulev1.UpdateBranchHolidayReq{ID: 999, Date: "2026-12-30", Name: "x"}); gerror.Code(err) != rescode.NotFound {
		t.Errorf("id không tồn tại phải 404, nhận %v", err)
	}
	global, _ := svc.CreateBranchHoliday(schedOwnerCtx(), &schedulev1.CreateBranchHolidayReq{Date: "2026-12-31", Name: "Toàn hệ thống"})
	mgr := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 2, Roles: []string{consts.RoleManager}, BranchIDs: []uint{1}})
	if _, err := svc.UpdateBranchHoliday(mgr, &schedulev1.UpdateBranchHolidayReq{ID: global.ID, Date: "2026-12-31", Name: "x"}); gerror.Code(err) != rescode.PermissionDenied {
		t.Errorf("quản lý chi nhánh sửa lịch toàn hệ thống phải 403, nhận %v", err)
	}
}

func TestScheduleExt_SubstituteAndReassignValidation(t *testing.T) {
	svc, repo, now := newSchedSvc()
	future := now.Add(48 * time.Hour)
	repo.classes[10] = &repository.ImpactedClassRow{ClassSchedule: entity.ClassSchedule{ID: 10, BranchID: 1, TrainerID: 5,
		Status: entity.ClassScheduleScheduled, StartAt: future, EndAt: future.Add(time.Hour)}}
	repo.classes[11] = &repository.ImpactedClassRow{ClassSchedule: entity.ClassSchedule{ID: 11, BranchID: 1, TrainerID: 5,
		Status: entity.ClassScheduleCancelled, StartAt: future, EndAt: future.Add(time.Hour)}}
	repo.ptSessions[20] = &repository.ImpactedPTSessionRow{PTSession: entity.PTSession{ID: 20, BranchID: 1, TrainerID: 5,
		Status: entity.PTSessionScheduled, StartAt: future, EndAt: future.Add(time.Hour)}}
	repo.ptSessions[21] = &repository.ImpactedPTSessionRow{PTSession: entity.PTSession{ID: 21, BranchID: 1, TrainerID: 5,
		Status: entity.PTSessionCompleted, StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour)}}
	repo.ineligible = map[uint]bool{99: true}
	repo.busy = map[uint]string{8: "PT", 9: "CLASS"}
	ctx := schedOwnerCtx()

	sub := func(id, trainer uint) error {
		_, err := svc.SubstituteClassTrainer(ctx, &schedulev1.SubstituteClassTrainerReq{ID: id, SubstituteTrainerID: trainer})
		return err
	}
	reassign := func(id, trainer uint) error {
		_, err := svc.ReassignPTSession(ctx, &schedulev1.ReassignPTSessionReq{ID: id, TrainerID: trainer})
		return err
	}
	checks := []struct {
		name string
		err  error
		want any
	}{
		{"lớp không tồn tại → 404", sub(404, 6), rescode.NotFound},
		{"lớp đã hủy → 400", sub(11, 6), rescode.BadRequest},
		{"HLV không tồn tại/khóa → 400 (không 500)", sub(10, 99), rescode.BadRequest},
		{"HLV dạy thay trùng PT → 409", sub(10, 8), rescode.Conflict},
		{"HLV dạy thay trùng lớp → 409", sub(10, 9), rescode.Conflict},
		{"dạy thay = HLV chính khi chưa có dạy thay → 400", sub(10, 5), rescode.BadRequest},
		{"PT không tồn tại → 404", reassign(404, 6), rescode.NotFound},
		{"PT đã hoàn thành → 400", reassign(21, 6), rescode.BadRequest},
		{"PT cùng HLV → 400", reassign(20, 5), rescode.BadRequest},
		{"PT HLV không hợp lệ → 400", reassign(20, 99), rescode.BadRequest},
		{"PT HLV trùng lịch → 409", reassign(20, 9), rescode.Conflict},
	}
	for _, c := range checks {
		if gerror.Code(c.err) != c.want {
			t.Errorf("%s: nhận %v", c.name, c.err)
		}
	}
	if repo.classes[10].SubstituteTrainerID != nil || repo.ptSessions[20].TrainerID != 5 {
		t.Fatalf("thao tác bị từ chối không được ghi dữ liệu")
	}

	if err := sub(10, 6); err != nil || repo.classes[10].SubstituteTrainerID == nil || *repo.classes[10].SubstituteTrainerID != 6 {
		t.Fatalf("phân công dạy thay hợp lệ thất bại: %v", err)
	}
	// Chọn lại HLV chính = bỏ dạy thay.
	if err := sub(10, 5); err != nil || repo.classes[10].SubstituteTrainerID != nil {
		t.Fatalf("bỏ dạy thay thất bại: %v", err)
	}
	if err := reassign(20, 6); err != nil || repo.ptSessions[20].TrainerID != 6 || *repo.ptSessions[20].OriginalTrainerID != 5 {
		t.Fatalf("chuyển PT hợp lệ thất bại: %v %+v", err, repo.ptSessions[20])
	}
}

func TestScheduleExt_SaveAvailabilityPerBranch(t *testing.T) {
	svc, repo, _ := newSchedSvc()
	ctx := schedOwnerCtx()
	repo.availabilities = []repository.TrainerAvailabilityRow{
		{TrainerAvailability: entity.TrainerAvailability{ID: 1, TrainerID: 6, BranchID: 2, DOW: 1, StartTime: "06:00", EndTime: "10:00"}},
	}
	if _, err := svc.SaveTrainerAvailability(ctx, &schedulev1.SaveTrainerAvailabilityReq{ID: 6}); gerror.Code(err) != rescode.BadRequest {
		t.Fatalf("thiếu branch_id phải 400, nhận %v", err)
	}
	res, err := svc.SaveTrainerAvailability(ctx, &schedulev1.SaveTrainerAvailabilityReq{ID: 6, BranchID: 1,
		Slots: []schedulev1.TrainerAvailabilitySlotInput{{DOW: 2, StartTime: "7:00", EndTime: "11:30"}}})
	if err != nil || len(res) != 1 || res[0].StartTime != "07:00" || res[0].BranchID != 1 {
		t.Fatalf("lưu lịch rảnh sai: %+v err=%v", res, err)
	}
	if repo.lastReplaceBranch != 1 || len(repo.availabilities) != 2 {
		t.Fatalf("lịch rảnh chi nhánh 2 bị xóa: %+v", repo.availabilities)
	}
	for _, bad := range []schedulev1.TrainerAvailabilitySlotInput{
		{DOW: 1, StartTime: "9:00", EndTime: "10:00", BranchID: 2},
		{DOW: 1, StartTime: "10:00", EndTime: "9:00"},
		{DOW: 1, StartTime: "7h", EndTime: "9:00"},
		{DOW: 1, StartTime: "09:00", EndTime: "10:00", EffectiveFrom: "2026-12-01", EffectiveTo: "2026-11-01"},
	} {
		if _, err := svc.SaveTrainerAvailability(ctx, &schedulev1.SaveTrainerAvailabilityReq{ID: 6, BranchID: 1,
			Slots: []schedulev1.TrainerAvailabilitySlotInput{bad}}); gerror.Code(err) != rescode.BadRequest {
			t.Errorf("slot %+v phải 400, nhận %v", bad, err)
		}
	}
	repo.ineligible = map[uint]bool{77: true}
	if _, err := svc.SaveTrainerAvailability(ctx, &schedulev1.SaveTrainerAvailabilityReq{ID: 77, BranchID: 1}); gerror.Code(err) != rescode.BadRequest {
		t.Errorf("HLV không hợp lệ phải 400, nhận %v", err)
	}
	// Danh sách rỗng = xóa cấu hình chi nhánh 1 (không giới hạn), giữ chi nhánh 2.
	if _, err := svc.SaveTrainerAvailability(ctx, &schedulev1.SaveTrainerAvailabilityReq{ID: 6, BranchID: 1}); err != nil || len(repo.availabilities) != 1 || repo.availabilities[0].BranchID != 2 {
		t.Fatalf("xóa lịch rảnh chi nhánh 1 sai: %v %+v", err, repo.availabilities)
	}
}

func TestScheduleExt_ListLeaveRequests_TrainerSeesOwnOnly(t *testing.T) {
	svc, repo, _ := newSchedSvc()
	b1 := uint(1)
	repo.leaves[1] = &repository.LeaveRequestRow{StaffLeaveRequest: entity.StaffLeaveRequest{ID: 1, StaffID: 6, BranchID: &b1, Status: "PENDING"}}
	repo.leaves[2] = &repository.LeaveRequestRow{StaffLeaveRequest: entity.StaffLeaveRequest{ID: 2, StaffID: 7, BranchID: &b1, Status: "PENDING"}}
	trainer := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 6, Roles: []string{consts.RoleTrainer},
		BranchIDs: []uint{1}, Permissions: []string{"pt.schedule", "pt.view"}})
	res, err := svc.ListLeaveRequests(trainer, &schedulev1.ListLeaveRequestsReq{StaffID: 7})
	if err != nil || res.Total != 1 || res.Items[0].StaffID != 6 {
		t.Fatalf("HLV chỉ được xem đơn của mình: %+v err=%v", res, err)
	}
	all, err := svc.ListLeaveRequests(schedOwnerCtx(), &schedulev1.ListLeaveRequestsReq{})
	if err != nil || all.Total != 2 {
		t.Fatalf("chủ xem tất cả: %+v err=%v", all, err)
	}
}
