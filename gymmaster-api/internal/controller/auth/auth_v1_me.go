package auth

import (
	"context"

	v1 "gymmaster-api/api/auth/v1"
)

func (c *ControllerV1) Me(ctx context.Context, req *v1.MeReq) (res *v1.MeRes, err error) {
	return c.authService.Me(ctx)
}
