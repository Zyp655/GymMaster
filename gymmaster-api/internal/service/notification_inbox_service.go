package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/notification/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// NotificationService: hộp thông báo in-app của người dùng hiện tại (nhân viên và hội viên dùng chung).
// Việc TẠO thông báo đi qua Notifier (notification_service.go).
type NotificationService interface {
	List(ctx context.Context, f v1.ListFilter) (*v1.ListRes, error)
	UnreadCount(ctx context.Context) (*v1.UnreadCount, error)
	MarkRead(ctx context.Context, id uint) (*v1.Notification, error)
	MarkAllRead(ctx context.Context) (*v1.ReadAllResult, error)
}

type notificationService struct {
	repo repository.NotificationRepository
	now  func() time.Time
}

func NewNotificationService(repo repository.NotificationRepository) NotificationService {
	return &notificationService{repo: repo, now: time.Now}
}

func notifCurrentUser(ctx context.Context) (uint, error) {
	if id := reqctx.UserID(ctx); id != nil {
		return *id, nil
	}
	return 0, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
}

func notifToDTO(n entity.Notification) *v1.Notification {
	return &v1.Notification{
		ID: n.ID, Type: n.Type, Title: n.Title, Body: n.Body, Link: n.Link,
		IsRead: n.ReadAt != nil, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt,
	}
}

func (s *notificationService) List(ctx context.Context, f v1.ListFilter) (*v1.ListRes, error) {
	uid, err := notifCurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	q := f.ToQuery([]string{"created_at", "id"}, "created_at")
	rows, total, err := s.repo.List(ctx, uid, f.Unread, f.Type, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, notifToDTO), total, q), nil
}

func (s *notificationService) UnreadCount(ctx context.Context) (*v1.UnreadCount, error) {
	uid, err := notifCurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.UnreadCount(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &v1.UnreadCount{Count: n}, nil
}

func (s *notificationService) MarkRead(ctx context.Context, id uint) (*v1.Notification, error) {
	uid, err := notifCurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.MarkRead(ctx, uid, id, s.now())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy thông báo")
	}
	if err != nil {
		return nil, err
	}
	return notifToDTO(*n), nil
}

func (s *notificationService) MarkAllRead(ctx context.Context) (*v1.ReadAllResult, error) {
	uid, err := notifCurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.MarkAllRead(ctx, uid, s.now())
	if err != nil {
		return nil, err
	}
	return &v1.ReadAllResult{Updated: n}, nil
}
