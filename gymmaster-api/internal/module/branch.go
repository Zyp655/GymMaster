package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	branchctrl "gymmaster-api/internal/controller/branch"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterBranch(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewBranchService(repository.NewBranchRepository(d.DB))
	admin.Bind(branchctrl.NewV1(svc))
}
