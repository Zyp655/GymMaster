package auth

import (
	"context"

	v1 "gymmaster-api/api/auth/v1"
)

func (c *ControllerV1) Refresh(ctx context.Context, req *v1.RefreshReq) (res *v1.RefreshRes, err error) {
	return c.authService.Refresh(ctx, req.RefreshToken, clientInfo(ctx))
}
