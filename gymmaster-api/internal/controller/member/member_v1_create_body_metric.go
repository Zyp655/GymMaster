package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) CreateBodyMetric(ctx context.Context, req *v1.CreateBodyMetricReq) (res *v1.CreateBodyMetricRes, err error) {
	return c.memberService.CreateBodyMetric(ctx, req)
}
