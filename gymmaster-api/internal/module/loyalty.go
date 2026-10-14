package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	loyaltyctrl "gymmaster-api/internal/controller/loyalty"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterLoyalty: tích điểm thưởng, đổi quà, huy hiệu, thử thách & giới thiệu bạn bè (/v1/admin & /v1/member).
func RegisterLoyalty(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := service.NewLoyaltyService(repository.NewLoyaltyRepository(d.DB), d.Tx, d.Audit)
	if admin != nil {
		admin.Bind(loyaltyctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(loyaltyctrl.NewMemberV1(svc))
	}
}
