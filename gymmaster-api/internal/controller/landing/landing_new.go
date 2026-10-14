package landing

import (
	"gymmaster-api/api/landing"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.LandingService
}

func NewV1(svc service.LandingService) landing.ILandingV1 {
	return &ControllerV1{svc: svc}
}
