package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"gymmaster-api/internal/pkg/rescode"
)

// Vé mời: lỗi unique uq_day_passes_guest_phone_date (đã bọc bởi TranslateError của GORM) → nhận diện để trả 409.
func TestIsGuestPassDupViolation(t *testing.T) {
	dup := &pgconn.PgError{Code: "23505", ConstraintName: guestPassDupConstraint}
	assert.True(t, isGuestPassDupViolation(dup))
	assert.True(t, isGuestPassDupViolation(fmt.Errorf("%w: %w", gorm.ErrDuplicatedKey, dup)))
	assert.False(t, isGuestPassDupViolation(&pgconn.PgError{Code: "23505", ConstraintName: "day_passes_qr_token_key"}))
	assert.False(t, isGuestPassDupViolation(errors.New("x")))
	assert.Equal(t, rescode.Conflict.Code(), codeOf(guestPassDupErr(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))))
}
