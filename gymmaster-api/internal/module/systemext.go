package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/controller/systemext"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterSystemExt(d *Deps, admin *ghttp.RouterGroup) {
	repo := repository.NewSystemExtRepository(d.DB)
	svc := service.NewSystemExtService(repo, d.Notifier)
	ctrl := systemext.New(svc)
	admin.Bind(ctrl)
}
