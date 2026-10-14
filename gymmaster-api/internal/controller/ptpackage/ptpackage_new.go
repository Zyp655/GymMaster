package ptpackage

import (
	"gymmaster-api/api/ptpackage"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.PTPackageService
}

func NewV1(svc service.PTPackageService) ptpackage.IPTPackageV1 {
	return &ControllerV1{svc: svc}
}
