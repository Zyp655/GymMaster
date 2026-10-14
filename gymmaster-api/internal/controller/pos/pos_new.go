package pos

import (
	"gymmaster-api/api/pos"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	posService service.PosService
}

func NewV1(posService service.PosService) pos.IPosV1 {
	return &ControllerV1{posService: posService}
}
