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

// checkinPickSubscription chọn gói dùng để vào tại branchID trong ngày today (YYYY-MM-DD, giờ VN).
//
// Gói hợp lệ: ACTIVE, start_date ≤ today ≤ end_date, đúng phạm vi chi nhánh, gói SESSION còn buổi.
// Ưu tiên: gói TIME trước gói SESSION, rồi gói hết hạn sớm nhất, rồi id nhỏ nhất.
// Không có gói hợp lệ: FROZEN → SUBSCRIPTION_FROZEN; gói đúng hạn/đúng chi nhánh hết buổi →
// NO_SESSIONS_LEFT; gói đúng hạn nhưng khác chi nhánh → WRONG_BRANCH; còn lại NO_ACTIVE_SUBSCRIPTION.
func checkinPickSubscription(subs []entity.CheckinSubscriptionView, branchID uint, today string) checkinPick {
	var valid []*entity.CheckinSubscriptionView
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
			if c := a.EndDate.Compare(b.EndDate); c != 0 {
				return c
			}
			return int(a.ID) - int(b.ID)
		})
		return checkinPick{chosen: valid[0]}
	}
	switch {
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
	trimmed := strings.TrimSpace(token)
	if strings.HasPrefix(trimmed, "TP-") {
		return s.scanTrialPass(ctx, trimmed, branch)
	}
	if strings.HasPrefix(trimmed, "DP-") {
		return s.scanDayPass(ctx, trimmed, branch)
	}
	memberID, err := qrtoken.ParseMemberID(token)
	if err != nil {
		// QR hỏng hoàn toàn: không xác định được hội viên → không ghi DB.
		return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
	}
	return s.process(ctx, checkinAttempt{method: entity.CheckinMethodQR, branch: branch, memberID: memberID, token: token})
}

func (s *checkinService) scanTrialPass(ctx context.Context, token string, branch *entity.Branch) (*v1.ScanResult, error) {
	tpRepo, ok := s.repo.(repository.CheckinTrialPassRepository)
	if !ok {
		return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
	}
	tp, err := tpRepo.FindTrialPassByToken(ctx, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
		}
		return nil, err
	}
	now := s.now()
	today := checkinDate(now.In(checkinVNLoc))
	memberInfo := &v1.CheckinMember{
		ID:       tp.LeadID,
		Code:     fmt.Sprintf("TRIAL-%d", tp.ID),
		FullName: tp.FullName + " (Tập thử)",
		Phone:    tp.Phone,
		Status:   "TRIAL",
	}
	deny := func(reason, msg string) *v1.ScanResult {
		if msg == "" {
			msg = checkinDenyMessages[reason]
		}
		return &v1.ScanResult{
			Result:      entity.CheckinDenied,
			DenyReason:  reason,
			DenyMessage: msg,
			Member:      memberInfo,
			Checkin: &v1.CheckinRef{
				ID:          tp.ID,
				CheckedInAt: now,
				BranchID:    branch.ID,
				BranchName:  branch.Name,
			},
			Warnings: []string{},
		}
	}
	if !checkinBranchOpen(branch, now) {
		return deny(entity.CheckinDenyBranchClosed, fmt.Sprintf("Ngoài giờ mở cửa của chi nhánh (%s - %s)", branch.OpenTime, branch.CloseTime)), nil
	}
	if tp.BranchID != branch.ID {
		return deny(entity.CheckinDenyWrongBranch, fmt.Sprintf("Vé tập thử chỉ áp dụng tại chi nhánh %s", tp.BranchName)), nil
	}
	if tp.UsedAt != nil {
		return deny(entity.CheckinDenyTrialPassUsed, ""), nil
	}
	if checkinDate(tp.ValidDate) != today {
		return deny(entity.CheckinDenyTrialPassExpired, fmt.Sprintf("Vé tập thử chỉ có hiệu lực ngày %s", tp.ValidDate.Format(checkinDisplayDateLayout))), nil
	}
	if err := tpRepo.MarkTrialPassUsed(ctx, tp.ID, now); err != nil {
		return nil, err
	}
	return &v1.ScanResult{
		Result: entity.CheckinAllowed,
		Member: memberInfo,
		Checkin: &v1.CheckinRef{
			ID:          tp.ID,
			CheckedInAt: now,
			BranchID:    branch.ID,
			BranchName:  branch.Name,
		},
		Warnings: []string{"Khách tập thử 1 ngày theo vé " + tp.QRToken},
	}, nil
}

func (s *checkinService) scanDayPass(ctx context.Context, token string, branch *entity.Branch) (*v1.ScanResult, error) {
	dpRepo, ok := s.repo.(repository.CheckinDayPassRepository)
	if !ok {
		return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
	}
	dp, err := dpRepo.FindDayPassByToken(ctx, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return checkinDeniedNoMember(entity.CheckinDenyQRInvalid), nil
		}
		return nil, err
	}
	now := s.now()
	today := checkinDate(now.In(checkinVNLoc))
	label := "Vé ngày"
	if dp.Type == entity.DayPassTypeGuest {
		label = "Khách mời"
	}
	memberInfo := &v1.CheckinMember{
		ID:       dp.ID,
		Code:     fmt.Sprintf("DP-%d", dp.ID),
		FullName: fmt.Sprintf("%s (%s)", dp.BuyerName, label),
		Phone:    dp.Phone,
		Status:   dp.Type,
	}
	deny := func(reason, msg string) *v1.ScanResult {
		if msg == "" {
			msg = checkinDenyMessages[reason]
		}
		return &v1.ScanResult{
			Result:      entity.CheckinDenied,
			DenyReason:  reason,
			DenyMessage: msg,
			Member:      memberInfo,
			Checkin: &v1.CheckinRef{
				ID:          dp.ID,
				CheckedInAt: now,
				BranchID:    branch.ID,
				BranchName:  branch.Name,
			},
			Warnings: []string{},
		}
	}
	if !checkinBranchOpen(branch, now) {
		return deny(entity.CheckinDenyBranchClosed, fmt.Sprintf("Ngoài giờ mở cửa của chi nhánh (%s - %s)", branch.OpenTime, branch.CloseTime)), nil
	}
	if dp.BranchID != branch.ID {
		return deny(entity.CheckinDenyWrongBranch, fmt.Sprintf("Vé chỉ áp dụng tại chi nhánh %s", dp.BranchName)), nil
	}
	if dp.UsedAt != nil {
		return deny(entity.CheckinDenyDayPassUsed, ""), nil
	}
	if checkinDate(dp.ValidDate) != today {
		return deny(entity.CheckinDenyDayPassExpired, fmt.Sprintf("Vé chỉ có hiệu lực ngày %s", dp.ValidDate.Format(checkinDisplayDateLayout))), nil
	}
	if err := dpRepo.MarkDayPassUsed(ctx, dp.ID, now); err != nil {
		return nil, err
	}
	return &v1.ScanResult{
		Result: entity.CheckinAllowed,
		Member: memberInfo,
		Checkin: &v1.CheckinRef{
			ID:          dp.ID,
			CheckedInAt: now,
			BranchID:    branch.ID,
			BranchName:  branch.Name,
		},
		Warnings: []string{fmt.Sprintf("%s theo mã %s", label, dp.QRToken)},
	}, nil
}

func (s *checkinService) Manual(ctx context.Context, memberID, branchID uint, note string) (*v1.ScanResult, error) {
	branch, err := s.checkBranch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	return s.process(ctx, checkinAttempt{
		method: entity.CheckinMethodManual, branch: branch, memberID: memberID, note: strings.TrimSpace(note),
	})
}

type checkinAttempt struct {
	method   string
	branch   *entity.Branch
	memberID uint
	token    string
	note     string
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
				if a.method == entity.CheckinMethodQR {
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
		if a.method == entity.CheckinMethodQR {
			switch err := qrtoken.Verify(a.token, m.QRSecret, now); {
			case errors.Is(err, qrtoken.ErrExpired):
				reason = entity.CheckinDenyQRExpired
			case err != nil:
				reason = entity.CheckinDenyQRInvalid
			}
		}
		if reason == "" && !checkinBranchOpen(a.branch, now) {
			reason = entity.CheckinDenyBranchClosed
			message = fmt.Sprintf("Ngoài giờ mở cửa của chi nhánh (%s - %s)", a.branch.OpenTime, a.branch.CloseTime)
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
			subs, err := s.repo.LockSubscriptions(ctx, m.ID)
			if err != nil {
				return err
			}
			pick = checkinPickSubscription(subs, a.branch.ID, today)

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
				reason, message = pick.reason, pick.message
			}
		}

		rec := &entity.Checkin{
			MemberID: m.ID, BranchID: a.branch.ID, CheckedInAt: now, Method: a.method,
			StaffID: reqctx.UserID(ctx), Note: a.note,
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
	rows, err := s.repo.Present(ctx, branchIDs, checkinStartOfDay(now))
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
	return &v1.CheckinItem{
		ID: r.ID, MemberID: r.MemberID, MemberCode: r.MemberCode, MemberName: r.MemberName, MemberPhone: r.MemberPhone,
		MemberAvatarURL: r.MemberAvatarURL, BranchID: r.BranchID, BranchName: r.BranchName,
		SubscriptionID: r.SubscriptionID, PlanName: r.PlanName, PlanType: r.PlanType,
		CheckedInAt: r.CheckedInAt, CheckedOutAt: r.CheckedOutAt, Method: r.Method, Result: r.Result,
		DenyReason: r.DenyReason, DenyMessage: checkinDenyMessages[r.DenyReason],
		StaffID: r.StaffID, StaffName: r.StaffName, Note: r.Note,
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
	rows, err := s.repo.Stats(ctx, branchIDs, *day, day.AddDate(0, 0, 1))
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
