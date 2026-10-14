package cryptoext

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

const (
	totpStepSeconds = 30
	totpIssuer      = "GymMaster"
)

// GenerateTOTPSecret sinh khóa bí mật TOTP ngẫu nhiên 20 bytes mã hóa Base32 (không padding).
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// BuildOTPAuthURL tạo chuỗi URL otpauth://totp chuẩn cho ứng dụng Google Authenticator / Authy.
func BuildOTPAuthURL(email, secret string) string {
	account := strings.TrimSpace(email)
	if account == "" {
		account = "user@gymmaster.vn"
	}
	label := url.PathEscape(fmt.Sprintf("%s:%s", totpIssuer, account))
	q := url.Values{}
	q.Set("secret", strings.ToUpper(strings.TrimSpace(secret)))
	q.Set("issuer", totpIssuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	return fmt.Sprintf("otpauth://totp/%s?%s", label, q.Encode())
}

// GenerateTOTPCode tính mã TOTP 6 chữ số tại thời điểm at theo RFC 6238 (HMAC-SHA1, 30s).
func GenerateTOTPCode(secret string, at time.Time) (string, error) {
	cleaned := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	cleaned = strings.TrimRight(cleaned, "=")
	if cleaned == "" {
		return "", errors.New("secret TOTP trống")
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(cleaned)
	if err != nil {
		return "", err
	}
	counter := uint64(at.Unix() / totpStepSeconds)
	return hotpCode(key, counter), nil
}

// VerifyTOTPCode xác thực mã TOTP 6 số tại thời điểm now với độ lệch cho phép ±1 bước (30s).
func VerifyTOTPCode(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	cleaned := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	cleaned = strings.TrimRight(cleaned, "=")
	if cleaned == "" {
		return false
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(cleaned)
	if err != nil {
		return false
	}
	baseCounter := now.Unix() / totpStepSeconds
	for offset := int64(-1); offset <= 1; offset++ {
		c := uint64(baseCounter + offset)
		expected := hotpCode(key, c)
		if hmac.Equal([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

func hotpCode(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binCode := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]&0xff) << 16) |
		(uint32(sum[offset+2]&0xff) << 8) |
		uint32(sum[offset+3]&0xff)
	otp := binCode % 1000000
	return fmt.Sprintf("%06d", otp)
}

func deriveAESKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// EncryptAESGCM mã hóa chuỗi plainText bằng AES-256-GCM và trả về chuỗi hex (nonce || ciphertext).
func EncryptAESGCM(plainText, key string) (string, error) {
	block, err := aes.NewCipher(deriveAESKey(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return hex.EncodeToString(sealed), nil
}

// DecryptAESGCM giải mã chuỗi hex được tạo bởi EncryptAESGCM.
func DecryptAESGCM(cipherHex, key string) (string, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(cipherHex))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveAESKey(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("dữ liệu mã hóa không hợp lệ")
	}
	nonce, cipherBytes := raw[:nonceSize], raw[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// MaskPhone che số điện thoại (VD: 0912345678 -> 0912***678).
func MaskPhone(phone string) string {
	s := strings.TrimSpace(phone)
	if s == "" {
		return ""
	}
	if len(s) < 7 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-7) + s[len(s)-3:]
}

// MaskIDCard che số CCCD/CMND (VD: 001099001289 -> 0010******89).
func MaskIDCard(idCard string) string {
	s := strings.TrimSpace(idCard)
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-6) + s[len(s)-2:]
}
