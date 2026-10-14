package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	authctrl "gymmaster-api/internal/controller/auth"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterAuth: login/refresh/forgot-password/reset-password/register-member/verify-member-otp là public;
// logout/me/change-password cần đăng nhập.
func RegisterAuth(d *Deps, public, authed *ghttp.RouterGroup) {
	svc := service.NewAuthServiceWithSecurity(
		repository.NewUserRepository(d.DB),
		repository.NewRefreshTokenRepository(d.DB),
		d.TokenMaker,
		d.Tx,
		d.Config.JWT.AccessTTL,
		d.Config.JWT.RefreshTTL,
		service.AuthSecurityDeps{
			PasswordResets: repository.NewPasswordResetRepository(d.DB),
			Limiter:        repository.NewLoginLimiter(d.DB),
			MemberRegister: repository.NewMemberRegisterRepository(d.DB),
			Mailer:         NewMailer(d),
			WebURL:         d.Config.WebURL,
			AdminURL:       d.Config.AdminURL,
		},
	)
	c := authctrl.NewV1(svc)
	public.Bind(
		c.Login,
		c.Refresh,
		c.ForgotPassword,
		c.ResetPassword,
		c.RegisterMember,
		c.VerifyMemberOTP,
	)
	authed.Bind(c.Logout, c.Me, c.ChangePassword)
}
