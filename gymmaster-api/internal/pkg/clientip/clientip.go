// Package clientip xác định IP thật của client.
package clientip

import (
	"os"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
)

// Real trả IP thật của kết nối TCP. Header X-Forwarded-For/X-Real-IP do client tự đặt được nên
// CHỈ được tin khi request đi qua proxy tin cậy khai báo trong biến môi trường TRUSTED_PROXIES
// (danh sách IP ngăn bởi dấu phẩy).
func Real(r *ghttp.Request) string {
	remote := r.GetRemoteIp()
	if trusted := trustedProxies(); len(trusted) > 0 && slices.Contains(trusted, remote) {
		if ip := r.GetClientIp(); ip != "" {
			return ip
		}
	}
	return remote
}

func trustedProxies() []string {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return nil
	}
	var out []string
	for p := range strings.SplitSeq(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
