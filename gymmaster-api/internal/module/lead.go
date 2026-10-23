package module

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	leadctrl "gymmaster-api/internal/controller/lead"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

var (
	leadMu       sync.Mutex
	leadServices = map[*Deps]service.LeadService{}
)

func LeadServiceOf(d *Deps) service.LeadService {
	leadMu.Lock()
	defer leadMu.Unlock()
	if s, ok := leadServices[d]; ok {
		return s
	}
	s := service.NewLeadService(
		repository.NewLeadRepository(d.DB),
		d.Tx,
		d.Audit,
		d.Notifier,
		NewMailer(d),
	)
	leadServices[d] = s

	if _, exists := cron.DefaultRegistry().Get(service.JobBirthdayGreetings); !exists {
		cron.Register(cron.Job{
			Name:        service.JobBirthdayGreetings,
			Spec:        "0 8 * * *",
			Description: "Gửi lời chúc mừng sinh nhật và mã ưu đãi giảm 10% cho hội viên có sinh nhật hôm nay",
			Run: func(ctx context.Context) (int, error) {
				return s.SendBirthdayGreetings(ctx, time.Now())
			},
		})
	}
	if _, exists := cron.DefaultRegistry().Get(service.JobChurnRiskScan); !exists {
		cron.Register(cron.Job{
			Name:        service.JobChurnRiskScan,
			Spec:        "0 6 * * *",
			Description: "Quét hội viên có gói đang hiệu lực nhưng hơn 14 ngày không check-in để nhắc lễ tân chăm sóc",
			Run: func(ctx context.Context) (int, error) {
				return s.ScanChurnRisk(ctx, time.Now())
			},
		})
	}
	return s
}

// RegisterLead: đăng ký toàn bộ route CRM khách tiềm năng, vé tập thử & chăm sóc hội viên (/v1/admin & /v1/public).
func RegisterLead(d *Deps, admin, public *ghttp.RouterGroup) {
	svc := LeadServiceOf(d)
	if admin != nil {
		admin.Bind(leadctrl.NewAdminV1(svc))
	}
	if public != nil {
		public.Bind(leadctrl.NewPublicV1(svc))
	}
}
