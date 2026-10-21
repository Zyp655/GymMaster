package module

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	lockerctrl "gymmaster-api/internal/controller/locker"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var (
	lockerMu       sync.Mutex
	lockerServices = map[*Deps]service.LockerService{}
)

func LockerServiceOf(d *Deps) service.LockerService {
	lockerMu.Lock()
	defer lockerMu.Unlock()
	if s, ok := lockerServices[d]; ok {
		return s
	}
	s := service.NewLockerServiceWithInvoices(
		repository.NewLockerRepository(d.DB),
		d.Tx,
		d.Audit,
		d.DB,
		d.Invoices,
	)
	lockerServices[d] = s

	if _, exists := cron.DefaultRegistry().Get(service.JobExpireLockerRentals); !exists {
		cron.Register(cron.Job{
			Name:        service.JobExpireLockerRentals,
			Spec:        "20 0 * * *",
			Description: "Tự động giải phóng các lượt thuê tủ đồ đã quá hạn",
			Run: func(ctx context.Context) (int, error) {
				return s.ExpireDueRentals(ctx, time.Now())
			},
		})
	}
	return s
}

// RegisterLocker đăng ký route tủ đồ cho nhân viên (/v1/admin) và cổng hội viên (/v1/member).
func RegisterLocker(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := LockerServiceOf(d)
	if admin != nil {
		admin.Bind(lockerctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(lockerctrl.NewMemberV1(svc))
	}
}
