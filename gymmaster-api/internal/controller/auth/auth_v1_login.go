package auth

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "gymmaster-api/api/auth/v1"
	"gymmaster-api/internal/pkg/clientip"
	"gymmaster-api/internal/service"
)

func (c *ControllerV1) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	return c.authService.Login(ctx, req.Email, req.Password, req.MFACode, clientInfo(ctx))
}

// clientInfo lấy User-Agent/IP ở tầng controller để service không phụ thuộc ghttp.
func clientInfo(ctx context.Context) service.ClientInfo {
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return service.ClientInfo{}
	}
	return service.ClientInfo{UserAgent: r.UserAgent(), IP: clientip.Real(r)}
}
