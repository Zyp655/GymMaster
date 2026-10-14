package cryptoext

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOTPGenerateAndVerify(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	require.NoError(t, err)
	assert.NotEmpty(t, secret)

	otpURL := BuildOTPAuthURL("admin@gymmaster.vn", secret)
	assert.Contains(t, otpURL, "otpauth://totp/")
	assert.Contains(t, otpURL, secret)

	now := time.Date(2026, 10, 14, 10, 0, 15, 0, time.UTC)
	code, err := GenerateTOTPCode(secret, now)
	require.NoError(t, err)
	assert.Len(t, code, 6)

	// Valid at exact time and ±30s step
	assert.True(t, VerifyTOTPCode(secret, code, now))
	assert.True(t, VerifyTOTPCode(secret, code, now.Add(30*time.Second)))
	assert.True(t, VerifyTOTPCode(secret, code, now.Add(-30*time.Second)))

	// Invalid outside ±1 step window (e.g. +90s)
	assert.False(t, VerifyTOTPCode(secret, code, now.Add(90*time.Second)))
	assert.False(t, VerifyTOTPCode(secret, "000000", now) && code != "000000")
}

func TestAESGCMEncryptDecrypt(t *testing.T) {
	key := "my-secret-key-for-totp"
	plain := "JBSWY3DPEHPK3PXP"

	enc, err := EncryptAESGCM(plain, key)
	require.NoError(t, err)
	assert.NotEmpty(t, enc)
	assert.NotEqual(t, plain, enc)

	dec, err := DecryptAESGCM(enc, key)
	require.NoError(t, err)
	assert.Equal(t, plain, dec)

	_, err = DecryptAESGCM(enc, "wrong-key")
	assert.Error(t, err)
}

func TestMaskPhoneAndIDCard(t *testing.T) {
	assert.Equal(t, "0912***678", MaskPhone("0912345678"))
	assert.Equal(t, "0010******89", MaskIDCard("001099001289"))
	assert.Equal(t, "", MaskPhone(""))
	assert.Equal(t, "", MaskIDCard(""))
}
