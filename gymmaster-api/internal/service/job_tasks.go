package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/mailer"
	"gymmaster-api/internal/repository"
)

// Tên & lịch các job hệ thống (giờ Asia/Ho_Chi_Minh). Đăng ký vào cron ở internal/module/system.go.
const (
	JobExpireSubscriptions = "expire_subscriptions"
	JobApplyFreezes        = "apply_freezes"
	JobRemindExpiring      = "remind_expiring"
	JobMarkClassNoShows    = "mark_class_no_shows"
	JobCleanupTokens       = "cleanup_tokens"
)

// JobSubscriptionOps: phần SubscriptionService mà job dùng (SubscriptionService thỏa interface này).
type JobSubscriptionOps interface {
	ExpireDue(ctx context.Context, today time.Time) (int, error)
	ApplyDueFreezes(ctx context.Context, today time.Time) (int, error)
	ListExpiring(ctx context.Context, days []int, today time.Time) ([]repository.ExpiringSubscription, error)
}

// JobBookingOps: phần BookingService mà job dùng.
type JobBookingOps interface {
	MarkNoShows(ctx context.Context, now time.Time) (int, error)
}

// JobTasks gom các hàm Run của job hệ thống. Mỗi hàm trả (số bản ghi bị ảnh hưởng, lỗi).
type JobTasks struct {
	subs     JobSubscriptionOps
	bookings JobBookingOps
	repo     repository.JobRepository
	settings SettingService
	notifier Notifier
	mail     mailer.Mailer
	now      func() time.Time
}

func NewJobTasks(subs JobSubscriptionOps, bookings JobBookingOps, repo repository.JobRepository,
	settings SettingService, notifier Notifier, mail mailer.Mailer) *JobTasks {
	return &JobTasks{subs: subs, bookings: bookings, repo: repo, settings: settings, notifier: notifier, mail: mail, now: time.Now}
}

const (
	jobMemberSubsLink       = "/member/subscriptions"
	jobTokenRetention       = 30 * 24 * time.Hour
	jobReceptionSummaryMax  = 10
	jobPermSubscriptionSell = "subscription.sell"
)

func jobFmtDate(t time.Time) string { return t.Format("02/01/2006") }

// ExpireSubscriptions (00:05): gói hết hạn → EXPIRED (+ hủy PENDING quá hạn thanh toán),
// rồi báo SUBSCRIPTION_EXPIRED cho hội viên có tài khoản cổng và chưa có gói mới.
func (t *JobTasks) ExpireSubscriptions(ctx context.Context) (int, error) {
	since, err := t.repo.DBNow(ctx)
	if err != nil {
		return 0, err
	}
	n, runErr := t.subs.ExpireDue(ctx, t.now())
	notices, err := t.repo.ExpiredNotices(ctx, since)
	if err != nil {
		return n, errors.Join(runErr, err)
	}
	var notifyErr error
	for _, nt := range notices {
		if nt.MemberUserID == nil {
			continue
		}
		body := fmt.Sprintf("Gói %s của bạn đã hết hạn ngày %s. Liên hệ lễ tân %s để gia hạn.",
			nt.PlanName, jobFmtDate(nt.EndDate), nt.BranchName)
		if err := t.notifier.Notify(ctx, []uint{*nt.MemberUserID}, NotifySubscriptionExpired,
			"Gói tập đã hết hạn", body, jobMemberSubsLink); err != nil && notifyErr == nil {
			notifyErr = err
		}
	}
	return n, errors.Join(runErr, notifyErr)
}

// ApplyFreezes (00:10): kết thúc bảo lưu đến hạn, áp bảo lưu đã duyệt đến ngày bắt đầu.
func (t *JobTasks) ApplyFreezes(ctx context.Context) (int, error) {
	return t.subs.ApplyDueFreezes(ctx, t.now())
}

// RemindExpiring (08:00): nhắc gói sắp hết hạn theo settings subscription.remind_before_days (mặc định [7, 1]):
// thông báo in-app + email cho hội viên, và một thông báo tổng hợp cho lễ tân từng chi nhánh.
func (t *JobTasks) RemindExpiring(ctx context.Context) (int, error) {
	days := t.settings.IntSlice(ctx, SettingSubscriptionRemindBeforeDays, []int{7, 1})
	rows, err := t.subs.ListExpiring(ctx, days, t.now())
	if err != nil {
		return 0, err
	}
	var firstErr error
	keep := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	byBranch := map[uint][]repository.ExpiringSubscription{}
	for _, r := range rows {
		byBranch[r.BranchID] = append(byBranch[r.BranchID], r)
		when := fmt.Sprintf("còn %d ngày", r.DaysLeft)
		if r.DaysLeft <= 0 {
			when = "hết hạn hôm nay"
		}
		if r.MemberUserID != nil {
			body := fmt.Sprintf("Gói %s của bạn sẽ hết hạn vào %s (%s). Gia hạn sớm để không gián đoạn tập luyện.",
				r.PlanName, jobFmtDate(r.EndDate), when)
			keep(t.notifier.Notify(ctx, []uint{*r.MemberUserID}, NotifySubscriptionExpiring, "Gói tập sắp hết hạn", body, jobMemberSubsLink))
		}
		if email := strings.TrimSpace(r.MemberEmail); email != "" && t.mail != nil {
			html, err := mailer.Render(mailer.TplSubscriptionExpiring, mailer.ExpiringData{
				MemberName: r.MemberName, PlanName: r.PlanName, EndDate: jobFmtDate(r.EndDate),
				DaysLeft: r.DaysLeft, BranchName: r.BranchName,
			})
			if err != nil {
				keep(err)
				continue
			}
			t.mail.SendAsync(mailer.Message{
				To: []string{email}, Subject: fmt.Sprintf("[GymMaster] Gói %s sắp hết hạn (%s)", r.PlanName, when), HTML: html,
			})
		}
	}

	branchIDs := make([]uint, 0, len(byBranch))
	for id := range byBranch {
		branchIDs = append(branchIDs, id)
	}
	sort.Slice(branchIDs, func(i, j int) bool { return branchIDs[i] < branchIDs[j] })
	for _, id := range branchIDs {
		list := byBranch[id]
		names := make([]string, 0, jobReceptionSummaryMax)
		for i, r := range list {
			if i == jobReceptionSummaryMax {
				names = append(names, fmt.Sprintf("… và %d hội viên khác", len(list)-jobReceptionSummaryMax))
				break
			}
			names = append(names, fmt.Sprintf("%s (%s, %s – %s)", r.MemberName, r.MemberPhone, r.PlanName, jobFmtDate(r.EndDate)))
		}
		bid := id
		title := fmt.Sprintf("%d hội viên sắp hết hạn gói tập – %s", len(list), list[0].BranchName)
		link := fmt.Sprintf("/admin/subscriptions?status=expiring&branch_id=%d", id)
		keep(t.notifier.NotifyPermission(ctx, jobPermSubscriptionSell, &bid, NotifySubscriptionExpiring, title, strings.Join(names, "\n"), link))
	}
	return len(rows), firstErr
}

// MarkClassNoShows (mỗi 15 phút): lượt BOOKED của lớp đã kết thúc → NO_SHOW, khóa đặt lớp khi vắng quá giới hạn.
func (t *JobTasks) MarkClassNoShows(ctx context.Context) (int, error) {
	return t.bookings.MarkNoShows(ctx, t.now())
}

// CleanupTokens (03:00): xóa refresh token hết hạn hoặc bị thu hồi quá 30 ngày.
func (t *JobTasks) CleanupTokens(ctx context.Context) (int, error) {
	n, err := t.repo.DeleteStaleRefreshTokens(ctx, t.now().Add(-jobTokenRetention))
	return int(n), err
}

// SendTestEmail gửi đồng bộ một email thử (dùng kiểm tra cấu hình SMTP).
func (t *JobTasks) SendTestEmail(ctx context.Context, to string) error {
	if t.mail == nil {
		return errors.New("chưa cấu hình mailer")
	}
	html, err := mailer.Render(mailer.TplGeneric, mailer.GenericData{
		Title: "Email thử từ GymMaster", Greeting: "Xin chào,",
		Paragraphs: []string{
			"Đây là email kiểm tra cấu hình gửi thư của hệ thống GymMaster.",
			"Thời điểm gửi: " + t.now().In(reportLoc).Format("15:04 02/01/2006") + " (" + consts.TimeZone + ").",
		},
	})
	if err != nil {
		return err
	}
	err = t.mail.Send(ctx, mailer.Message{To: []string{to}, Subject: "[GymMaster] Email thử", HTML: html})
	if err != nil {
		g.Log().Warningf(ctx, "gửi email thử: %v", err)
	}
	return err
}
