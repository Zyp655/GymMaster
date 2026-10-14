package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/notification/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/token"
)

// notifFakeRepo lưu thông báo trong bộ nhớ, kiểm tra service luôn lọc theo user hiện tại.
type notifFakeRepo struct {
	rows []entity.Notification
}

func (f *notifFakeRepo) List(_ context.Context, uid uint, unread bool, typ string, q dto.Query) ([]entity.Notification, int64, error) {
	var out []entity.Notification
	for _, n := range f.rows {
		if n.UserID == uid && (!unread || n.ReadAt == nil) && (typ == "" || n.Type == typ) {
			out = append(out, n)
		}
	}
	return out, int64(len(out)), nil
}

func (f *notifFakeRepo) UnreadCount(_ context.Context, uid uint) (int64, error) {
	var c int64
	for _, n := range f.rows {
		if n.UserID == uid && n.ReadAt == nil {
			c++
		}
	}
	return c, nil
}

func (f *notifFakeRepo) MarkRead(_ context.Context, uid, id uint, at time.Time) (*entity.Notification, error) {
	for i := range f.rows {
		if f.rows[i].ID == id && f.rows[i].UserID == uid {
			if f.rows[i].ReadAt == nil {
				f.rows[i].ReadAt = &at
			}
			return &f.rows[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *notifFakeRepo) MarkAllRead(_ context.Context, uid uint, at time.Time) (int64, error) {
	var c int64
	for i := range f.rows {
		if f.rows[i].UserID == uid && f.rows[i].ReadAt == nil {
			f.rows[i].ReadAt = &at
			c++
		}
	}
	return c, nil
}

func TestNotificationInbox(t *testing.T) {
	repo := &notifFakeRepo{rows: []entity.Notification{
		{ID: 1, UserID: 7, Type: NotifySubscriptionExpiring, Title: "a"},
		{ID: 2, UserID: 7, Type: NotifySystem, Title: "b"},
		{ID: 3, UserID: 8, Type: NotifySystem, Title: "c"},
	}}
	svc := NewNotificationService(repo)
	ctx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 7})

	_, err := svc.UnreadCount(context.Background())
	assert.Equal(t, 40100, reportErrCode(err)) // chưa đăng nhập

	cnt, err := svc.UnreadCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), cnt.Count)

	n, err := svc.MarkRead(ctx, 1)
	require.NoError(t, err)
	assert.True(t, n.IsRead)
	_, err = svc.MarkRead(ctx, 3) // của người khác
	assert.Equal(t, 40400, reportErrCode(err))

	list, err := svc.List(ctx, v1.ListFilter{PageReq: dto.PageReq{Page: 1, Limit: 10}, Unread: true})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, uint(2), list.Items[0].ID)

	all, err := svc.MarkAllRead(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), all.Updated)
	cnt, _ = svc.UnreadCount(ctx)
	assert.Equal(t, int64(0), cnt.Count)
	assert.Nil(t, repo.rows[2].ReadAt) // không đụng thông báo của user khác
}
