package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	workoutctrl "gymmaster-api/internal/controller/workout"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

// RegisterWorkout: thư viện bài tập, giáo án tập luyện & nhật ký tập của hội viên (/v1/admin & /v1/member).
func RegisterWorkout(d *Deps, admin, member *ghttp.RouterGroup) {
	svc := service.NewWorkoutService(repository.NewWorkoutRepository(d.DB), d.Tx, d.Audit)
	if admin != nil {
		admin.Bind(workoutctrl.NewAdminV1(svc))
	}
	if member != nil {
		member.Bind(workoutctrl.NewMemberV1(svc))
	}
}
