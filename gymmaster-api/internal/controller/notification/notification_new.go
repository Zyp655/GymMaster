package notification

import (
	"gymmaster-api/api/notification"
	"gymmaster-api/internal/service"
)

// ControllerV1: thông báo của nhân viên. MemberControllerV1: cùng service, route cổng hội viên.
type ControllerV1 struct {
	notificationService service.NotificationService
}

type MemberControllerV1 struct {
	notificationService service.NotificationService
}

func NewV1(svc service.NotificationService) notification.INotificationV1 {
	return &ControllerV1{notificationService: svc}
}

func NewMemberV1(svc service.NotificationService) notification.IMemberNotificationV1 {
	return &MemberControllerV1{notificationService: svc}
}
