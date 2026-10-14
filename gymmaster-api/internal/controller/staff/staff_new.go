package staff

import (
	"gymmaster-api/api/staff"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	staffService service.StaffService
}

func NewV1(staffService service.StaffService) staff.IStaffV1 {
	return &ControllerV1{staffService: staffService}
}
