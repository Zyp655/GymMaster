package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	reviewctrl "gymmaster-api/internal/controller/review"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterReview: đánh giá huấn luyện viên & lớp học (/v1/admin & /v1/member).
func RegisterReview(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := service.NewReviewService(repository.NewReviewRepository(d.DB), d.Tx, d.Audit)
	if admin != nil {
		admin.Bind(reviewctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(reviewctrl.NewMemberV1(svc))
	}
}
