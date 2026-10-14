package room

import (
	"context"

	v1 "gymmaster-api/api/room/v1"
)

func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	return c.roomService.Update(ctx, req.ID, req.RoomInput)
}
