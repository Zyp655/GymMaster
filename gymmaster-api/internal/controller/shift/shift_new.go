package shift

import (
	"gymmaster-api/api/shift"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	shiftService service.ShiftService
}

func NewV1(shiftService service.ShiftService) shift.IShiftV1 {
	return &ControllerV1{shiftService: shiftService}
}
