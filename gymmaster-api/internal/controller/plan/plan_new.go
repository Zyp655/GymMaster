package plan

import (
	"gymmaster-api/api/plan"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.MembershipPlanService
}

func NewV1(svc service.MembershipPlanService) plan.IPlanV1 {
	return &ControllerV1{svc: svc}
}
