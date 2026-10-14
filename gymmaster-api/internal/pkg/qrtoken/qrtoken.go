// Package qrtoken sinh và kiểm tra mã QR check-in động.
//
// Định dạng: GM1.<memberID>.<expUnix>.<sig>
//
//	sig = base64url(HMAC-SHA256(qr_secret_của_hội_viên, "<memberID>.<expUnix>"))[:22]
//
// Mã chỉ sống TTL giây (mặc định 60s, cấu hình `checkin.qr_ttl_seconds`) nên chụp màn hình
// gửi người khác cũng hết hiệu lực ngay. Bí mật nằm ở từng hội viên (members.qr_secret):
// đổi bí mật = vô hiệu mọi mã cũ của hội viên đó mà không ảnh hưởng người khác.
package qrtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const prefix = "GM1"

var (
	ErrMalformed = errors.New("mã QR không đúng định dạng")
	ErrExpired   = errors.New("mã QR đã hết hạn")
	ErrSignature = errors.New("mã QR không hợp lệ")
)

// Generate tạo mã QR cho hội viên, hết hạn sau ttl kể từ now.
func Generate(memberID uint, secret string, now time.Time, ttl time.Duration) (token string, expiresAt time.Time) {
	expiresAt = now.Add(ttl).Truncate(time.Second)
	payload := fmt.Sprintf("%d.%d", memberID, expiresAt.Unix())
	return prefix + "." + payload + "." + sign(secret, payload), expiresAt
}

// ParseMemberID đọc memberID (chưa xác thực) để tra qr_secret của hội viên.
func ParseMemberID(token string) (uint, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 4 || parts[0] != prefix {
		return 0, ErrMalformed
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || id == 0 {
		return 0, ErrMalformed
	}
	return uint(id), nil
}

// Verify kiểm tra chữ ký và hạn của mã với bí mật của hội viên.
func Verify(token, secret string, now time.Time) error {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 4 || parts[0] != prefix {
		return ErrMalformed
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return ErrMalformed
	}
	expected := sign(secret, parts[1]+"."+parts[2])
	if !hmac.Equal([]byte(expected), []byte(parts[3])) {
		return ErrSignature
	}
	if now.Unix() > exp {
		return ErrExpired
	}
	return nil
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:22]
}
