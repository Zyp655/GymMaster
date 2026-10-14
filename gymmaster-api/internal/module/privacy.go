package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	privacyctrl "gymmaster-api/internal/controller/privacy"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterPrivacy: xác thực 2 lớp (2FA TOTP), quản lý phiên & tuân thủ Nghị định 13/2023 (/v1/admin & /v1/member).
func RegisterPrivacy(d *Deps, admin, member *ghttp.RouterGroup) {
	encKey := ""
	if d.Config != nil {
		encKey = d.Config.JWT.Secret
	}
	svc := service.NewPrivacyService(repository.NewPrivacyRepository(d.DB), d.Tx, d.Audit, encKey)
	if admin != nil {
		admin.Bind(privacyctrl.NewSecurityV1(svc), privacyctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(privacyctrl.NewSecurityV1(svc), privacyctrl.NewMemberV1(svc))
	}
}
