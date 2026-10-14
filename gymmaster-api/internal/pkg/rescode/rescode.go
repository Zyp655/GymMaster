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
	ResetTokenInvalid   = gcode.New(40104, "RESET_TOKEN_INVALID", nil)
	TooManyAttempts     = gcode.New(40105, "TOO_MANY_ATTEMPTS", nil)
	PermissionDenied    = gcode.New(40300, "PERMISSION_DENIED", nil)
	AccountLocked       = gcode.New(40301, "ACCOUNT_LOCKED", nil)
	NotFound            = gcode.New(40400, "NOT_FOUND", nil)
	Conflict            = gcode.New(40900, "CONFLICT", nil)
	TooManyRequests     = gcode.New(42900, "TOO_MANY_REQUESTS", nil)

	PaymentInvalidSignature = gcode.New(44001, "PAYMENT_INVALID_SIGNATURE", nil)
	PaymentAmountMismatch   = gcode.New(44002, "PAYMENT_AMOUNT_MISMATCH", nil)
	PaymentTxnExpired       = gcode.New(44003, "PAYMENT_TXN_EXPIRED", nil)

	DeviceUnauthorized  = gcode.New(40106, "DEVICE_UNAUTHORIZED", nil)
	MFARequired         = gcode.New(40107, "MFA_REQUIRED", nil)
	MFAInvalidCode      = gcode.New(40108, "MFA_INVALID_CODE", nil)
	LockerUnavailable   = gcode.New(40901, "LOCKER_UNAVAILABLE", nil)
	PayrollLocked       = gcode.New(40902, "PAYROLL_LOCKED", nil)
	InsufficientPoints  = gcode.New(40903, "INSUFFICIENT_POINTS", nil)
	RewardOutOfStock    = gcode.New(40904, "REWARD_OUT_OF_STOCK", nil)
	ContractUnsigned    = gcode.New(42001, "CONTRACT_UNSIGNED", nil)
	OutsideAccessHours  = gcode.New(42002, "OUTSIDE_ACCESS_HOURS", nil)
	InstallmentOverdue  = gcode.New(42003, "INSTALLMENT_OVERDUE", nil)
	BranchHolidayClosed = gcode.New(42004, "BRANCH_HOLIDAY_CLOSED", nil)

	Internal = gcode.New(50000, "INTERNAL", nil)
)

// New tạo lỗi có mã nghiệp vụ và thông điệp hiển thị cho người dùng.
func New(code gcode.Code, msg string) error {
	return gerror.NewCode(code, msg)
}

// Wrap gắn mã nghiệp vụ vào lỗi gốc (giữ stack để log), thông điệp hiển thị là msg.
func Wrap(err error, code gcode.Code, msg string) error {
	return gerror.WrapCode(code, err, msg)
}
