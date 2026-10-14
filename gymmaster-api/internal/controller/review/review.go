package review

import (
	"context"

	v1 "gymmaster-api/api/review/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.ReviewService
}

type MemberControllerV1 struct {
	svc service.ReviewService
}

func NewAdminV1(svc service.ReviewService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.ReviewService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

func (c *AdminControllerV1) ListReviews(ctx context.Context, req *v1.ListReviewsReq) (res *v1.ListReviewsRes, err error) {
	return c.svc.List(ctx, req)
}

func (c *MemberControllerV1) CreateReview(ctx context.Context, req *v1.CreateReviewReq) (res *v1.CreateReviewRes, err error) {
	return c.svc.Create(ctx, req)
}

func (c *MemberControllerV1) MemberReviews(ctx context.Context, _ *v1.MemberReviewsReq) (res v1.MemberReviewsRes, err error) {
	return c.svc.ListMyReviews(ctx)
}
