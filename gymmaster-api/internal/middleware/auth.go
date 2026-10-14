package middleware

import (
	"errors"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

func (m *middleware) Authentication() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		header := r.GetHeader(consts.HeaderAuthorization)
		if !strings.HasPrefix(header, consts.BearerPrefix) {
			r.SetError(rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập"))
			return
		}
		payload, err := m.tokenMaker.VerifyToken(strings.TrimPrefix(header, consts.BearerPrefix))
		if err != nil {
			if errors.Is(err, token.ErrExpiredToken) {
				r.SetError(rescode.New(rescode.TokenExpired, "Phiên đăng nhập đã hết hạn"))
			} else {
				r.SetError(rescode.New(rescode.Unauthorized, "Token không hợp lệ"))
			}
			return
		}
		r.SetCtxVar(consts.CtxAuthKey, payload)
		r.Middleware.Next()
	}
}

// Authorization đọc yêu cầu truy cập từ g.Meta của Req đang được phục vụ:
//   - perm:"a.view,b.view" – cần ít nhất một quyền; OWNER luôn được phép.
//   - role:"MEMBER"        – cần đúng role (không có ngoại lệ OWNER), dùng cho cổng hội viên.
//
// Quyền khai báo ngay tại định nghĩa API (api/<module>/v1) nên router không phải liệt kê role cứng.
func (m *middleware) Authorization() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		var requiredPerm, requiredRole string
		if h := r.GetServeHandler(); h != nil {
			requiredPerm = h.GetMetaTag(consts.MetaTagPermission)
			requiredRole = h.GetMetaTag(consts.MetaTagRole)
		}
		if requiredPerm == "" && requiredRole == "" {
			r.Middleware.Next()
			return
		}

		auth := reqctx.Auth(r.Context())
		if auth == nil {
			r.SetError(rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập"))
			return
		}
		denied := rescode.New(rescode.PermissionDenied, "Bạn không có quyền thực hiện thao tác này")
		if requiredRole != "" && !anyOf(requiredRole, auth.HasRole) {
			r.SetError(denied)
			return
		}
		if requiredPerm != "" && !auth.HasRole(consts.RoleOwner) && !anyOf(requiredPerm, auth.HasPermission) {
			r.SetError(denied)
			return
		}
		r.Middleware.Next()
	}
}

// anyOf trả true nếu check() đúng với ít nhất một giá trị trong danh sách ngăn bởi dấu phẩy.
func anyOf(csv string, check func(string) bool) bool {
	for v := range strings.SplitSeq(csv, ",") {
		if check(strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}
