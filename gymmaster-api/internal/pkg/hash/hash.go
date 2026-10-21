package hash

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"

	"gymmaster-api/internal/pkg/rescode"
)

// MaxPasswordBytes: bcrypt chỉ xử lý tối đa 72 byte (tiếng Việt có dấu chiếm 2–3 byte/ký tự).
const MaxPasswordBytes = 72

// ErrPasswordTooLong là lỗi nghiệp vụ 400 (không phải 500) khi mật khẩu vượt quá giới hạn của bcrypt.
// Mang sẵn mã rescode.BadRequest nên mọi service gọi HashPassword trả lỗi trực tiếp là client nhận 400.
var ErrPasswordTooLong = rescode.New(rescode.BadRequest,
	"Mật khẩu quá dài (tối đa 72 byte, ký tự có dấu tính 2–3 byte), vui lòng chọn mật khẩu ngắn hơn")

// ValidatePasswordLength trả ErrPasswordTooLong nếu mật khẩu vượt quá 72 byte.
func ValidatePasswordLength(password string) error {
	if len(password) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}

// HashPassword băm mật khẩu bằng bcrypt. Mật khẩu > 72 byte trả ErrPasswordTooLong (400).
func HashPassword(password string) (string, error) {
	if err := ValidatePasswordLength(password); err != nil {
		return "", err
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword so khớp mật khẩu với chuỗi bcrypt đã lưu.
func CheckPassword(hashed, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
}

// RandomToken sinh chuỗi ngẫu nhiên an toàn (hex), dùng cho refresh token.
func RandomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SHA256 dùng để lưu refresh token dạng băm trong DB (lộ DB cũng không dùng lại được token).
func SHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
