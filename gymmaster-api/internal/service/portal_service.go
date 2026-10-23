package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/google/uuid"
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
	"gymmaster-api/internal/repository"
)

// Cổng hội viên (/v1/member/*). Mọi thao tác xác định hội viên từ tài khoản đăng nhập (members.user_id = reqctx.UserID)
// và chỉ đọc/ghi dữ liệu của chính hội viên đó – không bao giờ nhận member_id từ client.

const (
	portalAvatarMaxBytes     = 2 << 20 // ảnh đại diện ≤ 2 MB
	portalSlotStep           = 30 * time.Minute
	portalClassMaxRangeDays  = 31
	portalBookingMaxRangeDay = 366
	portalDefaultHistoryDays = 30
	portalDefaultCancelPTMin = 720

	// Lý do khung giờ không đặt được (trainer-availability).
	PortalSlotPast          = "PAST"
	PortalSlotTrainerBusy   = "TRAINER_BUSY"
	PortalSlotMemberBusy    = "MEMBER_BUSY"
	PortalSlotOutOfContract = "OUT_OF_CONTRACT"

	auditPortalUpdateProfile = "PORTAL_UPDATE_PROFILE"
)

var (
	errPortalNoMember   = rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
	errPortalBookingNF  = rescode.New(rescode.NotFound, "Không tìm thấy lượt đặt chỗ")
	errPortalInvoiceNF  = rescode.New(rescode.NotFound, "Không tìm thấy hóa đơn")
	errPortalContractNF = rescode.New(rescode.NotFound, "Không tìm thấy hợp đồng PT")
)

// Phụ thuộc thu hẹp (để test bằng fake). Các service dùng chung thỏa sẵn các interface này.
type (
	PortalMemberReader interface {
		Get(ctx context.Context, id uint) (*memberv1.Member, error)
		ListBodyMetrics(ctx context.Context, memberID uint) (memberv1.ListBodyMetricsRes, error)
	}
	PortalCheckinReader interface {
		IssueQR(ctx context.Context, memberID uint) (string, time.Time, error)
		ListMemberCheckins(ctx context.Context, memberID uint, page dto.PageReq) (*dto.PageRes[*checkinv1.CheckinItem], error)
	}
	PortalSubscriptionOps interface {
		ListMemberSubscriptions(ctx context.Context, memberID uint) ([]*subv1.Subscription, error)
		ListMemberInvoices(ctx context.Context, memberID uint, page dto.PageReq) (*invoicev1.ListRes, error)
		RequestFreezeByMember(ctx context.Context, memberID, subscriptionID uint, in subv1.FreezeInput) (*subv1.Freeze, error)
	}
	PortalInvoiceReader interface {
		FindInvoice(ctx context.Context, id uint) (*entity.Invoice, error)
		Detail(ctx context.Context, id uint) (*invoicev1.InvoiceDetail, error)
	}
	PortalBookingOps interface {
		Book(ctx context.Context, scheduleID, memberID uint, source string) (*entity.ClassBooking, error)
		Cancel(ctx context.Context, bookingID uint, byMember bool) error
		ListMemberBookings(ctx context.Context, memberID uint, from, to time.Time) ([]entity.ClassBookingDetail, error)
		GetBooking(ctx context.Context, id uint) (*entity.ClassBookingDetail, error)
	}
	PortalScheduleLister interface {
		List(ctx context.Context, f repository.ClassScheduleFilter) ([]entity.ClassScheduleDetail, error)
	}
	PortalPTOps interface {
		ListMemberContracts(ctx context.Context, memberID uint) ([]*ptv1.Contract, error)
		ListMemberSessions(ctx context.Context, memberID uint, from, to time.Time) ([]*ptv1.Session, error)
		BookSessionByMember(ctx context.Context, memberID, contractID uint, startAt time.Time) (*ptv1.Session, error)
		CancelSessionByMember(ctx context.Context, memberID, sessionID uint, reason string) (*ptv1.Session, error)
	}
)

// PortalDeps gom phụ thuộc của PortalService.
type PortalDeps struct {
	Repo          repository.PortalRepository
	Members       PortalMemberReader
	Checkins      PortalCheckinReader
	Subscriptions PortalSubscriptionOps
	Invoices      PortalInvoiceReader
	Bookings      PortalBookingOps
	Schedules     PortalScheduleLister
	PT            PortalPTOps
	Settings      SettingService
	Audit         AuditService
	// Thư mục / URL công khai lưu ảnh đại diện (cfg.Upload).
	UploadDir       string
	UploadPublicURL string
}

// PortalProfileInput: các trường hội viên được tự sửa.
type PortalProfileInput struct {
	Email                 string
	Address               string
	EmergencyContactName  string
	EmergencyContactPhone string
	AvatarURL             string
}

type (
	PortalService interface {
		Profile(ctx context.Context) (*v1.Profile, error)
		UpdateProfile(ctx context.Context, in PortalProfileInput) (*v1.Profile, error)
		UploadAvatar(ctx context.Context, size int64, file io.Reader) (string, error)
		QR(ctx context.Context) (*v1.QRRes, error)

		Subscriptions(ctx context.Context) ([]*v1.Subscription, error)
		RequestFreeze(ctx context.Context, subscriptionID uint, in subv1.FreezeInput) (*v1.Freeze, error)
		Checkins(ctx context.Context, page dto.PageReq) (*dto.PageRes[*checkinv1.CheckinItem], error)

		Branches(ctx context.Context) ([]*v1.Branch, error)
		Classes(ctx context.Context, branchID uint, from, to string) (*v1.ListClassesRes, error)
		BookClass(ctx context.Context, scheduleID uint) (*v1.Booking, error)
		Bookings(ctx context.Context, from, to string) ([]*v1.Booking, error)
		CancelBooking(ctx context.Context, bookingID uint) (*v1.Booking, error)

		PTContracts(ctx context.Context) ([]*ptv1.Contract, error)
		PTSessions(ctx context.Context, from, to string) (*v1.ListPTSessionsRes, error)
		BookPTSession(ctx context.Context, contractID uint, startAt string) (*v1.PTSession, error)
		CancelPTSession(ctx context.Context, sessionID uint, reason string) (*v1.PTSession, error)
		TrainerAvailability(ctx context.Context, contractID uint, date string) (*v1.TrainerAvailabilityRes, error)

		BodyMetrics(ctx context.Context) (memberv1.ListBodyMetricsRes, error)
		Invoices(ctx context.Context, page dto.PageReq) (*invoicev1.ListRes, error)
		Invoice(ctx context.Context, id uint) (*invoicev1.InvoiceDetail, error)
	}

	portalService struct {
		d   PortalDeps
		now func() time.Time
		loc *time.Location
	}
)

func NewPortalService(d PortalDeps) PortalService {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	d.UploadPublicURL = strings.TrimSuffix(d.UploadPublicURL, "/")
	return &portalService{d: d, now: time.Now, loc: loc}
}

// ---- Hội viên hiện tại ----------------------------------------------------------

// me trả hội viên gắn với tài khoản đang đăng nhập. Đây là chốt chặn duy nhất xác định "dữ liệu của tôi".
func (s *portalService) me(ctx context.Context) (*entity.Member, error) {
	p := reqctx.Auth(ctx)
	if p == nil || p.UserID == 0 {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	m, err := s.d.Repo.FindMemberByUserID(ctx, p.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errPortalNoMember
		}
		return nil, err
	}
	return m, nil
}

func (s *portalService) today() string { return s.now().In(s.loc).Format(consts.DateLayout) }

// dayStart: 00:00 (giờ VN) của ngày "YYYY-MM-DD".
func (s *portalService) dayStart(day string) (time.Time, error) {
	t, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(day), s.loc)
	if err != nil {
		return time.Time{}, rescode.New(rescode.BadRequest, "Ngày phải có dạng YYYY-MM-DD")
	}
	return t, nil
}

// dateRange đổi from/to (YYYY-MM-DD, tính cả hai đầu) thành [from 00:00, to+1 00:00), có mặc định và giới hạn số ngày.
func (s *portalService) dateRange(from, to string, defFrom, defTo, maxDays int) (time.Time, time.Time, error) {
	todayStart, _ := s.dayStart(s.today())
	f := todayStart.AddDate(0, 0, defFrom)
	t := todayStart.AddDate(0, 0, defTo)
	var err error
	if strings.TrimSpace(from) != "" {
		if f, err = s.dayStart(from); err != nil {
			return f, t, err
		}
		if strings.TrimSpace(to) == "" {
			t = f.AddDate(0, 0, defTo-defFrom)
		}
	}
	if strings.TrimSpace(to) != "" {
		if t, err = s.dayStart(to); err != nil {
			return f, t, err
		}
	}
	if t.Before(f) {
		return f, t, rescode.New(rescode.BadRequest, "Từ ngày phải trước hoặc bằng đến ngày")
	}
	end := t.AddDate(0, 0, 1)
	if end.Sub(f) > time.Duration(maxDays)*24*time.Hour+time.Hour {
		return f, t, rescode.New(rescode.BadRequest, fmt.Sprintf("Khoảng thời gian xem tối đa %d ngày", maxDays))
	}
	return f, end, nil
}

// ---- Hồ sơ ----------------------------------------------------------------------

func (s *portalService) Profile(ctx context.Context) (*v1.Profile, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	return s.profile(ctx, m)
}

func (s *portalService) profile(ctx context.Context, m *entity.Member) (*v1.Profile, error) {
	mem, err := s.d.Members.Get(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	mem.Note = "" // ghi chú nội bộ của nhân viên
	res := &v1.Profile{Member: mem}
	if b, err := s.d.Repo.FindBranch(ctx, m.HomeBranchID); err == nil {
		res.Branch = portalBranch(b, m.HomeBranchID)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	subs, err := s.subscriptions(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	today := s.today()
	res.CurrentSubscription = portalPickCurrent(subs, today)
	res.HasValidSubscription = portalHasValid(subs, today)
	return res, nil
}

func (s *portalService) UpdateProfile(ctx context.Context, in PortalProfileInput) (*v1.Profile, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email != "" && !strings.EqualFold(email, m.Email) {
		taken, err := s.d.Repo.EmailInUse(ctx, email, m.ID, m.UserID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, rescode.New(rescode.Conflict, "Email này đã được sử dụng bởi tài khoản khác")
		}
	}
	fields := map[string]any{
		"email":                  email,
		"address":                strings.TrimSpace(in.Address),
		"emergency_contact_name": strings.TrimSpace(in.EmergencyContactName),
		"updated_by":             reqctx.UserID(ctx),
	}
	phone := strings.TrimSpace(in.EmergencyContactPhone)
	if phone != "" {
		p, ok := NormalizeVNPhone(phone)
		if !ok {
			return nil, rescode.New(rescode.BadRequest, "Số điện thoại người liên hệ khẩn cấp không hợp lệ")
		}
		phone = p
	}
	fields["emergency_contact_phone"] = phone
	avatar := strings.TrimSpace(in.AvatarURL)
	if avatar != "" && !strings.HasPrefix(avatar, s.d.UploadPublicURL+"/") {
		return nil, rescode.New(rescode.BadRequest, "Ảnh đại diện không hợp lệ, vui lòng tải ảnh lên lại")
	}
	fields["avatar_url"] = avatar

	before := map[string]any{
		"email": m.Email, "address": m.Address, "emergency_contact_name": m.EmergencyContactName,
		"emergency_contact_phone": m.EmergencyContactPhone, "avatar_url": m.AvatarURL,
	}
	if err := s.d.Repo.UpdateProfile(ctx, m.ID, fields); err != nil {
		return nil, err
	}
	if s.d.Audit != nil {
		after := map[string]any{}
		for k := range before {
			after[k] = fields[k]
		}
		s.d.Audit.Log(ctx, auditPortalUpdateProfile, auditEntityMember, m.ID, before, after)
	}
	return s.profile(ctx, m)
}

// portalImageExt: chỉ nhận ảnh – định dạng lấy theo nội dung thật, không tin tên file của client.
var portalImageExt = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}

func (s *portalService) UploadAvatar(ctx context.Context, size int64, file io.Reader) (string, error) {
	if _, err := s.me(ctx); err != nil {
		return "", err
	}
	tooBig := rescode.New(rescode.BadRequest, fmt.Sprintf("Ảnh vượt quá dung lượng cho phép (%d MB)", portalAvatarMaxBytes>>20))
	if size > portalAvatarMaxBytes {
		return "", tooBig
	}
	data, err := io.ReadAll(io.LimitReader(file, portalAvatarMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > portalAvatarMaxBytes {
		return "", tooBig
	}
	if len(data) == 0 {
		return "", rescode.New(rescode.BadRequest, "Tệp rỗng")
	}
	ext, ok := portalImageExt[http.DetectContentType(data)]
	if !ok {
		return "", rescode.New(rescode.BadRequest, "Chỉ chấp nhận ảnh JPG, PNG hoặc WEBP")
	}
	now := s.now()
	rel := fmt.Sprintf("%04d/%02d/%s.%s", now.Year(), int(now.Month()), uuid.NewString(), ext)
	full := filepath.Join(s.d.UploadDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return s.d.UploadPublicURL + "/" + rel, nil
}

// ---- QR -------------------------------------------------------------------------

func (s *portalService) QR(ctx context.Context) (*v1.QRRes, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	token, exp, err := s.d.Checkins.IssueQR(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	subs, err := s.d.Subscriptions.ListMemberSubscriptions(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	wrapped := make([]*v1.Subscription, 0, len(subs))
	for _, sub := range subs {
		wrapped = append(wrapped, &v1.Subscription{Subscription: sub})
	}
	return &v1.QRRes{
		Token:                token,
		ExpiresAt:            exp,
		TTL:                  s.d.Settings.Int(ctx, SettingCheckinQRTTLSeconds, checkinDefaultQRTTLSeconds),
		ServerTime:           now,
		HasValidSubscription: m.Status == entity.MemberStatusActive && portalHasValid(wrapped, s.today()),
	}, nil
}

// ---- Gói tập --------------------------------------------------------------------

func (s *portalService) Subscriptions(ctx context.Context) ([]*v1.Subscription, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	return s.subscriptions(ctx, m.ID)
}

func (s *portalService) subscriptions(ctx context.Context, memberID uint) ([]*v1.Subscription, error) {
	subs, err := s.d.Subscriptions.ListMemberSubscriptions(ctx, memberID)
	if err != nil {
		return nil, err
	}
	plans, err := s.d.Repo.SubscriptionPlans(ctx, memberID)
	if err != nil {
		return nil, err
	}
	freezes, err := s.d.Repo.Freezes(ctx, memberID)
	if err != nil {
		return nil, err
	}
	planOf := make(map[uint]repository.PortalSubscriptionPlan, len(plans))
	for _, p := range plans {
		planOf[p.SubscriptionID] = p
	}
	freezesOf := map[uint][]*v1.Freeze{}
	for _, f := range freezes {
		freezesOf[f.SubscriptionID] = append(freezesOf[f.SubscriptionID], portalFreeze(f))
	}
	out := make([]*v1.Subscription, 0, len(subs))
	for _, sub := range subs {
		if sub.MemberID != memberID { // phòng thủ: chỉ dữ liệu của chính hội viên
			continue
		}
		sub.Note = ""
		item := &v1.Subscription{Subscription: sub, Freezes: freezesOf[sub.ID]}
		if item.Freezes == nil {
			item.Freezes = []*v1.Freeze{}
		}
		if p, ok := planOf[sub.ID]; ok {
			item.IncludesClasses, item.MaxFreezeDays, item.MaxFreezeTimes = p.IncludesClasses, p.MaxFreezeDays, p.MaxFreezeTimes
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *portalService) RequestFreeze(ctx context.Context, subscriptionID uint, in subv1.FreezeInput) (*v1.Freeze, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	f, err := s.d.Subscriptions.RequestFreezeByMember(ctx, m.ID, subscriptionID, in)
	if err != nil {
		return nil, err
	}
	return &v1.Freeze{
		ID: f.ID, SubscriptionID: f.SubscriptionID, StartDate: f.StartDate, EndDate: f.EndDate, Days: f.Days,
		Reason: f.Reason, Status: f.Status, ReviewNote: f.ReviewNote, ReviewedAt: f.ReviewedAt, CreatedAt: f.CreatedAt,
	}, nil
}

func (s *portalService) Checkins(ctx context.Context, page dto.PageReq) (*dto.PageRes[*checkinv1.CheckinItem], error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	return s.d.Checkins.ListMemberCheckins(ctx, m.ID, page)
}

// ---- Chi nhánh & lớp nhóm -------------------------------------------------------

func (s *portalService) Branches(ctx context.Context) ([]*v1.Branch, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	return s.branches(ctx, m)
}

// branches: chi nhánh hội viên dùng được = chi nhánh chính + chi nhánh của gói ACTIVE/FROZEN còn hạn
// (gói ALL_BRANCHES → mọi chi nhánh đang hoạt động). Chi nhánh chính đứng đầu.
func (s *portalService) branches(ctx context.Context, m *entity.Member) ([]*v1.Branch, error) {
	plans, err := s.d.Repo.SubscriptionPlans(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	all, err := s.d.Repo.ActiveBranches(ctx)
	if err != nil {
		return nil, err
	}
	today := s.today()
	allowAll := false
	allowed := map[uint]bool{m.HomeBranchID: true}
	for _, p := range plans {
		if (p.Status != entity.SubStatusActive && p.Status != entity.SubStatusFrozen) || p.EndDate.Format(consts.DateLayout) < today {
			continue
		}
		if p.AccessScope == "ALL_BRANCHES" {
			allowAll = true
		}
		allowed[p.BranchID] = true
	}
	out := []*v1.Branch{}
	for i := range all {
		b := &all[i]
		if !allowAll && !allowed[b.ID] {
			continue
		}
		out = append(out, portalBranch(b, m.HomeBranchID))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].IsHome && !out[j].IsHome })
	return out, nil
}

func (s *portalService) Classes(ctx context.Context, branchID uint, from, to string) (*v1.ListClassesRes, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	advance := s.d.Settings.Int(ctx, SettingBookingAdvanceDays, bookingDefaultAdvanceDays)
	cancelMin := s.d.Settings.Int(ctx, SettingBookingCancelBeforeMinutes, bookingDefaultCancelMinutes)
	f, end, err := s.dateRange(from, to, 0, advance, portalClassMaxRangeDays)
	if err != nil {
		return nil, err
	}
	res := &v1.ListClassesRes{
		From: f.Format(consts.DateLayout), To: end.AddDate(0, 0, -1).Format(consts.DateLayout),
		AdvanceDays: advance, CancelBeforeMinutes: cancelMin, Items: []*v1.ScheduleItem{},
	}
	branches, err := s.branches(ctx, m)
	if err != nil {
		return nil, err
	}
	if branchID == 0 {
		if len(branches) == 0 {
			return res, nil
		}
		plans, err := s.d.Repo.SubscriptionPlans(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		branchID = portalDefaultClassBranch(plans, branches, m.HomeBranchID, s.today())
	} else if !portalHasBranch(branches, branchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Gói tập của bạn không áp dụng tại chi nhánh này")
	}
	res.BranchID = branchID

	rows, err := s.d.Schedules.List(ctx, repository.ClassScheduleFilter{
		BranchIDs: []uint{branchID}, From: f, To: end, Status: entity.ClassScheduleScheduled,
	})
	if err != nil {
		return nil, err
	}
	bookings, err := s.d.Bookings.ListMemberBookings(ctx, m.ID, f, end)
	if err != nil {
		return nil, err
	}
	mine := map[uint]*entity.ClassBookingDetail{}
	for i := range bookings {
		b := &bookings[i]
		if b.MemberID == m.ID && b.Status != entity.BookingStatusCancelled {
			mine[b.ScheduleID] = b
		}
	}
	now := s.now()
	lastDay := now.In(s.loc).AddDate(0, 0, advance).Format(consts.DateLayout)
	blocked := m.BookingBlockedUntil != nil && m.BookingBlockedUntil.After(now)
	// Gói đang hoạt động của hội viên – để báo NO_SUBSCRIPTION (bỏ qua nếu dịch vụ đặt lớp không hỗ trợ).
	var subs []repository.ClassBookingSubscription
	subsKnown := false
	if sc, ok := s.d.Bookings.(portalClassSubscriptions); ok {
		if subs, err = sc.MemberClassSubscriptions(ctx, m.ID); err != nil {
			return nil, err
		}
		subsKnown = true
	}
	if blocked {
		res.BookingBlockedUntil = m.BookingBlockedUntil
	}
	for i := range rows {
		r := &rows[i]
		if !r.StartAt.After(now) {
			continue
		}
		reason, err := portalNotBookableReason(ctx, r, now, blocked, lastDay, subs, subsKnown, s.loc)
		if err != nil {
			return nil, err
		}
		cs := classScheduleToRes(r)
		// Hội viên thấy HLV THỰC DẠY (HLV dạy thay nếu đã phân công).
		if r.SubstituteTrainerID != nil {
			cs.TrainerID, cs.TrainerName = *r.SubstituteTrainerID, r.SubstituteTrainerName
		}
		item := &v1.ScheduleItem{
			ClassSchedule:     cs,
			CancelDeadline:    r.StartAt.Add(-time.Duration(cancelMin) * time.Minute),
			Bookable:          reason == "",
			NotBookableReason: reason,
		}
		if b, ok := mine[r.ID]; ok {
			item.MyBooking = ClassBookingToRes(b)
		}
		res.Items = append(res.Items, item)
	}
	return res, nil
}

// Lý do không đặt được lớp trên cổng hội viên (not_bookable_reason).
const (
	portalNotBookableStarted  = "STARTED"
	portalNotBookableBlocked  = "BLOCKED"
	portalNotBookableOutRange = "OUT_OF_RANGE"
	portalNotBookableHoliday  = "HOLIDAY"
	portalNotBookableLeave    = "TRAINER_LEAVE"
	portalNotBookableNoSub    = "NO_SUBSCRIPTION"
	portalNotBookableFull     = "FULL" // dự phòng: lớp hết chỗ hiện vẫn đặt được (vào danh sách chờ)
)

// portalClassSubscriptions: dịch vụ đặt lớp cung cấp gói ACTIVE của hội viên (bookingService cài).
type portalClassSubscriptions interface {
	MemberClassSubscriptions(ctx context.Context, memberID uint) ([]repository.ClassBookingSubscription, error)
}

// portalNotBookableReason trả "" nếu hội viên đặt được buổi lớp, ngược lại mã lý do (theo thứ tự kiểm tra của Book).
func portalNotBookableReason(ctx context.Context, r *entity.ClassScheduleDetail, now time.Time, blocked bool, lastDay string,
	subs []repository.ClassBookingSubscription, subsKnown bool, loc *time.Location) (string, error) {
	if !r.StartAt.After(now) {
		return portalNotBookableStarted, nil
	}
	if blocked {
		return portalNotBookableBlocked, nil
	}
	classDay := r.StartAt.In(loc).Format(consts.DateLayout)
	if classDay > lastDay {
		return portalNotBookableOutRange, nil
	}
	if err := bookingCheckScheduleGuard(ctx, &r.ClassSchedule); err != nil {
		switch gerror.Code(err) {
		case rescode.BranchHolidayClosed:
			return portalNotBookableHoliday, nil
		case rescode.TrainerOnLeave:
			return portalNotBookableLeave, nil
		}
		return "", err
	}
	if subsKnown {
		if _, err := bookingPickSubscription(subs, r.BranchID, classDay); err != nil {
			return portalNotBookableNoSub, nil
		}
	}
	return "", nil
}

func (s *portalService) booking(ctx context.Context, d *entity.ClassBookingDetail) *v1.Booking {
	cancelMin := s.d.Settings.Int(ctx, SettingBookingCancelBeforeMinutes, bookingDefaultCancelMinutes)
	deadline := d.StartAt.Add(-time.Duration(cancelMin) * time.Minute)
	active := d.Status == entity.BookingStatusBooked || d.Status == entity.BookingStatusWaitlist
	return &v1.Booking{
		ClassBooking:   ClassBookingToRes(d),
		CancelDeadline: deadline,
		CanCancel:      active && d.ScheduleStatus != entity.ClassScheduleCancelled && s.now().Before(deadline),
	}
}

func (s *portalService) BookClass(ctx context.Context, scheduleID uint) (*v1.Booking, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	b, err := s.d.Bookings.Book(ctx, scheduleID, m.ID, entity.BookingSourcePortal)
	if err != nil {
		return nil, err
	}
	d, err := s.d.Bookings.GetBooking(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	return s.booking(ctx, d), nil
}

func (s *portalService) Bookings(ctx context.Context, from, to string) ([]*v1.Booking, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	f, end, err := s.dateRange(from, to, -portalDefaultHistoryDays, portalDefaultHistoryDays, portalBookingMaxRangeDay)
	if err != nil {
		return nil, err
	}
	rows, err := s.d.Bookings.ListMemberBookings(ctx, m.ID, f, end)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Booking, 0, len(rows))
	for i := range rows {
		if rows[i].MemberID != m.ID {
			continue
		}
		out = append(out, s.booking(ctx, &rows[i]))
	}
	return out, nil
}

// CancelBooking: lượt đặt phải thuộc hội viên đang đăng nhập (khác → 404, không lộ sự tồn tại) rồi mới hủy theo hạn hủy.
func (s *portalService) CancelBooking(ctx context.Context, bookingID uint) (*v1.Booking, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.d.Bookings.GetBooking(ctx, bookingID)
	if err != nil {
		if portalIsNotFound(err) {
			return nil, errPortalBookingNF
		}
		return nil, err
	}
	if d.MemberID != m.ID {
		return nil, errPortalBookingNF
	}
	if err := s.d.Bookings.Cancel(ctx, bookingID, true); err != nil {
		return nil, err
	}
	if d, err = s.d.Bookings.GetBooking(ctx, bookingID); err != nil {
		return nil, err
	}
	return s.booking(ctx, d), nil
}

// ---- PT -------------------------------------------------------------------------

func (s *portalService) PTContracts(ctx context.Context) ([]*ptv1.Contract, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.d.PT.ListMemberContracts(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	out := make([]*ptv1.Contract, 0, len(rows))
	for _, c := range rows {
		if c.MemberID == m.ID {
			c.Note = ""
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *portalService) ptSession(sess *ptv1.Session, cancelMin int) *v1.PTSession {
	deadline := sess.StartAt.Add(-time.Duration(cancelMin) * time.Minute)
	return &v1.PTSession{
		Session:        sess,
		CancelDeadline: deadline,
		CanCancel:      sess.Status == entity.PTSessionScheduled && s.now().Before(deadline),
	}
}

func (s *portalService) ptCancelMinutes(ctx context.Context) int {
	return s.d.Settings.Int(ctx, SettingPTCancelBeforeMinutes, portalDefaultCancelPTMin)
}

func (s *portalService) PTSessions(ctx context.Context, from, to string) (*v1.ListPTSessionsRes, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	f, end, err := s.dateRange(from, to, -portalDefaultHistoryDays, portalDefaultHistoryDays, 62)
	if err != nil {
		return nil, err
	}
	rows, err := s.d.PT.ListMemberSessions(ctx, m.ID, f, end)
	if err != nil {
		return nil, err
	}
	cancelMin := s.ptCancelMinutes(ctx)
	res := &v1.ListPTSessionsRes{CancelBeforeMinutes: cancelMin, Items: make([]*v1.PTSession, 0, len(rows))}
	for _, r := range rows {
		if r.MemberID == m.ID {
			res.Items = append(res.Items, s.ptSession(r, cancelMin))
		}
	}
	return res, nil
}

func (s *portalService) BookPTSession(ctx context.Context, contractID uint, startAt string) (*v1.PTSession, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	start, err := s.parseTime(startAt)
	if err != nil {
		return nil, err
	}
	sess, err := s.d.PT.BookSessionByMember(ctx, m.ID, contractID, start)
	if err != nil {
		return nil, err
	}
	return s.ptSession(sess, s.ptCancelMinutes(ctx)), nil
}

func (s *portalService) CancelPTSession(ctx context.Context, sessionID uint, reason string) (*v1.PTSession, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Hội viên hủy qua cổng hội viên"
	}
	sess, err := s.d.PT.CancelSessionByMember(ctx, m.ID, sessionID, reason)
	if err != nil {
		return nil, err
	}
	return s.ptSession(sess, s.ptCancelMinutes(ctx)), nil
}

// TrainerAvailability: khung giờ 30 phút trong giờ mở cửa chi nhánh của hợp đồng; khung bận khi HLV hoặc chính hội viên
// đã có buổi PT/lớp giao nhau, đã qua, hoặc ngoài thời hạn hợp đồng.
func (s *portalService) TrainerAvailability(ctx context.Context, contractID uint, date string) (*v1.TrainerAvailabilityRes, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.d.Repo.FindContract(ctx, contractID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errPortalContractNF
		}
		return nil, err
	}
	if c.MemberID != m.ID {
		return nil, errPortalContractNF
	}
	if c.Status != entity.PTContractActive {
		return nil, rescode.New(rescode.BadRequest, "Hợp đồng PT không ở trạng thái đang tập, không thể đặt buổi")
	}
	day, err := s.dayStart(date)
	if err != nil {
		return nil, err
	}
	open, okOpen := portalClock(c.OpenTime)
	closeAt, okClose := portalClock(c.CloseTime)
	res := &v1.TrainerAvailabilityRes{
		Date: day.Format(consts.DateLayout), ContractID: c.ID, TrainerID: c.TrainerID, TrainerName: c.TrainerName,
		BranchID: c.BranchID, BranchName: c.BranchName, OpenTime: c.OpenTime, CloseTime: c.CloseTime,
		SessionMinutes: c.SessionMinutes, Slots: []*v1.Slot{},
	}
	if !okOpen || !okClose || c.SessionMinutes <= 0 {
		return res, nil
	}
	dayFrom, dayTo := day.Add(open), day.Add(closeAt)
	// Ngày lễ: đóng cửa → không có khung giờ; giờ mở cửa đặc biệt → chỉ sinh khung trong giờ đặc biệt.
	guard := currentScheduleGuard()
	holiday, err := guard.HolidayOn(ctx, c.BranchID, dayFrom)
	if err != nil {
		return nil, err
	}
	if holiday != nil && holiday.Closed {
		return res, nil
	}
	if holiday != nil && holiday.OpenTime != nil && holiday.CloseTime != nil {
		ho, ok1 := portalClock(*holiday.OpenTime)
		hc, ok2 := portalClock(*holiday.CloseTime)
		if ok1 && ok2 {
			dayFrom, dayTo = day.Add(ho), day.Add(hc)
			res.OpenTime, res.CloseTime = *holiday.OpenTime, *holiday.CloseTime
		}
	}
	trainerBusy, err := s.d.Repo.TrainerBusy(ctx, c.TrainerID, dayFrom, dayTo)
	if err != nil {
		return nil, err
	}
	memberBusy, err := s.d.Repo.MemberBusy(ctx, m.ID, dayFrom, dayTo)
	if err != nil {
		return nil, err
	}
	dayStr := res.Date
	inContract := dayStr >= c.StartDate.Format(consts.DateLayout) && dayStr <= c.EndDate.Format(consts.DateLayout)
	now := s.now()
	length := time.Duration(c.SessionMinutes) * time.Minute
	res.Slots = portalBuildSlots(dayFrom, dayTo, length, now, inContract, trainerBusy, memberBusy)
	// Khung ngoài lịch rảnh HLV đã đăng ký hoặc trùng đơn nghỉ phép đã duyệt → HLV bận.
	check, err := guard.TrainerSlotChecker(ctx, c.TrainerID, c.BranchID, dayFrom)
	if err != nil {
		return nil, err
	}
	for _, sl := range res.Slots {
		if sl.Available && check(sl.StartAt, sl.EndAt) != "" {
			sl.Available, sl.Reason = false, PortalSlotTrainerBusy
		}
	}
	return res, nil
}

// portalBuildSlots sinh khung giờ bước 30 phút trong [from, to), buổi dài length phải kết thúc trước to.
func portalBuildSlots(from, to time.Time, length time.Duration, now time.Time, inContract bool,
	trainerBusy, memberBusy []repository.PortalBusy) []*v1.Slot {
	out := []*v1.Slot{}
	for st := from; !st.Add(length).After(to); st = st.Add(portalSlotStep) {
		en := st.Add(length)
		slot := &v1.Slot{StartAt: st, EndAt: en, Available: true}
		switch {
		case !inContract:
			slot.Reason = PortalSlotOutOfContract
		case !st.After(now):
			slot.Reason = PortalSlotPast
		case portalOverlaps(trainerBusy, st, en):
			slot.Reason = PortalSlotTrainerBusy
		case portalOverlaps(memberBusy, st, en):
			slot.Reason = PortalSlotMemberBusy
		}
		slot.Available = slot.Reason == ""
		out = append(out, slot)
	}
	return out
}

func portalOverlaps(busy []repository.PortalBusy, start, end time.Time) bool {
	for _, b := range busy {
		if b.StartAt.Before(end) && b.EndAt.After(start) {
			return true
		}
	}
	return false
}

// portalClock đọc "HH:MM" thành khoảng thời gian tính từ 00:00.
func portalClock(v string) (time.Duration, bool) {
	t, err := time.Parse("15:04", strings.TrimSpace(v))
	if err != nil {
		return 0, false
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, true
}

func (s *portalService) parseTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, v, s.loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, rescode.New(rescode.BadRequest, "Giờ bắt đầu không hợp lệ")
}

// ---- Chỉ số cơ thể & hóa đơn ----------------------------------------------------

func (s *portalService) BodyMetrics(ctx context.Context) (memberv1.ListBodyMetricsRes, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	return s.d.Members.ListBodyMetrics(ctx, m.ID)
}

func (s *portalService) Invoices(ctx context.Context, page dto.PageReq) (*invoicev1.ListRes, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	return s.d.Subscriptions.ListMemberInvoices(ctx, m.ID, page)
}

// Invoice: hóa đơn phải thuộc hội viên đang đăng nhập (khác / khách vãng lai → 404).
func (s *portalService) Invoice(ctx context.Context, id uint) (*invoicev1.InvoiceDetail, error) {
	m, err := s.me(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := s.d.Invoices.FindInvoice(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || portalIsNotFound(err) {
			return nil, errPortalInvoiceNF
		}
		return nil, err
	}
	if inv.MemberID == nil || *inv.MemberID != m.ID {
		return nil, errPortalInvoiceNF
	}
	return s.d.Invoices.Detail(ctx, id)
}

// ---- Mapping / quy tắc chọn gói -------------------------------------------------

func portalBranch(b *entity.Branch, homeID uint) *v1.Branch {
	return &v1.Branch{
		ID: b.ID, Code: b.Code, Name: b.Name, Address: b.Address, Phone: b.Phone,
		OpenTime: b.OpenTime, CloseTime: b.CloseTime, IsHome: b.ID == homeID,
	}
}

// portalDefaultClassBranch chọn chi nhánh mặc định cho lịch lớp: chi nhánh chính nếu một gói đang hiệu lực dùng được tại đó
// (gói ALL_BRANCHES hoặc gói đăng ký tại chi nhánh chính); ngược lại chi nhánh của gói đang hiệu lực (ACTIVE, đúng hạn)
// đầu tiên – tránh mặc định về chi nhánh chính nơi hội viên không đặt được lớp. Không có gói hợp lệ → branches[0].
func portalDefaultClassBranch(plans []repository.PortalSubscriptionPlan, branches []*v1.Branch, homeID uint, today string) uint {
	var first uint
	for _, p := range plans {
		if p.Status != entity.SubStatusActive || p.StartDate.Format(consts.DateLayout) > today || p.EndDate.Format(consts.DateLayout) < today {
			continue
		}
		if (p.AccessScope == "ALL_BRANCHES" || p.BranchID == homeID) && portalHasBranch(branches, homeID) {
			return homeID
		}
		if first == 0 && portalHasBranch(branches, p.BranchID) {
			first = p.BranchID
		}
	}
	if first > 0 {
		return first
	}
	return branches[0].ID
}

func portalHasBranch(list []*v1.Branch, id uint) bool {
	for _, b := range list {
		if b.ID == id {
			return true
		}
	}
	return false
}

func portalFreeze(f repository.PortalFreezeRow) *v1.Freeze {
	return &v1.Freeze{
		ID: f.ID, SubscriptionID: f.SubscriptionID,
		StartDate: f.StartDate.Format(consts.DateLayout), EndDate: f.EndDate.Format(consts.DateLayout),
		Days: f.Days, Reason: f.Reason, Status: f.Status, ReviewNote: f.ReviewNote, ReviewedAt: f.ReviewedAt, CreatedAt: f.CreatedAt,
	}
}

// portalPickCurrent: gói ACTIVE/FROZEN đang trong hạn có ngày hết hạn xa nhất; không có thì gói ACTIVE sắp bắt đầu sớm nhất.
func portalPickCurrent(subs []*v1.Subscription, today string) *v1.Subscription {
	var cur, next *v1.Subscription
	for _, s := range subs {
		switch s.Status {
		case entity.SubStatusActive, entity.SubStatusFrozen:
		default:
			continue
		}
		if s.StartDate <= today && s.EndDate >= today {
			if cur == nil || s.EndDate > cur.EndDate {
				cur = s
			}
		} else if s.StartDate > today && s.Status == entity.SubStatusActive {
			if next == nil || s.StartDate < next.StartDate {
				next = s
			}
		}
	}
	if cur != nil {
		return cur
	}
	return next
}

// portalHasValid: có gói ACTIVE trong hạn hôm nay (gói theo buổi còn buổi).
func portalHasValid(subs []*v1.Subscription, today string) bool {
	for _, s := range subs {
		if s.Status != entity.SubStatusActive || s.StartDate > today || s.EndDate < today {
			continue
		}
		if s.PlanType == "SESSION" && (s.RemainingSessions == nil || *s.RemainingSessions <= 0) {
			continue
		}
		return true
	}
	return false
}

// portalIsNotFound: lỗi rescode.NotFound (40400) từ service dùng chung.
func portalIsNotFound(err error) bool {
	return gerror.Code(err).Code() == rescode.NotFound.Code()
}
