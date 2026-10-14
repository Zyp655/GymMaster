package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	engagementctrl "gymmaster-api/internal/controller/engagement"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterEngagement: dinh dưỡng, thực đơn, bảng tin & khảo sát NPS (/v1/admin & /v1/member).
func RegisterEngagement(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := service.NewEngagementService(repository.NewEngagementRepository(d.DB), d.Tx, d.Audit)
	if admin != nil {
		admin.Bind(engagementctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(engagementctrl.NewMemberV1(svc))
	}
}
