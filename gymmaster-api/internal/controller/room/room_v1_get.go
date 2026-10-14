package room

import (
	"context"

	v1 "gymmaster-api/api/room/v1"
)

func (c *ControllerV1) Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error) {
	return c.roomService.Get(ctx, req.ID)
}
