package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/controller/cms"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterCMS(d *Deps, admin, member, public *ghttp.RouterGroup) {
	repo := repository.NewCMSRepository(d.DB)
	svc := service.NewCMSService(repo, d.Notifier)
	pubCtrl := cms.NewPublic(svc)
	adminCtrl := cms.New(svc)
	sharedCtrl := cms.NewSharedPortal(svc)

	public.Bind(pubCtrl)
	public.GET("/calendar.ics", pubCtrl.CalendarICS)
	admin.Bind(adminCtrl, sharedCtrl)
	member.Bind(sharedCtrl)
}
