package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	checkinctrl "gymmaster-api/internal/controller/checkin"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterCheckin: QR động, quét/check-in thủ công, checkout, lịch sử ra vào (/v1/admin).
func RegisterCheckin(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewCheckinService(repository.NewCheckinRepository(d.DB), d.Tx, d.Settings)
	admin.Bind(checkinctrl.NewV1(svc))
}
