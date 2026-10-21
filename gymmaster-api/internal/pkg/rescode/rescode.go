// Package rescode định nghĩa mã lỗi nghiệp vụ.
//
// Quy ước: mã 5 chữ số; trường `code` trong body JSON luôn là mã nghiệp vụ này (frontend dựa vào nó).
// HTTP status được suy ra bằng HTTPStatus (xem bảng bên dưới) – luôn là status chuẩn của HTTP.
// Dải mã theo module: 400xx-409xx dùng chung, 41xxx subscription, 42xxx check-in,
// 43xxx lịch PT/booking, 44xxx thanh toán, 45xxx POS/tồn kho, 46xxx thiết bị & tủ đồ, 47xxx lương & ca thu ngân.
//
// Bảng ánh xạ mã nghiệp vụ → HTTP status:
//
//	400xx              → 400 Bad Request
//	401xx              → 401 Unauthorized
//	403xx              → 403 Forbidden
//	404xx              → 404 Not Found
//	409xx              → 409 Conflict
//	429xx              → 429 Too Many Requests
//	41xxx subscription → 409 Conflict             (trạng thái gói không cho phép thao tác)
//	42xxx check-in     → 403 Forbidden            (bị từ chối vào cửa: chưa ký HĐ, ngoài giờ, nợ trả góp, nghỉ lễ)
//	43xxx lịch PT      → 409 Conflict             (ngoài lịch rảnh HLV, HLV nghỉ phép)
//	44xxx thanh toán   → 400 Bad Request          (sai chữ ký, lệch số tiền, giao dịch hết hạn)
//	45xxx POS/tồn kho  → 409 Conflict
//	46xxx thiết bị/tủ  → 409 Conflict             (thiết bị offline, tủ đã được thuê)
//	47xxx lương/ca     → 47001 → 400 (kỳ lương không hợp lệ); 47002/47003 → 409 (trạng thái ca thu ngân)
//	4xxxx khác         → 400 Bad Request
//	5xxxx              → 500 Internal Server Error
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

	// 43xxx: Lịch rảnh HLV / ngoài khung giờ nhận PT (Sprint 25)
	PTOutsideAvailability = gcode.New(43005, "PT_OUTSIDE_AVAILABILITY", nil)
	TrainerOnLeave        = gcode.New(43006, "TRAINER_ON_LEAVE", nil)

	// 46xxx: Thiết bị ra vào & Tủ đồ (Sprint 18–19)
	DeviceOfflineError  = gcode.New(46001, "DEVICE_OFFLINE", nil)
	LockerAlreadyRented = gcode.New(46002, "LOCKER_ALREADY_RENTED", nil)

	// 47xxx: Lương, chi phí & đối soát ca (Sprint 20–21)
	PayrollPeriodInvalid   = gcode.New(47001, "PAYROLL_PERIOD_INVALID", nil)
	CashSessionAlreadyOpen = gcode.New(47002, "CASH_SESSION_ALREADY_OPEN", nil)
	CashSessionClosed      = gcode.New(47003, "CASH_SESSION_CLOSED", nil)

	Internal = gcode.New(50000, "INTERNAL", nil)
)

// HTTPStatus trả HTTP status chuẩn cho mã nghiệp vụ 5 chữ số (xem bảng ở đầu package).
// ok=false nếu code không phải mã nghiệp vụ 5 chữ số (10000–59999).
func HTTPStatus(code int) (status int, ok bool) {
	if code < 10000 || code > 59999 {
		return 0, false
	}
	if code >= 50000 {
		return 500, true
	}
	switch code / 100 {
	case 400:
		return 400, true
	case 401:
		return 401, true
	case 403:
		return 403, true
	case 404:
		return 404, true
	case 409:
		return 409, true
	case 429:
		return 429, true
	}
	switch code / 1000 {
	case 41, 43, 45, 46:
		return 409, true
	case 42:
		return 403, true
	case 44:
		return 400, true
	case 47:
		if code == PayrollPeriodInvalid.Code() {
			return 400, true
		}
		return 409, true
	}
	return 400, true
}

// New tạo lỗi có mã nghiệp vụ và thông điệp hiển thị cho người dùng.
func New(code gcode.Code, msg string) error {
	return gerror.NewCode(code, msg)
}

// Wrap gắn mã nghiệp vụ vào lỗi gốc (giữ stack để log), thông điệp hiển thị là msg.
func Wrap(err error, code gcode.Code, msg string) error {
	return gerror.WrapCode(code, err, msg)
}
