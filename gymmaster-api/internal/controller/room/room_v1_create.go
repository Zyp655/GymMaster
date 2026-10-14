package room

import (
	"context"

	v1 "gymmaster-api/api/room/v1"
)

func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	return c.roomService.Create(ctx, req.RoomInput)
}
