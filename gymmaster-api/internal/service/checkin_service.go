package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/checkin/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/qrtoken"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Giá trị mặc định khi bảng settings chưa có cấu hình.
const (
	checkinDefaultQRTTLSeconds   = 60
	checkinDefaultDuplicateMins  = 60
	checkinExpiringWarnDays      = 7 // gói còn ≤ số ngày này thì cảnh báo sắp hết hạn
	checkinLowSessionsWarn       = 3 // gói SESSION còn ≤ số buổi này thì cảnh báo
	checkinEventSessionUsed      = "SESSION_USED"
	checkinSubscriptionActive    = "ACTIVE"
	checkinSubscriptionFrozen    = "FROZEN"
	checkinDisplayDateLayout     = "02/01/2006"
	checkinDisplayTimeLayout     = "15:04"
	checkinListDefaultOrderField = "c.checked_in_at"

	// checkinDefaultInstallmentGraceDays: kỳ trả góp chưa trả quá số ngày này sau hạn → chặn check-in (PLAN mục trả góp).
	checkinDefaultInstallmentGraceDays = 7
	// SettingCheckinRequireSignedContract = 1: bắt buộc hội viên ký hợp đồng điện tử trước khi check-in.
	SettingCheckinRequireSignedContract = "checkin.require_signed_contract"
	// SettingCheckinInstallmentGraceDays: số ngày ân hạn kỳ trả góp trước khi chặn check-in.
	SettingCheckinInstallmentGraceDays = "checkin.installment_grace_days"
)

// checkinDenyMessages: thông điệp mặc định (tiếng Việt) cho từng lý do từ chối.
// Kết quả quét có thể trả thông điệp chi tiết hơn (giờ mở cửa, chi nhánh của gói, giờ check-in trước…).
var checkinDenyMessages = map[string]string{
	entity.CheckinDenyMemberInactive:       "Hội viên đang ngừng hoạt động hoặc bị cấm ra vào",
	entity.CheckinDenyNoActiveSubscription: "Hội viên không có gói tập còn hiệu lực",
	entity.CheckinDenyWrongBranch:          "Gói tập không áp dụng tại chi nhánh này",
	entity.CheckinDenyNoSessionsLeft:       "Gói tập theo buổi đã dùng hết số buổi",
	entity.CheckinDenySubscriptionFrozen:   "Gói tập đang bảo lưu",
	entity.CheckinDenyDuplicate:            "Hội viên vừa check-in, không cần quét lại",
	entity.CheckinDenyQRExpired:            "Mã QR đã hết hạn, vui lòng mở lại mã mới trên ứng dụng",
	entity.CheckinDenyQRInvalid:            "Mã QR không hợp lệ",
	entity.CheckinDenyBranchClosed:         "Ngoài giờ mở cửa của chi nhánh",
	entity.CheckinDenyTrialPassUsed:        "Vé tập thử này đã được sử dụng",
	entity.CheckinDenyTrialPassExpired:     "Vé tập thử không có hiệu lực trong ngày hôm nay",
	entity.CheckinDenyDayPassUsed:          "Vé ngày / vé khách này đã được sử dụng",
	entity.CheckinDenyDayPassExpired:       "Vé ngày / vé khách không có hiệu lực trong ngày hôm nay",
	entity.CheckinDenyInstallmentOverdue:   "Hội viên có kỳ trả góp quá hạn thanh toán",
	entity.CheckinDenyContractUnsigned:     "Hội viên chưa ký hợp đồng điện tử",
	entity.CheckinDenyOutsideAccessHours:   "Ngoài khung giờ được phép vào tập của gói Off-Peak",
}

// CheckinDenyMessage trả thông điệp tiếng Việt mặc định cho lý do từ chối (rỗng nếu không có).
func CheckinDenyMessage(reason string) string { return checkinDenyMessages[reason] }

type (
	// CheckinService: QR động, quét/check-in thủ công, checkout, lịch sử, thống kê.
	// IssueQR và ListMemberCheckins dùng chung cho cổng hội viên.
	CheckinService interface {
		// IssueQR sinh mã QR động (TTL = checkin.qr_ttl_seconds) cho hội viên.
		IssueQR(ctx context.Context, memberID uint) (token string, expiresAt time.Time, err error)
		MemberQR(ctx context.Context, memberID uint) (*v1.MemberQRRes, error)
		Scan(ctx context.Context, token string, branchID uint) (*v1.ScanResult, error)
		// ScanDevice: quét từ thiết bị kiểm soát ra vào (method DEVICE, gắn device_id, không gắn nhân viên).
		ScanDevice(ctx context.Context, token string, branchID, deviceID uint) (*v1.ScanResult, error)
		Manual(ctx context.Context, memberID, branchID uint, note string) (*v1.ScanResult, error)
		Checkout(ctx context.Context, id uint) (*v1.CheckinItem, error)
		Present(ctx context.Context, branchID uint) (v1.PresentRes, error)
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		Stats(ctx context.Context, branchID uint, date string) (*v1.StatsRes, error)
		// ListMemberCheckins: lịch sử ra vào của một hội viên (không lọc phạm vi chi nhánh – cổng hội viên).
		ListMemberCheckins(ctx context.Context, memberID uint, page dto.PageReq) (*dto.PageRes[*v1.CheckinItem], error)
	}

	checkinService struct {
		repo     repository.CheckinRepository
		tx       postgres.TxManager
		settings SettingService
		now      func() time.Time
	}
)

func NewCheckinService(repo repository.CheckinRepository, tx postgres.TxManager, settings SettingService) CheckinService {
	return &checkinService{repo: repo, tx: tx, settings: settings, now: time.Now}
}

var checkinVNLoc = memberLoadVNLoc()

// ---- QR động -------------------------------------------------------------------

func (s *checkinService) qrTTL(ctx context.Context) int {
	ttl := s.settings.Int(ctx, SettingCheckinQRTTLSeconds, checkinDefaultQRTTLSeconds)
	if ttl <= 0 {
		ttl = checkinDefaultQRTTLSeconds
	}
	return ttl
}

func (s *checkinService) IssueQR(ctx context.Context, memberID uint) (string, time.Time, error) {
	m, err := s.repo.FindMember(ctx, memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
		}
		return "", time.Time{}, err
	}
	if m.QRSecret == "" {
		return "", time.Time{}, rescode.New(rescode.Internal, "Hội viên chưa có bí mật QR")
	}
	token, exp := qrtoken.Generate(m.ID, m.QRSecret, s.now(), time.Duration(s.qrTTL(ctx))*time.Second)
	return token, exp, nil
}

func (s *checkinService) MemberQR(ctx context.Context, memberID uint) (*v1.MemberQRRes, error) {
	token, exp, err := s.IssueQR(ctx, memberID)
	if err != nil {
		return nil, err
	}
	return &v1.MemberQRRes{MemberID: memberID, Token: token, ExpiresAt: exp, TTLSeconds: s.qrTTL(ctx)}, nil
}

// ---- Quyết định cho vào --------------------------------------------------------

// checkinPick là kết quả chọn gói: chosen != nil khi có gói hợp lệ; ngược lại reason/related
// cho biết lý do từ chối và gói liên quan (để hiển thị).
type checkinPick struct {
	chosen  *entity.CheckinSubscriptionView
	reason  string
	related *entity.CheckinSubscriptionView
	message string
}

func checkinDate(t time.Time) string { return t.Format(consts.DateLayout) }

func checkinInDate(sub *entity.CheckinSubscriptionView, today string) bool {
	return checkinDate(sub.StartDate) <= today && today <= checkinDate(sub.EndDate)
}

func checkinBranchOK(sub *entity.CheckinSubscriptionView, branchID uint) bool {
	return sub.AccessScope == entity.AccessAllBranches || sub.BranchID == branchID
}

func checkinHasSessions(sub *entity.CheckinSubscriptionView) bool {
	return sub.PlanType != entity.PlanTypeSession || (sub.RemainingSessions != nil && *sub.RemainingSessions > 0)
}

// checkinPickSubscription chọn gói dùng để vào tại branchID trong ngày today (YYYY-MM-DD, giờ VN) lúc now (giờ VN).
//
// Gói hợp lệ: ACTIVE, start_date ≤ today ≤ end_date, đúng phạm vi chi nhánh, gói SESSION còn buổi và – với gói Off-Peak
// (access_hours) – now nằm trong khung giờ được phép (now.IsZero() = bỏ qua kiểm tra khung giờ). Mọi gói được xét: gói
// Off-Peak ngoài giờ KHÔNG che mất gói thường còn hiệu lực.
// Ưu tiên: gói TIME trước gói SESSION, rồi gói không giới hạn khung giờ, rồi gói hết hạn sớm nhất, rồi id nhỏ nhất.
// Không có gói hợp lệ: gói chỉ vướng khung giờ Off-Peak → OUTSIDE_ACCESS_HOURS; FROZEN → SUBSCRIPTION_FROZEN; gói đúng
// hạn/đúng chi nhánh hết buổi → NO_SESSIONS_LEFT; gói đúng hạn nhưng khác chi nhánh → WRONG_BRANCH; còn lại
// NO_ACTIVE_SUBSCRIPTION.
func checkinPickSubscription(subs []entity.CheckinSubscriptionView, branchID uint, today string, now time.Time) checkinPick {
	var valid []*entity.CheckinSubscriptionView
	var offPeak *entity.CheckinSubscriptionView
	var offPeakViolation *CheckinRuleViolation
	var frozen, exhausted, wrongBranch, future *entity.CheckinSubscriptionView
	for i := range subs {
		sub := &subs[i]
		switch {
		case sub.Status == checkinSubscriptionFrozen:
			if frozen == nil {
				frozen = sub
			}
		case sub.Status != checkinSubscriptionActive:
		case !checkinInDate(sub, today):
			if checkinDate(sub.StartDate) > today && checkinBranchOK(sub, branchID) &&
				(future == nil || sub.StartDate.Before(future.StartDate)) {
				future = sub
			}
		case !checkinBranchOK(sub, branchID):
			if wrongBranch == nil {
				wrongBranch = sub
			}
		case !checkinHasSessions(sub):
			if exhausted == nil {
				exhausted = sub
			}
		default:
			if !now.IsZero() {
				if v := EvaluateCheckinRules(context.Background(), &CheckinRuleInput{Now: now, AccessHoursJSON: sub.AccessHours},
					OffPeakHoursRule{}); v != nil {
					if offPeak == nil {
						offPeak, offPeakViolation = sub, v
					}
					continue
				}
			}
			valid = append(valid, sub)
		}
	}
	if len(valid) > 0 {
		slices.SortStableFunc(valid, func(a, b *entity.CheckinSubscriptionView) int {
			ta, tb := a.PlanType == entity.PlanTypeTime, b.PlanType == entity.PlanTypeTime
			if ta != tb {
				if ta {
					return -1
				}
				return 1
			}
			ua, ub := checkinUnrestricted(a), checkinUnrestricted(b)
			if ua != ub {
				if ua {
					return -1
				}
				return 1
			}
			if c := a.EndDate.Compare(b.EndDate); c != 0 {
				return c
			}
			return int(a.ID) - int(b.ID)
		})
		return checkinPick{chosen: valid[0]}
	}
	switch {
	case offPeak != nil:
		return checkinPick{reason: offPeakViolation.DenyReason, related: offPeak, message: offPeakViolation.Message}
	case frozen != nil:
		return checkinPick{reason: entity.CheckinDenySubscriptionFrozen, related: frozen,
			message: fmt.Sprintf("Gói tập \"%s\" đang bảo lưu, chưa thể vào tập", frozen.PlanName)}
	case exhausted != nil:
		return checkinPick{reason: entity.CheckinDenyNoSessionsLeft, related: exhausted,
			message: fmt.Sprintf("Gói \"%s\" đã dùng hết số buổi, vui lòng gia hạn", exhausted.PlanName)}
	case wrongBranch != nil:
		return checkinPick{reason: entity.CheckinDenyWrongBranch, related: wrongBranch,
			message: fmt.Sprintf("Gói tập chỉ dùng tại chi nhánh %s", wrongBranch.BranchName)}
	case future != nil:
		return checkinPick{reason: entity.CheckinDenyNoActiveSubscription, related: future,
			message: fmt.Sprintf("Gói tập \"%s\" bắt đầu từ ngày %s", future.PlanName, future.StartDate.Format(checkinDisplayDateLayout))}
	default:
		return checkinPick{reason: entity.CheckinDenyNoActiveSubscription,
			message: "Hội viên không có gói tập còn hiệu lực, vui lòng mua hoặc gia hạn gói"}
	}
}

// checkinUnrestricted: gói không giới hạn khung giờ vào tập (access_hours rỗng / []).
func checkinUnrestricted(sub *entity.CheckinSubscriptionView) bool {
	ah := strings.TrimSpace(sub.AccessHours)
	return ah == "" || ah == "[]" || ah == "null"
}

// checkinBranchOpen: giờ hiện tại (VN, HH:MM) nằm trong [open_time, close_time).
func checkinBranchOpen(b *entity.Branch, now time.Time) bool {
	hm := now.In(checkinVNLoc).Format(checkinDisplayTimeLayout)
	return hm >= b.OpenTime && hm < b.CloseTime
}

// checkinDaysLeft: số ngày từ today đến end (cả hai dạng YYYY-MM-DD).
func checkinDaysLeft(end time.Time, today string) int {
	t, _ := time.Parse(consts.DateLayout, today)
	e, _ := time.Parse(consts.DateLayout, checkinDate(end))
	return int(e.Sub(t).Hours() / 24)
}

func checkinStartOfDay(t time.Time) time.Time {
	t = t.In(checkinVNLoc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, checkinVNLoc)
}

// ---- Quét / thủ công ------------------------------------------------------------

func (s *checkinService) checkBranch(ctx context.Context, branchID uint) (*entity.Branch, error) {
	if !reqctx.CanAccessBranch(ctx, branchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được check-in tại chi nhánh này")
	}
	b, err := s.repo.FindBranch(ctx, branchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy chi nhánh")
		}
		return nil, err
	}
	if b.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đã ngừng hoạt động")
	}
	return b, nil
}

func checkinDeniedNoMember(reason string) *v1.ScanResult {
	return &v1.ScanResult{Result: entity.CheckinDenied, DenyReason: reason, DenyMessage: checkinDenyMessages[reason], Warnings: []string{}}
}

func (s *checkinService) Scan(ctx context.Context, token string, branchID uint) (*v1.ScanResult, error) {
	branch, err := s.checkBranch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	return s.scan(ctx, token, branch, checkinAttempt{method: entity.CheckinMethodQR, staffID: reqctx.UserID(ctx)})
}

// ScanDevice: thiết bị kiểm soát ra vào (cổng/Kiosk) đã xác thực bằng device key quét mã tại chi nhánh của thiết bị.
// Lượt check-in được ghi với method = DEVICE, device_id và không gắn nhân viên.
func (s *checkinService) ScanDevice(ctx context.Context, token string, branchID, deviceID uint) (*v1.ScanResult, error) {
	b, err := s.repo.FindBranch(ctx, branchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy chi nhánh")
		}
		return nil, err
	}
	if b.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đã ngừng hoạt động")
	}
	dev := deviceID
	return s.scan(ctx, token, b, checkinAttempt{method: entity.CheckinMethodDevice, deviceID: &dev})
}

func (s *checkinService) scan(ctx context.Context, token string, branch *entity.Branch, a checkinAttempt) (*v1.ScanResult, error) {
	trimmed := strings.TrimSpace(token)
	a.branch = branch
	if strings.HasPrefix(trimmed, "TP-") {
		return s.scanTrialPass(ctx, trimmed, a)
	}
	if strings.HasPrefix(trimmed, "DP-") {
		return s.scanDayPass(ctx, trimmed, a)
	}
	memberID, err := qrtoken.ParseMemberID(token)
	if err != nil {
		// QR hỏng hoàn toàn: không xác định được hội viên → không ghi DB.
		return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
	}
	a.memberID, a.token = memberID, token
	return s.process(ctx, a)
}

// checkinPass là dữ liệu chung của vé ngày / vé tập thử để quyết định cho vào.
type checkinPass struct {
	kind        string // DAY | TRIAL
	id          uint
	branchID    uint
	branchName  string
	validDate   time.Time
	usedAt      *time.Time
	qrToken     string
	member      *v1.CheckinMember
	usedReason  string
	expReason   string
	wrongBranch string
	expMessage  string
	warning     string
}

// processPass quyết định cho vào bằng vé trong MỘT transaction: khóa dòng vé → kiểm tra nghỉ lễ / giờ mở cửa /
// chi nhánh / đã dùng / ngày hiệu lực → đánh dấu đã dùng có điều kiện (used_at IS NULL) → ghi checkins.
// Hai lần quét đồng thời cùng một vé: lần sau chờ khóa, đọc lại used_at và bị từ chối (không thể dùng lại vé).
func (s *checkinService) processPass(ctx context.Context, a checkinAttempt, load func(ctx context.Context) (*checkinPass, error),
	markUsed func(ctx context.Context, id uint, at time.Time) (bool, error)) (*v1.ScanResult, error) {
	now := s.now()
	today := checkinDate(now.In(checkinVNLoc))
	var res *v1.ScanResult
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		p, err := load(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				res = checkinDeniedNoMember(entity.CheckinDenyQRInvalid)
				return nil
			}
			return err
		}
		reason, message := s.branchGateReason(ctx, a.branch, now, today)
		switch {
		case reason != "":
		case p.branchID != a.branch.ID:
			reason, message = entity.CheckinDenyWrongBranch, fmt.Sprintf(p.wrongBranch, p.branchName)
		case p.usedAt != nil:
			reason = p.usedReason
		case checkinDate(p.validDate) != today:
			reason, message = p.expReason, fmt.Sprintf(p.expMessage, p.validDate.Format(checkinDisplayDateLayout))
		default:
			ok, err := markUsed(ctx, p.id, now)
			if err != nil {
				return err
			}
			if !ok {
				reason = p.usedReason
			}
		}
		rec := &repository.CheckinPassRecord{
			BranchID: a.branch.ID, DeviceID: a.deviceID, CheckedInAt: now, Method: a.method,
			Result: entity.CheckinAllowed, StaffID: a.staffID, Note: p.qrToken,
		}
		if p.kind == "DAY" {
			rec.DayPassID = &p.id
		} else {
			rec.TrialPassID = &p.id
		}
		if reason != "" {
			rec.Result, rec.DenyReason = entity.CheckinDenied, reason
			if message == "" {
				message = checkinDenyMessages[reason]
			}
			rec.DenyMessage = message
		}
		id, err := s.passRepo().CreatePassCheckin(ctx, rec)
		if err != nil {
			return err
		}
		res = &v1.ScanResult{
			Result:      rec.Result,
			DenyReason:  reason,
			DenyMessage: message,
			Member:      p.member,
			Checkin:     &v1.CheckinRef{ID: id, CheckedInAt: now, BranchID: a.branch.ID, BranchName: a.branch.Name},
			Warnings:    []string{},
		}
		if reason == "" {
			res.Warnings = []string{p.warning}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// passRepo: repo thật luôn cài CheckinPassRepository; repo giả trong test có thể không → bỏ qua ghi lịch sử.
func (s *checkinService) passRepo() repository.CheckinPassRepository {
	if r, ok := s.repo.(repository.CheckinPassRepository); ok {
		return r
	}
	return checkinNoopPassRepo{}
}

type checkinNoopPassRepo struct{}

func (checkinNoopPassRepo) CreatePassCheckin(context.Context, *repository.CheckinPassRecord) (uint, error) {
	return 0, nil
}

func (s *checkinService) scanTrialPass(ctx context.Context, token string, a checkinAttempt) (*v1.ScanResult, error) {
	tpRepo, ok := s.repo.(repository.CheckinTrialPassRepository)
	if !ok {
		return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
	}
	return s.processPass(ctx, a, func(ctx context.Context) (*checkinPass, error) {
		tp, err := tpRepo.FindTrialPassByToken(ctx, token)
		if err != nil {
			return nil, err
		}
		return &checkinPass{
			kind: "TRIAL", id: tp.ID, branchID: tp.BranchID, branchName: tp.BranchName, validDate: tp.ValidDate,
			usedAt: tp.UsedAt, qrToken: tp.QRToken,
			member: &v1.CheckinMember{
				ID: tp.LeadID, Code: fmt.Sprintf("TRIAL-%d", tp.ID), FullName: tp.FullName + " (Tập thử)",
				Phone: tp.Phone, Status: "TRIAL",
			},
			usedReason: entity.CheckinDenyTrialPassUsed, expReason: entity.CheckinDenyTrialPassExpired,
			wrongBranch: "Vé tập thử chỉ áp dụng tại chi nhánh %s", expMessage: "Vé tập thử chỉ có hiệu lực ngày %s",
			warning: "Khách tập thử 1 ngày theo vé " + tp.QRToken,
		}, nil
	}, tpRepo.MarkTrialPassUsed)
}

func (s *checkinService) scanDayPass(ctx context.Context, token string, a checkinAttempt) (*v1.ScanResult, error) {
	dpRepo, ok := s.repo.(repository.CheckinDayPassRepository)
	if !ok {
		return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
	}
	return s.processPass(ctx, a, func(ctx context.Context) (*checkinPass, error) {
		dp, err := dpRepo.FindDayPassByToken(ctx, token)
		if err != nil {
			return nil, err
		}
		label := "Vé ngày"
		if dp.Type == entity.DayPassTypeGuest {
			label = "Khách mời"
		}
		return &checkinPass{
			kind: "DAY", id: dp.ID, branchID: dp.BranchID, branchName: dp.BranchName, validDate: dp.ValidDate,
			usedAt: dp.UsedAt, qrToken: dp.QRToken,
			member: &v1.CheckinMember{
				ID: dp.ID, Code: fmt.Sprintf("DP-%d", dp.ID), FullName: fmt.Sprintf("%s (%s)", dp.BuyerName, label),
				Phone: dp.Phone, Status: dp.Type,
			},
			usedReason: entity.CheckinDenyDayPassUsed, expReason: entity.CheckinDenyDayPassExpired,
			wrongBranch: "Vé chỉ áp dụng tại chi nhánh %s", expMessage: "Vé chỉ có hiệu lực ngày %s",
			warning: fmt.Sprintf("%s theo mã %s", label, dp.QRToken),
		}, nil
	}, dpRepo.MarkDayPassUsed)
}

// branchGateReason kiểm tra chi nhánh có mở cửa cho check-in lúc now không:
// lịch nghỉ lễ đóng cửa (BranchHolidayClosedRule) và giờ mở cửa (giờ đặc biệt của ngày lễ nếu có).
func (s *checkinService) branchGateReason(ctx context.Context, branch *entity.Branch, now time.Time, today string) (string, string) {
	open, closeT := branch.OpenTime, branch.CloseTime
	holidayName := ""
	if rr, ok := s.repo.(repository.CheckinRulesRepository); ok {
		if h, err := rr.FindBranchHoliday(ctx, branch.ID, today); err == nil && h != nil {
			if v := EvaluateCheckinRules(ctx, &CheckinRuleInput{
				Now: now, BranchID: branch.ID, BranchHolidayClosed: h.Closed, BranchHolidayName: h.Name,
			}, BranchHolidayClosedRule{}); v != nil {
				return v.DenyReason, v.Message + ", không thể check-in"
			}
			if h.OpenTime != nil && h.CloseTime != nil && *h.OpenTime != "" && *h.CloseTime != "" {
				open, closeT, holidayName = *h.OpenTime, *h.CloseTime, h.Name
			}
		}
	}
	hm := now.In(checkinVNLoc).Format(checkinDisplayTimeLayout)
	if hm < open || hm >= closeT {
		if holidayName != "" {
			return entity.CheckinDenyBranchClosed, fmt.Sprintf("Ngoài giờ mở cửa ngày lễ %s (%s - %s)", holidayName, open, closeT)
		}
		return entity.CheckinDenyBranchClosed, fmt.Sprintf("Ngoài giờ mở cửa của chi nhánh (%s - %s)", open, closeT)
	}
	return "", ""
}

// memberRuleViolation chạy các CheckinRule cấp hội viên: hợp đồng điện tử bắt buộc, kỳ trả góp quá hạn.
func (s *checkinService) memberRuleViolation(ctx context.Context, m *entity.Member, now time.Time) (*CheckinRuleViolation, error) {
	rr, ok := s.repo.(repository.CheckinRulesRepository)
	if !ok {
		return nil, nil
	}
	in := &CheckinRuleInput{Now: now, Member: m}
	if s.settings.Int(ctx, SettingCheckinRequireSignedContract, 0) == 1 {
		in.RequireSignedContract = true
		signed, err := rr.HasSignedContract(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		in.HasSignedContract = signed
	}
	grace := s.settings.Int(ctx, SettingCheckinInstallmentGraceDays, checkinDefaultInstallmentGraceDays)
	if grace < 0 {
		grace = 0
	}
	overdueBefore := checkinDate(now.In(checkinVNLoc).AddDate(0, 0, -grace))
	overdue, err := rr.HasOverdueInstallment(ctx, m.ID, overdueBefore)
	if err != nil {
		return nil, err
	}
	in.HasOverdueInstallment = overdue
	return EvaluateCheckinRules(ctx, in, ContractSignedRule{}, InstallmentNotOverdueRule{}), nil
}

func (s *checkinService) Manual(ctx context.Context, memberID, branchID uint, note string) (*v1.ScanResult, error) {
	branch, err := s.checkBranch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	return s.process(ctx, checkinAttempt{
		method: entity.CheckinMethodManual, branch: branch, memberID: memberID, note: strings.TrimSpace(note),
		staffID: reqctx.UserID(ctx),
	})
}

type checkinAttempt struct {
	method   string
	branch   *entity.Branch
	memberID uint
	token    string
	note     string
	staffID  *uint
	deviceID *uint
}

// process chạy toàn bộ quy trình quyết định trong MỘT transaction:
// khóa hội viên → kiểm tra QR → giờ mở cửa → trạng thái hội viên → khóa & chọn gói →
// chống trùng → ghi checkins (+ trừ buổi, ghi subscription_events nếu ALLOWED gói SESSION).
func (s *checkinService) process(ctx context.Context, a checkinAttempt) (*v1.ScanResult, error) {
	now := s.now()
	today := checkinDate(now.In(checkinVNLoc))
	var res *v1.ScanResult

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		m, err := s.repo.LockMember(ctx, a.memberID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if a.token != "" {
					res = checkinDeniedNoMember(entity.CheckinDenyQRInvalid)
					return nil
				}
				return rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
			}
			return err
		}
		res = &v1.ScanResult{
			Member:   &v1.CheckinMember{ID: m.ID, Code: m.MemberCode, FullName: m.FullName, AvatarURL: m.AvatarURL, Phone: m.Phone, Status: m.Status},
			Warnings: []string{},
		}

		var (
			reason, message string
			pick            checkinPick
			prev            *repository.CheckinRow
		)
		if a.token != "" {
			switch err := qrtoken.Verify(a.token, m.QRSecret, now); {
			case errors.Is(err, qrtoken.ErrExpired):
				reason = entity.CheckinDenyQRExpired
			case err != nil:
				reason = entity.CheckinDenyQRInvalid
			}
		}
		if reason == "" {
			reason, message = s.branchGateReason(ctx, a.branch, now, today)
		}
		if reason == "" && m.Status != entity.MemberStatusActive {
			reason = entity.CheckinDenyMemberInactive
			if m.Status == entity.MemberStatusBanned {
				message = "Hội viên đang bị cấm ra vào"
			} else {
				message = "Hội viên đã ngừng hoạt động"
			}
		}
		if reason == "" {
			v, err := s.memberRuleViolation(ctx, m, now)
			if err != nil {
				return err
			}
			if v != nil {
				reason, message = v.DenyReason, v.Message
			}
		}
		if reason == "" {
			subs, err := s.repo.LockSubscriptions(ctx, m.ID)
			if err != nil {
				return err
			}
			pick = checkinPickSubscription(subs, a.branch.ID, today, now.In(checkinVNLoc))

			window := s.settings.Int(ctx, SettingCheckinDuplicateWindowMinutes, checkinDefaultDuplicateMins)
			if window > 0 {
				prev, err = s.repo.LastAllowedSince(ctx, m.ID, now.Add(-time.Duration(window)*time.Minute))
				if err != nil {
					return err
				}
			}
			switch {
			case prev != nil:
				reason = entity.CheckinDenyDuplicate
				message = fmt.Sprintf("Hội viên đã check-in lúc %s tại %s (trong vòng %d phút), không cần quét lại",
					prev.CheckedInAt.In(checkinVNLoc).Format(checkinDisplayTimeLayout), prev.BranchName, window)
			case pick.chosen == nil:
				// Gồm cả trường hợp chỉ có gói Off-Peak ngoài khung giờ (OUTSIDE_ACCESS_HOURS) – đã xét mọi gói.
				reason, message = pick.reason, pick.message
			}
		}

		rec := &entity.Checkin{
			MemberID: m.ID, BranchID: a.branch.ID, CheckedInAt: now, Method: a.method,
			StaffID: a.staffID, DeviceID: a.deviceID, Note: a.note,
		}
		if reason == "" {
			rec.Result = entity.CheckinAllowed
			rec.SubscriptionID = &pick.chosen.ID
		} else {
			rec.Result, rec.DenyReason = entity.CheckinDenied, reason
		}
		if err := s.repo.Create(ctx, rec); err != nil {
			return err
		}
		// Lưu thông điệp chi tiết (nghỉ lễ, giờ đặc biệt, giờ check-in trước…) để lịch sử hiển thị đúng lý do.
		if reason != "" && message != "" && message != checkinDenyMessages[reason] {
			if w, ok := s.repo.(repository.CheckinDenyMessageWriter); ok {
				if err := w.SetDenyMessage(ctx, rec.ID, message); err != nil {
					return err
				}
			}
		}

		sub := pick.chosen
		if sub == nil {
			sub = pick.related
		}
		if rec.Result == entity.CheckinAllowed && sub.PlanType == entity.PlanTypeSession {
			remaining, err := s.repo.UseSession(ctx, sub.ID)
			if err != nil {
				return err
			}
			sub.RemainingSessions = &remaining
			if err := s.repo.CreateSubscriptionEvent(ctx, sub.ID, checkinEventSessionUsed,
				map[string]any{"checkin_id": rec.ID, "remaining": remaining, "branch_id": a.branch.ID}, rec.StaffID); err != nil {
				return err
			}
		}

		res.Result, res.DenyReason = rec.Result, reason
		if reason != "" {
			res.DenyMessage = message
			if res.DenyMessage == "" {
				res.DenyMessage = checkinDenyMessages[reason]
			}
		}
		res.Checkin = &v1.CheckinRef{ID: rec.ID, CheckedInAt: rec.CheckedInAt, BranchID: a.branch.ID, BranchName: a.branch.Name}
		if prev != nil {
			res.PreviousCheckin = &v1.CheckinRef{ID: prev.ID, CheckedInAt: prev.CheckedInAt, BranchID: prev.BranchID, BranchName: prev.BranchName}
		}
		if sub != nil {
			res.Subscription = &v1.CheckinSubscription{
				ID: sub.ID, PlanName: sub.PlanName, Type: sub.PlanType, Status: sub.Status,
				EndDate: checkinDate(sub.EndDate), DaysLeft: checkinDaysLeft(sub.EndDate, today),
				RemainingSessions: sub.RemainingSessions,
			}
			if rec.Result == entity.CheckinAllowed {
				res.Warnings = checkinWarnings(res.Subscription)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Tích điểm check-in (tối đa 1 lần/ngày) sau khi transaction đã commit – lỗi tích điểm không ảnh hưởng lượt vào.
	if res != nil && res.Result == entity.CheckinAllowed && res.Checkin != nil && res.Member != nil {
		awardCheckinPoints(ctx, res.Member.ID, res.Checkin.CheckedInAt)
	}
	return res, nil
}

// checkinWarnings: nhắc lễ tân khi gói sắp hết hạn hoặc sắp hết buổi.
func checkinWarnings(sub *v1.CheckinSubscription) []string {
	w := []string{}
	switch {
	case sub.DaysLeft == 0:
		w = append(w, "Gói hết hạn hôm nay")
	case sub.DaysLeft <= checkinExpiringWarnDays:
		w = append(w, fmt.Sprintf("Gói sắp hết hạn trong %d ngày", sub.DaysLeft))
	}
	if sub.Type == entity.PlanTypeSession && sub.RemainingSessions != nil {
		switch r := *sub.RemainingSessions; {
		case r == 0:
			w = append(w, "Đây là buổi cuối cùng của gói")
		case r <= checkinLowSessionsWarn:
			w = append(w, fmt.Sprintf("Gói chỉ còn %d buổi", r))
		}
	}
	return w
}

// ---- Checkout / đang tập ------------------------------------------------------

func (s *checkinService) Checkout(ctx context.Context, id uint) (*v1.CheckinItem, error) {
	row, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy lượt check-in")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Lượt check-in thuộc chi nhánh bạn không quản lý")
	}
	if row.Result != entity.CheckinAllowed {
		return nil, rescode.New(rescode.BadRequest, "Chỉ checkout được lượt check-in thành công")
	}
	if row.CheckedOutAt != nil {
		return nil, rescode.New(rescode.Conflict, "Hội viên đã checkout lúc "+row.CheckedOutAt.In(checkinVNLoc).Format(checkinDisplayTimeLayout))
	}
	now := s.now()
	if now.Before(row.CheckedInAt) {
		now = row.CheckedInAt
	}
	ok, err := s.repo.Checkout(ctx, id, now)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, rescode.New(rescode.Conflict, "Hội viên đã checkout")
	}
	row.CheckedOutAt = &now
	return checkinToItem(*row), nil
}

func (s *checkinService) Present(ctx context.Context, branchID uint) (v1.PresentRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, branchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem chi nhánh này")
	}
	now := s.now()
	// Cùng định nghĩa với Mật độ & dashboard: chưa checkout, vào trong repository.PresentWindow gần nhất.
	rows, err := s.repo.Present(ctx, branchIDs, repository.PresentSince(now))
	if err != nil {
		return nil, err
	}
	out := make(v1.PresentRes, 0, len(rows))
	for _, r := range rows {
		out = append(out, &v1.PresentItem{
			CheckinID: r.ID, MemberID: r.MemberID, MemberCode: r.MemberCode, MemberName: r.MemberName,
			MemberPhone: r.MemberPhone, MemberAvatarURL: r.MemberAvatarURL, BranchID: r.BranchID, BranchName: r.BranchName,
			PlanName: r.PlanName, CheckedInAt: r.CheckedInAt, Minutes: max(0, int(now.Sub(r.CheckedInAt).Minutes())),
		})
	}
	return out, nil
}

// ---- Lịch sử / thống kê --------------------------------------------------------

func checkinToItem(r repository.CheckinRow) *v1.CheckinItem {
	denyMsg := checkinDenyMessages[r.DenyReason]
	switch {
	case r.DenyReason == "":
		denyMsg = ""
	case r.StoredDenyMessage != "":
		denyMsg = r.StoredDenyMessage
	case r.HolidayDenyMessage != nil && *r.HolidayDenyMessage != "":
		denyMsg = *r.HolidayDenyMessage
	}
	return &v1.CheckinItem{
		ID: r.ID, MemberID: r.MemberID, MemberCode: r.MemberCode, MemberName: r.MemberName, MemberPhone: r.MemberPhone,
		MemberAvatarURL: r.MemberAvatarURL, BranchID: r.BranchID, BranchName: r.BranchName,
		SubscriptionID: r.SubscriptionID, PlanName: r.PlanName, PlanType: r.PlanType,
		CheckedInAt: r.CheckedInAt, CheckedOutAt: r.CheckedOutAt, Method: r.Method, Result: r.Result,
		DenyReason: r.DenyReason, DenyMessage: denyMsg,
		StaffID: r.StaffID, StaffName: r.StaffName, Note: r.Note,
		DeviceID: r.DeviceID, DayPassID: r.DayPassID, TrialPassID: r.TrialPassID,
	}
}

// checkinParseDay đọc "YYYY-MM-DD" (giờ VN) → 00:00 của ngày đó.
func checkinParseDay(s, field string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	d, err := time.ParseInLocation(consts.DateLayout, s, checkinVNLoc)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, field+" phải có dạng YYYY-MM-DD")
	}
	return &d, nil
}

func (s *checkinService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem chi nhánh này")
	}
	from, err := checkinParseDay(req.From, "Từ ngày")
	if err != nil {
		return nil, err
	}
	to, err := checkinParseDay(req.To, "Đến ngày")
	if err != nil {
		return nil, err
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, rescode.New(rescode.BadRequest, "Đến ngày phải sau hoặc bằng Từ ngày")
	}
	if to != nil {
		end := to.AddDate(0, 0, 1)
		to = &end
	}
	q := req.PageReq.ToQuery(nil, checkinListDefaultOrderField)
	rows, total, err := s.repo.List(ctx, q, repository.CheckinListFilter{
		BranchIDs: branchIDs, MemberID: req.MemberID, Result: req.Result, Method: req.Method, From: from, To: to,
	})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, checkinToItem), total, q), nil
}

func (s *checkinService) ListMemberCheckins(ctx context.Context, memberID uint, page dto.PageReq) (*dto.PageRes[*v1.CheckinItem], error) {
	q := page.ToQuery(nil, checkinListDefaultOrderField)
	rows, total, err := s.repo.List(ctx, q, repository.CheckinListFilter{MemberID: memberID})
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, checkinToItem), total, q), nil
}

func (s *checkinService) Stats(ctx context.Context, branchID uint, date string) (*v1.StatsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, branchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem chi nhánh này")
	}
	day, err := checkinParseDay(date, "Ngày")
	if err != nil {
		return nil, err
	}
	if day == nil {
		d := checkinStartOfDay(s.now())
		day = &d
	}
	// "Đang trong phòng tập" chỉ có nghĩa với hôm nay (định nghĩa chung repository.PresentSince); ngày khác → 0.
	var presentSince *time.Time
	if now := s.now(); day.Equal(checkinStartOfDay(now)) {
		ps := repository.PresentSince(now)
		presentSince = &ps
	}
	rows, err := s.repo.Stats(ctx, branchIDs, *day, day.AddDate(0, 0, 1), presentSince)
	if err != nil {
		return nil, err
	}
	res := &v1.StatsRes{
		Date: day.Format(consts.DateLayout), UniqueMembers: rows.UniqueMembers, Present: rows.Present,
		ByHour: make([]*v1.HourStat, 24), DenyReasons: make([]*v1.ReasonStat, 0, len(rows.ByReason)),
	}
	for h := range res.ByHour {
		res.ByHour[h] = &v1.HourStat{Hour: h}
	}
	for _, r := range rows.ByHour {
		if r.Hour < 0 || r.Hour > 23 {
			continue
		}
		if r.Result == entity.CheckinAllowed {
			res.ByHour[r.Hour].Allowed += r.Count
			res.Allowed += r.Count
		} else {
			res.ByHour[r.Hour].Denied += r.Count
			res.Denied += r.Count
		}
	}
	res.Total = res.Allowed + res.Denied
	for _, r := range rows.ByReason {
		res.DenyReasons = append(res.DenyReasons, &v1.ReasonStat{Reason: r.DenyReason, Message: checkinDenyMessages[r.DenyReason], Count: r.Count})
	}
	return res, nil
}
