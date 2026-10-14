package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/checkin/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/qrtoken"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// ---- Fake ----------------------------------------------------------------------

type checkinFakeEvent struct {
	subID uint
	typ   string
	data  map[string]any
}

type checkinFakeRepo struct {
	members  map[uint]*entity.Member
	branches map[uint]*entity.Branch
	subs     []entity.CheckinSubscriptionView
	checkins []entity.Checkin
	events   []checkinFakeEvent
}

func (f *checkinFakeRepo) FindMember(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		c := *m
		return &c, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *checkinFakeRepo) LockMember(ctx context.Context, id uint) (*entity.Member, error) {
	return f.FindMember(ctx, id)
}
func (f *checkinFakeRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	if b, ok := f.branches[id]; ok {
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *checkinFakeRepo) LockSubscriptions(_ context.Context, memberID uint) ([]entity.CheckinSubscriptionView, error) {
	var out []entity.CheckinSubscriptionView
	for _, s := range f.subs {
		if s.MemberID == memberID && (s.Status == "ACTIVE" || s.Status == "FROZEN") {
			if s.RemainingSessions != nil { // sao chép để service không sửa trực tiếp dữ liệu fake
				r := *s.RemainingSessions
				s.RemainingSessions = &r
			}
			out = append(out, s)
		}
	}
	return out, nil
}
func (f *checkinFakeRepo) row(c entity.Checkin) *repository.CheckinRow {
	r := &repository.CheckinRow{Checkin: c}
	if b, ok := f.branches[c.BranchID]; ok {
		r.BranchName = b.Name
	}
	if m, ok := f.members[c.MemberID]; ok {
		r.MemberName, r.MemberCode = m.FullName, m.MemberCode
	}
	return r
}
func (f *checkinFakeRepo) LastAllowedSince(_ context.Context, memberID uint, since time.Time) (*repository.CheckinRow, error) {
	var best *entity.Checkin
	for i := range f.checkins {
		c := &f.checkins[i]
		if c.MemberID == memberID && c.Result == entity.CheckinAllowed && !c.CheckedInAt.Before(since) &&
			(best == nil || c.CheckedInAt.After(best.CheckedInAt)) {
			best = c
		}
	}
	if best == nil {
		return nil, nil
	}
	return f.row(*best), nil
}
func (f *checkinFakeRepo) Create(_ context.Context, c *entity.Checkin) error {
	c.ID = uint(len(f.checkins) + 1)
	f.checkins = append(f.checkins, *c)
	return nil
}
func (f *checkinFakeRepo) UseSession(_ context.Context, id uint) (int, error) {
	for i := range f.subs {
		s := &f.subs[i]
		if s.ID == id && s.RemainingSessions != nil && *s.RemainingSessions > 0 {
			r := *s.RemainingSessions - 1
			s.RemainingSessions = &r
			return r, nil
		}
	}
	return 0, gorm.ErrRecordNotFound
}
func (f *checkinFakeRepo) CreateSubscriptionEvent(_ context.Context, subID uint, typ string, data map[string]any, _ *uint) error {
	f.events = append(f.events, checkinFakeEvent{subID: subID, typ: typ, data: data})
	return nil
}
func (f *checkinFakeRepo) FindByID(_ context.Context, id uint) (*repository.CheckinRow, error) {
	for _, c := range f.checkins {
		if c.ID == id {
			return f.row(c), nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *checkinFakeRepo) Checkout(_ context.Context, id uint, at time.Time) (bool, error) {
	for i := range f.checkins {
		if f.checkins[i].ID == id && f.checkins[i].CheckedOutAt == nil {
			f.checkins[i].CheckedOutAt = &at
			return true, nil
		}
	}
	return false, nil
}
func (f *checkinFakeRepo) List(_ context.Context, _ dto.Query, flt repository.CheckinListFilter) ([]repository.CheckinRow, int64, error) {
	var out []repository.CheckinRow
	for _, c := range f.checkins {
		if flt.MemberID == 0 || c.MemberID == flt.MemberID {
			out = append(out, *f.row(c))
		}
	}
	return out, int64(len(out)), nil
}
func (f *checkinFakeRepo) Present(_ context.Context, _ []uint, since time.Time) ([]repository.CheckinRow, error) {
	var out []repository.CheckinRow
	for _, c := range f.checkins {
		if c.Result == entity.CheckinAllowed && c.CheckedOutAt == nil && !c.CheckedInAt.Before(since) {
			out = append(out, *f.row(c))
		}
	}
	return out, nil
}
func (f *checkinFakeRepo) Stats(context.Context, []uint, time.Time, time.Time) (*repository.CheckinStatsRows, error) {
	return &repository.CheckinStatsRows{
		ByHour:   []repository.CheckinHourCount{{Hour: 8, Result: "ALLOWED", Count: 3}, {Hour: 8, Result: "DENIED", Count: 1}, {Hour: 18, Result: "ALLOWED", Count: 2}},
		ByReason: []repository.CheckinReasonCount{{DenyReason: "DUPLICATE", Count: 1}},
		Present:  2, UniqueMembers: 4,
	}, nil
}

type checkinFakeSettings struct{ values map[string]int }

func (s checkinFakeSettings) Int(_ context.Context, key string, def int) int {
	if v, ok := s.values[key]; ok {
		return v
	}
	return def
}
func (checkinFakeSettings) IntSlice(_ context.Context, _ string, def []int) []int { return def }
func (checkinFakeSettings) All(context.Context) ([]entity.Setting, error)         { return nil, nil }
func (checkinFakeSettings) Set(context.Context, string, json.RawMessage, *uint) error {
	return nil
}

// ---- Fixture -------------------------------------------------------------------

// 2026-10-14 09:00 giờ VN.
var checkinTestNow = time.Date(2026, 10, 14, 2, 0, 0, 0, time.UTC)

const (
	checkinTestMember = uint(10)
	checkinTestSecret = "secret-of-member-10"
)

func checkinDay(s string) time.Time {
	d, _ := time.Parse(consts.DateLayout, s)
	return d
}

func checkinIntPtr(v int) *int { return &v }

func checkinSub(id uint, typ, status, start, end string, branchID uint, scope string, remaining *int) entity.CheckinSubscriptionView {
	return entity.CheckinSubscriptionView{
		ID: id, MemberID: checkinTestMember, PlanID: id, BranchID: branchID, BranchName: map[uint]string{1: "Cầu Giấy", 2: "Hà Đông"}[branchID],
		StartDate: checkinDay(start), EndDate: checkinDay(end), RemainingSessions: remaining, TotalSessions: remaining,
		Status: status, PlanName: "Gói " + typ, PlanType: typ, AccessScope: scope,
	}
}

func newCheckinFixture() (*checkinService, *checkinFakeRepo) {
	repo := &checkinFakeRepo{
		members: map[uint]*entity.Member{
			checkinTestMember: {BaseEntity: entity.BaseEntity{ID: checkinTestMember}, MemberCode: "GM000010", FullName: "Trần Thị B",
				Phone: "0912345678", Status: entity.MemberStatusActive, QRSecret: checkinTestSecret, HomeBranchID: 1},
		},
		branches: map[uint]*entity.Branch{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Name: "Cầu Giấy", OpenTime: "05:00", CloseTime: "22:00", Status: "ACTIVE"},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Name: "Hà Đông", OpenTime: "05:00", CloseTime: "22:00", Status: "ACTIVE"},
			3: {BaseEntity: entity.BaseEntity{ID: 3}, Name: "Đóng cửa", OpenTime: "05:00", CloseTime: "22:00", Status: "INACTIVE"},
		},
	}
	svc := NewCheckinService(repo, noTx{}, checkinFakeSettings{values: map[string]int{}}).(*checkinService)
	svc.now = func() time.Time { return checkinTestNow }
	return svc, repo
}

func checkinOwnerCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: 1, Roles: []string{consts.RoleOwner}})
}

// checkinStaffCtx: lễ tân chỉ thuộc chi nhánh 1.
func checkinStaffCtx() context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{
		UserID: 2, Roles: []string{consts.RoleReceptionist}, BranchIDs: []uint{1},
		Permissions: []string{consts.PermCheckinScan, consts.PermCheckinView},
	})
}

func checkinValidToken() string {
	tok, _ := qrtoken.Generate(checkinTestMember, checkinTestSecret, checkinTestNow, time.Minute)
	return tok
}

func checkinErrCode(err error) int { return gerror.Code(err).Code() }

// ---- Bảng quyết định ALLOWED / DENIED -------------------------------------------

func TestCheckinScan_Decisions(t *testing.T) {
	const timeOK, sessOK = "2026-10-01", "2026-12-31"
	cases := []struct {
		name       string
		token      func() string
		branchID   uint
		now        time.Time
		status     string
		subs       []entity.CheckinSubscriptionView
		prior      []entity.Checkin
		wantResult string
		wantReason string
		wantSubID  uint // gói được chọn / liên quan (0 = không có)
		wantStored bool // có ghi bảng checkins
	}{
		{name: "gói TIME hợp lệ", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			wantResult: "ALLOWED", wantSubID: 1, wantStored: true},
		{name: "QR sai định dạng – không ghi DB", token: func() string { return "hello" },
			wantResult: "DENIED", wantReason: "QR_INVALID"},
		{name: "QR của hội viên không tồn tại – không ghi DB", token: func() string {
			tok, _ := qrtoken.Generate(999, "x", checkinTestNow, time.Minute)
			return tok
		}, wantResult: "DENIED", wantReason: "QR_INVALID"},
		{name: "QR sai chữ ký – ghi DB", token: func() string {
			tok, _ := qrtoken.Generate(checkinTestMember, "secret-cu", checkinTestNow, time.Minute)
			return tok
		}, subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "QR_INVALID", wantStored: true},
		{name: "QR hết hạn – ghi DB", token: func() string {
			tok, _ := qrtoken.Generate(checkinTestMember, checkinTestSecret, checkinTestNow.Add(-2*time.Minute), time.Minute)
			return tok
		}, subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "QR_EXPIRED", wantStored: true},
		{name: "ngoài giờ mở cửa (23:00)", now: time.Date(2026, 10, 14, 16, 0, 0, 0, time.UTC),
			subs:       []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "BRANCH_CLOSED", wantStored: true},
		{name: "trước giờ mở cửa (04:59)", now: time.Date(2026, 10, 13, 21, 59, 0, 0, time.UTC),
			subs:       []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "BRANCH_CLOSED", wantStored: true},
		{name: "hội viên INACTIVE", status: "INACTIVE",
			subs:       []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "MEMBER_INACTIVE", wantStored: true},
		{name: "hội viên BANNED", status: "BANNED", wantResult: "DENIED", wantReason: "MEMBER_INACTIVE", wantStored: true},
		{name: "không có gói", wantResult: "DENIED", wantReason: "NO_ACTIVE_SUBSCRIPTION", wantStored: true},
		{name: "gói đã quá end_date", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", "2026-09-01", "2026-10-13", 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "NO_ACTIVE_SUBSCRIPTION", wantStored: true},
		{name: "gói nối tiếp chưa tới start_date", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", "2026-10-15", "2026-11-15", 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "NO_ACTIVE_SUBSCRIPTION", wantSubID: 1, wantStored: true},
		{name: "gói hết hạn đúng hôm nay vẫn vào được", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, "2026-10-14", 1, "SINGLE_BRANCH", nil)},
			wantResult: "ALLOWED", wantSubID: 1, wantStored: true},
		{name: "gói bảo lưu", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "FROZEN", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "SUBSCRIPTION_FROZEN", wantSubID: 1, wantStored: true},
		{name: "gói SINGLE_BRANCH ở chi nhánh khác", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 2, "SINGLE_BRANCH", nil)},
			wantResult: "DENIED", wantReason: "WRONG_BRANCH", wantSubID: 1, wantStored: true},
		{name: "gói ALL_BRANCHES mua ở chi nhánh khác vẫn vào", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 2, "ALL_BRANCHES", nil)},
			wantResult: "ALLOWED", wantSubID: 1, wantStored: true},
		{name: "gói SESSION hết buổi", subs: []entity.CheckinSubscriptionView{checkinSub(1, "SESSION", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", checkinIntPtr(0))},
			wantResult: "DENIED", wantReason: "NO_SESSIONS_LEFT", wantSubID: 1, wantStored: true},
		{name: "hết buổi ở chi nhánh này được ưu tiên báo hơn WRONG_BRANCH", subs: []entity.CheckinSubscriptionView{
			checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 2, "SINGLE_BRANCH", nil),
			checkinSub(2, "SESSION", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", checkinIntPtr(0)),
		}, wantResult: "DENIED", wantReason: "NO_SESSIONS_LEFT", wantSubID: 2, wantStored: true},
		{name: "FROZEN được ưu tiên báo khi không có gói hợp lệ", subs: []entity.CheckinSubscriptionView{
			checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 2, "SINGLE_BRANCH", nil),
			checkinSub(2, "TIME", "FROZEN", timeOK, sessOK, 1, "SINGLE_BRANCH", nil),
		}, wantResult: "DENIED", wantReason: "SUBSCRIPTION_FROZEN", wantSubID: 2, wantStored: true},
		{name: "gói FROZEN + gói ACTIVE hợp lệ → vào bằng gói ACTIVE", subs: []entity.CheckinSubscriptionView{
			checkinSub(1, "TIME", "FROZEN", timeOK, sessOK, 1, "SINGLE_BRANCH", nil),
			checkinSub(2, "SESSION", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", checkinIntPtr(5)),
		}, wantResult: "ALLOWED", wantSubID: 2, wantStored: true},
		{name: "trùng trong cửa sổ 60 phút", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			prior:      []entity.Checkin{{MemberID: checkinTestMember, BranchID: 2, CheckedInAt: checkinTestNow.Add(-30 * time.Minute), Result: "ALLOWED", Method: "QR"}},
			wantResult: "DENIED", wantReason: "DUPLICATE", wantSubID: 1, wantStored: true},
		{name: "lần trước bị DENIED thì không tính trùng", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			prior:      []entity.Checkin{{MemberID: checkinTestMember, BranchID: 1, CheckedInAt: checkinTestNow.Add(-1 * time.Minute), Result: "DENIED", DenyReason: "QR_EXPIRED", Method: "QR"}},
			wantResult: "ALLOWED", wantSubID: 1, wantStored: true},
		{name: "ngoài cửa sổ chống trùng", subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 1, "SINGLE_BRANCH", nil)},
			prior:      []entity.Checkin{{MemberID: checkinTestMember, BranchID: 1, CheckedInAt: checkinTestNow.Add(-61 * time.Minute), Result: "ALLOWED", Method: "QR"}},
			wantResult: "ALLOWED", wantSubID: 1, wantStored: true},
		{name: "chi nhánh 2 vẫn quét được (OWNER)", branchID: 2, subs: []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", timeOK, sessOK, 2, "SINGLE_BRANCH", nil)},
			wantResult: "ALLOWED", wantSubID: 1, wantStored: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newCheckinFixture()
			if !tc.now.IsZero() {
				svc.now = func() time.Time { return tc.now }
			}
			if tc.status != "" {
				repo.members[checkinTestMember].Status = tc.status
			}
			repo.subs = tc.subs
			for _, c := range tc.prior {
				_ = repo.Create(context.Background(), &c)
			}
			before := len(repo.checkins)
			tok := checkinValidToken()
			if !tc.now.IsZero() {
				tok, _ = qrtoken.Generate(checkinTestMember, checkinTestSecret, tc.now, time.Minute)
			}
			if tc.token != nil {
				tok = tc.token()
			}
			branchID := tc.branchID
			if branchID == 0 {
				branchID = 1
			}
			res, err := svc.Scan(checkinOwnerCtx(), tok, branchID)
			if err != nil {
				t.Fatalf("Scan lỗi: %v", err)
			}
			if res.Result != tc.wantResult || res.DenyReason != tc.wantReason {
				t.Fatalf("kết quả = %s/%s, muốn %s/%s (%s)", res.Result, res.DenyReason, tc.wantResult, tc.wantReason, res.DenyMessage)
			}
			if tc.wantReason != "" && res.DenyMessage == "" {
				t.Errorf("thiếu deny_message")
			}
			if tc.wantReason == "" && res.DenyMessage != "" {
				t.Errorf("ALLOWED không được có deny_message: %q", res.DenyMessage)
			}
			stored := len(repo.checkins) > before
			if stored != tc.wantStored {
				t.Fatalf("ghi DB = %v, muốn %v", stored, tc.wantStored)
			}
			if stored {
				rec := repo.checkins[len(repo.checkins)-1]
				if rec.Result != tc.wantResult || rec.DenyReason != tc.wantReason || rec.Method != "QR" || rec.BranchID != branchID {
					t.Errorf("bản ghi sai: %+v", rec)
				}
				if (rec.Result == "ALLOWED") != (rec.SubscriptionID != nil) {
					t.Errorf("subscription_id chỉ gắn khi ALLOWED: %+v", rec)
				}
				if res.Checkin == nil || res.Checkin.ID != rec.ID || res.Member == nil || res.Member.Code != "GM000010" {
					t.Errorf("response thiếu checkin/member: %+v", res)
				}
			} else if res.Checkin != nil || res.Member != nil {
				t.Errorf("QR hỏng hoàn toàn không có member/checkin: %+v", res)
			}
			gotSub := uint(0)
			if res.Subscription != nil {
				gotSub = res.Subscription.ID
			}
			if gotSub != tc.wantSubID {
				t.Errorf("gói = %d, muốn %d", gotSub, tc.wantSubID)
			}
			if tc.wantReason == "DUPLICATE" && (res.PreviousCheckin == nil || res.PreviousCheckin.BranchName != "Hà Đông") {
				t.Errorf("DUPLICATE phải trả lần check-in trước: %+v", res.PreviousCheckin)
			}
		})
	}
}

// ---- Chọn gói ưu tiên ----------------------------------------------------------

func TestCheckinPickSubscription_Priority(t *testing.T) {
	const today = "2026-10-14"
	cases := []struct {
		name string
		subs []entity.CheckinSubscriptionView
		want uint
	}{
		{"TIME trước SESSION dù SESSION hết hạn sớm hơn", []entity.CheckinSubscriptionView{
			checkinSub(1, "SESSION", "ACTIVE", "2026-10-01", "2026-10-20", 1, "SINGLE_BRANCH", checkinIntPtr(5)),
			checkinSub(2, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil),
		}, 2},
		{"hai gói TIME → hết hạn sớm nhất", []entity.CheckinSubscriptionView{
			checkinSub(1, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil),
			checkinSub(2, "TIME", "ACTIVE", "2026-10-01", "2026-11-30", 1, "ALL_BRANCHES", nil),
		}, 2},
		{"hai gói SESSION → bỏ gói hết buổi, chọn hết hạn sớm nhất", []entity.CheckinSubscriptionView{
			checkinSub(1, "SESSION", "ACTIVE", "2026-10-01", "2026-10-31", 1, "SINGLE_BRANCH", checkinIntPtr(0)),
			checkinSub(2, "SESSION", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", checkinIntPtr(3)),
			checkinSub(3, "SESSION", "ACTIVE", "2026-10-01", "2026-11-30", 1, "SINGLE_BRANCH", checkinIntPtr(8)),
		}, 3},
		{"bỏ gói TIME khác chi nhánh, dùng SESSION tại đây", []entity.CheckinSubscriptionView{
			checkinSub(1, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 2, "SINGLE_BRANCH", nil),
			checkinSub(2, "SESSION", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", checkinIntPtr(1)),
		}, 2},
		{"cùng ngày hết hạn → id nhỏ hơn", []entity.CheckinSubscriptionView{
			checkinSub(5, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil),
			checkinSub(4, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil),
		}, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := checkinPickSubscription(tc.subs, 1, today)
			if p.chosen == nil || p.chosen.ID != tc.want {
				t.Fatalf("chọn %+v, muốn gói %d", p.chosen, tc.want)
			}
		})
	}
}

// ---- Trừ buổi SESSION + cảnh báo ------------------------------------------------

func TestCheckinScan_SessionUsed(t *testing.T) {
	svc, repo := newCheckinFixture()
	repo.subs = []entity.CheckinSubscriptionView{checkinSub(7, "SESSION", "ACTIVE", "2026-10-01", "2026-10-17", 1, "SINGLE_BRANCH", checkinIntPtr(3))}

	res, err := svc.Scan(checkinStaffCtx(), checkinValidToken(), 1)
	if err != nil || res.Result != "ALLOWED" {
		t.Fatalf("muốn ALLOWED, được %+v err=%v", res, err)
	}
	if got := *repo.subs[0].RemainingSessions; got != 2 {
		t.Fatalf("remaining_sessions = %d, muốn 2", got)
	}
	if res.Subscription.RemainingSessions == nil || *res.Subscription.RemainingSessions != 2 || res.Subscription.DaysLeft != 3 {
		t.Errorf("subscription trả về sai: %+v", res.Subscription)
	}
	if len(repo.events) != 1 || repo.events[0].typ != "SESSION_USED" || repo.events[0].subID != 7 ||
		repo.events[0].data["checkin_id"] != res.Checkin.ID || repo.events[0].data["remaining"] != 2 {
		t.Fatalf("subscription_events sai: %+v", repo.events)
	}
	want := []string{"Gói sắp hết hạn trong 3 ngày", "Gói chỉ còn 2 buổi"}
	if len(res.Warnings) != 2 || res.Warnings[0] != want[0] || res.Warnings[1] != want[1] {
		t.Errorf("warnings = %v, muốn %v", res.Warnings, want)
	}
	if staff := repo.checkins[0].StaffID; staff == nil || *staff != 2 {
		t.Errorf("staff_id phải là người quét")
	}

	// Quét lại ngay: DUPLICATE, không trừ thêm buổi.
	res, err = svc.Scan(checkinStaffCtx(), checkinValidToken(), 1)
	if err != nil || res.DenyReason != "DUPLICATE" {
		t.Fatalf("muốn DUPLICATE, được %+v err=%v", res, err)
	}
	if got := *repo.subs[0].RemainingSessions; got != 2 || len(repo.events) != 1 {
		t.Fatalf("DENIED không được trừ buổi: remaining=%d events=%d", got, len(repo.events))
	}

	// Gói TIME không trừ buổi, không ghi event.
	svc2, repo2 := newCheckinFixture()
	repo2.subs = []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil)}
	res, _ = svc2.Scan(checkinOwnerCtx(), checkinValidToken(), 1)
	if res.Result != "ALLOWED" || len(repo2.events) != 0 || len(res.Warnings) != 0 {
		t.Errorf("gói TIME: %+v events=%v", res, repo2.events)
	}
}

func TestCheckinWarnings(t *testing.T) {
	cases := []struct {
		sub  checkinSubWarnCase
		want []string
	}{
		{checkinSubWarnCase{"TIME", 30, nil}, []string{}},
		{checkinSubWarnCase{"TIME", 0, nil}, []string{"Gói hết hạn hôm nay"}},
		{checkinSubWarnCase{"TIME", 7, nil}, []string{"Gói sắp hết hạn trong 7 ngày"}},
		{checkinSubWarnCase{"SESSION", 30, checkinIntPtr(0)}, []string{"Đây là buổi cuối cùng của gói"}},
		{checkinSubWarnCase{"SESSION", 30, checkinIntPtr(4)}, []string{}},
	}
	for _, tc := range cases {
		got := checkinWarnings(tc.sub.toV1())
		if len(got) != len(tc.want) {
			t.Fatalf("%+v: %v, muốn %v", tc.sub, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%+v: %v, muốn %v", tc.sub, got, tc.want)
			}
		}
	}
}

// ---- Thủ công, phạm vi chi nhánh, lỗi hệ thống ----------------------------------

func TestCheckinManual(t *testing.T) {
	svc, repo := newCheckinFixture()
	repo.subs = []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil)}

	res, err := svc.Manual(checkinStaffCtx(), checkinTestMember, 1, "  Quên điện thoại ")
	if err != nil || res.Result != "ALLOWED" {
		t.Fatalf("muốn ALLOWED: %+v err=%v", res, err)
	}
	if rec := repo.checkins[0]; rec.Method != "MANUAL" || rec.Note != "Quên điện thoại" {
		t.Errorf("bản ghi thủ công sai: %+v", rec)
	}

	errCases := []struct {
		name     string
		ctx      context.Context
		memberID uint
		branchID uint
		code     int
	}{
		{"hội viên không tồn tại → 404", checkinOwnerCtx(), 999, 1, rescode.NotFound.Code()},
		{"chi nhánh ngoài phạm vi → 403", checkinStaffCtx(), checkinTestMember, 2, rescode.PermissionDenied.Code()},
		{"chi nhánh không tồn tại → 404", checkinOwnerCtx(), checkinTestMember, 99, rescode.NotFound.Code()},
		{"chi nhánh ngừng hoạt động → 400", checkinOwnerCtx(), checkinTestMember, 3, rescode.BadRequest.Code()},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			n := len(repo.checkins)
			_, err := svc.Manual(tc.ctx, tc.memberID, tc.branchID, "")
			if checkinErrCode(err) != tc.code {
				t.Fatalf("mã lỗi = %d, muốn %d (%v)", checkinErrCode(err), tc.code, err)
			}
			if len(repo.checkins) != n {
				t.Errorf("lỗi hệ thống/validate không được ghi checkins")
			}
		})
	}

	// Quét QR tại chi nhánh ngoài phạm vi cũng là lỗi 403 (không phải DENIED).
	if _, err := svc.Scan(checkinStaffCtx(), checkinValidToken(), 2); checkinErrCode(err) != rescode.PermissionDenied.Code() {
		t.Errorf("Scan ngoài phạm vi: %v", err)
	}
}

// ---- Checkout / đang tập / lịch sử ---------------------------------------------

func TestCheckinCheckout(t *testing.T) {
	svc, repo := newCheckinFixture()
	repo.subs = []entity.CheckinSubscriptionView{checkinSub(1, "TIME", "ACTIVE", "2026-10-01", "2026-12-31", 1, "SINGLE_BRANCH", nil)}
	ok, _ := svc.Scan(checkinOwnerCtx(), checkinValidToken(), 1)
	denied, _ := svc.Scan(checkinOwnerCtx(), checkinValidToken(), 1) // DUPLICATE

	present, err := svc.Present(checkinStaffCtx(), 0)
	if err != nil || len(present) != 1 || present[0].CheckinID != ok.Checkin.ID {
		t.Fatalf("present = %+v err=%v", present, err)
	}

	svc.now = func() time.Time { return checkinTestNow.Add(90 * time.Minute) }
	item, err := svc.Checkout(checkinStaffCtx(), ok.Checkin.ID)
	if err != nil || item.CheckedOutAt == nil || !item.CheckedOutAt.Equal(checkinTestNow.Add(90*time.Minute)) {
		t.Fatalf("checkout: %+v err=%v", item, err)
	}
	if _, err := svc.Checkout(checkinStaffCtx(), ok.Checkin.ID); checkinErrCode(err) != rescode.Conflict.Code() {
		t.Errorf("checkout 2 lần phải 409: %v", err)
	}
	if _, err := svc.Checkout(checkinStaffCtx(), denied.Checkin.ID); checkinErrCode(err) != rescode.BadRequest.Code() {
		t.Errorf("checkout lượt DENIED phải 400: %v", err)
	}
	if _, err := svc.Checkout(checkinStaffCtx(), 999); checkinErrCode(err) != rescode.NotFound.Code() {
		t.Errorf("checkout không tồn tại phải 404: %v", err)
	}
	if present, _ := svc.Present(checkinStaffCtx(), 0); len(present) != 0 {
		t.Errorf("sau checkout không còn đang tập: %+v", present)
	}
	if _, err := svc.Present(checkinStaffCtx(), 2); checkinErrCode(err) != rescode.PermissionDenied.Code() {
		t.Errorf("present chi nhánh ngoài phạm vi phải 403: %v", err)
	}

	// Checkout lượt ở chi nhánh ngoài phạm vi.
	repo.checkins = append(repo.checkins, entity.Checkin{ID: 50, MemberID: checkinTestMember, BranchID: 2, CheckedInAt: checkinTestNow, Result: "ALLOWED", Method: "QR"})
	if _, err := svc.Checkout(checkinStaffCtx(), 50); checkinErrCode(err) != rescode.PermissionDenied.Code() {
		t.Errorf("checkout ngoài phạm vi phải 403: %v", err)
	}

	page, err := svc.ListMemberCheckins(context.Background(), checkinTestMember, dto.PageReq{})
	if err != nil || page.Total != 3 || page.Limit != dto.DefaultLimit || page.Items[1].DenyMessage == "" {
		t.Errorf("ListMemberCheckins: %+v err=%v", page, err)
	}
}

func TestCheckinStats(t *testing.T) {
	svc, _ := newCheckinFixture()
	res, err := svc.Stats(checkinStaffCtx(), 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Date != "2026-10-14" || res.Total != 6 || res.Allowed != 5 || res.Denied != 1 || res.Present != 2 ||
		len(res.ByHour) != 24 || res.ByHour[8].Allowed != 3 || res.ByHour[8].Denied != 1 || res.ByHour[18].Allowed != 2 ||
		len(res.DenyReasons) != 1 || res.DenyReasons[0].Message == "" {
		t.Fatalf("stats sai: %+v", res)
	}
	if _, err := svc.Stats(checkinStaffCtx(), 0, "14/10/2026"); checkinErrCode(err) != rescode.BadRequest.Code() {
		t.Errorf("ngày sai định dạng phải 400: %v", err)
	}
	if _, err := svc.Stats(checkinStaffCtx(), 2, ""); checkinErrCode(err) != rescode.PermissionDenied.Code() {
		t.Errorf("chi nhánh ngoài phạm vi phải 403: %v", err)
	}
}

// ---- QR động -------------------------------------------------------------------

func TestCheckinIssueQR(t *testing.T) {
	svc, _ := newCheckinFixture()
	svc.settings = checkinFakeSettings{values: map[string]int{SettingCheckinQRTTLSeconds: 45}}

	tok, exp, err := svc.IssueQR(context.Background(), checkinTestMember)
	if err != nil {
		t.Fatal(err)
	}
	if !exp.Equal(checkinTestNow.Add(45 * time.Second)) {
		t.Errorf("expires_at = %v, muốn now+45s", exp)
	}
	if err := qrtoken.Verify(tok, checkinTestSecret, checkinTestNow.Add(44*time.Second)); err != nil {
		t.Errorf("mã phải hợp lệ trong TTL: %v", err)
	}
	if err := qrtoken.Verify(tok, checkinTestSecret, checkinTestNow.Add(46*time.Second)); !errors.Is(err, qrtoken.ErrExpired) {
		t.Errorf("mã phải hết hạn sau TTL: %v", err)
	}
	res, err := svc.MemberQR(checkinStaffCtx(), checkinTestMember)
	if err != nil || res.TTLSeconds != 45 || res.Token == "" {
		t.Errorf("MemberQR: %+v err=%v", res, err)
	}
	if _, _, err := svc.IssueQR(context.Background(), 999); checkinErrCode(err) != rescode.NotFound.Code() {
		t.Errorf("hội viên không tồn tại phải 404: %v", err)
	}

	// TTL cấu hình sai (≤ 0) → dùng mặc định 60s.
	svc.settings = checkinFakeSettings{values: map[string]int{SettingCheckinQRTTLSeconds: 0}}
	if _, exp, _ := svc.IssueQR(context.Background(), checkinTestMember); !exp.Equal(checkinTestNow.Add(60 * time.Second)) {
		t.Errorf("TTL mặc định sai: %v", exp)
	}
}

type checkinSubWarnCase struct {
	typ       string
	daysLeft  int
	remaining *int
}

func (c checkinSubWarnCase) toV1() *v1.CheckinSubscription {
	return &v1.CheckinSubscription{Type: c.typ, DaysLeft: c.daysLeft, RemainingSessions: c.remaining}
}
