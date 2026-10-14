package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	memberctrl "gymmaster-api/internal/controller/member"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterMember(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewMemberService(
		repository.NewMemberRepository(d.DB),
		repository.NewBodyMetricRepository(d.DB),
		repository.NewUserRepository(d.DB),
		d.Tx,
		d.Audit,
	)
	admin.Bind(memberctrl.NewV1(svc))
}
