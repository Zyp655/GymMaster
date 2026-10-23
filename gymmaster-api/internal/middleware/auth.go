package middleware

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

func (m *middleware) Authentication() ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		raw, ok := BearerToken(r.GetHeader(consts.HeaderAuthorization))
		if !ok {
			r.SetError(rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập"))
			return
		}
		payload, err := m.tokenMaker.VerifyToken(raw)
		if err != nil {
			if errors.Is(err, token.ErrExpiredToken) {
				r.SetError(rescode.New(rescode.TokenExpired, "Phiên đăng nhập đã hết hạn"))
			} else {
				r.SetError(rescode.New(rescode.Unauthorized, "Token không hợp lệ"))
			}
			return
		}
		// Access token (JWT) không thể thu hồi → kiểm tra trạng thái tài khoản để user bị khóa/xóa
		// bị chặn ngay, không phải đợi access token hết hạn.
		if err := checkUserActive(r.Context(), payload.UserID); err != nil {
			r.SetError(err)
			return
		}
		// Phiên (refresh token) đã bị đăng xuất/thu hồi → access token gắn phiên đó hết hiệu lực ngay.
		if err := checkSessionActive(r.Context(), payload.SessionID); err != nil {
			r.SetError(err)
			return
		}
		r.SetCtxVar(consts.CtxAuthKey, payload)
		r.Middleware.Next()
	}
}

// BearerToken tách access token từ header Authorization. Tên scheme không phân biệt hoa thường (RFC 7235):
// "Bearer x", "bearer x", "BEARER  x" đều hợp lệ.
func BearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	scheme, tok, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, strings.TrimSpace(consts.BearerPrefix)) {
		return "", false
	}
	tok = strings.TrimSpace(tok)
	return tok, tok != ""
}

// UserStatusLookup trả trạng thái hiện tại của user (ACTIVE/LOCKED...);
// gorm.ErrRecordNotFound = tài khoản không còn tồn tại (đã xóa mềm).
type UserStatusLookup func(ctx context.Context, userID uint) (string, error)

var userStatusLookup atomic.Pointer[UserStatusLookup]

// SetUserStatusLookup đăng ký hàm tra trạng thái user (module auth gọi lúc khởi động). nil = tắt kiểm tra.
func SetUserStatusLookup(fn UserStatusLookup) {
	if fn == nil {
		userStatusLookup.Store(nil)
		return
	}
	userStatusLookup.Store(&fn)
}

// CheckUserActive dùng cho các handler tự xác thực token ngoài middleware (vd. SSE).
func CheckUserActive(ctx context.Context, userID uint) error {
	return checkUserActive(ctx, userID)
}

// checkUserActive tra trạng thái user theo khóa chính ở MỖI request (truy vấn rất rẻ) để khóa tài khoản
// có hiệu lực ngay lập tức – không cache vì việc khóa xảy ra ở module khác.
func checkUserActive(ctx context.Context, userID uint) error {
	fnPtr := userStatusLookup.Load()
	if fnPtr == nil {
		return nil
	}
	status, err := (*fnPtr)(ctx, userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			g.Log().Errorf(ctx, "kiểm tra trạng thái tài khoản %d: %v", userID, err)
			return rescode.Wrap(err, rescode.Internal, "Hệ thống đang gặp sự cố, vui lòng thử lại sau")
		}
		status = "DELETED"
	}
	switch status {
	case consts.StatusActive:
		return nil
	case "DELETED":
		return rescode.New(rescode.Unauthorized, "Tài khoản không còn tồn tại, vui lòng đăng nhập lại")
	default:
		return rescode.New(rescode.Unauthorized, "Tài khoản đã bị khóa, vui lòng liên hệ quản trị viên")
	}
}

// SessionRevokedLookup trả true nếu phiên (refresh token, claim "sid") đã bị thu hồi chủ động: đăng xuất,
// xóa phiên, đăng xuất thiết bị khác, đổi/đặt lại mật khẩu, khóa tài khoản. Phiên chỉ được xoay vòng khi
// refresh KHÔNG tính là thu hồi (access token cũ còn hiệu lực tới hạn để tránh race giữa các request song song).
type SessionRevokedLookup func(ctx context.Context, sessionID uint) (bool, error)

var sessionRevokedLookup atomic.Pointer[SessionRevokedLookup]

// SetSessionRevokedLookup đăng ký hàm tra phiên đã thu hồi (module auth gọi lúc khởi động). nil = tắt kiểm tra.
func SetSessionRevokedLookup(fn SessionRevokedLookup) {
	if fn == nil {
		sessionRevokedLookup.Store(nil)
		return
	}
	sessionRevokedLookup.Store(&fn)
}

// CheckSessionActive dùng cho các handler tự xác thực token ngoài middleware (vd. SSE).
func CheckSessionActive(ctx context.Context, sessionID uint) error {
	return checkSessionActive(ctx, sessionID)
}

// checkSessionActive: token không có sid (cấp trước khi có claim này) thì bỏ qua; lỗi DB không chặn request
// (đã có kiểm tra trạng thái tài khoản), chỉ ghi log.
func checkSessionActive(ctx context.Context, sessionID uint) error {
	fnPtr := sessionRevokedLookup.Load()
	if fnPtr == nil || sessionID == 0 {
		return nil
	}
	revoked, err := (*fnPtr)(ctx, sessionID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			g.Log().Errorf(ctx, "kiểm tra phiên %d: %v", sessionID, err)
			return nil
		}
		revoked = true
	}
	if revoked {
		return rescode.New(rescode.Unauthorized, "Phiên đăng nhập đã bị đăng xuất, vui lòng đăng nhập lại")
	}
	return nil
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

		auth := reqctx.Auth(r.Context())
		if auth == nil {
			r.SetError(rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập"))
			return
		}
		denied := rescode.New(rescode.PermissionDenied, "Bạn không có quyền thực hiện thao tác này")
		// Nhóm /v1/admin/* chỉ dành cho nhân sự (có ít nhất một role khác MEMBER).
		if strings.HasPrefix(r.URL.Path, "/v1/admin/") && !hasStaffRole(auth) {
			r.SetError(denied)
			return
		}
		if requiredPerm == "" && requiredRole == "" {
			r.Middleware.Next()
			return
		}
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

// hasStaffRole trả true nếu người dùng có ít nhất một vai trò quản trị/nhân sự (khác MEMBER).
func hasStaffRole(auth *token.Payload) bool {
	if auth == nil {
		return false
	}
	for _, role := range auth.Roles {
		r := strings.TrimSpace(role)
		if r != "" && r != consts.RoleMember {
			return true
		}
	}
	return false
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
