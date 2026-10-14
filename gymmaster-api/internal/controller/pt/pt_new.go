package pt

import (
	"gymmaster-api/api/pt"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.PTService
}

func NewV1(svc service.PTService) pt.IPtV1 {
	return &ControllerV1{svc: svc}
}
