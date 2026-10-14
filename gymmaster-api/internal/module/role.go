package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	rolectrl "gymmaster-api/internal/controller/role"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterRole(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewRoleService(repository.NewRoleRepository(d.DB), d.Tx)
	admin.Bind(rolectrl.NewV1(svc))
}
