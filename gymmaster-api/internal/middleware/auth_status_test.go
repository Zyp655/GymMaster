package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/rescode"
)

func TestCheckUserActive(t *testing.T) {
	t.Cleanup(func() { SetUserStatusLookup(nil) })
	statuses := map[uint]string{1: consts.StatusActive, 2: consts.StatusLocked}
	SetUserStatusLookup(func(_ context.Context, id uint) (string, error) {
		if st, ok := statuses[id]; ok {
			return st, nil
		}
		if id == 99 {
			return "", errors.New("db down")
		}
		return "", gorm.ErrRecordNotFound
	})
	ctx := context.Background()

	assert.NoError(t, checkUserActive(ctx, 1))
	assert.Equal(t, rescode.Unauthorized.Code(), gerror.Code(checkUserActive(ctx, 2)).Code(), "tài khoản LOCKED → 401")
	assert.Equal(t, rescode.Unauthorized.Code(), gerror.Code(checkUserActive(ctx, 3)).Code(), "tài khoản đã xóa → 401")
	assert.Equal(t, rescode.Internal.Code(), gerror.Code(checkUserActive(ctx, 99)).Code())

	// Khóa có hiệu lực ngay (không cache).
	statuses[1] = consts.StatusLocked
	assert.Error(t, checkUserActive(ctx, 1))

	SetUserStatusLookup(nil)
	assert.NoError(t, checkUserActive(ctx, 2), "chưa cấu hình lookup → bỏ qua kiểm tra")
}
