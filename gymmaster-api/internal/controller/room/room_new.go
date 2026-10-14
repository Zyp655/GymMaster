package room

import (
	"gymmaster-api/api/room"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	roomService service.RoomService
}

func NewV1(roomService service.RoomService) room.IRoomV1 {
	return &ControllerV1{roomService: roomService}
}
