package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	roomctrl "gymmaster-api/internal/controller/room"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/service"
)

func RegisterRoom(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewRoomService(repository.NewRoomRepository(d.DB))
	admin.Bind(roomctrl.NewV1(svc))
}
