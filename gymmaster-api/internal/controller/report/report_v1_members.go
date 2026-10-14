package report

import (
	"context"

	v1 "gymmaster-api/api/report/v1"
)

func (c *ControllerV1) Members(ctx context.Context, req *v1.MembersReq) (res *v1.MembersRes, err error) {
	return c.reportService.Members(ctx, req)
}
