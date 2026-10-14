package portal

import (
	"gymmaster-api/api/portal"
	"gymmaster-api/internal/service"
)

// ControllerV1: cổng hội viên (/v1/member/*). Controller mỏng – mọi quy tắc nằm trong PortalService.
type ControllerV1 struct {
	portalService service.PortalService
}

func NewV1(svc service.PortalService) portal.IPortalV1 {
	return &ControllerV1{portalService: svc}
}
