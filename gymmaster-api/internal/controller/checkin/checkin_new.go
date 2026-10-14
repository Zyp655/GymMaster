package checkin

import (
	"gymmaster-api/api/checkin"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	checkinService service.CheckinService
}

func NewV1(checkinService service.CheckinService) checkin.ICheckinV1 {
	return &ControllerV1{checkinService: checkinService}
}
