package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	staffctrl "gymmaster-api/internal/controller/staff"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterStaff: nhân viên, HLV (/staff, /trainers/options).
func RegisterStaff(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewStaffService(
		repository.NewStaffRepository(d.DB),
		repository.NewRoleRepository(d.DB),
		repository.NewRefreshTokenRepository(d.DB),
		d.Tx,
		d.Audit,
	)
	admin.Bind(staffctrl.NewV1(svc))
}
