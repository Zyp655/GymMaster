package auth

import (
	"context"

	v1 "gymmaster-api/api/auth/v1"
)

func (c *ControllerV1) Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error) {
	return nil, c.authService.Logout(ctx, req.RefreshToken)
}
