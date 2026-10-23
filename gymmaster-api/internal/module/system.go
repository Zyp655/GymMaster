package module

import (
	"sync"

	"github.com/gogf/gf/v2/net/ghttp"

	auditctrl "gymmaster-api/internal/controller/auditlog"
	jobctrl "gymmaster-api/internal/controller/job"
	notifctrl "gymmaster-api/internal/controller/notification"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/pkg/mailer"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var systemJobsOnce sync.Once

// NewMailer trả mailer dùng chung (SMTP từ cfg.Mail). Module khác cần gửi email dùng hàm này.
func NewMailer(d *Deps) mailer.Mailer {
	return mailer.Default(d.Config.Mail)
}

// RegisterSystem: thông báo của nhân viên, nhật ký thao tác, job định kỳ (+ đăng ký các job hệ thống vào cron).
func RegisterSystem(d *Deps, admin *ghttp.RouterGroup) {
	jobRepo := repository.NewJobRepository(d.DB)
	tasks := service.NewJobTasks(d.Subscriptions, NewClassBookingService(d), jobRepo, d.Settings, d.Notifier, NewMailer(d))
	systemJobsOnce.Do(func() { registerSystemJobs(tasks) })

	admin.Bind(
		notifctrl.NewV1(service.NewNotificationService(repository.NewNotificationRepository(d.DB))),
		auditctrl.NewV1(service.NewAuditLogService(repository.NewAuditLogRepository(d.DB))),
		jobctrl.NewV1(service.NewJobService(service.RunnerScheduler{Runner: cron.Default(d.DB)}, jobRepo), tasks),
	)
}

// RegisterMemberNotification: thông báo của hội viên (/v1/member/notifications, role MEMBER).
func RegisterMemberNotification(d *Deps, member *ghttp.RouterGroup) {
	member.Bind(notifctrl.NewMemberV1(service.NewNotificationService(repository.NewNotificationRepository(d.DB))))
}

// registerSystemJobs: các job hệ thống. Module khác thêm job bằng cron.Register(...) trong Register<M> của mình
// (xem docs/api/reports-system.md §Job định kỳ).
func registerSystemJobs(t *service.JobTasks) {
	cron.Register(cron.Job{
		Name: service.JobExpireSubscriptions, Spec: "5 0 * * *",
		Description: "Chuyển gói tập quá hạn sang Hết hạn, hủy gói chờ thanh toán quá 3 ngày, báo hội viên",
		Run:         t.ExpireSubscriptions,
	})
	cron.Register(cron.Job{
		Name: service.JobApplyFreezes, Spec: "10 0 * * *",
		Description: "Kết thúc bảo lưu đến hạn và áp các bảo lưu đã duyệt đến ngày bắt đầu",
		Run:         t.ApplyFreezes,
	})
	cron.Register(cron.Job{
		Name: service.JobRemindExpiring, Spec: "0 8 * * *",
		Description: "Nhắc gia hạn gói sắp hết hạn (in-app + email) và gửi danh sách cho lễ tân",
		Run:         t.RemindExpiring,
	})
	cron.Register(cron.Job{
		Name: service.JobMarkClassNoShows, Spec: "*/15 * * * *",
		Description: "Đánh dấu Vắng mặt lượt đặt lớp đã kết thúc, khóa đặt lớp khi vắng quá giới hạn",
		Run:         t.MarkClassNoShows,
	})
	cron.Register(cron.Job{
		Name: service.JobCleanupTokens, Spec: "0 3 * * *",
		Description: "Xóa refresh token hết hạn hoặc bị thu hồi quá 30 ngày",
		Run:         t.CleanupTokens,
	})
}
