package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gcode"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
)

// CheckinRuleInput chứa dữ liệu ngữ cảnh để chuỗi CheckinRule đánh giá độc lập.
// Thiết kế theo Mục 16.6 (PLAN.md): tách mỗi quy tắc kiểm tra check-in thành struct riêng
// cài đặt interface CheckinRule để dễ bảo trì, mở rộng và unit test từng rule.
type CheckinRuleInput struct {
	Now                   time.Time
	BranchID              uint
	BranchHolidayClosed   bool
	BranchHolidayName     string
	Member                *entity.Member
	RequireSignedContract bool
	HasSignedContract     bool
	HasOverdueInstallment bool
	Subscription          *entity.Subscription
	Plan                  *entity.MembershipPlan
	AccessHoursWindow     string
	LastCheckinAt         *time.Time
	CooldownMinutes       int
}

// CheckinRuleViolation mô tả lý do từ chối check-in của một rule.
type CheckinRuleViolation struct {
	RuleName   string
	DenyReason string
	Code       gcode.Code
	Message    string
}

// CheckinRule là giao diện chuẩn cho từng quy tắc kiểm tra check-in.
type CheckinRule interface {
	Name() string
	Evaluate(ctx context.Context, in *CheckinRuleInput) *CheckinRuleViolation
}

// DefaultCheckinRules trả về chuỗi quy tắc kiểm tra check-in theo thứ tự ưu tiên nghiệp vụ.
func DefaultCheckinRules() []CheckinRule {
	return []CheckinRule{
		MemberActiveRule{},
		BranchHolidayClosedRule{},
		ContractSignedRule{},
		InstallmentNotOverdueRule{},
		SubscriptionScopeAndDateRule{},
		OffPeakHoursRule{},
		SessionBalanceRule{},
		DuplicateCheckinCooldownRule{},
	}
}

// EvaluateCheckinRules chạy lần lượt từng rule, dừng ngay ở vi phạm đầu tiên.
func EvaluateCheckinRules(ctx context.Context, in *CheckinRuleInput, rules ...CheckinRule) *CheckinRuleViolation {
	if len(rules) == 0 {
		rules = DefaultCheckinRules()
	}
	for _, r := range rules {
		if v := r.Evaluate(ctx, in); v != nil {
			return v
		}
	}
	return nil
}

// 1. MemberActiveRule: hội viên phải tồn tại và có trạng thái ACTIVE.
type MemberActiveRule struct{}

func (MemberActiveRule) Name() string { return "member_active" }

func (MemberActiveRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in == nil || in.Member == nil || in.Member.Status != entity.MemberStatusActive {
		return &CheckinRuleViolation{
			RuleName:   "member_active",
			DenyReason: entity.CheckinDenyMemberInactive,
			Code:       rescode.PermissionDenied,
			Message:    "Hội viên đang bị khóa hoặc ngừng hoạt động",
		}
	}
	return nil
}

// 2. BranchHolidayClosedRule: chi nhánh đóng cửa nghỉ lễ hôm nay (Sprint 25 – 42004).
type BranchHolidayClosedRule struct{}

func (BranchHolidayClosedRule) Name() string { return "branch_holiday_closed" }

func (BranchHolidayClosedRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in != nil && in.BranchHolidayClosed {
		msg := "Chi nhánh đang đóng cửa nghỉ lễ hôm nay"
		if in.BranchHolidayName != "" {
			msg = fmt.Sprintf("Chi nhánh nghỉ lễ (%s)", in.BranchHolidayName)
		}
		return &CheckinRuleViolation{
			RuleName:   "branch_holiday_closed",
			DenyReason: entity.CheckinDenyBranchClosed,
			Code:       rescode.BranchHolidayClosed,
			Message:    msg,
		}
	}
	return nil
}

// 3. ContractSignedRule: bắt buộc ký hợp đồng điện tử nếu được yêu cầu (Sprint 24 – 42001).
type ContractSignedRule struct{}

func (ContractSignedRule) Name() string { return "contract_signed" }

func (ContractSignedRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in != nil && in.RequireSignedContract && !in.HasSignedContract {
		return &CheckinRuleViolation{
			RuleName:   "contract_signed",
			DenyReason: entity.CheckinDenyContractUnsigned,
			Code:       rescode.ContractUnsigned,
			Message:    "Hội viên chưa ký hợp đồng điện tử bắt buộc",
		}
	}
	return nil
}

// 4. InstallmentNotOverdueRule: chặn check-in nếu có kỳ trả góp quá hạn (Sprint 21 – 42003).
type InstallmentNotOverdueRule struct{}

func (InstallmentNotOverdueRule) Name() string { return "installment_not_overdue" }

func (InstallmentNotOverdueRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in != nil && in.HasOverdueInstallment {
		return &CheckinRuleViolation{
			RuleName:   "installment_not_overdue",
			DenyReason: entity.CheckinDenyInstallmentOverdue,
			Code:       rescode.InstallmentOverdue,
			Message:    "Hội viên có kỳ trả góp quá hạn chưa thanh toán",
		}
	}
	return nil
}

// 5. SubscriptionScopeAndDateRule: có gói ACTIVE hợp lệ tại chi nhánh và trong thời hạn.
type SubscriptionScopeAndDateRule struct{}

func (SubscriptionScopeAndDateRule) Name() string { return "subscription_scope_and_date" }

func (SubscriptionScopeAndDateRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in == nil || in.Subscription == nil {
		return &CheckinRuleViolation{
			RuleName:   "subscription_scope_and_date",
			DenyReason: entity.CheckinDenyNoActiveSubscription,
			Code:       rescode.BadRequest,
			Message:    "Hội viên chưa có gói tập đang hoạt động",
		}
	}
	sub := in.Subscription
	if sub.Status == entity.SubStatusFrozen {
		return &CheckinRuleViolation{
			RuleName:   "subscription_scope_and_date",
			DenyReason: entity.CheckinDenySubscriptionFrozen,
			Code:       rescode.BadRequest,
			Message:    "Gói tập đang trong thời gian bảo lưu",
		}
	}
	if sub.Status != entity.SubStatusActive {
		return &CheckinRuleViolation{
			RuleName:   "subscription_scope_and_date",
			DenyReason: entity.CheckinDenyNoActiveSubscription,
			Code:       rescode.BadRequest,
			Message:    "Gói tập không ở trạng thái hoạt động",
		}
	}
	today := dateOnly(in.Now)
	if !sub.EndDate.IsZero() && dateOnly(sub.EndDate).Before(today) {
		return &CheckinRuleViolation{
			RuleName:   "subscription_scope_and_date",
			DenyReason: entity.CheckinDenyNoActiveSubscription,
			Code:       rescode.BadRequest,
			Message:    "Gói tập đã hết hạn",
		}
	}
	if in.Plan != nil && in.Plan.AccessScope == entity.AccessSingleBranch && in.BranchID > 0 && sub.BranchID != in.BranchID {
		return &CheckinRuleViolation{
			RuleName:   "subscription_scope_and_date",
			DenyReason: entity.CheckinDenyWrongBranch,
			Code:       rescode.BadRequest,
			Message:    "Gói tập chỉ áp dụng tại chi nhánh đăng ký",
		}
	}
	return nil
}

// 6. OffPeakHoursRule: kiểm tra khung giờ được phép vào cửa (VD "09:00-16:00", Sprint 24 – 42002).
type OffPeakHoursRule struct{}

func (OffPeakHoursRule) Name() string { return "off_peak_hours" }

func (OffPeakHoursRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in == nil {
		return nil
	}
	window := strings.TrimSpace(in.AccessHoursWindow)
	if window == "" && in.Plan != nil && len(in.Plan.AccessHours) > 0 {
		window = strings.Trim(string(in.Plan.AccessHours), " \"[]")
	}
	if window == "" {
		return nil
	}
	if !withinAccessHoursWindow(in.Now, window) {
		return &CheckinRuleViolation{
			RuleName:   "off_peak_hours",
			DenyReason: entity.CheckinDenyOutsideAccessHours,
			Code:       rescode.OutsideAccessHours,
			Message:    fmt.Sprintf("Gói tập Off-Peak chỉ cho phép check-in trong khung giờ %s", window),
		}
	}
	return nil
}

// 7. SessionBalanceRule: gói theo buổi phải còn số buổi > 0.
type SessionBalanceRule struct{}

func (SessionBalanceRule) Name() string { return "session_balance" }

func (SessionBalanceRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in == nil || in.Subscription == nil || in.Plan == nil {
		return nil
	}
	if in.Plan.Type == entity.PlanTypeSession {
		if in.Subscription.RemainingSessions == nil || *in.Subscription.RemainingSessions <= 0 {
			return &CheckinRuleViolation{
				RuleName:   "session_balance",
				DenyReason: entity.CheckinDenyNoSessionsLeft,
				Code:       rescode.BadRequest,
				Message:    "Gói tập theo buổi đã hết lượt sử dụng",
			}
		}
	}
	return nil
}

// 8. DuplicateCheckinCooldownRule: chống quét trùng trong khoảng CooldownMinutes.
type DuplicateCheckinCooldownRule struct{}

func (DuplicateCheckinCooldownRule) Name() string { return "duplicate_checkin_cooldown" }

func (DuplicateCheckinCooldownRule) Evaluate(_ context.Context, in *CheckinRuleInput) *CheckinRuleViolation {
	if in == nil || in.LastCheckinAt == nil {
		return nil
	}
	cooldown := in.CooldownMinutes
	if cooldown <= 0 {
		cooldown = 60
	}
	if in.Now.Sub(*in.LastCheckinAt) < time.Duration(cooldown)*time.Minute {
		return &CheckinRuleViolation{
			RuleName:   "duplicate_checkin_cooldown",
			DenyReason: entity.CheckinDenyDuplicate,
			Code:       rescode.Conflict,
			Message:    fmt.Sprintf("Hội viên vừa check-in trong vòng %d phút gần đây", cooldown),
		}
	}
	return nil
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func withinAccessHoursWindow(now time.Time, window string) bool {
	parts := strings.Split(window, "-")
	if len(parts) != 2 {
		return true
	}
	startMin, ok1 := parseHHMM(parts[0])
	endMin, ok2 := parseHHMM(parts[1])
	if !ok1 || !ok2 {
		return true
	}
	curMin := now.Hour()*60 + now.Minute()
	if startMin <= endMin {
		return curMin >= startMin && curMin <= endMin
	}
	return curMin >= startMin || curMin <= endMin
}

func parseHHMM(s string) (int, bool) {
	p := strings.Split(strings.TrimSpace(s), ":")
	if len(p) != 2 {
		return 0, false
	}
	h, err1 := strconv.Atoi(p[0])
	m, err2 := strconv.Atoi(p[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
