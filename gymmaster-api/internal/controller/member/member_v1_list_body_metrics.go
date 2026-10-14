package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) ListBodyMetrics(ctx context.Context, req *v1.ListBodyMetricsReq) (res *v1.ListBodyMetricsRes, err error) {
	items, err := c.memberService.ListBodyMetrics(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &items, nil
}
