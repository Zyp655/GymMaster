package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gymmaster-api/internal/pkg/rescode"
)

// REG-A6: LOGIN_FAILED lý do TOO_MANY_ATTEMPTS phải ghi actor_id khi email tồn tại.
func TestLogin_TooManyAttemptsAuditHasActor(t *testing.T) {
	svc, _, _, _, _, _, _ := newAuthSecurityFixture(t)
	spy := &authAuditSpy{}
	svc.audit = spy
	ctx := context.Background()
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	client := ClientInfo{IP: "10.0.0.9"}
	for i := 0; i < 5; i++ {
		_, _ = svc.Login(ctx, "lt@gm.vn", "wrong-pass", "", client)
	}
	_, err := svc.Login(ctx, "lt@gm.vn", testPassword, "", client)
	assert.Equal(t, rescode.TooManyAttempts.Code(), codeOf(err))

	last := spy.entries[len(spy.entries)-1]
	assert.Equal(t, AuditLoginFailed, last.action)
	require.NotNil(t, last.actor, "actor_id không được NULL khi email tồn tại")
	assert.Equal(t, uint(2), *last.actor)

	// Email không tồn tại bị khóa → vẫn không có actor.
	for i := 0; i < 6; i++ {
		_, _ = svc.Login(ctx, "khong-co@gm.vn", "wrong-pass", "", client)
	}
	last = spy.entries[len(spy.entries)-1]
	assert.Nil(t, last.actor)
}
