package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

const (
	// corsAllowHeaders: header chuẩn + header tùy biến mà admin/web gửi lên
	// (X-Branch-Id: chi nhánh đang chọn; X-Device-Key: khóa thiết bị kiosk check-in).
	corsAllowHeaders = "Origin,Content-Type,Accept,Accept-Language,Authorization,X-Requested-With,X-Branch-Id,X-Device-Key"
	// corsExposeHeaders: header client cần đọc được (tên file khi tải báo cáo/xuất Excel).
	corsExposeHeaders = "Content-Disposition"
	corsAllowMethods  = "GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD"
	corsMaxAge        = "86400"
)

// CORS chỉ trả Access-Control-Allow-Origin (kèm Allow-Credentials) khi request có header Origin
// thuộc whitelist CORS_ORIGINS. Không bao giờ trả "*" kèm Credentials=true, không suy origin từ Referer
// (khác hành vi mặc định của GoFrame) để mọi method (GET/HEAD/OPTIONS...) nhất quán.
func (m *middleware) CORS() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		h := r.Response.Header()
		h.Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if origin != "" && m.originAllowed(origin) {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", corsAllowMethods)
			h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
			h.Set("Access-Control-Expose-Headers", corsExposeHeaders)
			h.Set("Access-Control-Max-Age", corsMaxAge)
		}
		// Preflight: trả ngay, không đi tiếp vào handler.
		if r.Method == http.MethodOptions {
			r.Response.WriteHeader(http.StatusNoContent)
			r.ExitAll()
			return
		}
		r.Middleware.Next()
	}
}

// originAllowed so khớp CHÍNH XÁC Origin với whitelist (không còn chấp nhận subdomain hay cổng khác):
//   - mục có scheme ("https://admin.gymmaster.vn", "http://localhost:3001") → so khớp scheme + host + cổng;
//   - mục chỉ có host[:port] ("localhost:3000") → so khớp host + cổng, scheme http/https.
func (m *middleware) originAllowed(origin string) bool {
	return OriginAllowed(origin, m.corsOrigins)
}

// OriginAllowed: xem originAllowed.
func OriginAllowed(origin string, whitelist []string) bool {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	full := strings.ToLower(u.Scheme) + "://" + host
	for _, d := range whitelist {
		d = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(d), "/"))
		if d == "" {
			continue
		}
		if strings.Contains(d, "://") {
			if d == full {
				return true
			}
		} else if d == host {
			return true
		}
	}
	return false
}

// ParseCORSOrigins tách danh sách origin (giữ nguyên scheme để so khớp chính xác) từ chuỗi CORS_ORIGINS.
func ParseCORSOrigins(raw string) []string {
	var out []string
	for p := range strings.SplitSeq(raw, ",") {
		if p = strings.TrimSuffix(strings.TrimSpace(p), "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// SecurityHeaders gắn các header bảo mật cơ bản cho mọi response API.
// Swagger UI (/swagger) cần nhúng script/CSS từ CDN nên không đặt CSP chặt; chỉ chặn bị nhúng iframe.
func (m *middleware) SecurityHeaders() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		h := r.Response.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if !strings.HasPrefix(r.URL.Path, "/swagger") {
			h.Set("X-Frame-Options", "DENY")
			h.Set("Content-Security-Policy", "frame-ancestors 'none'")
		}
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
