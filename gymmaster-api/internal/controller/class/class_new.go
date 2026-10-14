package class

import (
	"gymmaster-api/api/class"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	typeService     service.ClassTypeService
	scheduleService service.ClassScheduleService
	bookingService  service.BookingService
}

func NewV1(typeService service.ClassTypeService, scheduleService service.ClassScheduleService,
	bookingService service.BookingService) class.IClassV1 {
	return &ControllerV1{typeService: typeService, scheduleService: scheduleService, bookingService: bookingService}
}
