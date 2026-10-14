package auth

import (
	"context"

	v1 "gymmaster-api/api/auth/v1"
)

func (c *ControllerV1) ChangePassword(ctx context.Context, req *v1.ChangePasswordReq) (res *v1.ChangePasswordRes, err error) {
	return nil, c.authService.ChangePassword(ctx, req.OldPassword, req.NewPassword)
}
