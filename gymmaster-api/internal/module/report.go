package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	reportctrl "gymmaster-api/internal/controller/report"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterReport: báo cáo & dashboard (/v1/admin/reports/*).
func RegisterReport(d *Deps, admin *ghttp.RouterGroup) {
	admin.Bind(reportctrl.NewV1(service.NewReportService(repository.NewReportRepository(d.DB))))
}
