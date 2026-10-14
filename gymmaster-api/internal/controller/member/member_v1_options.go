package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) Options(ctx context.Context, req *v1.OptionsReq) (res *v1.OptionsRes, err error) {
	items, err := c.memberService.Options(ctx, req.Keyword, req.BranchID)
	if err != nil {
		return nil, err
	}
	return &items, nil
}
