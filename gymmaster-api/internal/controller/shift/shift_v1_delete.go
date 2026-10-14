package shift

import (
	"context"

	v1 "gymmaster-api/api/shift/v1"
)

func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	return nil, c.shiftService.Delete(ctx, req.ID)
}
