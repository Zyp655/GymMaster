package auditlog

import (
	"context"

	v1 "gymmaster-api/api/auditlog/v1"
)

func (c *ControllerV1) Options(ctx context.Context, req *v1.OptionsReq) (res *v1.OptionsRes, err error) {
	return c.auditLogService.Options(ctx)
}
