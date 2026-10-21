package v1

import (
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type LoyaltyLedgerItem struct {
	ID        uint      `json:"id"`
	MemberID  uint      `json:"member_id"`
	Points    int       `json:"points"`
	Reason    string    `json:"reason"`
	RefType   string    `json:"ref_type"`
	RefID     *uint     `json:"ref_id"`
	Note      string    `json:"note"`
	CreatedBy *uint     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type LoyaltyRewardItem struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CostPoints  int       `json:"cost_points"`
	RewardType  string    `json:"reward_type"`
	RefID       *uint     `json:"ref_id"`
	DiscountPct float64   `json:"discount_pct"`
	Stock       int       `json:"stock"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LoyaltyRewardInput struct {
	Name        string  `json:"name"         v:"required|max-length:150#Vui lòng nhập tên phần thưởng"`
	Description string  `json:"description"  v:"max-length:500"`
	CostPoints  int     `json:"cost_points"  v:"required|min:1#Điểm quy đổi phải lớn hơn 0"`
	RewardType  string  `json:"reward_type"  d:"VOUCHER" v:"required|in:VOUCHER,PRODUCT#Loại phần thưởng không hợp lệ"`
	RefID       *uint   `json:"ref_id"`
	DiscountPct float64 `json:"discount_pct" d:"10"      v:"between:0,100"`
	Stock       int     `json:"stock"        d:"100"     v:"min:0#Số lượng tồn phải >= 0"`
	Status      string  `json:"status"       d:"ACTIVE"  v:"in:ACTIVE,INACTIVE"`
}

type BadgeItem struct {
	ID          uint            `json:"id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	Rule        json.RawMessage `json:"rule"`
	Earned      bool            `json:"earned"`
	EarnedAt    *time.Time      `json:"earned_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type BadgeInput struct {
	Code        string         `json:"code"        v:"required|max-length:50#Vui lòng nhập mã huy hiệu"`
	Name        string         `json:"name"        v:"required|max-length:150#Vui lòng nhập tên huy hiệu"`
	Description string         `json:"description" v:"max-length:255"`
	Icon        string         `json:"icon"        d:"trophy"`
	Rule        map[string]any `json:"rule"`
}

type ChallengeItem struct {
	ID               uint       `json:"id"`
	BranchID         *uint      `json:"branch_id"`
	BranchName       string     `json:"branch_name,omitempty"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Metric           string     `json:"metric"`
	Target           int        `json:"target"`
	StartDate        string     `json:"start_date"`
	EndDate          string     `json:"end_date"`
	RewardPoints     int        `json:"reward_points"`
	Status           string     `json:"status"`
	ParticipantCount int64      `json:"participant_count,omitempty"`
	Joined           bool       `json:"joined"`
	Progress         int        `json:"progress"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	JoinedAt         *time.Time `json:"joined_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ChallengeInput struct {
	BranchID     *uint  `json:"branch_id"`
	Name         string `json:"name"          v:"required|max-length:150#Vui lòng nhập tên thử thách"`
	Description  string `json:"description"   v:"max-length:500"`
	Metric       string `json:"metric"        d:"CHECKINS" v:"required|in:CHECKINS,CLASSES,PT_SESSIONS#Chỉ số thử thách không hợp lệ"`
	Target       int    `json:"target"        v:"required|min:1#Mục tiêu phải lớn hơn 0"`
	StartDate    string `json:"start_date"    v:"required#Vui lòng chọn ngày bắt đầu"`
	EndDate      string `json:"end_date"      v:"required#Vui lòng chọn ngày kết thúc"`
	RewardPoints int    `json:"reward_points" d:"100" v:"min:0"`
	Status       string `json:"status"        d:"ACTIVE" v:"in:ACTIVE,ENDED,DRAFT"`
}

type LeaderboardEntry struct {
	Rank          int    `json:"rank"`
	MemberID      uint   `json:"member_id"`
	MemberCode    string `json:"member_code"`
	FullName      string `json:"full_name"`
	AvatarURL     string `json:"avatar_url"`
	LoyaltyPoints int    `json:"loyalty_points"`
	CheckinCount  int64  `json:"checkin_count"`
}

type ReferredFriend struct {
	MemberID   uint      `json:"member_id"`
	MemberCode string    `json:"member_code"`
	FullName   string    `json:"full_name"`
	JoinedAt   time.Time `json:"joined_at"`
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListLoyaltyRewardsReq struct {
	g.Meta `path:"/loyalty-rewards" method:"get" tags:"Tích điểm & Thử thách" summary:"Danh sách quà đổi điểm" perm:"loyalty.view,loyalty.manage"`
	dto.PageReq
	Status     string `json:"status"      in:"query"`
	RewardType string `json:"reward_type" in:"query"`
}
type ListLoyaltyRewardsRes = dto.PageRes[*LoyaltyRewardItem]

type CreateLoyaltyRewardReq struct {
	g.Meta `path:"/loyalty-rewards" method:"post" tags:"Tích điểm & Thử thách" summary:"Thêm quà đổi điểm" perm:"loyalty.manage"`
	LoyaltyRewardInput
}
type CreateLoyaltyRewardRes = LoyaltyRewardItem

type UpdateLoyaltyRewardReq struct {
	g.Meta `path:"/loyalty-rewards/{id}" method:"put" tags:"Tích điểm & Thử thách" summary:"Cập nhật quà đổi điểm" perm:"loyalty.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	LoyaltyRewardInput
}
type UpdateLoyaltyRewardRes = LoyaltyRewardItem

type DeleteLoyaltyRewardReq struct {
	g.Meta `path:"/loyalty-rewards/{id}" method:"delete" tags:"Tích điểm & Thử thách" summary:"Xóa quà đổi điểm" perm:"loyalty.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteLoyaltyRes struct{}

type ListBadgesReq struct {
	g.Meta `path:"/badges" method:"get" tags:"Tích điểm & Thử thách" summary:"Danh sách huy hiệu" perm:"loyalty.view,loyalty.manage"`
	dto.PageReq
}
type ListBadgesRes = dto.PageRes[*BadgeItem]

type CreateBadgeReq struct {
	g.Meta `path:"/badges" method:"post" tags:"Tích điểm & Thử thách" summary:"Thêm huy hiệu mới" perm:"loyalty.manage"`
	BadgeInput
}
type CreateBadgeRes = BadgeItem

type DeleteBadgeReq struct {
	g.Meta `path:"/badges/{id}" method:"delete" tags:"Tích điểm & Thử thách" summary:"Xóa huy hiệu" perm:"loyalty.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type ListChallengesReq struct {
	g.Meta `path:"/challenges" method:"get" tags:"Tích điểm & Thử thách" summary:"Danh sách thử thách" perm:"loyalty.view,loyalty.manage"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListChallengesRes = dto.PageRes[*ChallengeItem]

type CreateChallengeReq struct {
	g.Meta `path:"/challenges" method:"post" tags:"Tích điểm & Thử thách" summary:"Tạo thử thách mới" perm:"loyalty.manage"`
	ChallengeInput
}
type CreateChallengeRes = ChallengeItem

type UpdateChallengeReq struct {
	g.Meta `path:"/challenges/{id}" method:"put" tags:"Tích điểm & Thử thách" summary:"Cập nhật thử thách" perm:"loyalty.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ChallengeInput
}
type UpdateChallengeRes = ChallengeItem

type DeleteChallengeReq struct {
	g.Meta `path:"/challenges/{id}" method:"delete" tags:"Tích điểm & Thử thách" summary:"Xóa thử thách" perm:"loyalty.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}

type AdjustMemberPointsReq struct {
	g.Meta `path:"/members/{id}/points/adjust" method:"post" tags:"Tích điểm & Thử thách" summary:"Điều chỉnh điểm thưởng hội viên" perm:"loyalty.manage"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Points int    `json:"points" v:"required#Vui lòng nhập số điểm điều chỉnh"`
	Reason string `json:"reason" d:"ADJUST"`
	Note   string `json:"note"   v:"max-length:255"`
}
type AdjustMemberPointsRes struct {
	MemberID uint               `json:"member_id"`
	Points   int                `json:"points"`
	Ledger   *LoyaltyLedgerItem `json:"ledger"`
}

// AdminLoyaltyLedgerItem: dòng lịch sử biến động điểm kèm thông tin hội viên / người thao tác.
type AdminLoyaltyLedgerItem struct {
	LoyaltyLedgerItem
	MemberCode    string `json:"member_code"`
	MemberName    string `json:"member_name"`
	BranchID      uint   `json:"branch_id"`
	BranchName    string `json:"branch_name"`
	CreatedByName string `json:"created_by_name"`
}

type ListLoyaltyLedgersReq struct {
	g.Meta `path:"/loyalty-ledgers" method:"get" tags:"Tích điểm & Thử thách" summary:"Lịch sử biến động điểm thưởng (theo chi nhánh được phân quyền)" perm:"loyalty.view,loyalty.manage"`
	dto.PageReq
	MemberID uint   `json:"member_id" in:"query"`
	BranchID uint   `json:"branch_id" in:"query"`
	Reason   string `json:"reason"    in:"query" v:"in:CHECKIN,RENEW,REFERRAL,REVIEW,REDEEM,CHALLENGE,ADJUST#Lý do biến động điểm không hợp lệ"`
}
type ListLoyaltyLedgersRes = dto.PageRes[*AdminLoyaltyLedgerItem]

// ---- Member (/v1/member) --------------------------------------------------------

type MemberLoyaltyReq struct {
	g.Meta `path:"/loyalty" method:"get" tags:"Cổng hội viên" summary:"Tổng quan điểm thưởng & quà quy đổi" role:"MEMBER"`
}
type MemberLoyaltyRes struct {
	Points       int                  `json:"points"`
	ReferralCode string               `json:"referral_code"`
	Ledgers      []*LoyaltyLedgerItem `json:"ledgers"`
	Rewards      []*LoyaltyRewardItem `json:"rewards"`
}

type RedeemRewardReq struct {
	g.Meta   `path:"/loyalty/redeem" method:"post" tags:"Cổng hội viên" summary:"Đổi quà bằng điểm thưởng" role:"MEMBER"`
	RewardID uint `json:"reward_id" v:"required|min:1#Vui lòng chọn phần thưởng"`
}
type RedeemRewardRes struct {
	Points      int                `json:"points"`
	VoucherCode string             `json:"voucher_code,omitempty"`
	Reward      *LoyaltyRewardItem `json:"reward"`
	Ledger      *LoyaltyLedgerItem `json:"ledger"`
}

type MemberBadgesReq struct {
	g.Meta `path:"/badges" method:"get" tags:"Cổng hội viên" summary:"Danh sách huy hiệu của tôi" role:"MEMBER"`
}
type MemberBadgesRes []*BadgeItem

type MemberChallengesReq struct {
	g.Meta `path:"/challenges" method:"get" tags:"Cổng hội viên" summary:"Danh sách thử thách đang diễn ra" role:"MEMBER"`
}
type MemberChallengesRes []*ChallengeItem

type JoinChallengeReq struct {
	g.Meta `path:"/challenges/{id}/join" method:"post" tags:"Cổng hội viên" summary:"Tham gia thử thách" role:"MEMBER"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type JoinChallengeRes = ChallengeItem

type MemberLeaderboardReq struct {
	g.Meta `path:"/leaderboard" method:"get" tags:"Cổng hội viên" summary:"Bảng xếp hạng hội viên tích cực" role:"MEMBER"`
	Limit  int `json:"limit" in:"query" d:"10"`
}
type MemberLeaderboardRes struct {
	ByPoints   []*LeaderboardEntry `json:"by_points"`
	ByCheckins []*LeaderboardEntry `json:"by_checkins"`
	Items      []*LeaderboardEntry `json:"items"`
}

type MemberReferralReq struct {
	g.Meta `path:"/referral" method:"get" tags:"Cổng hội viên" summary:"Mã giới thiệu bạn bè & thống kê" role:"MEMBER"`
}
type MemberReferralRes struct {
	ReferralCode  string            `json:"referral_code"`
	ReferredCount int64             `json:"referred_count"`
	TotalPoints   int               `json:"total_points"`
	Referrals     []*ReferredFriend `json:"referrals"`
}
