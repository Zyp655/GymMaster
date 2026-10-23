package service

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/loyalty/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
)

type fakeLoyaltyRepo struct {
	nextRewardID    uint
	nextBadgeID     uint
	nextChallengeID uint
	nextLedgerID    uint
	nextPromoID     uint

	rewards      map[uint]*entity.LoyaltyReward
	badges       map[uint]*entity.Badge
	memberBadges map[[2]uint]entity.MemberBadge
	challenges   map[uint]*entity.Challenge
	participants map[[2]uint]*entity.ChallengeParticipant
	members      map[uint]*entity.Member
	ledgers      []entity.LoyaltyLedger
	promotions   []entity.Promotion
	metricCounts map[string]int
}

func (f *fakeLoyaltyRepo) ListLedgers(_ context.Context, _ dto.Query, branchIDs []uint, memberID uint, reason string) ([]repository.LoyaltyLedgerRow, int64, error) {
	out := []repository.LoyaltyLedgerRow{}
	for _, l := range f.ledgers {
		m := f.members[l.MemberID]
		if m == nil || (memberID > 0 && l.MemberID != memberID) || (reason != "" && l.Reason != reason) {
			continue
		}
		if branchIDs != nil && !slices.Contains(branchIDs, m.HomeBranchID) {
			continue
		}
		out = append(out, repository.LoyaltyLedgerRow{LoyaltyLedger: l, MemberCode: m.MemberCode, MemberName: m.FullName, BranchID: m.HomeBranchID})
	}
	return out, int64(len(out)), nil
}

func newFakeLoyaltyRepo() *fakeLoyaltyRepo {
	return &fakeLoyaltyRepo{
		rewards:      map[uint]*entity.LoyaltyReward{},
		badges:       map[uint]*entity.Badge{},
		memberBadges: map[[2]uint]entity.MemberBadge{},
		challenges:   map[uint]*entity.Challenge{},
		participants: map[[2]uint]*entity.ChallengeParticipant{},
		members:      map[uint]*entity.Member{},
		metricCounts: map[string]int{},
	}
}

func (f *fakeLoyaltyRepo) ListRewards(_ context.Context, _ dto.Query, status, rewardType string) ([]entity.LoyaltyReward, int64, error) {
	var out []entity.LoyaltyReward
	for _, r := range f.rewards {
		if status != "" && r.Status != status {
			continue
		}
		if rewardType != "" && r.RewardType != rewardType {
			continue
		}
		out = append(out, *r)
	}
	return out, int64(len(out)), nil
}

func (f *fakeLoyaltyRepo) ListActiveRewards(_ context.Context) ([]entity.LoyaltyReward, error) {
	var out []entity.LoyaltyReward
	for _, r := range f.rewards {
		if r.Status == consts.StatusActive {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) FindRewardByID(_ context.Context, id uint) (*entity.LoyaltyReward, error) {
	if r, ok := f.rewards[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLoyaltyRepo) CreateReward(_ context.Context, r *entity.LoyaltyReward) error {
	f.nextRewardID++
	r.ID = f.nextRewardID
	cp := *r
	f.rewards[r.ID] = &cp
	return nil
}

func (f *fakeLoyaltyRepo) UpdateReward(_ context.Context, r *entity.LoyaltyReward) error {
	cp := *r
	f.rewards[r.ID] = &cp
	return nil
}

func (f *fakeLoyaltyRepo) DeleteReward(_ context.Context, id uint) error {
	delete(f.rewards, id)
	return nil
}

func (f *fakeLoyaltyRepo) DecrementRewardStock(_ context.Context, id uint) (bool, error) {
	r, ok := f.rewards[id]
	if !ok || r.Stock <= 0 {
		return false, nil
	}
	r.Stock--
	return true, nil
}

func (f *fakeLoyaltyRepo) ListBadges(_ context.Context, _ dto.Query) ([]entity.Badge, int64, error) {
	out, _ := f.ListAllBadges(context.Background())
	return out, int64(len(out)), nil
}

func (f *fakeLoyaltyRepo) ListAllBadges(_ context.Context) ([]entity.Badge, error) {
	var out []entity.Badge
	for _, b := range f.badges {
		out = append(out, *b)
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) FindBadgeByID(_ context.Context, id uint) (*entity.Badge, error) {
	if b, ok := f.badges[id]; ok {
		cp := *b
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLoyaltyRepo) ExistsBadgeCode(_ context.Context, code string) (bool, error) {
	for _, b := range f.badges {
		if strings.EqualFold(b.Code, strings.TrimSpace(code)) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeLoyaltyRepo) CreateBadge(_ context.Context, b *entity.Badge) error {
	f.nextBadgeID++
	b.ID = f.nextBadgeID
	cp := *b
	f.badges[b.ID] = &cp
	return nil
}

func (f *fakeLoyaltyRepo) DeleteBadge(_ context.Context, id uint) error {
	delete(f.badges, id)
	return nil
}

func (f *fakeLoyaltyRepo) ListMemberBadges(_ context.Context, memberID uint) ([]entity.MemberBadge, error) {
	var out []entity.MemberBadge
	for k, mb := range f.memberBadges {
		if k[0] == memberID {
			out = append(out, mb)
		}
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) AwardBadge(_ context.Context, mb *entity.MemberBadge) error {
	f.memberBadges[[2]uint{mb.MemberID, mb.BadgeID}] = *mb
	return nil
}

func (f *fakeLoyaltyRepo) ListChallenges(_ context.Context, _ dto.Query, _ uint, status string) ([]repository.ChallengeRow, int64, error) {
	var out []repository.ChallengeRow
	for _, c := range f.challenges {
		if status != "" && c.Status != status {
			continue
		}
		out = append(out, repository.ChallengeRow{Challenge: *c})
	}
	return out, int64(len(out)), nil
}

func (f *fakeLoyaltyRepo) ListActiveChallenges(_ context.Context, _ uint) ([]repository.ChallengeRow, error) {
	var out []repository.ChallengeRow
	for _, c := range f.challenges {
		if c.Status == consts.StatusActive {
			out = append(out, repository.ChallengeRow{Challenge: *c})
		}
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) FindChallengeByID(_ context.Context, id uint) (*repository.ChallengeRow, error) {
	if c, ok := f.challenges[id]; ok {
		return &repository.ChallengeRow{Challenge: *c}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLoyaltyRepo) CreateChallenge(_ context.Context, c *entity.Challenge) error {
	f.nextChallengeID++
	c.ID = f.nextChallengeID
	cp := *c
	f.challenges[c.ID] = &cp
	return nil
}

func (f *fakeLoyaltyRepo) UpdateChallenge(_ context.Context, c *entity.Challenge) error {
	cp := *c
	f.challenges[c.ID] = &cp
	return nil
}

func (f *fakeLoyaltyRepo) DeleteChallenge(_ context.Context, id uint) error {
	delete(f.challenges, id)
	return nil
}

func (f *fakeLoyaltyRepo) FindChallengeParticipant(_ context.Context, challengeID, memberID uint) (*entity.ChallengeParticipant, error) {
	if cp, ok := f.participants[[2]uint{challengeID, memberID}]; ok {
		out := *cp
		return &out, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLoyaltyRepo) ListMemberParticipants(_ context.Context, memberID uint) ([]entity.ChallengeParticipant, error) {
	var out []entity.ChallengeParticipant
	for k, cp := range f.participants {
		if k[1] == memberID {
			out = append(out, *cp)
		}
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) SaveChallengeParticipant(_ context.Context, cp *entity.ChallengeParticipant) error {
	copyCP := *cp
	f.participants[[2]uint{cp.ChallengeID, cp.MemberID}] = &copyCP
	return nil
}

func (f *fakeLoyaltyRepo) CountMemberMetric(_ context.Context, _ uint, metric string, _, _ *time.Time) (int, error) {
	return f.metricCounts[strings.ToUpper(metric)], nil
}

func (f *fakeLoyaltyRepo) FindMemberByID(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLoyaltyRepo) FindMemberByUserID(_ context.Context, userID uint) (*entity.Member, error) {
	for _, m := range f.members {
		if m.UserID != nil && *m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLoyaltyRepo) AddMemberPoints(_ context.Context, memberID uint, delta int) (int, error) {
	m, ok := f.members[memberID]
	if !ok {
		return 0, gorm.ErrRecordNotFound
	}
	if m.LoyaltyPoints+delta < 0 {
		return 0, repository.ErrInsufficientPoints
	}
	m.LoyaltyPoints += delta
	return m.LoyaltyPoints, nil
}

func (f *fakeLoyaltyRepo) LockMember(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeLoyaltyRepo) CreateLedgerOnce(ctx context.Context, l *entity.LoyaltyLedger) (bool, error) {
	if l.RefID != nil {
		for _, e := range f.ledgers {
			if e.MemberID == l.MemberID && e.Reason == l.Reason && e.RefType == l.RefType && e.RefID != nil && *e.RefID == *l.RefID {
				return false, nil
			}
		}
	}
	return true, f.CreateLedger(ctx, l)
}

func (f *fakeLoyaltyRepo) UpdateMemberReferralCode(_ context.Context, memberID uint, code string) error {
	if m, ok := f.members[memberID]; ok {
		m.ReferralCode = &code
	}
	return nil
}

func (f *fakeLoyaltyRepo) CreateLedger(_ context.Context, l *entity.LoyaltyLedger) error {
	f.nextLedgerID++
	l.ID = f.nextLedgerID
	l.CreatedAt = time.Now()
	f.ledgers = append(f.ledgers, *l)
	return nil
}

func (f *fakeLoyaltyRepo) ListMemberLedgers(_ context.Context, memberID uint, _ int) ([]entity.LoyaltyLedger, error) {
	var out []entity.LoyaltyLedger
	for _, l := range f.ledgers {
		if l.MemberID == memberID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) CreatePromotion(_ context.Context, p *entity.Promotion) error {
	f.nextPromoID++
	p.ID = f.nextPromoID
	f.promotions = append(f.promotions, *p)
	return nil
}

func (f *fakeLoyaltyRepo) TopMembersByPoints(_ context.Context, _ int) ([]repository.LeaderboardRow, error) {
	var out []repository.LeaderboardRow
	for _, m := range f.members {
		out = append(out, repository.LeaderboardRow{
			MemberID:      m.ID,
			MemberCode:    m.MemberCode,
			FullName:      m.FullName,
			LoyaltyPoints: m.LoyaltyPoints,
		})
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) TopMembersByCheckins(ctx context.Context, limit int) ([]repository.LeaderboardRow, error) {
	return f.TopMembersByPoints(ctx, limit)
}

func (f *fakeLoyaltyRepo) ListReferredMembers(_ context.Context, memberID uint) ([]entity.Member, error) {
	var out []entity.Member
	for _, m := range f.members {
		if m.ReferredBy != nil && *m.ReferredBy == memberID {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (f *fakeLoyaltyRepo) SumReferralPoints(_ context.Context, memberID uint) (int, error) {
	sum := 0
	for _, l := range f.ledgers {
		if l.MemberID == memberID && l.Reason == entity.LoyaltyReasonReferral {
			sum += l.Points
		}
	}
	return sum, nil
}

func TestLoyaltyService_RedeemAndChallenges(t *testing.T) {
	repo := newFakeLoyaltyRepo()
	svc := NewLoyaltyService(repo, nil, nil)
	// Cố định "hôm nay" trong khoảng thời gian thử thách (01–31/10/2026) để test không phụ thuộc ngày chạy.
	svc.(*loyaltyService).now = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.Local) }

	uid := uint(101)
	repo.members[1] = &entity.Member{
		BaseEntity:    entity.BaseEntity{ID: 1},
		UserID:        &uid,
		MemberCode:    "GM000001",
		FullName:      "Nguyễn Văn A",
		LoyaltyPoints: 150,
		Status:        entity.MemberStatusActive,
	}

	adminCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: 999, Roles: []string{consts.RoleOwner}})
	memberCtx := reqctx.WithAuth(context.Background(), &token.Payload{UserID: uid, Roles: []string{consts.RoleMember}})

	// Create rewards: 1 voucher (200 pts), 1 out-of-stock reward (50 pts)
	voucherReward, err := svc.CreateReward(adminCtx, v1.LoyaltyRewardInput{
		Name:        "Voucher giảm 15%",
		CostPoints:  200,
		RewardType:  entity.RewardTypeVoucher,
		DiscountPct: 15,
		Stock:       5,
		Status:      consts.StatusActive,
	})
	require.NoError(t, err)

	emptyReward, err := svc.CreateReward(adminCtx, v1.LoyaltyRewardInput{
		Name:       "Bình nước thể thao",
		CostPoints: 50,
		RewardType: entity.RewardTypeProduct,
		Stock:      0,
		Status:     consts.StatusActive,
	})
	require.NoError(t, err)

	// 1. Đổi quà khi thiếu điểm -> InsufficientPoints
	_, err = svc.RedeemReward(memberCtx, &v1.RedeemRewardReq{RewardID: voucherReward.ID})
	assert.Equal(t, rescode.InsufficientPoints.Code(), codeOf(err))

	// 2. Đổi quà khi hết hàng -> RewardOutOfStock
	_, err = svc.RedeemReward(memberCtx, &v1.RedeemRewardReq{RewardID: emptyReward.ID})
	assert.Equal(t, rescode.RewardOutOfStock.Code(), codeOf(err))

	// Admin cộng thêm 100 điểm -> member có 250 điểm
	adjRes, err := svc.AdjustMemberPoints(adminCtx, &v1.AdjustMemberPointsReq{
		ID:     1,
		Points: 100,
		Reason: entity.LoyaltyReasonAdjust,
		Note:   "Thưởng tích cực",
	})
	require.NoError(t, err)
	assert.Equal(t, 250, adjRes.Points)

	// 3. Đổi voucher thành công -> trừ 200 điểm còn 50, tạo mã khuyến mãi RWD-GM000001-...
	redeemRes, err := svc.RedeemReward(memberCtx, &v1.RedeemRewardReq{RewardID: voucherReward.ID})
	require.NoError(t, err)
	assert.Equal(t, 50, redeemRes.Points)
	assert.True(t, strings.HasPrefix(redeemRes.VoucherCode, "RWD-GM000001-"))
	require.Len(t, repo.promotions, 1)
	assert.Equal(t, 15.0, repo.promotions[0].Value)

	// 4. Tham gia thử thách và hoàn thành ngay khi đủ chỉ số
	ch, err := svc.CreateChallenge(adminCtx, v1.ChallengeInput{
		Name:         "Thử thách 10 buổi check-in",
		Metric:       entity.ChallengeMetricCheckins,
		Target:       10,
		StartDate:    "2026-10-01",
		EndDate:      "2026-10-31",
		RewardPoints: 120,
		Status:       consts.StatusActive,
	})
	require.NoError(t, err)

	repo.metricCounts[entity.ChallengeMetricCheckins] = 12
	joined, err := svc.JoinChallenge(memberCtx, &v1.JoinChallengeReq{ID: ch.ID})
	require.NoError(t, err)
	assert.True(t, joined.Joined)
	assert.Equal(t, 12, joined.Progress)
	assert.NotNil(t, joined.CompletedAt)

	// Điểm hội viên tăng từ 50 + 120 = 170
	overview, err := svc.MemberLoyalty(memberCtx)
	require.NoError(t, err)
	assert.Equal(t, 170, overview.Points)
	assert.Equal(t, "REF-GM000001", overview.ReferralCode)
}
