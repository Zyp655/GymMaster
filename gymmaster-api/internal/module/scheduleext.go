package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	scheduleextctrl "gymmaster-api/internal/controller/scheduleext"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterScheduleExt: lịch rảnh HLV, đơn nghỉ phép, dạy thay & ngày nghỉ lễ (/v1/admin & /v1/public).
func RegisterScheduleExt(d *Deps, admin, public *ghttp.RouterGroup) {
	svc := service.NewScheduleExtService(repository.NewScheduleExtRepository(d.DB), d.Tx, d.Audit)
	if admin != nil {
		admin.Bind(scheduleextctrl.NewAdminV1(svc))
	}
	if public != nil {
		public.Bind(scheduleextctrl.NewPublicV1(svc))
	}
}
