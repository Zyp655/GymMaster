package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	shiftctrl "gymmaster-api/internal/controller/shift"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterShift(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewShiftService(repository.NewShiftRepository(d.DB), d.Tx)
	admin.Bind(shiftctrl.NewV1(svc))
}
