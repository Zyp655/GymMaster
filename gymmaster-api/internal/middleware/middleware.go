package middleware

import (
	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/pkg/token"
)

type Middleware interface {
	CORS() ghttp.HandlerFunc
	Response() ghttp.HandlerFunc
	RequestLog() ghttp.HandlerFunc
	// Authentication bắt buộc Bearer access token hợp lệ.
	Authentication() ghttp.HandlerFunc
	// Authorization kiểm tra quyền khai báo ở tag `perm` trong g.Meta của Req.
	Authorization() ghttp.HandlerFunc
}

type middleware struct {
	tokenMaker  token.Maker
	corsOrigins []string
}

func New(tokenMaker token.Maker, corsOrigins []string) Middleware {
	return &middleware{tokenMaker: tokenMaker, corsOrigins: corsOrigins}
}
