package auditlog

import (
	"gymmaster-api/api/auditlog"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	auditLogService service.AuditLogService
}

func NewV1(svc service.AuditLogService) auditlog.IAuditlogV1 {
	return &ControllerV1{auditLogService: svc}
}
