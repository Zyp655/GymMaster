package auth

import (
	"gymmaster-api/api/auth"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	authService service.AuthService
}

func NewV1(authService service.AuthService) auth.IAuthV1 {
	return &ControllerV1{authService: authService}
}
