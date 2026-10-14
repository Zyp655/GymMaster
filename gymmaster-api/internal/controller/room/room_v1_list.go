package room

import (
	"context"

	v1 "gymmaster-api/api/room/v1"
)

func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	return c.roomService.List(ctx, req)
}
