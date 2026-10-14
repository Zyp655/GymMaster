package subscription

import (
	"gymmaster-api/api/subscription"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.SubscriptionService
}

func NewV1(svc service.SubscriptionService) subscription.ISubscriptionV1 {
	return &ControllerV1{svc: svc}
}
