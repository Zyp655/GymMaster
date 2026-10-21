package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1loyalty "gymmaster-api/api/loyalty/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
)

func TestPublicAvailableSlots_NeverNegative(t *testing.T) {
	assert.Equal(t, 20, publicAvailableSlots(20, 0))
	assert.Equal(t, 5, publicAvailableSlots(20, 15))
	assert.Equal(t, 0, publicAvailableSlots(20, 20))
	assert.Equal(t, 0, publicAvailableSlots(20, 25), "overbooked → 0, không âm")
}

func TestValidatePostSlug(t *testing.T) {
	for _, ok := range []string{"lo-trinh-tang-co", "abc", "bai-1-tuan-2"} {
		assert.NoError(t, validatePostSlug(ok), ok)
	}
	for _, bad := range []string{"", "ab", "Lo-Trinh", "lo trinh", "-abc", "abc-", "a--b", "lộ-trình", "a/b", "../x", string(make([]byte, 181))} {
		assert.Error(t, validatePostSlug(bad), bad)
	}
}

func TestValidatePostEnums(t *testing.T) {
	assert.NoError(t, validatePostCategory("TAP_LUYEN"))
	assert.NoError(t, validatePostCategory("DINH_DUONG"))
	assert.NoError(t, validatePostCategory("SU_KIEN"))
	assert.Error(t, validatePostCategory("tap_luyen"))
	assert.Error(t, validatePostCategory("HACK"))
	assert.NoError(t, validatePostStatus("DRAFT"))
	assert.NoError(t, validatePostStatus("PUBLISHED"))
	assert.NoError(t, validatePostStatus("ARCHIVED"))
	assert.Error(t, validatePostStatus("DELETED"))
}

func TestPostIsPublic_OnlyPublishedAndDue(t *testing.T) {
	now := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	assert.True(t, postIsPublic(&entity.Post{Status: entity.PostStatusPublished, PublishedAt: &past}, now))
	assert.True(t, postIsPublic(&entity.Post{Status: entity.PostStatusPublished}, now))
	assert.False(t, postIsPublic(&entity.Post{Status: entity.PostStatusPublished, PublishedAt: &future}, now), "hẹn giờ chưa tới")
	assert.False(t, postIsPublic(&entity.Post{Status: entity.PostStatusDraft, PublishedAt: &past}, now), "bản nháp")
	assert.False(t, postIsPublic(&entity.Post{Status: entity.PostStatusArchived, PublishedAt: &past}, now))
	assert.False(t, postIsPublic(nil, now))
}

func TestChallengeEnded_ByCalendarDay(t *testing.T) {
	end := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	assert.False(t, challengeEnded(end, time.Date(2026, 10, 20, 23, 59, 0, 0, time.Local)), "còn trong ngày kết thúc")
	assert.True(t, challengeEnded(end, time.Date(2026, 10, 21, 0, 1, 0, 0, time.Local)))
	assert.False(t, challengeEnded(end, time.Date(2026, 10, 1, 8, 0, 0, 0, time.Local)))
}

func TestLoyaltyChallenge_RejectPastActiveAndEndedJoin(t *testing.T) {
	repo := newFakeLoyaltyRepo()
	svcI := NewLoyaltyService(repo, nil, nil)
	svc := svcI.(*loyaltyService)
	today := time.Date(2026, 10, 21, 9, 0, 0, 0, time.Local)
	svc.now = func() time.Time { return today }

	uid := uint(202)
	repo.members[7] = &entity.Member{
		BaseEntity: entity.BaseEntity{ID: 7}, UserID: &uid, MemberCode: "GM000007",
		FullName: "Trần B", HomeBranchID: 1, Status: entity.MemberStatusActive,
	}
	adminCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 999, Roles: []string{consts.RoleOwner}})
	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid, Roles: []string{consts.RoleMember}})

	// Admin tạo thử thách ACTIVE nhưng ngày kết thúc đã qua → 400.
	_, err := svc.CreateChallenge(adminCtx, v1loyalty.ChallengeInput{
		Name: "Hết hạn", Metric: entity.ChallengeMetricCheckins, Target: 5,
		StartDate: "2026-09-01", EndDate: "2026-10-20", Status: consts.StatusActive,
	})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Trạng thái ENDED với ngày quá khứ vẫn hợp lệ (lưu trữ lịch sử).
	ended, err := svc.CreateChallenge(adminCtx, v1loyalty.ChallengeInput{
		Name: "Đã kết thúc", Metric: entity.ChallengeMetricCheckins, Target: 5,
		StartDate: "2026-09-01", EndDate: "2026-10-20", Status: "ENDED",
	})
	require.NoError(t, err)
	_, err = svc.JoinChallenge(memberCtx, &v1loyalty.JoinChallengeReq{ID: ended.ID})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// ACTIVE, kết thúc hôm nay → vẫn tham gia được.
	open, err := svc.CreateChallenge(adminCtx, v1loyalty.ChallengeInput{
		Name: "Kết thúc hôm nay", Metric: entity.ChallengeMetricCheckins, Target: 5,
		StartDate: "2026-10-01", EndDate: "2026-10-21", Status: consts.StatusActive,
	})
	require.NoError(t, err)
	_, err = svc.JoinChallenge(memberCtx, &v1loyalty.JoinChallengeReq{ID: open.ID})
	require.NoError(t, err)

	// Thời gian trôi qua sang ngày hôm sau → thử thách ACTIVE (chưa đóng) nhưng đã hết hạn → từ chối.
	svc.now = func() time.Time { return today.AddDate(0, 0, 1) }
	_, err = svc.JoinChallenge(memberCtx, &v1loyalty.JoinChallengeReq{ID: open.ID})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	// Thử thách của chi nhánh khác → 403.
	svc.now = func() time.Time { return today }
	other := uint(2)
	branchOnly, err := svc.CreateChallenge(adminCtx, v1loyalty.ChallengeInput{
		BranchID: &other, Name: "CN2", Metric: entity.ChallengeMetricCheckins, Target: 5,
		StartDate: "2026-10-01", EndDate: "2026-10-31", Status: consts.StatusActive,
	})
	require.NoError(t, err)
	_, err = svc.JoinChallenge(memberCtx, &v1loyalty.JoinChallengeReq{ID: branchOnly.ID})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}

func TestLoyaltyLedgers_BranchScopedAndAdjustScope(t *testing.T) {
	repo := newFakeLoyaltyRepo()
	svc := NewLoyaltyService(repo, nil, nil)
	repo.members[1] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 1}, MemberCode: "GM1", FullName: "CN1", HomeBranchID: 1}
	repo.members[2] = &entity.Member{BaseEntity: entity.BaseEntity{ID: 2}, MemberCode: "GM2", FullName: "CN2", HomeBranchID: 2}

	owner := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 9, Roles: []string{consts.RoleOwner}})
	mgr1 := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 10, Roles: []string{"MANAGER"}, BranchIDs: []uint{1}})

	_, err := svc.AdjustMemberPoints(owner, &v1loyalty.AdjustMemberPointsReq{ID: 1, Points: 50, Note: "Thưởng"})
	require.NoError(t, err)
	_, err = svc.AdjustMemberPoints(owner, &v1loyalty.AdjustMemberPointsReq{ID: 2, Points: 30, Note: "Thưởng"})
	require.NoError(t, err)

	// Owner thấy cả 2 chi nhánh.
	all, err := svc.ListLedgers(owner, &v1loyalty.ListLoyaltyLedgersReq{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), all.Total)

	// Quản lý CN1 chỉ thấy ledger của hội viên CN1; lọc CN2 → 403.
	mine, err := svc.ListLedgers(mgr1, &v1loyalty.ListLoyaltyLedgersReq{})
	require.NoError(t, err)
	require.Len(t, mine.Items, 1)
	assert.Equal(t, "GM1", mine.Items[0].MemberCode)
	assert.Equal(t, 50, mine.Items[0].Points)
	_, err = svc.ListLedgers(mgr1, &v1loyalty.ListLoyaltyLedgersReq{BranchID: 2})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))

	// Quản lý CN1 không được điều chỉnh điểm hội viên CN2.
	_, err = svc.AdjustMemberPoints(mgr1, &v1loyalty.AdjustMemberPointsReq{ID: 2, Points: 10})
	assert.Equal(t, rescode.PermissionDenied.Code(), codeOf(err))
}
