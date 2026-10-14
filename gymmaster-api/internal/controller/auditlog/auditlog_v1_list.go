package auditlog

import (
	"context"

	v1 "gymmaster-api/api/auditlog/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	return c.auditLogService.List(ctx, req)
}
