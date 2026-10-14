package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	kpictrl "gymmaster-api/internal/controller/kpi"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterKPI: mật độ phòng tập realtime, chỉ tiêu doanh số KPI & báo cáo Cohort Retention (/v1/admin, /v1/member, /v1/public).
func RegisterKPI(d *Deps, admin, member, public *ghttp.RouterGroup) {
	svc := service.NewKPIService(repository.NewKPIRepository(d.DB), d.Audit)
	if admin != nil {
		admin.Bind(kpictrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(kpictrl.NewMemberV1(svc))
	}
	if public != nil {
		public.Bind(kpictrl.NewPublicV1(svc))
	}
}
