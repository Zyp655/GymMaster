package v1

import "github.com/gogf/gf/v2/frame/g"

// PublicPlan là gói tập hiển thị trên trang công khai.
type PublicPlan struct {
	ID              uint    `json:"id"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	DurationDays    int     `json:"duration_days"`
	SessionCount    *int    `json:"session_count"`
	Price           float64 `json:"price"`
	AccessScope     string  `json:"access_scope"`
	MaxFreezeDays   int     `json:"max_freeze_days"`
	IncludesClasses bool    `json:"includes_classes"`
}

// PublicPTPackage là gói PT hiển thị trên trang công khai.
type PublicPTPackage struct {
	ID             uint    `json:"id"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	SessionCount   int     `json:"session_count"`
	SessionMinutes int     `json:"session_minutes"`
	ValidityDays   int     `json:"validity_days"`
	Price          float64 `json:"price"`
}

type PublicBranch struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Phone     string `json:"phone"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

type PlansReq struct {
	g.Meta `path:"/plans" method:"get" tags:"Công khai" summary:"Bảng giá gói tập và gói PT đang bán"`
}
type PlansRes struct {
	MembershipPlans []*PublicPlan      `json:"membership_plans"`
	PTPackages      []*PublicPTPackage `json:"pt_packages"`
}

type BranchesReq struct {
	g.Meta `path:"/branches" method:"get" tags:"Công khai" summary:"Danh sách chi nhánh đang hoạt động"`
}
type BranchesRes []*PublicBranch
