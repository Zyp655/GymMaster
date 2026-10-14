package role

import (
	"gymmaster-api/api/role"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	roleService service.RoleService
}

func NewV1(roleService service.RoleService) role.IRoleV1 {
	return &ControllerV1{roleService: roleService}
}
