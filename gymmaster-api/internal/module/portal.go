package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	portalctrl "gymmaster-api/internal/controller/portal"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterPortal: cổng hội viên (/v1/member/*, role MEMBER) – hồ sơ, QR, gói tập, bảo lưu, check-in,
// lịch lớp/đặt chỗ, PT, chỉ số cơ thể, hóa đơn. Dùng lại service của các module nghiệp vụ.
func RegisterPortal(d *Deps, member *ghttp.RouterGroup) {
	subs := d.Subscriptions
	if subs == nil {
		subs = service.NewSubscriptionService(repository.NewSubscriptionRepository(d.DB), d.Invoices, d.Tx, d.Audit, d.Notifier)
		d.Subscriptions = subs
	}
	svc := service.NewPortalService(service.PortalDeps{
		Repo: repository.NewPortalRepository(d.DB),
		Members: service.NewMemberService(
			repository.NewMemberRepository(d.DB),
			repository.NewBodyMetricRepository(d.DB),
			repository.NewUserRepository(d.DB),
			d.Tx, d.Audit,
		),
		Checkins:        service.NewCheckinService(repository.NewCheckinRepository(d.DB), d.Tx, d.Settings),
		Subscriptions:   subs,
		Invoices:        d.Invoices,
		Bookings:        NewClassBookingService(d),
		Schedules:       repository.NewClassScheduleRepository(d.DB),
		PT:              PTServiceOf(d),
		Settings:        d.Settings,
		Audit:           d.Audit,
		UploadDir:       d.Config.Upload.Dir,
		UploadPublicURL: d.Config.Upload.PublicURL,
	})
	member.Bind(portalctrl.NewV1(svc))
}
