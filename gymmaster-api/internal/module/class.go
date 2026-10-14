package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	classctrl "gymmaster-api/internal/controller/class"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// NewClassBookingService dựng BookingService dùng chung (module cổng hội viên, cron gọi lại).
func NewClassBookingService(d *Deps) service.BookingService {
	return service.NewBookingService(
		repository.NewClassScheduleRepository(d.DB),
		repository.NewClassBookingRepository(d.DB),
		d.Tx, d.Settings, d.Notifier,
	)
}

func RegisterClass(d *Deps, admin *ghttp.RouterGroup) {
	scheduleRepo := repository.NewClassScheduleRepository(d.DB)
	bookingRepo := repository.NewClassBookingRepository(d.DB)
	typeSvc := service.NewClassTypeService(repository.NewClassTypeRepository(d.DB), d.Audit)
	scheduleSvc := service.NewClassScheduleService(scheduleRepo, bookingRepo, d.Tx, d.Audit, d.Notifier)
	admin.Bind(classctrl.NewV1(typeSvc, scheduleSvc, NewClassBookingService(d)))
}
