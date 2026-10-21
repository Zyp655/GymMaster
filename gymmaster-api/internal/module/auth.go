package module

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"

	authctrl "gymmaster-api/internal/controller/auth"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/middleware"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterAuth: login/refresh/forgot-password/reset-password/register-member/verify-member-otp là public;
// logout cũng public (thu hồi bằng chính refresh token, không cần access token còn hạn);
// me/change-password cần đăng nhập.
func RegisterAuth(d *Deps, public, authed *ghttp.RouterGroup) {
	// Middleware Authentication tra trạng thái user (LOCKED/đã xóa → 401 ngay, không đợi JWT hết hạn).
	middleware.SetUserStatusLookup(func(ctx context.Context, userID uint) (string, error) {
		var u entity.User
		err := d.DB.WithContext(ctx).Select("status").Where("id = ?", userID).Take(&u).Error
		return u.Status, err
	})

	// 2FA khi đăng nhập dùng chung khóa mã hóa với module privacy (JWT secret).
	encKey := ""
	if d.Config != nil {
		encKey = d.Config.JWT.Secret
	}
	mfa := service.NewPrivacyService(repository.NewPrivacyRepository(d.DB), d.Tx, d.Audit, encKey)

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
			MFA:            mfa,
			Audit:          d.Audit,
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
		c.Logout,
	)
	authed.Bind(c.Me, c.ChangePassword)
}
