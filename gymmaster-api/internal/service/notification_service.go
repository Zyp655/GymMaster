package service

import (
	"context"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/realtime"
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
	NotifyPaymentSuccess       = "PAYMENT_SUCCESS"
	NotifyReviewPrompt         = "REVIEW_PROMPT"
	NotifyMaintenanceDue       = "MAINTENANCE_DUE"
	NotifyBirthday             = "BIRTHDAY"
	NotifyLeadNew              = "LEAD_NEW"
	NotifyCheckin              = "CHECKIN"
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
	db  *gorm.DB
	hub *realtime.Hub
}

func NewNotifier(db *gorm.DB) Notifier {
	return &notifier{db: db, hub: realtime.Default()}
}

func (n *notifier) Notify(ctx context.Context, userIDs []uint, typ, title, body, link string) error {
	if len(userIDs) == 0 {
		return nil
	}
	disabledUsers := make(map[uint]bool)
	if typ != NotifyPaymentSuccess && typ != NotifySystem {
		var disabledIDs []uint
		_ = postgres.Conn(ctx, n.db).Model(&entity.NotificationPreference{}).
			Where("user_id IN ? AND type = ? AND channel = 'IN_APP' AND enabled = FALSE", userIDs, typ).
			Pluck("user_id", &disabledIDs).Error
		for _, id := range disabledIDs {
			disabledUsers[id] = true
		}
	}
	rows := make([]entity.Notification, 0, len(userIDs))
	for _, id := range userIDs {
		if disabledUsers[id] {
			continue
		}
		rows = append(rows, entity.Notification{UserID: id, Type: typ, Title: title, Body: body, Link: link})
	}
	if len(rows) == 0 {
		return nil
	}
	if err := postgres.Conn(ctx, n.db).Create(&rows).Error; err != nil {
		return err
	}
	if n.hub != nil {
		for _, row := range rows {
			n.hub.Publish(realtime.Event{
				ID:        row.ID,
				UserID:    row.UserID,
				Type:      row.Type,
				Title:     row.Title,
				Body:      row.Body,
				Link:      row.Link,
				CreatedAt: row.CreatedAt,
			})
		}
	}
	return nil
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

