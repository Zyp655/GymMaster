package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) DeleteBodyMetric(ctx context.Context, req *v1.DeleteBodyMetricReq) (res *v1.DeleteBodyMetricRes, err error) {
	return nil, c.memberService.DeleteBodyMetric(ctx, req.ID, req.MetricID)
}
