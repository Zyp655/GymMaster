package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/qrtoken"
	"gymmaster-api/internal/repository"
)

// checkinHolidayRepo: repo giả có lịch ngày lễ (CheckinRulesRepository).
type checkinHolidayRepo struct {
	*checkinFakeRepo
	holiday *entity.BranchHoliday
}

func (r *checkinHolidayRepo) FindBranchHoliday(_ context.Context, _ uint, day string) (*entity.BranchHoliday, error) {
	if r.holiday != nil && r.holiday.Date.Format("2006-01-02") == day {
		h := *r.holiday
		return &h, nil
	}
	return nil, nil
}
func (r *checkinHolidayRepo) HasSignedContract(context.Context, uint) (bool, error) { return true, nil }
func (r *checkinHolidayRepo) HasOverdueInstallment(context.Context, uint, string) (bool, error) {
	return false, nil
}

var _ repository.CheckinRulesRepository = (*checkinHolidayRepo)(nil)

const checkinOffPeakHours = `[{"dow":[1,2,3,4,5],"from":"09:00","to":"16:00"}]`

func checkinOffPeakSub(id uint) entity.CheckinSubscriptionView {
	s := checkinSub(id, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil)
	s.PlanName, s.AccessHours = "Off-Peak", checkinOffPeakHours
	return s
}

func TestCheckinPickSubscription_OffPeak(t *testing.T) {
	const today = "2026-10-14" // Thứ 4
	in := time.Date(2026, 10, 14, 10, 0, 0, 0, checkinVNLoc)
	out := time.Date(2026, 10, 14, 18, 0, 0, 0, checkinVNLoc)
	normalSess := checkinSub(2, "SESSION", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", checkinIntPtr(5))
	normalTime := checkinSub(3, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil)

	// Ngoài giờ Off-Peak: gói thường còn hiệu lực phải được dùng (lỗi cũ: chọn Off-Peak rồi từ chối).
	if p := checkinPickSubscription([]entity.CheckinSubscriptionView{checkinOffPeakSub(1), normalSess}, 1, today, out); p.chosen == nil || p.chosen.ID != 2 {
		t.Fatalf("ngoài giờ Off-Peak phải dùng gói SESSION thường: %+v", p)
	}
	// Trong giờ: TIME Off-Peak (không trừ buổi) vẫn ưu tiên hơn SESSION.
	if p := checkinPickSubscription([]entity.CheckinSubscriptionView{checkinOffPeakSub(1), normalSess}, 1, today, in); p.chosen == nil || p.chosen.ID != 1 {
		t.Fatalf("trong giờ Off-Peak chọn gói TIME Off-Peak: %+v", p)
	}
	// Hai gói TIME: ưu tiên gói không giới hạn khung giờ.
	if p := checkinPickSubscription([]entity.CheckinSubscriptionView{checkinOffPeakSub(1), normalTime}, 1, today, in); p.chosen == nil || p.chosen.ID != 3 {
		t.Fatalf("ưu tiên gói không giới hạn: %+v", p)
	}
	// Chỉ có Off-Peak ngoài giờ → OUTSIDE_ACCESS_HOURS kèm gói liên quan.
	p := checkinPickSubscription([]entity.CheckinSubscriptionView{checkinOffPeakSub(1)}, 1, today, out)
	if p.chosen != nil || p.reason != entity.CheckinDenyOutsideAccessHours || p.related == nil || p.related.ID != 1 ||
		!strings.Contains(p.message, "09:00-16:00") {
		t.Fatalf("Off-Peak ngoài giờ phải từ chối OUTSIDE_ACCESS_HOURS: %+v", p)
	}
	// Off-Peak ngoài giờ được báo trước gói bảo lưu.
	frozen := checkinSub(4, "TIME", "FROZEN", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil)
	if p := checkinPickSubscription([]entity.CheckinSubscriptionView{frozen, checkinOffPeakSub(1)}, 1, today, out); p.reason != entity.CheckinDenyOutsideAccessHours {
		t.Fatalf("muốn OUTSIDE_ACCESS_HOURS, nhận %+v", p)
	}
}

func TestCheckinScan_OffPeakFallsBackToNormalPlan(t *testing.T) {
	svc, repo := newCheckinFixture()
	at := time.Date(2026, 10, 14, 18, 0, 0, 0, checkinVNLoc) // ngoài 09:00-16:00
	svc.now = func() time.Time { return at }
	repo.subs = []entity.CheckinSubscriptionView{checkinOffPeakSub(1),
		checkinSub(2, "SESSION", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", checkinIntPtr(5))}
	tok, _ := qrtoken.Generate(checkinTestMember, checkinTestSecret, at, time.Minute)
	res, err := svc.Scan(checkinOwnerCtx(), tok, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Result != entity.CheckinAllowed || res.Subscription == nil || res.Subscription.ID != 2 {
		t.Fatalf("muốn ALLOWED bằng gói 2, nhận %+v", res)
	}
	if rem := *res.Subscription.RemainingSessions; rem != 4 {
		t.Errorf("phải trừ 1 buổi gói SESSION, còn %d", rem)
	}

	svc, repo = newCheckinFixture()
	svc.now = func() time.Time { return at }
	repo.subs = []entity.CheckinSubscriptionView{checkinOffPeakSub(1)}
	res, err = svc.Scan(checkinOwnerCtx(), tok, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Result != entity.CheckinDenied || res.DenyReason != entity.CheckinDenyOutsideAccessHours || res.Subscription == nil || res.Subscription.ID != 1 {
		t.Fatalf("chỉ Off-Peak ngoài giờ phải DENIED OUTSIDE_ACCESS_HOURS: %+v", res)
	}
}

func TestCheckinScan_HolidaySpecialHours(t *testing.T) {
	open, closeT := "08:00", "12:00"
	day := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		holiday    *entity.BranchHoliday
		at         time.Time
		wantReason string
		wantMsg    string
	}{
		{"trong giờ đặc biệt → vào", &entity.BranchHoliday{Date: day, Name: "Giờ lễ", OpenTime: &open, CloseTime: &closeT},
			time.Date(2026, 10, 14, 9, 0, 0, 0, checkinVNLoc), "", ""},
		{"ngoài giờ đặc biệt (dù trong giờ thường) → từ chối", &entity.BranchHoliday{Date: day, Name: "Giờ lễ", OpenTime: &open, CloseTime: &closeT},
			time.Date(2026, 10, 14, 13, 0, 0, 0, checkinVNLoc), entity.CheckinDenyBranchClosed, "Ngoài giờ mở cửa ngày lễ Giờ lễ (08:00 - 12:00)"},
		{"đúng giờ đóng cửa đặc biệt → từ chối", &entity.BranchHoliday{Date: day, Name: "Giờ lễ", OpenTime: &open, CloseTime: &closeT},
			time.Date(2026, 10, 14, 12, 0, 0, 0, checkinVNLoc), entity.CheckinDenyBranchClosed, "08:00 - 12:00"},
		{"đóng cửa cả ngày → từ chối nghỉ lễ", &entity.BranchHoliday{Date: day, Name: "Tết", Closed: true},
			time.Date(2026, 10, 14, 9, 0, 0, 0, checkinVNLoc), entity.CheckinDenyBranchClosed, "Chi nhánh nghỉ lễ (Tết)"},
		{"ngày lễ không giờ đặc biệt → giờ thường", &entity.BranchHoliday{Date: day, Name: "Lễ"},
			time.Date(2026, 10, 14, 20, 0, 0, 0, checkinVNLoc), "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, base := newCheckinFixture()
			base.subs = []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil)}
			svc.repo = &checkinHolidayRepo{checkinFakeRepo: base, holiday: tc.holiday}
			svc.now = func() time.Time { return tc.at }
			tok, _ := qrtoken.Generate(checkinTestMember, checkinTestSecret, tc.at, time.Minute)
			res, err := svc.Scan(checkinOwnerCtx(), tok, 1)
			if err != nil {
				t.Fatal(err)
			}
			if res.DenyReason != tc.wantReason || !strings.Contains(res.DenyMessage, tc.wantMsg) {
				t.Fatalf("nhận %s/%q, muốn %s/%q", res.DenyReason, res.DenyMessage, tc.wantReason, tc.wantMsg)
			}
		})
	}
}

func TestCheckinToItem_HolidayDenyMessage(t *testing.T) {
	msg := "Chi nhánh nghỉ lễ (Tết), không thể check-in"
	row := repository.CheckinRow{Checkin: entity.Checkin{Result: entity.CheckinDenied, DenyReason: entity.CheckinDenyBranchClosed}, HolidayDenyMessage: &msg}
	if got := checkinToItem(row).DenyMessage; got != msg {
		t.Fatalf("lịch sử phải hiện thông điệp nghỉ lễ, nhận %q", got)
	}
	// Thông điệp đã lưu lúc quét được ưu tiên (không đổi theo lịch lễ sửa sau).
	row.StoredDenyMessage = "Ngoài giờ mở cửa ngày lễ X (05:30 - 06:00)"
	if got := checkinToItem(row).DenyMessage; got != row.StoredDenyMessage {
		t.Fatalf("phải dùng thông điệp đã lưu, nhận %q", got)
	}
	row.HolidayDenyMessage, row.StoredDenyMessage = nil, ""
	if got := checkinToItem(row).DenyMessage; got != "Ngoài giờ mở cửa của chi nhánh" {
		t.Fatalf("ngày thường giữ thông điệp ngoài giờ, nhận %q", got)
	}
}
