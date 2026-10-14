package qrtoken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateVerify(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok, exp := Generate(42, "secret-a", now, time.Minute)
	assert.Equal(t, now.Add(time.Minute), exp)

	id, err := ParseMemberID(tok)
	require.NoError(t, err)
	assert.Equal(t, uint(42), id)

	assert.NoError(t, Verify(tok, "secret-a", now.Add(30*time.Second)))
	assert.ErrorIs(t, Verify(tok, "secret-a", now.Add(61*time.Second)), ErrExpired)
	assert.ErrorIs(t, Verify(tok, "secret-b", now), ErrSignature, "sai bí mật")

	// Sửa memberID trong mã → chữ ký không còn khớp.
	forged := "GM1.43" + tok[len("GM1.42"):]
	assert.ErrorIs(t, Verify(forged, "secret-a", now), ErrSignature)

	_, err = ParseMemberID("abc")
	assert.ErrorIs(t, err, ErrMalformed)
}
