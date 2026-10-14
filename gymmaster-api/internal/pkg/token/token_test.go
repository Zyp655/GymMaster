package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker(t *testing.T) {
	_, err := NewJWTMaker("short-secret", "gymmaster", time.Minute)
	assert.Error(t, err, "secret ngắn hơn 32 ký tự phải báo lỗi")

	secret := "0123456789abcdef0123456789abcdef"
	maker, err := NewJWTMaker(secret, "gymmaster", time.Minute)
	require.NoError(t, err)

	tok, payload, err := maker.CreateToken(42, []string{"MANAGER"}, []string{"member.view", "pos.sell"}, []uint{1, 2})
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
	assert.True(t, payload.HasRole("MANAGER"))
	assert.False(t, payload.HasRole("OWNER"))
	assert.True(t, payload.HasPermission("pos.sell"))

	verified, err := maker.VerifyToken(tok)
	require.NoError(t, err)
	assert.Equal(t, uint(42), verified.UserID)
	assert.Equal(t, []uint{1, 2}, verified.BranchIDs)

	// Sai issuer hoặc sai chữ ký -> ErrInvalidToken
	otherMaker, err := NewJWTMaker(secret, "other-issuer", time.Minute)
	require.NoError(t, err)
	_, err = otherMaker.VerifyToken(tok)
	assert.ErrorIs(t, err, ErrInvalidToken)

	// Token hết hạn -> ErrExpiredToken
	expiredMaker, err := NewJWTMaker(secret, "gymmaster", -time.Second)
	require.NoError(t, err)
	expTok, _, err := expiredMaker.CreateToken(1, nil, nil, nil)
	require.NoError(t, err)
	_, err = maker.VerifyToken(expTok)
	assert.ErrorIs(t, err, ErrExpiredToken)
}
