package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/loyalty/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityLoyaltyReward = "loyalty_reward"
	auditEntityBadge         = "badge"
	auditEntityChallenge     = "challenge"
	auditEntityLoyaltyPoints = "loyalty_points"
)

var (
	loyaltyRewardOrderable = []string{"id", "name", "cost_points", "stock", "created_at"}
	badgeOrderable         = []string{"id", "code", "name", "created_at"}
	challengeOrderable     = []string{"id", "name", "start_date", "end_date", "reward_points", "created_at"}
	validChallengeMetrics  = []string{
		entity.ChallengeMetricCheckins,
		entity.ChallengeMetricClasses,
		entity.ChallengeMetricPTSessions,
	}
	validLoyaltyReasons = []string{
		entity.LoyaltyReasonCheckin,
		entity.LoyaltyReasonRenew,
		entity.LoyaltyReasonReferral,
		entity.LoyaltyReasonReview,
		entity.LoyaltyReasonRedeem,
		entity.LoyaltyReasonChallenge,
		entity.LoyaltyReasonAdjust,
	}
)

type (
	LoyaltyService interface {
		ListRewards(ctx context.Context, req *v1.ListLoyaltyRewardsReq) (*v1.ListLoyaltyRewardsRes, error)
		CreateReward(ctx context.Context, in v1.LoyaltyRewardInput) (*v1.LoyaltyRewardItem, error)
		UpdateReward(ctx context.Context, id uint, in v1.LoyaltyRewardInput) (*v1.LoyaltyRewardItem, error)
		DeleteReward(ctx context.Context, id uint) error

		ListBadges(ctx context.Context, req *v1.ListBadgesReq) (*v1.ListBadgesRes, error)
		CreateBadge(ctx context.Context, in v1.BadgeInput) (*v1.BadgeItem, error)
		DeleteBadge(ctx context.Context, id uint) error

		ListChallenges(ctx context.Context, req *v1.ListChallengesReq) (*v1.ListChallengesRes, error)
		CreateChallenge(ctx context.Context, in v1.ChallengeInput) (*v1.ChallengeItem, error)
		UpdateChallenge(ctx context.Context, id uint, in v1.ChallengeInput) (*v1.ChallengeItem, error)
		DeleteChallenge(ctx context.Context, id uint) error

		AdjustMemberPoints(ctx context.Context, req *v1.AdjustMemberPointsReq) (*v1.AdjustMemberPointsRes, error)
		ListLedgers(ctx context.Context, req *v1.ListLoyaltyLedgersReq) (*v1.ListLoyaltyLedgersRes, error)

		MemberLoyalty(ctx context.Context) (*v1.MemberLoyaltyRes, error)
		RedeemReward(ctx context.Context, req *v1.RedeemRewardReq) (*v1.RedeemRewardRes, error)
		MemberBadges(ctx context.Context) (v1.MemberBadgesRes, error)
		MemberChallenges(ctx context.Context) (v1.MemberChallengesRes, error)
		JoinChallenge(ctx context.Context, req *v1.JoinChallengeReq) (*v1.ChallengeItem, error)
		MemberLeaderboard(ctx context.Context, req *v1.MemberLeaderboardReq) (*v1.MemberLeaderboardRes, error)
		MemberReferral(ctx context.Context) (*v1.MemberReferralRes, error)
	}

	loyaltyService struct {
		repo  repository.LoyaltyRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewLoyaltyService(repo repository.LoyaltyRepository, tx postgres.TxManager, audit AuditService) LoyaltyService {
	return &loyaltyService{repo: repo, tx: tx, audit: audit, now: time.Now}
}

func (s *loyaltyService) runTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx != nil {
		return s.tx.WithTx(ctx, fn)
	}
	return fn(ctx)
}

func (s *loyaltyService) currentMember(ctx context.Context) (*entity.Member, error) {
	uid := reqctx.UserID(ctx)
	if uid == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	m, err := s.repo.FindMemberByUserID(ctx, *uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
		}
		return nil, err
	}
	return m, nil
}

func (s *loyaltyService) ensureReferralCode(ctx context.Context, m *entity.Member) string {
	if m.ReferralCode != nil && strings.TrimSpace(*m.ReferralCode) != "" {
		return *m.ReferralCode
	}
	base := strings.TrimSpace(m.MemberCode)
	if base == "" {
		base = fmt.Sprintf("M%06d", m.ID)
	}
	code := "REF-" + strings.ToUpper(base)
	_ = s.repo.UpdateMemberReferralCode(ctx, m.ID, code)
	m.ReferralCode = &code
	return code
}

// ---- Admin: Loyalty Rewards -----------------------------------------------------

func (s *loyaltyService) ListRewards(ctx context.Context, req *v1.ListLoyaltyRewardsReq) (*v1.ListLoyaltyRewardsRes, error) {
	q := req.PageReq.ToQuery(loyaltyRewardOrderable, "id")
	items, total, err := s.repo.ListRewards(ctx, q, strings.TrimSpace(req.Status), strings.TrimSpace(req.RewardType))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, toLoyaltyRewardItem), total, q), nil
}

func (s *loyaltyService) CreateReward(ctx context.Context, in v1.LoyaltyRewardInput) (*v1.LoyaltyRewardItem, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên phần thưởng")
	}
	if in.CostPoints <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Điểm quy đổi phải lớn hơn 0")
	}
	rt := strings.ToUpper(strings.TrimSpace(in.RewardType))
	if rt == "" {
		rt = entity.RewardTypeVoucher
	}
	if rt != entity.RewardTypeVoucher && rt != entity.RewardTypeProduct {
		return nil, rescode.New(rescode.BadRequest, "Loại phần thưởng không hợp lệ")
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = consts.StatusActive
	}
	discountPct := in.DiscountPct
	if rt == entity.RewardTypeVoucher && discountPct <= 0 {
		discountPct = 10
	}
	item := &entity.LoyaltyReward{
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		CostPoints:  in.CostPoints,
		RewardType:  rt,
		RefID:       in.RefID,
		DiscountPct: discountPct,
		Stock:       in.Stock,
		Status:      status,
	}
	if err := s.repo.CreateReward(ctx, item); err != nil {
		return nil, err
	}
	res := toLoyaltyRewardItem(*item)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityLoyaltyReward, item.ID, nil, res)
	}
	return res, nil
}

func (s *loyaltyService) UpdateReward(ctx context.Context, id uint, in v1.LoyaltyRewardInput) (*v1.LoyaltyRewardItem, error) {
	item, err := s.repo.FindRewardByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy phần thưởng")
		}
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên phần thưởng")
	}
	if in.CostPoints <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Điểm quy đổi phải lớn hơn 0")
	}
	rt := strings.ToUpper(strings.TrimSpace(in.RewardType))
	if rt == "" {
		rt = item.RewardType
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = item.Status
	}
	before := toLoyaltyRewardItem(*item)
	item.Name = name
	item.Description = strings.TrimSpace(in.Description)
	item.CostPoints = in.CostPoints
	item.RewardType = rt
	item.RefID = in.RefID
	item.DiscountPct = in.DiscountPct
	item.Stock = in.Stock
	item.Status = status
	if err := s.repo.UpdateReward(ctx, item); err != nil {
		return nil, err
	}
	res := toLoyaltyRewardItem(*item)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityLoyaltyReward, item.ID, before, res)
	}
	return res, nil
}

func (s *loyaltyService) DeleteReward(ctx context.Context, id uint) error {
	item, err := s.repo.FindRewardByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy phần thưởng")
		}
		return err
	}
	if err := s.repo.DeleteReward(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityLoyaltyReward, id, toLoyaltyRewardItem(*item), nil)
	}
	return nil
}

// ---- Admin: Badges --------------------------------------------------------------

func (s *loyaltyService) ListBadges(ctx context.Context, req *v1.ListBadgesReq) (*v1.ListBadgesRes, error) {
	q := req.PageReq.ToQuery(badgeOrderable, "id")
	items, total, err := s.repo.ListBadges(ctx, q)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(items, func(b entity.Badge) *v1.BadgeItem {
		return toBadgeItem(b, nil)
	}), total, q), nil
}

func (s *loyaltyService) CreateBadge(ctx context.Context, in v1.BadgeInput) (*v1.BadgeItem, error) {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập mã và tên huy hiệu")
	}
	dup, err := s.repo.ExistsBadgeCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Mã huy hiệu đã tồn tại")
	}
	icon := strings.TrimSpace(in.Icon)
	if icon == "" {
		icon = "trophy"
	}
	ruleBytes := []byte("{}")
	if in.Rule != nil {
		if b, err := json.Marshal(in.Rule); err == nil {
			ruleBytes = b
		}
	}
	item := &entity.Badge{
		Code:        code,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		Icon:        icon,
		Rule:        datatypes.JSON(ruleBytes),
	}
	if err := s.repo.CreateBadge(ctx, item); err != nil {
		return nil, mapDuplicate(err, "Mã huy hiệu đã tồn tại")
	}
	res := toBadgeItem(*item, nil)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityBadge, item.ID, nil, res)
	}
	return res, nil
}

func (s *loyaltyService) DeleteBadge(ctx context.Context, id uint) error {
	b, err := s.repo.FindBadgeByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy huy hiệu")
		}
		return err
	}
	if err := s.repo.DeleteBadge(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityBadge, id, toBadgeItem(*b, nil), nil)
	}
	return nil
}

// ---- Admin: Challenges ----------------------------------------------------------

func (s *loyaltyService) ListChallenges(ctx context.Context, req *v1.ListChallengesReq) (*v1.ListChallengesRes, error) {
	q := req.PageReq.ToQuery(challengeOrderable, "id")
	q.OrderBy = "c." + q.OrderBy
	rows, total, err := s.repo.ListChallenges(ctx, q, req.BranchID, strings.TrimSpace(req.Status))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.ChallengeRow) *v1.ChallengeItem {
		return toChallengeItem(r, nil)
	}), total, q), nil
}

func (s *loyaltyService) CreateChallenge(ctx context.Context, in v1.ChallengeInput) (*v1.ChallengeItem, error) {
	c, err := s.parseChallengeInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateChallenge(ctx, c); err != nil {
		return nil, err
	}
	row, err := s.repo.FindChallengeByID(ctx, c.ID)
	if err != nil {
		row = &repository.ChallengeRow{Challenge: *c}
	}
	res := toChallengeItem(*row, nil)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityChallenge, c.ID, nil, res)
	}
	return res, nil
}

func (s *loyaltyService) UpdateChallenge(ctx context.Context, id uint, in v1.ChallengeInput) (*v1.ChallengeItem, error) {
	existing, err := s.repo.FindChallengeByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thử thách")
		}
		return nil, err
	}
	parsed, err := s.parseChallengeInput(in)
	if err != nil {
		return nil, err
	}
	before := toChallengeItem(*existing, nil)
	c := existing.Challenge
	c.BranchID = parsed.BranchID
	c.Name = parsed.Name
	c.Description = parsed.Description
	c.Metric = parsed.Metric
	c.Target = parsed.Target
	c.StartDate = parsed.StartDate
	c.EndDate = parsed.EndDate
	c.RewardPoints = parsed.RewardPoints
	c.Status = parsed.Status
	if err := s.repo.UpdateChallenge(ctx, &c); err != nil {
		return nil, err
	}
	row, err := s.repo.FindChallengeByID(ctx, id)
	if err != nil {
		row = &repository.ChallengeRow{Challenge: c}
	}
	res := toChallengeItem(*row, nil)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityChallenge, id, before, res)
	}
	return res, nil
}

func (s *loyaltyService) DeleteChallenge(ctx context.Context, id uint) error {
	existing, err := s.repo.FindChallengeByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy thử thách")
		}
		return err
	}
	if err := s.repo.DeleteChallenge(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntityChallenge, id, toChallengeItem(*existing, nil), nil)
	}
	return nil
}

// challengeEnded: ngày kết thúc (theo ngày lịch) đã qua so với hôm nay – thử thách hết hạn từ 00:00 ngày kế tiếp.
func challengeEnded(endDate, now time.Time) bool {
	return endDate.Format(consts.DateLayout) < now.Format(consts.DateLayout)
}

func (s *loyaltyService) parseChallengeInput(in v1.ChallengeInput) (*entity.Challenge, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên thử thách")
	}
	metric := strings.ToUpper(strings.TrimSpace(in.Metric))
	if metric == "" {
		metric = entity.ChallengeMetricCheckins
	}
	if !slices.Contains(validChallengeMetrics, metric) {
		return nil, rescode.New(rescode.BadRequest, "Chỉ số thử thách không hợp lệ")
	}
	if in.Target <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Mục tiêu thử thách phải lớn hơn 0")
	}
	startDate, err := time.Parse(consts.DateLayout, strings.TrimSpace(in.StartDate))
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày bắt đầu phải có định dạng YYYY-MM-DD")
	}
	endDate, err := time.Parse(consts.DateLayout, strings.TrimSpace(in.EndDate))
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Ngày kết thúc phải có định dạng YYYY-MM-DD")
	}
	if endDate.Before(startDate) {
		return nil, rescode.New(rescode.BadRequest, "Ngày kết thúc phải sau hoặc bằng ngày bắt đầu")
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = consts.StatusActive
	}
	if status == consts.StatusActive && challengeEnded(endDate, s.now()) {
		return nil, rescode.New(rescode.BadRequest, "Không thể mở thử thách có ngày kết thúc trong quá khứ – hãy chọn ngày kết thúc từ hôm nay trở đi hoặc chuyển trạng thái sang Đã kết thúc")
	}
	rewardPts := in.RewardPoints
	if rewardPts < 0 {
		rewardPts = 0
	}
	return &entity.Challenge{
		BranchID:     in.BranchID,
		Name:         name,
		Description:  strings.TrimSpace(in.Description),
		Metric:       metric,
		Target:       in.Target,
		StartDate:    startDate,
		EndDate:      endDate,
		RewardPoints: rewardPts,
		Status:       status,
	}, nil
}

// ---- Admin: Adjust Member Points ------------------------------------------------

func (s *loyaltyService) AdjustMemberPoints(ctx context.Context, req *v1.AdjustMemberPointsReq) (*v1.AdjustMemberPointsRes, error) {
	if req.Points == 0 {
		return nil, rescode.New(rescode.BadRequest, "Số điểm điều chỉnh phải khác 0")
	}
	m, err := s.repo.FindMemberByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, m.HomeBranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền điều chỉnh điểm của hội viên chi nhánh khác")
	}
	if m.LoyaltyPoints+req.Points < 0 {
		return nil, rescode.New(rescode.InsufficientPoints, "Số điểm thưởng của hội viên không đủ để trừ")
	}
	reason := strings.ToUpper(strings.TrimSpace(req.Reason))
	if reason == "" || !slices.Contains(validLoyaltyReasons, reason) {
		reason = entity.LoyaltyReasonAdjust
	}
	uid := reqctx.UserID(ctx)
	var (
		newPoints int
		ledger    *entity.LoyaltyLedger
	)
	err = s.runTx(ctx, func(txCtx context.Context) error {
		pts, err := s.repo.AddMemberPoints(txCtx, m.ID, req.Points)
		if err != nil {
			return err
		}
		newPoints = pts
		ledger = &entity.LoyaltyLedger{
			MemberID:  m.ID,
			Points:    req.Points,
			Reason:    reason,
			RefType:   "MANUAL",
			Note:      strings.TrimSpace(req.Note),
			CreatedBy: uid,
		}
		return s.repo.CreateLedger(txCtx, ledger)
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityLoyaltyPoints, m.ID, m.LoyaltyPoints, newPoints)
	}
	return &v1.AdjustMemberPointsRes{
		MemberID: m.ID,
		Points:   newPoints,
		Ledger:   toLoyaltyLedgerItem(*ledger),
	}, nil
}

var ledgerOrderable = []string{"id", "created_at", "points"}

// ListLedgers: lịch sử biến động điểm cho admin, giới hạn trong chi nhánh được phân quyền.
func (s *loyaltyService) ListLedgers(ctx context.Context, req *v1.ListLoyaltyLedgersReq) (*v1.ListLoyaltyLedgersRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu của chi nhánh này")
	}
	q := req.PageReq.ToQuery(ledgerOrderable, "id")
	q.OrderBy = "l." + q.OrderBy
	rows, total, err := s.repo.ListLedgers(ctx, q, branchIDs, req.MemberID, strings.ToUpper(strings.TrimSpace(req.Reason)))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.LoyaltyLedgerRow) *v1.AdminLoyaltyLedgerItem {
		return &v1.AdminLoyaltyLedgerItem{
			LoyaltyLedgerItem: *toLoyaltyLedgerItem(r.LoyaltyLedger),
			MemberCode:        r.MemberCode,
			MemberName:        r.MemberName,
			BranchID:          r.BranchID,
			BranchName:        r.BranchName,
			CreatedByName:     r.CreatedByName,
		}
	}), total, q), nil
}

// ---- Member Endpoints -----------------------------------------------------------

func (s *loyaltyService) MemberLoyalty(ctx context.Context) (*v1.MemberLoyaltyRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	refCode := s.ensureReferralCode(ctx, m)
	ledgers, err := s.repo.ListMemberLedgers(ctx, m.ID, 50)
	if err != nil {
		return nil, err
	}
	rewards, err := s.repo.ListActiveRewards(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.MemberLoyaltyRes{
		Points:       m.LoyaltyPoints,
		ReferralCode: refCode,
		Ledgers:      dto.Map(ledgers, toLoyaltyLedgerItem),
		Rewards:      dto.Map(rewards, toLoyaltyRewardItem),
	}, nil
}

func randomCodeSuffix4() string {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "9999"
	}
	return strings.ToUpper(hex.EncodeToString(b[:]))
}

func (s *loyaltyService) RedeemReward(ctx context.Context, req *v1.RedeemRewardReq) (*v1.RedeemRewardRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	reward, err := s.repo.FindRewardByID(ctx, req.RewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy phần thưởng")
		}
		return nil, err
	}
	if reward.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Phần thưởng hiện không khả dụng")
	}
	if reward.Stock <= 0 {
		return nil, rescode.New(rescode.RewardOutOfStock, "Phần thưởng đã hết hàng")
	}
	if m.LoyaltyPoints < reward.CostPoints {
		return nil, rescode.New(rescode.InsufficientPoints, "Điểm thưởng của bạn không đủ để đổi phần thưởng này")
	}

	var (
		newPoints   int
		voucherCode string
		ledger      *entity.LoyaltyLedger
	)
	now := s.now()
	uid := reqctx.UserID(ctx)

	err = s.runTx(ctx, func(txCtx context.Context) error {
		ok, err := s.repo.DecrementRewardStock(txCtx, reward.ID)
		if err != nil {
			return err
		}
		if !ok {
			return rescode.New(rescode.RewardOutOfStock, "Phần thưởng đã hết hàng")
		}
		pts, err := s.repo.AddMemberPoints(txCtx, m.ID, -reward.CostPoints)
		if err != nil {
			return err
		}
		newPoints = pts
		reward.Stock--

		note := "Đổi quà: " + reward.Name
		if reward.RewardType == entity.RewardTypeVoucher {
			mCode := strings.TrimSpace(m.MemberCode)
			if mCode == "" {
				mCode = fmt.Sprintf("M%d", m.ID)
			}
			voucherCode = fmt.Sprintf("RWD-%s-%s", strings.ToUpper(mCode), randomCodeSuffix4())
			if len(voucherCode) > 30 {
				voucherCode = voucherCode[:30]
			}
			one := 1
			disc := reward.DiscountPct
			if disc <= 0 {
				disc = 10
			}
			promo := &entity.Promotion{
				BaseEntity: entity.BaseEntity{CreatedBy: uid, UpdatedBy: uid},
				Code:       voucherCode,
				Name:       fmt.Sprintf("Voucher đổi điểm (%s) - %s", mCode, reward.Name),
				Type:       entity.PromotionPercent,
				Value:      disc,
				MinAmount:  0,
				AppliesTo:  entity.AppliesAll,
				StartAt:    now,
				EndAt:      now.AddDate(0, 0, 30),
				UsageLimit: &one,
				UsedCount:  0,
				Status:     consts.StatusActive,
			}
			if err := s.repo.CreatePromotion(txCtx, promo); err != nil {
				return err
			}
			note = fmt.Sprintf("Đổi quà: %s (Mã: %s)", reward.Name, voucherCode)
		}

		ledger = &entity.LoyaltyLedger{
			MemberID:  m.ID,
			Points:    -reward.CostPoints,
			Reason:    entity.LoyaltyReasonRedeem,
			RefType:   "REWARD",
			RefID:     &reward.ID,
			Note:      note,
			CreatedBy: uid,
		}
		return s.repo.CreateLedger(txCtx, ledger)
	})
	if err != nil {
		return nil, err
	}

	return &v1.RedeemRewardRes{
		Points:      newPoints,
		VoucherCode: voucherCode,
		Reward:      toLoyaltyRewardItem(*reward),
		Ledger:      toLoyaltyLedgerItem(*ledger),
	}, nil
}

func (s *loyaltyService) MemberBadges(ctx context.Context) (v1.MemberBadgesRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	badges, err := s.repo.ListAllBadges(ctx)
	if err != nil {
		return nil, err
	}
	earnedRows, err := s.repo.ListMemberBadges(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	earnedMap := make(map[uint]time.Time, len(earnedRows))
	for _, eb := range earnedRows {
		earnedMap[eb.BadgeID] = eb.EarnedAt
	}

	// Auto-evaluate badge rules if not earned yet
	now := s.now()
	metricCache := map[string]int{}
	for _, b := range badges {
		if _, ok := earnedMap[b.ID]; ok {
			continue
		}
		var rule struct {
			Metric string `json:"metric"`
			Target int    `json:"target"`
		}
		if len(b.Rule) > 0 && json.Unmarshal(b.Rule, &rule) == nil && rule.Metric != "" && rule.Target > 0 {
			cnt, cached := metricCache[rule.Metric]
			if !cached {
				cnt, _ = s.repo.CountMemberMetric(ctx, m.ID, rule.Metric, nil, nil)
				metricCache[rule.Metric] = cnt
			}
			if cnt >= rule.Target {
				mb := &entity.MemberBadge{MemberID: m.ID, BadgeID: b.ID, EarnedAt: now}
				if err := s.repo.AwardBadge(ctx, mb); err == nil {
					earnedMap[b.ID] = now
				}
			}
		}
	}

	out := make([]*v1.BadgeItem, 0, len(badges))
	for _, b := range badges {
		var earnedAt *time.Time
		if t, ok := earnedMap[b.ID]; ok {
			tCopy := t
			earnedAt = &tCopy
		}
		out = append(out, toBadgeItem(b, earnedAt))
	}
	return out, nil
}

func (s *loyaltyService) MemberChallenges(ctx context.Context) (v1.MemberChallengesRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListActiveChallenges(ctx, m.HomeBranchID)
	if err != nil {
		return nil, err
	}
	participants, err := s.repo.ListMemberParticipants(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	partByChallenge := make(map[uint]*entity.ChallengeParticipant, len(participants))
	for i := range participants {
		cp := participants[i]
		partByChallenge[cp.ChallengeID] = &cp
	}

	out := make([]*v1.ChallengeItem, 0, len(rows))
	for _, r := range rows {
		cp := partByChallenge[r.ID]
		out = append(out, toChallengeItem(r, cp))
	}
	return out, nil
}

func (s *loyaltyService) JoinChallenge(ctx context.Context, req *v1.JoinChallengeReq) (*v1.ChallengeItem, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	chRow, err := s.repo.FindChallengeByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy thử thách")
		}
		return nil, err
	}
	if chRow.Status != consts.StatusActive {
		return nil, rescode.New(rescode.BadRequest, "Thử thách này không còn mở đăng ký")
	}
	now := s.now()
	if challengeEnded(chRow.EndDate, now) {
		return nil, rescode.New(rescode.BadRequest, "Thử thách đã kết thúc, không thể tham gia")
	}
	if chRow.BranchID != nil && *chRow.BranchID != m.HomeBranchID {
		return nil, rescode.New(rescode.PermissionDenied, "Thử thách này chỉ dành cho hội viên của chi nhánh khác")
	}

	var cp *entity.ChallengeParticipant
	err = s.runTx(ctx, func(txCtx context.Context) error {
		existing, err := s.repo.FindChallengeParticipant(txCtx, chRow.ID, m.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		isNew := false
		if existing == nil {
			isNew = true
			existing = &entity.ChallengeParticipant{
				ChallengeID: chRow.ID,
				MemberID:    m.ID,
				JoinedAt:    now,
			}
		}

		start := chRow.StartDate
		end := chRow.EndDate
		progress, err := s.repo.CountMemberMetric(txCtx, m.ID, chRow.Metric, &start, &end)
		if err != nil {
			return err
		}
		existing.Progress = progress

		if progress >= chRow.Target && existing.CompletedAt == nil {
			completedAt := now
			existing.CompletedAt = &completedAt
			if chRow.RewardPoints > 0 {
				if _, err := s.repo.AddMemberPoints(txCtx, m.ID, chRow.RewardPoints); err != nil {
					return err
				}
				chID := chRow.ID
				ledger := &entity.LoyaltyLedger{
					MemberID:  m.ID,
					Points:    chRow.RewardPoints,
					Reason:    entity.LoyaltyReasonChallenge,
					RefType:   "CHALLENGE",
					RefID:     &chID,
					Note:      "Hoàn thành thử thách: " + chRow.Name,
					CreatedBy: reqctx.UserID(ctx),
				}
				if err := s.repo.CreateLedger(txCtx, ledger); err != nil {
					return err
				}
			}
		}

		if err := s.repo.SaveChallengeParticipant(txCtx, existing); err != nil {
			return err
		}
		if isNew {
			chRow.ParticipantCount++
		}
		cp = existing
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toChallengeItem(*chRow, cp), nil
}

func (s *loyaltyService) MemberLeaderboard(ctx context.Context, req *v1.MemberLeaderboardReq) (*v1.MemberLeaderboardRes, error) {
	if _, err := s.currentMember(ctx); err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}
	byPts, err := s.repo.TopMembersByPoints(ctx, limit)
	if err != nil {
		return nil, err
	}
	byCheckins, err := s.repo.TopMembersByCheckins(ctx, limit)
	if err != nil {
		return nil, err
	}
	ptsEntries := toLeaderboardEntries(byPts)
	chkEntries := toLeaderboardEntries(byCheckins)
	return &v1.MemberLeaderboardRes{
		ByPoints:   ptsEntries,
		ByCheckins: chkEntries,
		Items:      ptsEntries,
	}, nil
}

func (s *loyaltyService) MemberReferral(ctx context.Context) (*v1.MemberReferralRes, error) {
	m, err := s.currentMember(ctx)
	if err != nil {
		return nil, err
	}
	code := s.ensureReferralCode(ctx, m)
	friends, err := s.repo.ListReferredMembers(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	totalPts, err := s.repo.SumReferralPoints(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	refs := make([]*v1.ReferredFriend, 0, len(friends))
	for _, f := range friends {
		refs = append(refs, &v1.ReferredFriend{
			MemberID:   f.ID,
			MemberCode: f.MemberCode,
			FullName:   f.FullName,
			JoinedAt:   f.CreatedAt,
		})
	}
	return &v1.MemberReferralRes{
		ReferralCode:  code,
		ReferredCount: int64(len(friends)),
		TotalPoints:   totalPts,
		Referrals:     refs,
	}, nil
}

// ---- Mappers --------------------------------------------------------------------

func toLoyaltyLedgerItem(l entity.LoyaltyLedger) *v1.LoyaltyLedgerItem {
	return &v1.LoyaltyLedgerItem{
		ID:        l.ID,
		MemberID:  l.MemberID,
		Points:    l.Points,
		Reason:    l.Reason,
		RefType:   l.RefType,
		RefID:     l.RefID,
		Note:      l.Note,
		CreatedBy: l.CreatedBy,
		CreatedAt: l.CreatedAt,
	}
}

func toLoyaltyRewardItem(r entity.LoyaltyReward) *v1.LoyaltyRewardItem {
	return &v1.LoyaltyRewardItem{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		CostPoints:  r.CostPoints,
		RewardType:  r.RewardType,
		RefID:       r.RefID,
		DiscountPct: r.DiscountPct,
		Stock:       r.Stock,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toBadgeItem(b entity.Badge, earnedAt *time.Time) *v1.BadgeItem {
	ruleRaw := json.RawMessage("{}")
	if len(b.Rule) > 0 {
		ruleRaw = json.RawMessage(b.Rule)
	}
	return &v1.BadgeItem{
		ID:          b.ID,
		Code:        b.Code,
		Name:        b.Name,
		Description: b.Description,
		Icon:        b.Icon,
		Rule:        ruleRaw,
		Earned:      earnedAt != nil,
		EarnedAt:    earnedAt,
		CreatedAt:   b.CreatedAt,
	}
}

func toChallengeItem(r repository.ChallengeRow, cp *entity.ChallengeParticipant) *v1.ChallengeItem {
	item := &v1.ChallengeItem{
		ID:               r.ID,
		BranchID:         r.BranchID,
		BranchName:       r.BranchName,
		Name:             r.Name,
		Description:      r.Description,
		Metric:           r.Metric,
		Target:           r.Target,
		StartDate:        r.StartDate.Format(consts.DateLayout),
		EndDate:          r.EndDate.Format(consts.DateLayout),
		RewardPoints:     r.RewardPoints,
		Status:           r.Status,
		ParticipantCount: r.ParticipantCount,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
	if cp != nil {
		item.Joined = true
		item.Progress = cp.Progress
		item.CompletedAt = cp.CompletedAt
		jAt := cp.JoinedAt
		item.JoinedAt = &jAt
	}
	return item
}

func toLeaderboardEntries(rows []repository.LeaderboardRow) []*v1.LeaderboardEntry {
	out := make([]*v1.LeaderboardEntry, 0, len(rows))
	for i, r := range rows {
		out = append(out, &v1.LeaderboardEntry{
			Rank:          i + 1,
			MemberID:      r.MemberID,
			MemberCode:    r.MemberCode,
			FullName:      r.FullName,
			AvatarURL:     r.AvatarURL,
			LoyaltyPoints: r.LoyaltyPoints,
			CheckinCount:  r.CheckinCount,
		})
	}
	return out
}
