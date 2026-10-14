package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	authctrl "gymmaster-api/internal/controller/auth"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterAuth: login/refresh là public; logout/me/change-password cần đăng nhập.
func RegisterAuth(d *Deps, public, authed *ghttp.RouterGroup) {
	svc := service.NewAuthService(
		repository.NewUserRepository(d.DB),
		repository.NewRefreshTokenRepository(d.DB),
		d.TokenMaker,
		d.Tx,
		d.Config.JWT.AccessTTL,
		d.Config.JWT.RefreshTTL,
	)
	c := authctrl.NewV1(svc)
	public.Bind(c.Login, c.Refresh)
	authed.Bind(c.Logout, c.Me, c.ChangePassword)
}
