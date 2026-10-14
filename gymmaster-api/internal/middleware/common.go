package middleware

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func (m *middleware) CORS() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		opts := r.Response.DefaultCORSOptions()
		// AllowDomain so khớp theo host (vd "localhost:3000", "gymmaster.vn" – gồm cả subdomain).
		opts.AllowDomain = m.corsOrigins
		opts.AllowHeaders = "Origin,Content-Type,Accept,Authorization,X-Branch-Id"
		r.Response.CORS(opts)
		r.Middleware.Next()
	}
}

func (m *middleware) RequestLog() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		start := time.Now()
		r.Middleware.Next()
		g.Log().Infof(r.Context(), "%s %s %d %s", r.Method, r.URL.Path, r.Response.Status, time.Since(start))
	}
}
