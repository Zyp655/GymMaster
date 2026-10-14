package setting

import (
	"gymmaster-api/api/setting"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.SettingAdminService
}

func NewV1(svc service.SettingAdminService) setting.ISettingV1 {
	return &ControllerV1{svc: svc}
}
