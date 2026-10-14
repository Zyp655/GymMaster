package loyalty

import (
	"context"

	v1 "gymmaster-api/api/loyalty/v1"
	"gymmaster-api/internal/service"
)

type AdminControllerV1 struct {
	svc service.LoyaltyService
}

type MemberControllerV1 struct {
	svc service.LoyaltyService
}

func NewAdminV1(svc service.LoyaltyService) *AdminControllerV1 {
	return &AdminControllerV1{svc: svc}
}

func NewMemberV1(svc service.LoyaltyService) *MemberControllerV1 {
	return &MemberControllerV1{svc: svc}
}

// ---- Admin ----------------------------------------------------------------------

func (c *AdminControllerV1) ListRewards(ctx context.Context, req *v1.ListLoyaltyRewardsReq) (res *v1.ListLoyaltyRewardsRes, err error) {
	return c.svc.ListRewards(ctx, req)
}

func (c *AdminControllerV1) CreateReward(ctx context.Context, req *v1.CreateLoyaltyRewardReq) (res *v1.CreateLoyaltyRewardRes, err error) {
	return c.svc.CreateReward(ctx, req.LoyaltyRewardInput)
}

func (c *AdminControllerV1) UpdateReward(ctx context.Context, req *v1.UpdateLoyaltyRewardReq) (res *v1.UpdateLoyaltyRewardRes, err error) {
	return c.svc.UpdateReward(ctx, req.ID, req.LoyaltyRewardInput)
}

func (c *AdminControllerV1) DeleteReward(ctx context.Context, req *v1.DeleteLoyaltyRewardReq) (res *v1.DeleteLoyaltyRes, err error) {
	return &v1.DeleteLoyaltyRes{}, c.svc.DeleteReward(ctx, req.ID)
}

func (c *AdminControllerV1) ListBadges(ctx context.Context, req *v1.ListBadgesReq) (res *v1.ListBadgesRes, err error) {
	return c.svc.ListBadges(ctx, req)
}

func (c *AdminControllerV1) CreateBadge(ctx context.Context, req *v1.CreateBadgeReq) (res *v1.CreateBadgeRes, err error) {
	return c.svc.CreateBadge(ctx, req.BadgeInput)
}

func (c *AdminControllerV1) DeleteBadge(ctx context.Context, req *v1.DeleteBadgeReq) (res *v1.DeleteLoyaltyRes, err error) {
	return &v1.DeleteLoyaltyRes{}, c.svc.DeleteBadge(ctx, req.ID)
}

func (c *AdminControllerV1) ListChallenges(ctx context.Context, req *v1.ListChallengesReq) (res *v1.ListChallengesRes, err error) {
	return c.svc.ListChallenges(ctx, req)
}

func (c *AdminControllerV1) CreateChallenge(ctx context.Context, req *v1.CreateChallengeReq) (res *v1.CreateChallengeRes, err error) {
	return c.svc.CreateChallenge(ctx, req.ChallengeInput)
}

func (c *AdminControllerV1) UpdateChallenge(ctx context.Context, req *v1.UpdateChallengeReq) (res *v1.UpdateChallengeRes, err error) {
	return c.svc.UpdateChallenge(ctx, req.ID, req.ChallengeInput)
}

func (c *AdminControllerV1) DeleteChallenge(ctx context.Context, req *v1.DeleteChallengeReq) (res *v1.DeleteLoyaltyRes, err error) {
	return &v1.DeleteLoyaltyRes{}, c.svc.DeleteChallenge(ctx, req.ID)
}

func (c *AdminControllerV1) AdjustMemberPoints(ctx context.Context, req *v1.AdjustMemberPointsReq) (res *v1.AdjustMemberPointsRes, err error) {
	return c.svc.AdjustMemberPoints(ctx, req)
}

// ---- Member ---------------------------------------------------------------------

func (c *MemberControllerV1) MemberLoyalty(ctx context.Context, _ *v1.MemberLoyaltyReq) (res *v1.MemberLoyaltyRes, err error) {
	return c.svc.MemberLoyalty(ctx)
}

func (c *MemberControllerV1) RedeemReward(ctx context.Context, req *v1.RedeemRewardReq) (res *v1.RedeemRewardRes, err error) {
	return c.svc.RedeemReward(ctx, req)
}

func (c *MemberControllerV1) MemberBadges(ctx context.Context, _ *v1.MemberBadgesReq) (res v1.MemberBadgesRes, err error) {
	return c.svc.MemberBadges(ctx)
}

func (c *MemberControllerV1) MemberChallenges(ctx context.Context, _ *v1.MemberChallengesReq) (res v1.MemberChallengesRes, err error) {
	return c.svc.MemberChallenges(ctx)
}

func (c *MemberControllerV1) JoinChallenge(ctx context.Context, req *v1.JoinChallengeReq) (res *v1.JoinChallengeRes, err error) {
	return c.svc.JoinChallenge(ctx, req)
}

func (c *MemberControllerV1) MemberLeaderboard(ctx context.Context, req *v1.MemberLeaderboardReq) (res *v1.MemberLeaderboardRes, err error) {
	return c.svc.MemberLeaderboard(ctx, req)
}

func (c *MemberControllerV1) MemberReferral(ctx context.Context, _ *v1.MemberReferralReq) (res *v1.MemberReferralRes, err error) {
	return c.svc.MemberReferral(ctx)
}
