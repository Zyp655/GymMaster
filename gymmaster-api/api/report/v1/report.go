package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// Filter là bộ lọc chung của mọi báo cáo. from/to là ngày giờ Việt Nam (tính cả hai đầu).
// Bỏ trống: from = ngày 1 tháng hiện tại, to = hôm nay.
type Filter struct {
	BranchID uint   `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh (bỏ trống = mọi chi nhánh được phép xem)"`
	From     string `json:"from"      in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"Từ ngày (YYYY-MM-DD)"`
	To       string `json:"to"        in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"Đến ngày (YYYY-MM-DD, tính cả ngày)"`
}

// GroupBy chọn độ mịn của trục thời gian.
type GroupBy struct {
	GroupBy string `json:"group_by" in:"query" d:"day" v:"in:day,week,month#Nhóm theo phải là day, week hoặc month" dc:"day | week | month"`
}

// Period mô tả khoảng thời gian thực tế của báo cáo (sau khi áp mặc định).
type Period struct {
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days"`
}

type NameValue struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Pct   float64 `json:"pct"` // % trên tổng (0-100, làm tròn 1 chữ số)
}

type NameCount struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Count int64   `json:"count"`
	Pct   float64 `json:"pct"`
}

// ---------------------------------------------------------------------------
// Overview (dashboard)
// ---------------------------------------------------------------------------

type OverviewReq struct {
	g.Meta   `path:"/reports/overview" method:"get" tags:"Báo cáo" summary:"KPI tổng quan cho dashboard" perm:"report.view"`
	BranchID uint `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh"`
}

type Overview struct {
	Date    string          `json:"date"`
	Revenue OverviewRevenue `json:"revenue"`
	Members OverviewMembers `json:"members"`
	Checkin OverviewCheckin `json:"checkins"`
	Classes OverviewClasses `json:"classes"`
	PT      OverviewPT      `json:"pt"`
	// Việc cần xử lý
	PendingFreezes int64          `json:"pending_freezes"`
	LowStock       int64          `json:"low_stock_count"`
	LowStockItems  []LowStockItem `json:"low_stock_items"`
}

type OverviewRevenue struct {
	Today           float64  `json:"today"`
	Month           float64  `json:"month"`
	PrevMonthToDate float64  `json:"prev_month_to_date"`
	PrevMonth       float64  `json:"prev_month"`
	ChangePct       *float64 `json:"change_pct"` // month so với prev_month_to_date; null khi kỳ trước = 0
}

type OverviewMembers struct {
	Active       int64    `json:"active"`
	NewThisMonth int64    `json:"new_this_month"`
	NewPrevMonth int64    `json:"new_prev_month"`
	NewChangePct *float64 `json:"new_change_pct"`
	ExpiringIn7d int64    `json:"expiring_7d"`
}

type OverviewCheckin struct {
	Today int64 `json:"today" dc:"Lượt check-in hợp lệ trong ngày"`
	// Present dùng CHUNG định nghĩa với Mật độ (/occupancy): người check-in ALLOWED chưa checkout trong
	// present_window_hours giờ gần nhất (từ present_since), đếm theo người.
	Present            int64     `json:"present" dc:"Đang trong phòng tập (cùng định nghĩa Mật độ)"`
	PresentSince       time.Time `json:"present_since" dc:"Mốc bắt đầu cửa sổ đang trong phòng tập"`
	PresentWindowHours int       `json:"present_window_hours" dc:"Độ dài cửa sổ (giờ)"`
}

type OverviewClasses struct {
	Today    int64   `json:"today"`
	Capacity int64   `json:"capacity"`
	Booked   int64   `json:"booked"`
	FillRate float64 `json:"fill_rate"` // %
}

type OverviewPT struct {
	Today     int64 `json:"today"`
	Completed int64 `json:"completed"`
}

type LowStockItem struct {
	ProductID  uint   `json:"product_id"`
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	BranchID   uint   `json:"branch_id"`
	BranchName string `json:"branch_name"`
	Quantity   int    `json:"quantity"`
	MinStock   int    `json:"min_stock"`
}

type OverviewRes = Overview

// ---------------------------------------------------------------------------
// Revenue
// ---------------------------------------------------------------------------

type RevenueReq struct {
	g.Meta `path:"/reports/revenue" method:"get" tags:"Báo cáo" summary:"Doanh thu thực thu theo thời gian, loại, phương thức, chi nhánh" perm:"report.view"`
	Filter
	GroupBy
}

type RevenuePoint struct {
	Period     string  `json:"period"` // ngày đầu của nhóm (YYYY-MM-DD)
	Label      string  `json:"label"`  // nhãn hiển thị trục X
	Total      float64 `json:"total"`
	Membership float64 `json:"membership"`
	PT         float64 `json:"pt"`
	Product    float64 `json:"product"`
	Fee        float64 `json:"fee"`
	Other      float64 `json:"other"`
	Payments   float64 `json:"payments"`
	Refunds    float64 `json:"refunds"`
}

type RevenueTotals struct {
	Net      float64 `json:"net"` // thực thu = payments − refunds
	Payments float64 `json:"payments"`
	Refunds  float64 `json:"refunds"`
}

type RevenueCompare struct {
	Period
	Net       float64  `json:"net"`
	ChangePct *float64 `json:"change_pct"`
}

type BranchAmount struct {
	BranchID   uint    `json:"branch_id"`
	BranchName string  `json:"branch_name"`
	Amount     float64 `json:"amount"`
	Pct        float64 `json:"pct"`
}

type RevenueRes struct {
	Period   Period         `json:"period"`
	GroupBy  string         `json:"group_by"`
	Totals   RevenueTotals  `json:"totals"`
	Previous RevenueCompare `json:"previous"`
	Series   []RevenuePoint `json:"series"`
	ByType   []NameValue    `json:"by_type"`
	ByMethod []NameValue    `json:"by_method"`
	ByBranch []BranchAmount `json:"by_branch"`
}

// ---------------------------------------------------------------------------
// Members
// ---------------------------------------------------------------------------

type MembersReq struct {
	g.Meta `path:"/reports/members" method:"get" tags:"Báo cáo" summary:"Hội viên mới, phân bố, trạng thái" perm:"report.view"`
	Filter
	GroupBy
	Population string `json:"population" in:"query" d:"new" v:"in:new,all#population phải là new hoặc all" dc:"Phân bố trên: new = hội viên mới trong kỳ, all = toàn bộ"`
}

type CountPoint struct {
	Period string `json:"period"`
	Label  string `json:"label"`
	Count  int64  `json:"count"`
}

type MembersRes struct {
	Period       Period       `json:"period"`
	GroupBy      string       `json:"group_by"`
	Population   string       `json:"population"`
	NewTotal     int64        `json:"new_total"`
	NewSeries    []CountPoint `json:"new_series"`
	BySource     []NameCount  `json:"by_source"`
	ByGender     []NameCount  `json:"by_gender"`
	ByAge        []NameCount  `json:"by_age"`
	ByBranch     []NameCount  `json:"by_branch"`
	Status       []NameCount  `json:"status"` // ACTIVE | FROZEN | PENDING | EXPIRED | NONE (hiện tại, không phụ thuộc kỳ)
	TotalMembers int64        `json:"total_members"`
}

// ---------------------------------------------------------------------------
// Retention
// ---------------------------------------------------------------------------

type RetentionReq struct {
	g.Meta `path:"/reports/retention" method:"get" tags:"Báo cáo" summary:"Tỷ lệ gia hạn & churn" perm:"report.view"`
	Filter
	WindowDays int `json:"window_days" in:"query" d:"30" v:"between:1,180#Số ngày cửa sổ gia hạn từ 1 đến 180" dc:"Gia hạn trong vòng N ngày sau hết hạn (mặc định 30)"`
}

type RetentionPoint struct {
	Month       string  `json:"month"` // YYYY-MM
	Label       string  `json:"label"`
	Expired     int64   `json:"expired"`      // số hội viên có gói hết hạn
	Renewed     int64   `json:"renewed"`      // đã có gói mới trong cửa sổ
	Pending     int64   `json:"pending"`      // chưa gia hạn nhưng còn trong cửa sổ
	Churned     int64   `json:"churned"`      // quá cửa sổ mà không gia hạn
	RenewalRate float64 `json:"renewal_rate"` // % renewed / expired
	ChurnRate   float64 `json:"churn_rate"`   // % churned / expired
}

type RetentionRes struct {
	Period     Period           `json:"period"`
	WindowDays int              `json:"window_days"`
	Summary    RetentionPoint   `json:"summary"`
	Months     []RetentionPoint `json:"months"`
}

// ---------------------------------------------------------------------------
// Check-in heatmap
// ---------------------------------------------------------------------------

type CheckinHeatmapReq struct {
	g.Meta `path:"/reports/checkin-heatmap" method:"get" tags:"Báo cáo" summary:"Heatmap check-in 7 thứ × 24 giờ" perm:"report.view"`
	Filter
}

type HeatmapDay struct {
	Dow    int     `json:"dow"`   // 1 = Thứ 2 … 7 = Chủ nhật
	Label  string  `json:"label"` // "T2" … "CN"
	Hours  []int64 `json:"hours"` // 24 phần tử, index = giờ
	Total  int64   `json:"total"`
	AvgDay float64 `json:"avg_per_day"` // trung bình lượt / ngày thứ đó trong kỳ
}

type PeakSlot struct {
	Dow   int    `json:"dow"`
	Hour  int    `json:"hour"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type CheckinHeatmapRes struct {
	Period        Period       `json:"period"`
	Matrix        []HeatmapDay `json:"matrix"`  // 7 dòng
	ByHour        []int64      `json:"by_hour"` // 24 phần tử
	Max           int64        `json:"max"`     // giá trị ô lớn nhất (để tô màu)
	Total         int64        `json:"total"`
	Denied        int64        `json:"denied"`
	UniqueMembers int64        `json:"unique_members"`
	AvgPerDay     float64      `json:"avg_per_day"`
	Peak          *PeakSlot    `json:"peak"`
	PeakHours     []int        `json:"peak_hours"` // top 3 giờ đông nhất
	Daily         []CountPoint `json:"daily"`
}

// ---------------------------------------------------------------------------
// Class occupancy
// ---------------------------------------------------------------------------

type ClassOccupancyReq struct {
	g.Meta `path:"/reports/class-occupancy" method:"get" tags:"Báo cáo" summary:"Tỷ lệ lấp đầy lớp theo loại lớp/HLV/khung giờ" perm:"report.view"`
	Filter
}

type OccupancyRow struct {
	Key            string  `json:"key"`
	Label          string  `json:"label"`
	ID             uint    `json:"id,omitempty"`
	Color          string  `json:"color,omitempty"`
	Sessions       int64   `json:"sessions"`
	Capacity       int64   `json:"capacity"`
	Booked         int64   `json:"booked"`
	Attended       int64   `json:"attended"`
	NoShow         int64   `json:"no_show"`
	Cancelled      int64   `json:"cancelled"`
	Waitlist       int64   `json:"waitlist"`
	FillRate       float64 `json:"fill_rate"`       // % booked / capacity
	AttendanceRate float64 `json:"attendance_rate"` // % attended / (attended + no_show)
}

type ClassOccupancyRes struct {
	Period      Period         `json:"period"`
	Summary     OccupancyRow   `json:"summary"`
	ByClassType []OccupancyRow `json:"by_class_type"`
	ByTrainer   []OccupancyRow `json:"by_trainer"`
	ByTimeSlot  []OccupancyRow `json:"by_time_slot"` // theo giờ bắt đầu "HH:00"
	ByWeekday   []OccupancyRow `json:"by_weekday"`
}

// ---------------------------------------------------------------------------
// Trainer performance
// ---------------------------------------------------------------------------

type TrainerPerformanceReq struct {
	g.Meta `path:"/reports/trainer-performance" method:"get" tags:"Báo cáo" summary:"Hiệu suất & hoa hồng HLV" perm:"report.view"`
	Filter
}

type TrainerPerformance struct {
	TrainerID       uint    `json:"trainer_id"`
	TrainerName     string  `json:"trainer_name"`
	Level           string  `json:"level"`
	PTCompleted     int64   `json:"pt_completed"`
	PTScheduled     int64   `json:"pt_scheduled"`
	PTNoShow        int64   `json:"pt_no_show"`
	PTCancelled     int64   `json:"pt_cancelled"`
	ClassesTaught   int64   `json:"classes_taught"`
	ClassAttendees  int64   `json:"class_attendees"`
	ActivePTMembers int64   `json:"active_pt_members"`
	PTContractsSold int64   `json:"pt_contracts_sold"`
	PTRevenue       float64 `json:"pt_revenue"`
	Commission      float64 `json:"commission"`
	CommissionPaid  float64 `json:"commission_paid"`
	RatingAvg       float64 `json:"rating_avg"`
	RatingCount     int     `json:"rating_count"`
}

type TrainerPerformanceRes struct {
	Period Period               `json:"period"`
	Items  []TrainerPerformance `json:"items"`
	Totals TrainerPerformance   `json:"totals"`
}

// ---------------------------------------------------------------------------
// Products
// ---------------------------------------------------------------------------

type SalesProductsReq struct {
	g.Meta `path:"/reports/sales-products" method:"get" tags:"Báo cáo" summary:"Top sản phẩm, doanh thu, lợi nhuận gộp" perm:"report.view"`
	Filter
	Limit int `json:"limit" in:"query" d:"10" v:"between:1,100#limit từ 1 đến 100" dc:"Số sản phẩm top"`
}

type ProductSales struct {
	ProductID    uint    `json:"product_id"`
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	CategoryName string  `json:"category_name"`
	Quantity     int64   `json:"quantity"`
	Revenue      float64 `json:"revenue"`
	Cost         float64 `json:"cost"`
	GrossProfit  float64 `json:"gross_profit"`
	Margin       float64 `json:"margin"` // % lợi nhuận gộp / doanh thu
}

type CategorySales struct {
	CategoryName string  `json:"category_name"`
	Quantity     int64   `json:"quantity"`
	Revenue      float64 `json:"revenue"`
	GrossProfit  float64 `json:"gross_profit"`
	Pct          float64 `json:"pct"`
}

type SalesProductsRes struct {
	Period     Period          `json:"period"`
	Totals     ProductSales    `json:"totals"`
	Top        []ProductSales  `json:"top"`
	ByCategory []CategorySales `json:"by_category"`
	Products   int             `json:"product_count"` // số sản phẩm có bán trong kỳ
}

// ---------------------------------------------------------------------------
// Subscriptions
// ---------------------------------------------------------------------------

type SubscriptionsReq struct {
	g.Meta `path:"/reports/subscriptions" method:"get" tags:"Báo cáo" summary:"Doanh số gói tập & biến động đăng ký" perm:"report.view"`
	Filter
	GroupBy
}

type SubscriptionCounts struct {
	New         int64 `json:"new"`
	Renewed     int64 `json:"renewed"`
	Upgraded    int64 `json:"upgraded"`
	Transferred int64 `json:"transferred"`
	Cancelled   int64 `json:"cancelled"`
	Expired     int64 `json:"expired"`
	Frozen      int64 `json:"frozen"`
}

type SubscriptionPoint struct {
	Period string `json:"period"`
	Label  string `json:"label"`
	SubscriptionCounts
}

type PlanSales struct {
	PlanID     uint    `json:"plan_id"`
	PlanCode   string  `json:"plan_code"`
	PlanName   string  `json:"plan_name"`
	PlanType   string  `json:"plan_type"`
	Sold       int64   `json:"sold"`
	NewSales   int64   `json:"new_sales"`
	Continued  int64   `json:"continued"` // gia hạn / nâng cấp / chuyển nhượng (có gói trước)
	SoldAmount float64 `json:"sold_amount"`
	Collected  float64 `json:"collected"`
	Pct        float64 `json:"pct"` // % collected trên tổng
}

type SubscriptionsRes struct {
	Period  Period              `json:"period"`
	GroupBy string              `json:"group_by"`
	Counts  SubscriptionCounts  `json:"counts"`
	Series  []SubscriptionPoint `json:"series"`
	ByPlan  []PlanSales         `json:"by_plan"`
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

type ExportReq struct {
	g.Meta `path:"/reports/export" method:"get" tags:"Báo cáo" summary:"Xuất báo cáo Excel (.xlsx)" perm:"report.export"`
	Filter
	Type string `json:"type" in:"query" v:"required|in:revenue,members,checkins,subscriptions,invoices#Vui lòng chọn loại báo cáo|Loại báo cáo không hợp lệ" dc:"revenue | members | checkins | subscriptions | invoices"`
}
type ExportRes struct{}
