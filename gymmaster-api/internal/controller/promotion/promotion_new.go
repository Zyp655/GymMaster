package promotion

import (
	"gymmaster-api/api/promotion"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	svc service.PromotionService
}

func NewV1(svc service.PromotionService) promotion.IPromotionV1 {
	return &ControllerV1{svc: svc}
}
