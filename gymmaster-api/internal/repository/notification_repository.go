package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// NotificationRepository: hộp thông báo của từng người dùng (tạo thông báo đi qua service.Notifier).
	NotificationRepository interface {
		List(ctx context.Context, userID uint, unreadOnly bool, typ string, q dto.Query) ([]entity.Notification, int64, error)
		UnreadCount(ctx context.Context, userID uint) (int64, error)
		// MarkRead đánh dấu đã đọc một thông báo của userID; trả false nếu không tồn tại / không thuộc user.
		MarkRead(ctx context.Context, userID, id uint, at time.Time) (*entity.Notification, error)
		MarkAllRead(ctx context.Context, userID uint, at time.Time) (int64, error)
	}

	notificationRepository struct {
		db *gorm.DB
	}
)

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) List(ctx context.Context, userID uint, unreadOnly bool, typ string, q dto.Query) ([]entity.Notification, int64, error) {
	db := postgres.Conn(ctx, r.db).Model(&entity.Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		db = db.Where("read_at IS NULL")
	}
	if typ != "" {
		db = db.Where("type = ?", typ)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.Notification
	err := db.Scopes(q.Paginate).Order("id DESC").Find(&rows).Error
	return rows, total, err
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID uint) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&n).Error
	return n, err
}

func (r *notificationRepository) MarkRead(ctx context.Context, userID, id uint, at time.Time) (*entity.Notification, error) {
	conn := postgres.Conn(ctx, r.db)
	if err := conn.Model(&entity.Notification{}).Where("id = ? AND user_id = ? AND read_at IS NULL", id, userID).
		Update("read_at", at).Error; err != nil {
		return nil, err
	}
	var n entity.Notification
	if err := conn.Where("id = ? AND user_id = ?", id, userID).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uint, at time.Time) (int64, error) {
	res := postgres.Conn(ctx, r.db).Model(&entity.Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Update("read_at", at)
	return res.RowsAffected, res.Error
}
