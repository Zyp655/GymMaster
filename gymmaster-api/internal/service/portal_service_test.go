package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	checkinv1 "gymmaster-api/api/checkin/v1"
	invoicev1 "gymmaster-api/api/invoice/v1"
	memberv1 "gymmaster-api/api/member/v1"
	v1 "gymmaster-api/api/portal/v1"
	ptv1 "gymmaster-api/api/pt/v1"
	subv1 "gymmaster-api/api/subscription/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

// Hai hội viên: A (member 1, user 101) và B (member 2, user 102). Mọi test kiểm tra A không đọc/sửa được dữ liệu của B.
const (
	ptA, ptB         uint = 1, 2
	ptUserA, ptUserB uint = 101, 102
)

// ---- fakes ----------------------------------------------------------------------

type fakePortalRepo struct {
	members   map[uint]*entity.Member // theo user_id
	branches  []entity.Branch
	plans     map[uint][]repository.PortalSubscriptionPlan
	contracts map[uint]*repository.PortalContract
	trainer   []repository.PortalBusy
	memberBz  []repository.PortalBusy
	updated   map[uint]map[string]any
	// email tài khoản (users) khác đã dùng – giả lập cho kiểm tra trùng email.
	takenUserEmails []string
}

func (f *fakePortalRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	if m, ok := f.members[userID]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakePortalRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	for i := range f.branches {
		if f.branches[i].ID == id {
			return &f.branches[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakePortalRepo) ActiveBranches(context.Context) ([]entity.Branch, error) {
	return f.branches, nil
}
func (f *fakePortalRepo) SubscriptionPlans(_ context.Context, memberID uint) ([]repository.PortalSubscriptionPlan, error) {
	return f.plans[memberID], nil
}
func (f *fakePortalRepo) Freezes(context.Context, uint) ([]repository.PortalFreezeRow, error) {
	return nil, nil
}
func (f *fakePortalRepo) UpdateProfile(_ context.Context, memberID uint, fields map[string]any) error {
	if f.updated == nil {
		f.updated = map[uint]map[string]any{}
	}
	f.updated[memberID] = fields
	return nil
}
func (f *fakePortalRepo) EmailInUse(_ context.Context, email string, memberID uint, _ *uint) (bool, error) {
	for _, m := range f.members {
		if m.ID != memberID && strings.EqualFold(m.Email, email) {
			return true, nil
		}
	}
	for _, e := range f.takenUserEmails {
		if strings.EqualFold(e, email) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakePortalRepo) FindContract(_ context.Context, id uint) (*repository.PortalContract, error) {
	if c, ok := f.contracts[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakePortalRepo) TrainerBusy(context.Context, uint, time.Time, time.Time) ([]repository.PortalBusy, error) {
	return f.trainer, nil
}
func (f *fakePortalRepo) MemberBusy(context.Context, uint, time.Time, time.Time) ([]repository.PortalBusy, error) {
	return f.memberBz, nil
}

type fakePortalMembers struct{ gotID uint }

func (f *fakePortalMembers) Get(_ context.Context, id uint) (*memberv1.Member, error) {
	f.gotID = id
	return &memberv1.Member{ID: id, Note: "ghi chú nội bộ"}, nil
}
func (f *fakePortalMembers) ListBodyMetrics(_ context.Context, memberID uint) (memberv1.ListBodyMetricsRes, error) {
	f.gotID = memberID
	return memberv1.ListBodyMetricsRes{{MemberID: memberID}}, nil
}

type fakePortalCheckins struct{ gotID uint }

func (f *fakePortalCheckins) IssueQR(_ context.Context, memberID uint) (string, time.Time, error) {
	f.gotID = memberID
	return "GM1.x", time.Now().Add(time.Minute), nil
}
func (f *fakePortalCheckins) ListMemberCheckins(_ context.Context, memberID uint, _ dto.PageReq) (*dto.PageRes[*checkinv1.CheckinItem], error) {
	f.gotID = memberID
	return &dto.PageRes[*checkinv1.CheckinItem]{Items: []*checkinv1.CheckinItem{}}, nil
}

type fakePortalSubs struct {
	subs      map[uint][]*subv1.Subscription
	freezeFor uint
}

func (f *fakePortalSubs) ListMemberSubscriptions(_ context.Context, memberID uint) ([]*subv1.Subscription, error) {
	return f.subs[memberID], nil
}
func (f *fakePortalSubs) ListMemberInvoices(context.Context, uint, dto.PageReq) (*invoicev1.ListRes, error) {
	return &invoicev1.ListRes{}, nil
}
func (f *fakePortalSubs) RequestFreezeByMember(_ context.Context, memberID, subscriptionID uint, _ subv1.FreezeInput) (*subv1.Freeze, error) {
	f.freezeFor = memberID
	for _, s := range f.subs[memberID] {
		if s.ID == subscriptionID {
			return &subv1.Freeze{ID: 9, SubscriptionID: subscriptionID, Status: "PENDING"}, nil
		}
	}
	return nil, rescode.New(rescode.NotFound, "Không tìm thấy gói tập của hội viên")
}

type fakePortalInvoices struct {
	invoices     map[uint]*entity.Invoice
	detailCalled bool
}

func (f *fakePortalInvoices) FindInvoice(_ context.Context, id uint) (*entity.Invoice, error) {
	if inv, ok := f.invoices[id]; ok {
		return inv, nil
	}
	return nil, rescode.New(rescode.NotFound, "Không tìm thấy hóa đơn")
}
func (f *fakePortalInvoices) Detail(_ context.Context, id uint) (*invoicev1.InvoiceDetail, error) {
	f.detailCalled = true
	return &invoicev1.InvoiceDetail{}, nil
}

type fakePortalBookings struct {
	bookings  map[uint]*entity.ClassBookingDetail
	cancelled []uint
	byMember  bool
	bookedFor uint
}

func (f *fakePortalBookings) Book(_ context.Context, scheduleID, memberID uint, source string) (*entity.ClassBooking, error) {
	f.bookedFor = memberID
	d := &entity.ClassBookingDetail{ClassBooking: entity.ClassBooking{ID: 77, ScheduleID: scheduleID, MemberID: memberID, Status: "BOOKED", Source: source}}
	d.StartAt = time.Now().Add(48 * time.Hour)
	f.bookings[77] = d
	return &d.ClassBooking, nil
}
func (f *fakePortalBookings) Cancel(_ context.Context, id uint, byMember bool) error {
	f.cancelled = append(f.cancelled, id)
	f.byMember = byMember
	f.bookings[id].Status = entity.BookingStatusCancelled
	return nil
}
func (f *fakePortalBookings) ListMemberBookings(_ context.Context, memberID uint, _, _ time.Time) ([]entity.ClassBookingDetail, error) {
	out := []entity.ClassBookingDetail{}
	for _, b := range f.bookings {
		// Cố tình trả cả lượt của người khác để kiểm tra lớp lọc phòng thủ của portal.
		_ = memberID
		out = append(out, *b)
	}
	return out, nil
}
func (f *fakePortalBookings) GetBooking(_ context.Context, id uint) (*entity.ClassBookingDetail, error) {
	if b, ok := f.bookings[id]; ok {
		return b, nil
	}
	return nil, rescode.New(rescode.NotFound, "Không tìm thấy lượt đặt chỗ")
}

type fakePortalSchedules struct{ rows []entity.ClassScheduleDetail }

func (f *fakePortalSchedules) List(context.Context, repository.ClassScheduleFilter) ([]entity.ClassScheduleDetail, error) {
	return f.rows, nil
}

type fakePortalPT struct {
	bookedFor, cancelFor uint
}

func (f *fakePortalPT) ListMemberContracts(_ context.Context, memberID uint) ([]*ptv1.Contract, error) {
	return []*ptv1.Contract{{ID: 1, MemberID: memberID}, {ID: 2, MemberID: memberID + 100}}, nil
}
func (f *fakePortalPT) ListMemberSessions(_ context.Context, memberID uint, _, _ time.Time) ([]*ptv1.Session, error) {
	return []*ptv1.Session{
		{ID: 1, MemberID: memberID, Status: "SCHEDULED", StartAt: time.Now().Add(48 * time.Hour)},
		{ID: 2, MemberID: memberID, Status: "SCHEDULED", StartAt: time.Now().Add(time.Hour)},
	}, nil
}
func (f *fakePortalPT) BookSessionByMember(_ context.Context, memberID, contractID uint, startAt time.Time) (*ptv1.Session, error) {
	f.bookedFor = memberID
	return &ptv1.Session{ID: 5, ContractID: contractID, MemberID: memberID, StartAt: startAt, Status: "SCHEDULED"}, nil
}
func (f *fakePortalPT) CancelSessionByMember(_ context.Context, memberID, sessionID uint, _ string) (*ptv1.Session, error) {
	f.cancelFor = memberID
	return &ptv1.Session{ID: sessionID, MemberID: memberID, Status: "CANCELLED"}, nil
}

type fakePortalSettings struct{}

func (fakePortalSettings) Int(_ context.Context, _ string, def int) int              { return def }
func (fakePortalSettings) IntSlice(_ context.Context, _ string, def []int) []int     { return def }
func (fakePortalSettings) All(context.Context) ([]entity.Setting, error)             { return nil, nil }
func (fakePortalSettings) Set(context.Context, string, json.RawMessage, *uint) error { return nil }

// ---- dựng service ---------------------------------------------------------------

type portalFixture struct {
	svc      *portalService
	repo     *fakePortalRepo
	members  *fakePortalMembers
	checkins *fakePortalCheckins
	subs     *fakePortalSubs
	invoices *fakePortalInvoices
	bookings *fakePortalBookings
	pt       *fakePortalPT
	now      time.Time
}

func newPortalFixture(t *testing.T) *portalFixture {
	t.Helper()
	loc, _ := time.LoadLocation(consts.TimeZone)
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, loc)
	memberA := &entity.Member{UserID: portalPtr(ptUserA), HomeBranchID: 1, Status: entity.MemberStatusActive}
	memberA.ID = ptA
	memberB := &entity.Member{UserID: portalPtr(ptUserB), HomeBranchID: 2, Status: entity.MemberStatusActive}
	memberB.ID = ptB
	b1 := entity.Branch{Name: "Cầu Giấy", OpenTime: "06:00", CloseTime: "09:00"}
	b1.ID = 1
	b2 := entity.Branch{Name: "Hà Đông", OpenTime: "06:00", CloseTime: "22:00"}
	b2.ID = 2
	b3 := entity.Branch{Name: "Thanh Xuân", OpenTime: "06:00", CloseTime: "22:00"}
	b3.ID = 3

	startB := now.Add(72 * time.Hour)
	f := &portalFixture{
		now: now,
		repo: &fakePortalRepo{
			members:  map[uint]*entity.Member{ptUserA: memberA, ptUserB: memberB},
			branches: []entity.Branch{b1, b2, b3},
			plans: map[uint][]repository.PortalSubscriptionPlan{
				ptA: {{SubscriptionID: 10, BranchID: 1, Status: "ACTIVE", AccessScope: "SINGLE_BRANCH", EndDate: now.AddDate(0, 1, 0)}},
			},
			contracts: map[uint]*repository.PortalContract{
				20: {PTContract: entity.PTContract{ID: 20, MemberID: ptA, TrainerID: 50, BranchID: 1, SessionMinutes: 60, Status: "ACTIVE",
					StartDate: now.AddDate(0, 0, -5), EndDate: now.AddDate(0, 1, 0)}, OpenTime: "06:00", CloseTime: "09:00"},
				21: {PTContract: entity.PTContract{ID: 21, MemberID: ptB, TrainerID: 50, BranchID: 2, SessionMinutes: 60, Status: "ACTIVE",
					StartDate: now.AddDate(0, 0, -5), EndDate: now.AddDate(0, 1, 0)}, OpenTime: "06:00", CloseTime: "22:00"},
			},
		},
		members:  &fakePortalMembers{},
		checkins: &fakePortalCheckins{},
		subs: &fakePortalSubs{subs: map[uint][]*subv1.Subscription{
			ptA: {{ID: 10, MemberID: ptA, Status: "ACTIVE", PlanType: "TIME", StartDate: "2026-10-01", EndDate: "2026-11-14", Note: "nội bộ"}},
			ptB: {{ID: 11, MemberID: ptB, Status: "ACTIVE", PlanType: "TIME", StartDate: "2026-10-01", EndDate: "2026-11-14"}},
		}},
		invoices: &fakePortalInvoices{invoices: map[uint]*entity.Invoice{
			30: {ID: 30, MemberID: portalPtr(ptA)},
			31: {ID: 31, MemberID: portalPtr(ptB)},
			32: {ID: 32, MemberID: nil}, // POS khách vãng lai
		}},
		bookings: &fakePortalBookings{bookings: map[uint]*entity.ClassBookingDetail{
			40: {ClassBooking: entity.ClassBooking{ID: 40, ScheduleID: 1, MemberID: ptA, Status: "BOOKED"}, StartAt: startB, ScheduleStatus: "SCHEDULED"},
			41: {ClassBooking: entity.ClassBooking{ID: 41, ScheduleID: 1, MemberID: ptB, Status: "BOOKED"}, StartAt: startB, ScheduleStatus: "SCHEDULED"},
		}},
		pt: &fakePortalPT{},
	}
	svc := NewPortalService(PortalDeps{
		Repo: f.repo, Members: f.members, Checkins: f.checkins, Subscriptions: f.subs, Invoices: f.invoices,
		Bookings: f.bookings, PT: f.pt, Settings: fakePortalSettings{},
		Schedules: &fakePortalSchedules{rows: []entity.ClassScheduleDetail{
			{ClassSchedule: entity.ClassSchedule{ID: 1, BranchID: 1, StartAt: startB, EndAt: startB.Add(time.Hour), Capacity: 10, BookedCount: 2, Status: "SCHEDULED"}},
			{ClassSchedule: entity.ClassSchedule{ID: 2, BranchID: 1, StartAt: now.Add(-time.Hour), EndAt: now, Capacity: 10, Status: "SCHEDULED"}},
		}},
		UploadDir: t.TempDir(), UploadPublicURL: "http://localhost:8042/uploads/",
	}).(*portalService)
	svc.now = func() time.Time { return now }
	f.svc = svc
	return f
}

func portalPtr[T any](v T) *T { return &v }

func portalCtx(userID uint) context.Context {
	return reqctx.WithAuth(context.Background(), &token.Payload{UserID: userID, Roles: []string{consts.RoleMember}})
}

func portalCode(err error) int { return gerror.Code(err).Code() }

// ---- tests ----------------------------------------------------------------------

func TestPortal_RequiresLinkedMember(t *testing.T) {
	f := newPortalFixture(t)
	_, err := f.svc.Profile(context.Background())
	assert.Equal(t, rescode.Unauthorized.Code(), portalCode(err))

	_, err = f.svc.Profile(portalCtx(999)) // tài khoản MEMBER chưa liên kết hội viên
	assert.Equal(t, rescode.NotFound.Code(), portalCode(err))
}

func TestPortal_ProfileUsesOwnMemberAndHidesNote(t *testing.T) {
	f := newPortalFixture(t)
	p, err := f.svc.Profile(portalCtx(ptUserA))
	require.NoError(t, err)
	assert.Equal(t, ptA, f.members.gotID)
	assert.Empty(t, p.Member.Note)
	require.NotNil(t, p.CurrentSubscription)
	assert.Equal(t, uint(10), p.CurrentSubscription.ID)
	assert.Empty(t, p.CurrentSubscription.Note)
	assert.True(t, p.HasValidSubscription)
}

func TestPortal_QRAndCheckinsAndMetricsUseOwnMember(t *testing.T) {
	f := newPortalFixture(t)
	q, err := f.svc.QR(portalCtx(ptUserB))
	require.NoError(t, err)
	assert.Equal(t, ptB, f.checkins.gotID)
	assert.True(t, q.HasValidSubscription)
	assert.Equal(t, 60, q.TTL)

	_, err = f.svc.Checkins(portalCtx(ptUserA), dto.PageReq{})
	require.NoError(t, err)
	assert.Equal(t, ptA, f.checkins.gotID)

	m, err := f.svc.BodyMetrics(portalCtx(ptUserA))
	require.NoError(t, err)
	assert.Equal(t, ptA, m[0].MemberID)
}

func TestPortal_CancelBookingOnlyOwn(t *testing.T) {
	f := newPortalFixture(t)
	_, err := f.svc.CancelBooking(portalCtx(ptUserA), 41) // lượt của B
	assert.Equal(t, rescode.NotFound.Code(), portalCode(err))
	assert.Empty(t, f.bookings.cancelled, "không được gọi Cancel cho lượt của người khác")

	_, err = f.svc.CancelBooking(portalCtx(ptUserA), 999)
	assert.Equal(t, rescode.NotFound.Code(), portalCode(err))

	res, err := f.svc.CancelBooking(portalCtx(ptUserA), 40)
	require.NoError(t, err)
	assert.Equal(t, []uint{40}, f.bookings.cancelled)
	assert.True(t, f.bookings.byMember, "hội viên hủy phải áp hạn hủy")
	assert.Equal(t, entity.BookingStatusCancelled, res.Status)
}

func TestPortal_BookingsAndClassesShowOnlyOwn(t *testing.T) {
	f := newPortalFixture(t)
	list, err := f.svc.Bookings(portalCtx(ptUserA), "", "")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, uint(40), list[0].ID)
	assert.True(t, list[0].CanCancel)

	res, err := f.svc.Classes(portalCtx(ptUserA), 0, "", "")
	require.NoError(t, err)
	assert.Equal(t, uint(1), res.BranchID, "mặc định chi nhánh chính")
	require.Len(t, res.Items, 1, "bỏ buổi đã bắt đầu")
	require.NotNil(t, res.Items[0].MyBooking)
	assert.Equal(t, uint(40), res.Items[0].MyBooking.ID, "chỉ gắn lượt đặt của chính mình")
	assert.Equal(t, 8, res.Items[0].AvailableSlots)

	_, err = f.svc.Classes(portalCtx(ptUserA), 2, "", "") // gói SINGLE_BRANCH chi nhánh 1
	assert.Equal(t, rescode.PermissionDenied.Code(), portalCode(err))
}

func TestPortal_BranchesFollowSubscriptionScope(t *testing.T) {
	f := newPortalFixture(t)
	list, err := f.svc.Branches(portalCtx(ptUserA))
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.True(t, list[0].IsHome)

	f.repo.plans[ptA] = append(f.repo.plans[ptA], repository.PortalSubscriptionPlan{
		SubscriptionID: 12, BranchID: 2, Status: "ACTIVE", AccessScope: "ALL_BRANCHES", EndDate: f.now.AddDate(0, 1, 0)})
	list, err = f.svc.Branches(portalCtx(ptUserA))
	require.NoError(t, err)
	assert.Len(t, list, 3)
	assert.Equal(t, uint(1), list[0].ID, "chi nhánh chính đứng đầu")
}

func TestPortal_BookUsesLoggedInMember(t *testing.T) {
	f := newPortalFixture(t)
	b, err := f.svc.BookClass(portalCtx(ptUserB), 1)
	require.NoError(t, err)
	assert.Equal(t, ptB, f.bookings.bookedFor)
	assert.Equal(t, entity.BookingSourcePortal, b.Source)

	_, err = f.svc.BookPTSession(portalCtx(ptUserA), 20, "2026-10-16T07:00")
	require.NoError(t, err)
	assert.Equal(t, ptA, f.pt.bookedFor)

	_, err = f.svc.CancelPTSession(portalCtx(ptUserB), 5, "")
	require.NoError(t, err)
	assert.Equal(t, ptB, f.pt.cancelFor)

	_, err = f.svc.RequestFreeze(portalCtx(ptUserA), 11, subv1.FreezeInput{}) // gói của B
	assert.Equal(t, rescode.NotFound.Code(), portalCode(err))
	assert.Equal(t, ptA, f.subs.freezeFor)
}

func TestPortal_PTListsFilterOthersAndCancelDeadline(t *testing.T) {
	f := newPortalFixture(t)
	cs, err := f.svc.PTContracts(portalCtx(ptUserA))
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, ptA, cs[0].MemberID)

	f.svc.now = time.Now
	ss, err := f.svc.PTSessions(portalCtx(ptUserA), "", "")
	require.NoError(t, err)
	require.Len(t, ss.Items, 2)
	assert.True(t, ss.Items[0].CanCancel, "còn 48 giờ > hạn 12 giờ")
	assert.False(t, ss.Items[1].CanCancel, "còn 1 giờ < hạn 12 giờ")
}

func TestPortal_InvoiceOnlyOwn(t *testing.T) {
	f := newPortalFixture(t)
	for _, id := range []uint{31, 32, 999} {
		_, err := f.svc.Invoice(portalCtx(ptUserA), id)
		assert.Equal(t, rescode.NotFound.Code(), portalCode(err), "hóa đơn %d", id)
	}
	assert.False(t, f.invoices.detailCalled)
	_, err := f.svc.Invoice(portalCtx(ptUserA), 30)
	require.NoError(t, err)
	assert.True(t, f.invoices.detailCalled)
}

func TestPortal_TrainerAvailability(t *testing.T) {
	f := newPortalFixture(t)
	_, err := f.svc.TrainerAvailability(portalCtx(ptUserA), 21, "2026-10-15") // hợp đồng của B
	assert.Equal(t, rescode.NotFound.Code(), portalCode(err))

	loc := f.svc.loc
	day := time.Date(2026, 10, 15, 0, 0, 0, 0, loc)
	f.repo.trainer = []repository.PortalBusy{{StartAt: day.Add(7 * time.Hour), EndAt: day.Add(8 * time.Hour)}}
	f.repo.memberBz = []repository.PortalBusy{{StartAt: day.Add(8 * time.Hour), EndAt: day.Add(8*time.Hour + 30*time.Minute)}}
	res, err := f.svc.TrainerAvailability(portalCtx(ptUserA), 20, "2026-10-15")
	require.NoError(t, err)
	// 06:00–09:00, buổi 60 phút, bước 30 phút → 06:00, 06:30, 07:00, 07:30, 08:00.
	got := []string{}
	for _, s := range res.Slots {
		got = append(got, s.StartAt.In(loc).Format("15:04")+"="+s.Reason)
	}
	assert.Equal(t, "06:00=,06:30=TRAINER_BUSY,07:00=TRAINER_BUSY,07:30=TRAINER_BUSY,08:00=MEMBER_BUSY", strings.Join(got, ","))

	res, err = f.svc.TrainerAvailability(portalCtx(ptUserA), 20, "2026-10-14") // hôm nay, now = 09:00
	require.NoError(t, err)
	for _, s := range res.Slots {
		assert.Equal(t, PortalSlotPast, s.Reason)
	}
	res, err = f.svc.TrainerAvailability(portalCtx(ptUserA), 20, "2027-01-01")
	require.NoError(t, err)
	assert.Equal(t, PortalSlotOutOfContract, res.Slots[0].Reason)
}

func TestPortal_UpdateProfileValidatesAndTargetsOwnMember(t *testing.T) {
	f := newPortalFixture(t)
	_, err := f.svc.UpdateProfile(portalCtx(ptUserA), PortalProfileInput{AvatarURL: "http://evil.example/x.png"})
	assert.Equal(t, rescode.BadRequest.Code(), portalCode(err))
	_, err = f.svc.UpdateProfile(portalCtx(ptUserA), PortalProfileInput{EmergencyContactPhone: "123"})
	assert.Equal(t, rescode.BadRequest.Code(), portalCode(err))
	assert.Empty(t, f.repo.updated)

	_, err = f.svc.UpdateProfile(portalCtx(ptUserA), PortalProfileInput{
		Email: " A@Mail.VN ", EmergencyContactPhone: "+84 912 345 678",
		AvatarURL: "http://localhost:8042/uploads/2026/10/a.png",
	})
	require.NoError(t, err)
	require.Contains(t, f.repo.updated, ptA)
	assert.NotContains(t, f.repo.updated, ptB)
	assert.Equal(t, "a@mail.vn", f.repo.updated[ptA]["email"])
	assert.Equal(t, "0912345678", f.repo.updated[ptA]["emergency_contact_phone"])
}

func TestPortal_UpdateProfileRejectsEmailOfAnotherAccount(t *testing.T) {
	f := newPortalFixture(t)
	f.repo.members[ptUserB].Email = "b@mail.vn"
	f.repo.takenUserEmails = []string{"owner@gymmaster.vn"}

	// Email của hội viên khác (không phân biệt hoa thường) → 409, không ghi DB.
	_, err := f.svc.UpdateProfile(portalCtx(ptUserA), PortalProfileInput{Email: " B@Mail.VN "})
	assert.Equal(t, rescode.Conflict.Code(), portalCode(err))
	// Email của tài khoản nhân viên/người dùng khác → 409.
	_, err = f.svc.UpdateProfile(portalCtx(ptUserA), PortalProfileInput{Email: "owner@gymmaster.vn"})
	assert.Equal(t, rescode.Conflict.Code(), portalCode(err))
	assert.Empty(t, f.repo.updated)

	// Giữ nguyên email của chính mình / email mới chưa ai dùng → OK.
	f.repo.members[ptUserA].Email = "a@mail.vn"
	_, err = f.svc.UpdateProfile(portalCtx(ptUserA), PortalProfileInput{Email: "A@mail.vn"})
	require.NoError(t, err)
	_, err = f.svc.UpdateProfile(portalCtx(ptUserA), PortalProfileInput{Email: "new@mail.vn"})
	require.NoError(t, err)
	assert.Equal(t, "new@mail.vn", f.repo.updated[ptA]["email"])
}

func TestPortal_UploadAvatarRejectsNonImage(t *testing.T) {
	f := newPortalFixture(t)
	_, err := f.svc.UploadAvatar(portalCtx(ptUserA), 4, strings.NewReader("text"))
	assert.Equal(t, rescode.BadRequest.Code(), portalCode(err))
	_, err = f.svc.UploadAvatar(portalCtx(ptUserA), 3<<20, strings.NewReader("x"))
	assert.Equal(t, rescode.BadRequest.Code(), portalCode(err))
	png := "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"
	url, err := f.svc.UploadAvatar(portalCtx(ptUserA), int64(len(png)), strings.NewReader(png))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(url, "http://localhost:8042/uploads/2026/10/"))
	assert.True(t, strings.HasSuffix(url, ".png"))
}

func TestPortal_PickCurrentSubscription(t *testing.T) {
	two := 2
	zero := 0
	subs := []*v1.Subscription{
		{Subscription: &subv1.Subscription{ID: 1, Status: "EXPIRED", StartDate: "2026-01-01", EndDate: "2026-02-01"}},
		{Subscription: &subv1.Subscription{ID: 2, Status: "ACTIVE", PlanType: "SESSION", RemainingSessions: &zero, StartDate: "2026-10-01", EndDate: "2026-10-30"}},
		{Subscription: &subv1.Subscription{ID: 3, Status: "ACTIVE", StartDate: "2026-11-01", EndDate: "2026-12-01"}},
	}
	assert.Equal(t, uint(2), portalPickCurrent(subs, "2026-10-14").ID)
	assert.False(t, portalHasValid(subs, "2026-10-14"), "gói theo buổi hết buổi không hợp lệ")
	subs[1].RemainingSessions = &two
	assert.True(t, portalHasValid(subs, "2026-10-14"))
	assert.Equal(t, uint(3), portalPickCurrent(subs[2:], "2026-10-14").ID, "chỉ còn gói sắp bắt đầu")
	assert.Nil(t, portalPickCurrent(subs[:1], "2026-10-14"))
}
