// Package rescode định nghĩa mã lỗi nghiệp vụ.
//
// Quy ước: mã 5 chữ số, HTTP status = code / 100 (xem middleware.Response).
// Dải mã theo module: 400xx-409xx dùng chung, 41xxx subscription, 42xxx check-in,
// 43xxx booking, 44xxx thanh toán, 45xxx POS/tồn kho.
package rescode

import (
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

var (
	BadRequest          = gcode.New(40000, "BAD_REQUEST", nil)
	Unauthorized        = gcode.New(40100, "UNAUTHORIZED", nil)
	TokenExpired        = gcode.New(40101, "TOKEN_EXPIRED", nil)
	InvalidCredentials  = gcode.New(40102, "INVALID_CREDENTIALS", nil)
	RefreshTokenInvalid = gcode.New(40103, "REFRESH_TOKEN_INVALID", nil)
	PermissionDenied    = gcode.New(40300, "PERMISSION_DENIED", nil)
	AccountLocked       = gcode.New(40301, "ACCOUNT_LOCKED", nil)
	NotFound            = gcode.New(40400, "NOT_FOUND", nil)
	Conflict            = gcode.New(40900, "CONFLICT", nil)
	Internal            = gcode.New(50000, "INTERNAL", nil)
)

// New tạo lỗi có mã nghiệp vụ và thông điệp hiển thị cho người dùng.
func New(code gcode.Code, msg string) error {
	return gerror.NewCode(code, msg)
}

// Wrap gắn mã nghiệp vụ vào lỗi gốc (giữ stack để log), thông điệp hiển thị là msg.
func Wrap(err error, code gcode.Code, msg string) error {
	return gerror.WrapCode(code, err, msg)
}
