package service

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

// Loại thông báo (notifications.type).
const (
	NotifySubscriptionExpiring = "SUBSCRIPTION_EXPIRING"
	NotifySubscriptionExpired  = "SUBSCRIPTION_EXPIRED"
	NotifyFreezeRequest        = "FREEZE_REQUEST"
	NotifyFreezeResult         = "FREEZE_RESULT"
	NotifyBookingPromoted      = "BOOKING_PROMOTED"
	NotifyClassCancelled       = "CLASS_CANCELLED"
	NotifyPTScheduled          = "PT_SCHEDULED"
	NotifyPTCancelled          = "PT_CANCELLED"
	NotifyLowStock             = "LOW_STOCK"
	NotifySystem               = "SYSTEM"
)

// Notifier tạo thông báo in-app. Các module nghiệp vụ dùng interface này;
// API đọc/đánh dấu đã đọc nằm ở module notification.
type Notifier interface {
	Notify(ctx context.Context, userIDs []uint, typ, title, body, link string) error
	// NotifyPermission gửi cho mọi user (đang hoạt động) có quyền perm, tùy chọn giới hạn chi nhánh.
	NotifyPermission(ctx context.Context, perm string, branchID *uint, typ, title, body, link string) error
}

type notifier struct {
	db *gorm.DB
}

func NewNotifier(db *gorm.DB) Notifier {
	return &notifier{db: db}
}

func (n *notifier) Notify(ctx context.Context, userIDs []uint, typ, title, body, link string) error {
	if len(userIDs) == 0 {
		return nil
	}
	rows := make([]entity.Notification, 0, len(userIDs))
	for _, id := range userIDs {
		rows = append(rows, entity.Notification{UserID: id, Type: typ, Title: title, Body: body, Link: link})
	}
	return postgres.Conn(ctx, n.db).Create(&rows).Error
}

func (n *notifier) NotifyPermission(ctx context.Context, perm string, branchID *uint, typ, title, body, link string) error {
	q := postgres.Conn(ctx, n.db).Table("users u").
		Distinct("u.id").
		Joins("JOIN user_roles ur ON ur.user_id = u.id").
		Joins("JOIN roles r ON r.id = ur.role_id AND r.deleted_at IS NULL").
		Where("u.deleted_at IS NULL AND u.status = 'ACTIVE'").
		Where(`r.code = 'OWNER' OR EXISTS (
			SELECT 1 FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id
			WHERE rp.role_id = r.id AND p.code = ?)`, perm)
	if branchID != nil {
		q = q.Where("ur.branch_id IS NULL OR ur.branch_id = ?", *branchID)
	}
	var ids []uint
	if err := q.Pluck("u.id", &ids).Error; err != nil {
		return err
	}
	return n.Notify(ctx, ids, typ, title, body, link)
}
