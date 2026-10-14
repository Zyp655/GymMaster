package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndRandomToken(t *testing.T) {
	hashed, err := HashPassword("GymMaster@2026")
	require.NoError(t, err)
	assert.True(t, CheckPassword(hashed, "GymMaster@2026"))
	assert.False(t, CheckPassword(hashed, "wrong-password"))

	tok1, err := RandomToken(32)
	require.NoError(t, err)
	tok2, err := RandomToken(32)
	require.NoError(t, err)
	assert.Len(t, tok1, 64)
	assert.NotEqual(t, tok1, tok2)

	h1 := SHA256(tok1)
	assert.Len(t, h1, 64)
	assert.Equal(t, h1, SHA256(tok1))
	assert.NotEqual(t, h1, SHA256(tok2))
}
