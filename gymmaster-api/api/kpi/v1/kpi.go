package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type BranchOccupancyItem struct {
	BranchID     uint    `json:"branch_id"`
	BranchCode   string  `json:"branch_code"`
	BranchName   string  `json:"branch_name"`
	PresentCount int     `json:"present_count"`
	Capacity     int     `json:"capacity"`
	OccupancyPct float64 `json:"occupancy_pct"`
	Level        string  `json:"level"` // QUIET | NORMAL | BUSY | FULL
}

type SalesTargetItem struct {
	ID          uint      `json:"id"`
	BranchID    uint      `json:"branch_id"`
	BranchName  string    `json:"branch_name,omitempty"`
	StaffID     *uint     `json:"staff_id"`
	StaffName   string    `json:"staff_name,omitempty"`
	Period      string    `json:"period"`
	Metric      string    `json:"metric"`
	TargetValue float64   `json:"target_value"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type KPIMetricComparison struct {
	Metric         string  `json:"metric"`
	Label          string  `json:"label"`
	Actual         float64 `json:"actual"`
	TargetValue    float64 `json:"target_value"`
	AchievementPct float64 `json:"achievement_pct"`
	// HasTarget = false → chưa đặt chỉ tiêu (FE hiện "Chưa đặt chỉ tiêu" thay vì 0%).
	HasTarget bool `json:"has_target"`
	// TargetBranchCount: số chi nhánh có chỉ tiêu cho chỉ số này. Khi xem toàn hệ thống, Actual chỉ cộng
	// các chi nhánh này (so cùng phạm vi với chỉ tiêu) – nếu chưa chi nhánh nào đặt thì Actual là toàn hệ thống.
	TargetBranchCount int `json:"target_branch_count"`
}

type CohortRow struct {
	CohortMonth      string    `json:"cohort_month"`
	Size             int       `json:"size"`
	RetentionByMonth []float64 `json:"retention_by_month"`
	ActiveCounts     []int     `json:"active_counts"`
}

// ---- Public & Member ------------------------------------------------------------

type PublicOccupancyReq struct {
	g.Meta `path:"/branches/occupancy" method:"get" tags:"Công khai" summary:"Mật độ hội viên đang tập realtime tại các chi nhánh"`
}
type PublicOccupancyRes []*BranchOccupancyItem

type MemberOccupancyReq struct {
	g.Meta `path:"/occupancy" method:"get" tags:"Cổng hội viên" summary:"Mật độ phòng tập realtime" role:"MEMBER"`
}
type MemberOccupancyRes []*BranchOccupancyItem

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListSalesTargetsReq struct {
	g.Meta `path:"/sales-targets" method:"get" tags:"KPI & Phân tích" summary:"Danh sách chỉ tiêu doanh số KPI" perm:"kpi.manage,report.view"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query"`
	Period   string `json:"period"    in:"query"`
	Metric   string `json:"metric"    in:"query"`
}
type ListSalesTargetsRes = dto.PageRes[*SalesTargetItem]

type CreateSalesTargetReq struct {
	g.Meta      `path:"/sales-targets" method:"post" tags:"KPI & Phân tích" summary:"Thiết lập chỉ tiêu KPI tháng" perm:"kpi.manage"`
	BranchID    uint    `json:"branch_id"    v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	StaffID     *uint   `json:"staff_id"`
	Period      string  `json:"period"       v:"required#Vui lòng nhập kỳ KPI (YYYY-MM)"`
	Metric      string  `json:"metric"       v:"required|in:REVENUE,NEW_MEMBERS,RENEWALS,PT_SALES#Vui lòng chọn chỉ số KPI|Chỉ số KPI không hợp lệ"`
	TargetValue float64 `json:"target_value" v:"required|min:0|max:999999999999#Vui lòng nhập giá trị mục tiêu|Giá trị mục tiêu không được âm|Giá trị mục tiêu vượt quá giới hạn cho phép"`
}
type CreateSalesTargetRes = SalesTargetItem

type DeleteSalesTargetReq struct {
	g.Meta `path:"/sales-targets/{id}" method:"delete" tags:"KPI & Phân tích" summary:"Xóa chỉ tiêu KPI" perm:"kpi.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteKPIRes struct{}

type KPIReportReq struct {
	g.Meta   `path:"/reports/kpi" method:"get" tags:"KPI & Phân tích" summary:"Báo cáo thực tế vs mục tiêu KPI" perm:"kpi.manage,report.view"`
	Period   string `json:"period"    in:"query"`
	BranchID uint   `json:"branch_id" in:"query"`
}
type KPIReportRes struct {
	Period   string                 `json:"period"`
	BranchID uint                   `json:"branch_id"`
	Metrics  []*KPIMetricComparison `json:"metrics"`
	Items    []*KPIMetricComparison `json:"items"`
}

type CohortReportReq struct {
	g.Meta   `path:"/reports/cohort" method:"get" tags:"KPI & Phân tích" summary:"Báo cáo ma trận giữ chân hội viên (Cohort Retention)" perm:"kpi.manage,report.view"`
	BranchID uint `json:"branch_id" in:"query"`
	Months   int  `json:"months"    in:"query" d:"6"`
}
type CohortReportRes struct {
	Months  int          `json:"months"`
	Cohorts []*CohortRow `json:"cohorts"`
	Items   []*CohortRow `json:"items"`
}
