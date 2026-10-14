package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	contractpassctrl "gymmaster-api/internal/controller/contractpass"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterContractPass: hợp đồng điện tử, vé ngày & vé mời khách (/v1/admin & /v1/member).
func RegisterContractPass(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := service.NewContractPassService(repository.NewContractPassRepository(d.DB), d.Tx, d.Audit)
	if admin != nil {
		admin.Bind(contractpassctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(contractpassctrl.NewMemberV1(svc))
	}
}
