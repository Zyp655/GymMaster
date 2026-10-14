package member

import (
	"context"

	v1 "gymmaster-api/api/member/v1"
)

func (c *ControllerV1) Import(ctx context.Context, req *v1.ImportReq) (res *v1.ImportRes, err error) {
	f, err := req.File.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return c.memberService.Import(ctx, req.BranchID, f)
}
