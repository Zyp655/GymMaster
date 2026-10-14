package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"gymmaster-api/internal/storage/postgres"
)

// ReportScope là phạm vi chung của một truy vấn báo cáo.
//   - BranchIDs nil = toàn hệ thống.
//   - Start/End: khoảng thời điểm [Start, End) (đã quy đổi từ ngày giờ VN).
//   - FromDate/ToDate: cùng khoảng dạng ngày "YYYY-MM-DD" (tính cả hai đầu) cho cột DATE và generate_series.
type ReportScope struct {
	BranchIDs []uint
	Start     time.Time
	End       time.Time
	FromDate  string
	ToDate    string
}

type (
	// ReportRevenueRow: thực thu (thanh toán dương, hoàn tiền âm) đã phân bổ theo loại dòng hóa đơn.
	ReportRevenueRow struct {
		Bucket   string  `gorm:"column:bucket"`
		BranchID uint    `gorm:"column:branch_id"`
		Method   string  `gorm:"column:method"`
		ItemType string  `gorm:"column:item_type"`
		Kind     string  `gorm:"column:kind"` // PAYMENT | REFUND
		Amount   float64 `gorm:"column:amount"`
	}

	ReportOverviewRow struct {
		RevenueToday           float64 `gorm:"column:revenue_today"`
		RevenueMonth           float64 `gorm:"column:revenue_month"`
		RevenuePrevMonthToDate float64 `gorm:"column:revenue_prev_month_to_date"`
		RevenuePrevMonth       float64 `gorm:"column:revenue_prev_month"`
		ActiveMembers          int64   `gorm:"column:active_members"`
		NewMembersMonth        int64   `gorm:"column:new_members_month"`
		NewMembersPrevMonth    int64   `gorm:"column:new_members_prev_month"`
		Expiring7d             int64   `gorm:"column:expiring_7d"`
		CheckinsToday          int64   `gorm:"column:checkins_today"`
		PresentNow             int64   `gorm:"column:present_now"`
		ClassesToday           int64   `gorm:"column:classes_today"`
		ClassCapacityToday     int64   `gorm:"column:class_capacity_today"`
		ClassBookedToday       int64   `gorm:"column:class_booked_today"`
		PTSessionsToday        int64   `gorm:"column:pt_sessions_today"`
		PTCompletedToday       int64   `gorm:"column:pt_completed_today"`
		PendingFreezes         int64   `gorm:"column:pending_freezes"`
		LowStock               int64   `gorm:"column:low_stock"`
	}

	ReportOverviewParams struct {
		BranchIDs      []uint
		Now            time.Time
		TodayStart     time.Time
		TomorrowStart  time.Time
		MonthStart     time.Time
		PrevMonthStart time.Time
		PrevMonthSame  time.Time // mốc cùng kỳ tháng trước (để so sánh month-to-date)
		Today          string    // YYYY-MM-DD
		Today7         string    // today + 7
	}

	ReportLowStockRow struct {
		ProductID  uint   `gorm:"column:product_id"`
		SKU        string `gorm:"column:sku"`
		Name       string `gorm:"column:name"`
		BranchID   uint   `gorm:"column:branch_id"`
		BranchName string `gorm:"column:branch_name"`
		Quantity   int    `gorm:"column:quantity"`
		MinStock   int    `gorm:"column:min_stock"`
	}

	ReportCountRow struct {
		Bucket string `gorm:"column:bucket"`
		Count  int64  `gorm:"column:count"`
	}

	ReportDimRow struct {
		Dim   string `gorm:"column:dim"`
		Key   string `gorm:"column:key"`
		Count int64  `gorm:"column:count"`
	}

	ReportRetentionRow struct {
		Month   *string `gorm:"column:month"` // NULL = dòng tổng
		Expired int64   `gorm:"column:expired"`
		Renewed int64   `gorm:"column:renewed"`
		Pending int64   `gorm:"column:pending"`
	}

	ReportHeatmapRow struct {
		Dow   int   `gorm:"column:dow"` // 1 = Thứ 2 … 7 = Chủ nhật (ISODOW)
		Hour  int   `gorm:"column:hour"`
		Count int64 `gorm:"column:count"`
	}

	ReportCheckinTotalsRow struct {
		Allowed       int64 `gorm:"column:allowed"`
		Denied        int64 `gorm:"column:denied"`
		UniqueMembers int64 `gorm:"column:unique_members"`
	}

	ReportClassRow struct {
		ScheduleID    uint   `gorm:"column:schedule_id"`
		ClassTypeID   uint   `gorm:"column:class_type_id"`
		ClassTypeName string `gorm:"column:class_type_name"`
		Color         string `gorm:"column:color"`
		TrainerID     uint   `gorm:"column:trainer_id"`
		TrainerName   string `gorm:"column:trainer_name"`
		Hour          int    `gorm:"column:hour"`
		Dow           int    `gorm:"column:dow"`
		Capacity      int64  `gorm:"column:capacity"`
		Booked        int64  `gorm:"column:booked"`
		Attended      int64  `gorm:"column:attended"`
		NoShow        int64  `gorm:"column:no_show"`
		Cancelled     int64  `gorm:"column:cancelled"`
		Waitlist      int64  `gorm:"column:waitlist"`
	}

	ReportTrainerRow struct {
		TrainerID       uint    `gorm:"column:trainer_id"`
		TrainerName     string  `gorm:"column:trainer_name"`
		Level           string  `gorm:"column:level"`
		PTCompleted     int64   `gorm:"column:pt_completed"`
		PTScheduled     int64   `gorm:"column:pt_scheduled"`
		PTNoShow        int64   `gorm:"column:pt_no_show"`
		PTCancelled     int64   `gorm:"column:pt_cancelled"`
		ClassesTaught   int64   `gorm:"column:classes_taught"`
		ClassAttendees  int64   `gorm:"column:class_attendees"`
		ActivePTMembers int64   `gorm:"column:active_pt_members"`
		PTContractsSold int64   `gorm:"column:pt_contracts_sold"`
		PTRevenue       float64 `gorm:"column:pt_revenue"`
		Commission      float64 `gorm:"column:commission"`
		CommissionPaid  float64 `gorm:"column:commission_paid"`
	}

	ReportProductRow struct {
		ProductID    uint    `gorm:"column:product_id"`
		SKU          string  `gorm:"column:sku"`
		Name         string  `gorm:"column:name"`
		CategoryID   *uint   `gorm:"column:category_id"`
		CategoryName string  `gorm:"column:category_name"`
		Quantity     int64   `gorm:"column:quantity"`
		Revenue      float64 `gorm:"column:revenue"`
		Cost         float64 `gorm:"column:cost"`
	}

	ReportSubEventRow struct {
		Bucket string `gorm:"column:bucket"`
		Kind   string `gorm:"column:kind"` // NEW | RENEWED | UPGRADED | TRANSFERRED | CANCELLED | EXPIRED | FROZEN
		Count  int64  `gorm:"column:count"`
	}

	ReportPlanRow struct {
		PlanID     uint    `gorm:"column:plan_id"`
		PlanCode   string  `gorm:"column:plan_code"`
		PlanName   string  `gorm:"column:plan_name"`
		PlanType   string  `gorm:"column:plan_type"`
		Sold       int64   `gorm:"column:sold"`
		NewSales   int64   `gorm:"column:new_sales"`
		Continued  int64   `gorm:"column:continued"`
		SoldAmount float64 `gorm:"column:sold_amount"`
		Collected  float64 `gorm:"column:collected"`
	}

	// --- Xuất Excel ---

	ReportMemberExportRow struct {
		MemberCode string     `gorm:"column:member_code"`
		FullName   string     `gorm:"column:full_name"`
		Gender     string     `gorm:"column:gender"`
		Dob        *time.Time `gorm:"column:dob"`
		Phone      string     `gorm:"column:phone"`
		Email      string     `gorm:"column:email"`
		BranchName string     `gorm:"column:branch_name"`
		Source     string     `gorm:"column:source"`
		Status     string     `gorm:"column:status"`
		PlanName   string     `gorm:"column:plan_name"`
		EndDate    *time.Time `gorm:"column:end_date"`
		CreatedAt  time.Time  `gorm:"column:created_at"`
	}

	ReportCheckinExportRow struct {
		CheckedInAt  time.Time  `gorm:"column:checked_in_at"`
		CheckedOutAt *time.Time `gorm:"column:checked_out_at"`
		MemberCode   string     `gorm:"column:member_code"`
		MemberName   string     `gorm:"column:member_name"`
		BranchName   string     `gorm:"column:branch_name"`
		Method       string     `gorm:"column:method"`
		Result       string     `gorm:"column:result"`
		DenyReason   string     `gorm:"column:deny_reason"`
		StaffName    string     `gorm:"column:staff_name"`
	}

	ReportSubExportRow struct {
		ID         uint      `gorm:"column:id"`
		MemberCode string    `gorm:"column:member_code"`
		MemberName string    `gorm:"column:member_name"`
		PlanName   string    `gorm:"column:plan_name"`
		BranchName string    `gorm:"column:branch_name"`
		StartDate  time.Time `gorm:"column:start_date"`
		EndDate    time.Time `gorm:"column:end_date"`
		Price      float64   `gorm:"column:price"`
		Status     string    `gorm:"column:status"`
		SoldBy     string    `gorm:"column:sold_by"`
		CreatedAt  time.Time `gorm:"column:created_at"`
	}

	ReportInvoiceExportRow struct {
		InvoiceNo  string    `gorm:"column:invoice_no"`
		IssuedAt   time.Time `gorm:"column:issued_at"`
		MemberCode string    `gorm:"column:member_code"`
		MemberName string    `gorm:"column:member_name"`
		BranchName string    `gorm:"column:branch_name"`
		Source     string    `gorm:"column:source"`
		Subtotal   float64   `gorm:"column:subtotal"`
		Discount   float64   `gorm:"column:discount"`
		Total      float64   `gorm:"column:total"`
		Paid       float64   `gorm:"column:paid_amount"`
		Refunded   float64   `gorm:"column:refunded_amount"`
		Status     string    `gorm:"column:status"`
		IssuedBy   string    `gorm:"column:issued_by"`
	}

	ReportRepository interface {
		BranchNames(ctx context.Context) (map[uint]string, error)
		RevenueFlows(ctx context.Context, s ReportScope, unit string) ([]ReportRevenueRow, error)
		Overview(ctx context.Context, p ReportOverviewParams) (*ReportOverviewRow, error)
		LowStock(ctx context.Context, branchIDs []uint, limit int) ([]ReportLowStockRow, error)
		NewMemberSeries(ctx context.Context, s ReportScope, unit string) ([]ReportCountRow, error)
		MemberDistribution(ctx context.Context, s ReportScope, onlyNew bool) ([]ReportDimRow, error)
		MemberStatusCounts(ctx context.Context, branchIDs []uint) ([]ReportDimRow, error)
		Retention(ctx context.Context, s ReportScope, windowDays int, today string) ([]ReportRetentionRow, error)
		CheckinHeatmap(ctx context.Context, s ReportScope) ([]ReportHeatmapRow, error)
		CheckinDaily(ctx context.Context, s ReportScope) ([]ReportCountRow, error)
		CheckinTotals(ctx context.Context, s ReportScope) (*ReportCheckinTotalsRow, error)
		ClassScheduleStats(ctx context.Context, s ReportScope) ([]ReportClassRow, error)
		TrainerPerformance(ctx context.Context, s ReportScope, now time.Time) ([]ReportTrainerRow, error)
		ProductSales(ctx context.Context, s ReportScope) ([]ReportProductRow, error)
		SubscriptionEvents(ctx context.Context, s ReportScope, unit string) ([]ReportSubEventRow, error)
		SubscriptionPlans(ctx context.Context, s ReportScope) ([]ReportPlanRow, error)

		ExportMembers(ctx context.Context, s ReportScope, limit int) ([]ReportMemberExportRow, error)
		ExportCheckins(ctx context.Context, s ReportScope, limit int) ([]ReportCheckinExportRow, error)
		ExportSubscriptions(ctx context.Context, s ReportScope, limit int) ([]ReportSubExportRow, error)
		ExportInvoices(ctx context.Context, s ReportScope, limit int) ([]ReportInvoiceExportRow, error)
	}

	reportRepository struct {
		db *gorm.DB
	}
)

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

// reportTZ: mọi phép quy đổi giờ trong báo cáo dùng giờ Việt Nam.
const reportTZ = `'Asia/Ho_Chi_Minh'`

// reportBranch trả điều kiện lọc chi nhánh trên cột col (rỗng khi xem toàn hệ thống).
func reportBranch(col string, ids []uint) string {
	if ids == nil {
		return ""
	}
	return " AND " + col + " IN @branches"
}

// reportSQL thay {tz} bằng múi giờ VN cho dễ đọc câu SQL.
func reportSQL(q string) string {
	return strings.ReplaceAll(q, "{tz}", reportTZ)
}

func (s ReportScope) args(extra map[string]any) map[string]any {
	m := map[string]any{
		"start": s.Start, "end": s.End, "from_date": s.FromDate, "to_date": s.ToDate, "branches": s.BranchIDs,
	}
	if s.BranchIDs != nil && len(s.BranchIDs) == 0 {
		m["branches"] = []uint{0} // không có chi nhánh nào → không khớp bản ghi nào
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func (r *reportRepository) raw(ctx context.Context, dest any, q string, args map[string]any) error {
	q = reportSQL(q)
	if !strings.Contains(q, "@") { // không có tham số đặt tên (vd. không lọc chi nhánh) → không truyền map
		return postgres.Conn(ctx, r.db).Raw(q).Scan(dest).Error
	}
	return postgres.Conn(ctx, r.db).Raw(q, args).Scan(dest).Error
}

func (r *reportRepository) BranchNames(ctx context.Context) (map[uint]string, error) {
	var rows []struct {
		ID   uint
		Name string
	}
	if err := postgres.Conn(ctx, r.db).Raw("SELECT id, name FROM branches").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, b := range rows {
		out[b.ID] = b.Name
	}
	return out, nil
}

// reportFlowsCTE: dòng tiền thực thu trong [start,end) – thanh toán (dương) và hoàn tiền (âm) – kèm tỷ lệ phân bổ
// theo từng dòng hóa đơn (amount dòng / tổng amount các dòng; hóa đơn toàn dòng 0đ chia đều).
const reportFlowsCTE = `
flows AS (
	SELECT p.invoice_id, p.paid_at AS at, p.amount AS amount, p.method, 'PAYMENT' AS kind
	FROM payments p WHERE p.paid_at >= @start AND p.paid_at < @end
	UNION ALL
	SELECT r.invoice_id, r.created_at AS at, -r.amount AS amount, r.method, 'REFUND' AS kind
	FROM refunds r WHERE r.created_at >= @start AND r.created_at < @end
),
alloc AS (
	SELECT ii.invoice_id, ii.item_type, ii.ref_id,
		CASE WHEN SUM(ii.amount) OVER w > 0 THEN ii.amount / SUM(ii.amount) OVER w
		     ELSE 1.0 / COUNT(*) OVER w END AS ratio
	FROM invoice_items ii
	WHERE ii.invoice_id IN (SELECT invoice_id FROM flows)
	WINDOW w AS (PARTITION BY ii.invoice_id)
)`

func (r *reportRepository) RevenueFlows(ctx context.Context, s ReportScope, unit string) ([]ReportRevenueRow, error) {
	q := `WITH ` + reportFlowsCTE + `
SELECT to_char(date_trunc(@unit, f.at AT TIME ZONE {tz}), 'YYYY-MM-DD') AS bucket,
	i.branch_id, f.method, COALESCE(a.item_type, 'OTHER') AS item_type, f.kind,
	SUM(f.amount * COALESCE(a.ratio, 1)) AS amount
FROM flows f
JOIN invoices i ON i.id = f.invoice_id
LEFT JOIN alloc a ON a.invoice_id = f.invoice_id
WHERE TRUE` + reportBranch("i.branch_id", s.BranchIDs) + `
GROUP BY 1, 2, 3, 4, 5
ORDER BY 1`
	var rows []ReportRevenueRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"unit": unit}))
	return rows, err
}

func (r *reportRepository) Overview(ctx context.Context, p ReportOverviewParams) (*ReportOverviewRow, error) {
	b := p.BranchIDs
	q := `
WITH flows AS (
	SELECT p.paid_at AS at, p.amount AS amount
	FROM payments p JOIN invoices i ON i.id = p.invoice_id
	WHERE p.paid_at >= @prev_month_start AND p.paid_at < @tomorrow` + reportBranch("i.branch_id", b) + `
	UNION ALL
	SELECT r.created_at AS at, -r.amount AS amount
	FROM refunds r JOIN invoices i ON i.id = r.invoice_id
	WHERE r.created_at >= @prev_month_start AND r.created_at < @tomorrow` + reportBranch("i.branch_id", b) + `
), rev AS (
	SELECT
		COALESCE(SUM(amount) FILTER (WHERE at >= @today_start), 0) AS revenue_today,
		COALESCE(SUM(amount) FILTER (WHERE at >= @month_start), 0) AS revenue_month,
		COALESCE(SUM(amount) FILTER (WHERE at >= @prev_month_start AND at < @prev_month_same), 0) AS revenue_prev_month_to_date,
		COALESCE(SUM(amount) FILTER (WHERE at < @month_start), 0) AS revenue_prev_month
	FROM flows
)
SELECT rev.*,
	(SELECT count(DISTINCT s.member_id) FROM subscriptions s JOIN members m ON m.id = s.member_id AND m.deleted_at IS NULL
	  WHERE s.status = 'ACTIVE'` + reportBranch("s.branch_id", b) + `) AS active_members,
	(SELECT count(*) FROM members m WHERE m.deleted_at IS NULL AND m.created_at >= @month_start AND m.created_at < @tomorrow` + reportBranch("m.home_branch_id", b) + `) AS new_members_month,
	(SELECT count(*) FROM members m WHERE m.deleted_at IS NULL AND m.created_at >= @prev_month_start AND m.created_at < @month_start` + reportBranch("m.home_branch_id", b) + `) AS new_members_prev_month,
	(SELECT count(DISTINCT s.member_id) FROM subscriptions s JOIN members m ON m.id = s.member_id AND m.deleted_at IS NULL
	  WHERE s.status = 'ACTIVE' AND s.end_date BETWEEN CAST(@today AS date) AND CAST(@today7 AS date)` + reportBranch("s.branch_id", b) + `
	  AND NOT EXISTS (SELECT 1 FROM subscriptions n WHERE n.member_id = s.member_id AND n.id <> s.id
	      AND n.status IN ('PENDING','ACTIVE','FROZEN') AND n.end_date > s.end_date)) AS expiring_7d,
	(SELECT count(*) FROM checkins c WHERE c.result = 'ALLOWED' AND c.checked_in_at >= @today_start AND c.checked_in_at < @tomorrow` + reportBranch("c.branch_id", b) + `) AS checkins_today,
	(SELECT count(*) FROM checkins c WHERE c.result = 'ALLOWED' AND c.checked_out_at IS NULL AND c.checked_in_at >= @today_start` + reportBranch("c.branch_id", b) + `) AS present_now,
	(SELECT count(*) FROM class_schedules cs WHERE cs.status <> 'CANCELLED' AND cs.start_at >= @today_start AND cs.start_at < @tomorrow` + reportBranch("cs.branch_id", b) + `) AS classes_today,
	(SELECT COALESCE(SUM(cs.capacity), 0) FROM class_schedules cs WHERE cs.status <> 'CANCELLED' AND cs.start_at >= @today_start AND cs.start_at < @tomorrow` + reportBranch("cs.branch_id", b) + `) AS class_capacity_today,
	(SELECT COALESCE(SUM(cs.booked_count), 0) FROM class_schedules cs WHERE cs.status <> 'CANCELLED' AND cs.start_at >= @today_start AND cs.start_at < @tomorrow` + reportBranch("cs.branch_id", b) + `) AS class_booked_today,
	(SELECT count(*) FROM pt_sessions ps WHERE ps.status IN ('SCHEDULED','COMPLETED') AND ps.start_at >= @today_start AND ps.start_at < @tomorrow` + reportBranch("ps.branch_id", b) + `) AS pt_sessions_today,
	(SELECT count(*) FROM pt_sessions ps WHERE ps.status = 'COMPLETED' AND ps.start_at >= @today_start AND ps.start_at < @tomorrow` + reportBranch("ps.branch_id", b) + `) AS pt_completed_today,
	(SELECT count(*) FROM subscription_freezes f JOIN subscriptions s ON s.id = f.subscription_id WHERE f.status = 'PENDING'` + reportBranch("s.branch_id", b) + `) AS pending_freezes,
	(SELECT count(*) FROM stock_levels sl JOIN products pr ON pr.id = sl.product_id
	  WHERE pr.deleted_at IS NULL AND pr.status = 'ACTIVE' AND pr.min_stock > 0 AND sl.quantity <= pr.min_stock` + reportBranch("sl.branch_id", b) + `) AS low_stock
FROM rev`
	args := ReportScope{BranchIDs: b}.args(map[string]any{
		"today_start": p.TodayStart, "tomorrow": p.TomorrowStart, "month_start": p.MonthStart,
		"prev_month_start": p.PrevMonthStart, "prev_month_same": p.PrevMonthSame, "today": p.Today, "today7": p.Today7,
	})
	var row ReportOverviewRow
	if err := r.raw(ctx, &row, q, args); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *reportRepository) LowStock(ctx context.Context, branchIDs []uint, limit int) ([]ReportLowStockRow, error) {
	q := `
SELECT pr.id AS product_id, pr.sku, pr.name, sl.branch_id, b.name AS branch_name, sl.quantity, pr.min_stock
FROM stock_levels sl
JOIN products pr ON pr.id = sl.product_id
JOIN branches b ON b.id = sl.branch_id
WHERE pr.deleted_at IS NULL AND pr.status = 'ACTIVE' AND pr.min_stock > 0 AND sl.quantity <= pr.min_stock` + reportBranch("sl.branch_id", branchIDs) + `
ORDER BY (sl.quantity::numeric / pr.min_stock), pr.name
LIMIT @limit`
	var rows []ReportLowStockRow
	err := r.raw(ctx, &rows, q, ReportScope{BranchIDs: branchIDs}.args(map[string]any{"limit": limit}))
	return rows, err
}

func (r *reportRepository) NewMemberSeries(ctx context.Context, s ReportScope, unit string) ([]ReportCountRow, error) {
	q := `
SELECT to_char(g.b, 'YYYY-MM-DD') AS bucket, COALESCE(x.cnt, 0) AS count
FROM generate_series(date_trunc(@unit, CAST(@from_date AS timestamp)), CAST(@to_date AS timestamp), ('1 ' || @unit)::interval) AS g(b)
LEFT JOIN (
	SELECT date_trunc(@unit, m.created_at AT TIME ZONE {tz}) AS k, count(*) AS cnt
	FROM members m
	WHERE m.deleted_at IS NULL AND m.created_at >= @start AND m.created_at < @end` + reportBranch("m.home_branch_id", s.BranchIDs) + `
	GROUP BY 1
) x ON x.k = g.b
ORDER BY g.b`
	var rows []ReportCountRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"unit": unit}))
	return rows, err
}

func (r *reportRepository) MemberDistribution(ctx context.Context, s ReportScope, onlyNew bool) ([]ReportDimRow, error) {
	cond := ""
	if onlyNew {
		cond = " AND m.created_at >= @start AND m.created_at < @end"
	}
	q := `
WITH pop AS (
	SELECT m.source, m.gender, m.home_branch_id,
		CASE WHEN m.dob IS NULL THEN 'UNKNOWN'
		     ELSE (CASE
		        WHEN date_part('year', age(m.dob)) < 18 THEN 'UNDER_18'
		        WHEN date_part('year', age(m.dob)) < 25 THEN '18_24'
		        WHEN date_part('year', age(m.dob)) < 35 THEN '25_34'
		        WHEN date_part('year', age(m.dob)) < 45 THEN '35_44'
		        WHEN date_part('year', age(m.dob)) < 55 THEN '45_54'
		        ELSE '55_PLUS' END) END AS age_group
	FROM members m
	WHERE m.deleted_at IS NULL` + cond + reportBranch("m.home_branch_id", s.BranchIDs) + `
)
SELECT 'source' AS dim, source AS key, count(*) AS count FROM pop GROUP BY source
UNION ALL SELECT 'gender', gender, count(*) FROM pop GROUP BY gender
UNION ALL SELECT 'age', age_group, count(*) FROM pop GROUP BY age_group
UNION ALL SELECT 'branch', home_branch_id::text, count(*) FROM pop GROUP BY home_branch_id`
	var rows []ReportDimRow
	err := r.raw(ctx, &rows, q, s.args(nil))
	return rows, err
}

// MemberStatusCounts phân loại hội viên theo gói hiện có: ACTIVE > FROZEN > PENDING > EXPIRED (từng có gói) > NONE.
func (r *reportRepository) MemberStatusCounts(ctx context.Context, branchIDs []uint) ([]ReportDimRow, error) {
	q := `
SELECT 'status' AS dim, st AS key, count(*) AS count FROM (
	SELECT CASE
		WHEN bool_or(s.status = 'ACTIVE') THEN 'ACTIVE'
		WHEN bool_or(s.status = 'FROZEN') THEN 'FROZEN'
		WHEN bool_or(s.status = 'PENDING') THEN 'PENDING'
		WHEN count(s.id) > 0 THEN 'EXPIRED'
		ELSE 'NONE' END AS st
	FROM members m
	LEFT JOIN subscriptions s ON s.member_id = m.id AND s.status <> 'CANCELLED'
	WHERE m.deleted_at IS NULL` + reportBranch("m.home_branch_id", branchIDs) + `
	GROUP BY m.id
) t GROUP BY st`
	var rows []ReportDimRow
	err := r.raw(ctx, &rows, q, ReportScope{BranchIDs: branchIDs}.args(nil))
	return rows, err
}

// Retention: gói đã kết thúc (end_date trong kỳ và < hôm nay). Hội viên được tính "gia hạn" khi có gói khác
// (đã kích hoạt) kéo dài quá end_date và được tạo trước end_date + window ngày.
func (r *reportRepository) Retention(ctx context.Context, s ReportScope, windowDays int, today string) ([]ReportRetentionRow, error) {
	q := `
WITH ended AS (
	SELECT s.member_id, s.end_date, to_char(s.end_date, 'YYYY-MM') AS month,
		EXISTS (
			SELECT 1 FROM subscriptions n
			WHERE n.member_id = s.member_id AND n.id <> s.id
			  AND n.status IN ('ACTIVE','FROZEN','EXPIRED','TRANSFERRED')
			  AND n.end_date > s.end_date
			  AND n.created_at < ((s.end_date + CAST(@window AS int) + 1)::timestamp AT TIME ZONE {tz})
		) AS renewed
	FROM subscriptions s
	JOIN members m ON m.id = s.member_id AND m.deleted_at IS NULL
	WHERE s.status IN ('ACTIVE','FROZEN','EXPIRED')
	  AND s.end_date BETWEEN CAST(@from_date AS date) AND CAST(@to_date AS date)
	  AND s.end_date < CAST(@today AS date)` + reportBranch("s.branch_id", s.BranchIDs) + `
)
SELECT month,
	count(DISTINCT member_id) AS expired,
	count(DISTINCT member_id) FILTER (WHERE renewed) AS renewed,
	count(DISTINCT member_id) FILTER (WHERE NOT renewed AND end_date + CAST(@window AS int) >= CAST(@today AS date)) AS pending
FROM ended
GROUP BY GROUPING SETS ((month), ())
ORDER BY month NULLS LAST`
	var rows []ReportRetentionRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"window": windowDays, "today": today}))
	return rows, err
}

func (r *reportRepository) CheckinHeatmap(ctx context.Context, s ReportScope) ([]ReportHeatmapRow, error) {
	q := `
SELECT EXTRACT(ISODOW FROM c.checked_in_at AT TIME ZONE {tz})::int AS dow,
	EXTRACT(HOUR FROM c.checked_in_at AT TIME ZONE {tz})::int AS hour,
	count(*) AS count
FROM checkins c
WHERE c.result = 'ALLOWED' AND c.checked_in_at >= @start AND c.checked_in_at < @end` + reportBranch("c.branch_id", s.BranchIDs) + `
GROUP BY 1, 2`
	var rows []ReportHeatmapRow
	err := r.raw(ctx, &rows, q, s.args(nil))
	return rows, err
}

func (r *reportRepository) CheckinDaily(ctx context.Context, s ReportScope) ([]ReportCountRow, error) {
	q := `
SELECT to_char(g.d, 'YYYY-MM-DD') AS bucket, COALESCE(x.cnt, 0) AS count
FROM generate_series(CAST(@from_date AS date), CAST(@to_date AS date), interval '1 day') AS g(d)
LEFT JOIN (
	SELECT (c.checked_in_at AT TIME ZONE {tz})::date AS day, count(*) AS cnt
	FROM checkins c
	WHERE c.result = 'ALLOWED' AND c.checked_in_at >= @start AND c.checked_in_at < @end` + reportBranch("c.branch_id", s.BranchIDs) + `
	GROUP BY 1
) x ON x.day = g.d::date
ORDER BY g.d`
	var rows []ReportCountRow
	err := r.raw(ctx, &rows, q, s.args(nil))
	return rows, err
}

func (r *reportRepository) CheckinTotals(ctx context.Context, s ReportScope) (*ReportCheckinTotalsRow, error) {
	q := `
SELECT count(*) FILTER (WHERE c.result = 'ALLOWED') AS allowed,
	count(*) FILTER (WHERE c.result = 'DENIED') AS denied,
	count(DISTINCT c.member_id) FILTER (WHERE c.result = 'ALLOWED') AS unique_members
FROM checkins c
WHERE c.checked_in_at >= @start AND c.checked_in_at < @end` + reportBranch("c.branch_id", s.BranchIDs)
	var row ReportCheckinTotalsRow
	if err := r.raw(ctx, &row, q, s.args(nil)); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *reportRepository) ClassScheduleStats(ctx context.Context, s ReportScope) ([]ReportClassRow, error) {
	q := `
SELECT cs.id AS schedule_id, cs.class_type_id, ct.name AS class_type_name, ct.color,
	cs.trainer_id, u.full_name AS trainer_name,
	EXTRACT(HOUR FROM cs.start_at AT TIME ZONE {tz})::int AS hour,
	EXTRACT(ISODOW FROM cs.start_at AT TIME ZONE {tz})::int AS dow,
	cs.capacity,
	count(b.id) FILTER (WHERE b.status IN ('BOOKED','ATTENDED','NO_SHOW')) AS booked,
	count(b.id) FILTER (WHERE b.status = 'ATTENDED') AS attended,
	count(b.id) FILTER (WHERE b.status = 'NO_SHOW') AS no_show,
	count(b.id) FILTER (WHERE b.status = 'CANCELLED') AS cancelled,
	count(b.id) FILTER (WHERE b.status = 'WAITLIST') AS waitlist
FROM class_schedules cs
JOIN class_types ct ON ct.id = cs.class_type_id
JOIN users u ON u.id = cs.trainer_id
LEFT JOIN class_bookings b ON b.schedule_id = cs.id
WHERE cs.status <> 'CANCELLED' AND cs.start_at >= @start AND cs.start_at < @end` + reportBranch("cs.branch_id", s.BranchIDs) + `
GROUP BY cs.id, ct.name, ct.color, u.full_name
ORDER BY cs.start_at`
	var rows []ReportClassRow
	err := r.raw(ctx, &rows, q, s.args(nil))
	return rows, err
}

func (r *reportRepository) TrainerPerformance(ctx context.Context, s ReportScope, now time.Time) ([]ReportTrainerRow, error) {
	b := s.BranchIDs
	trainerCond := ""
	if b != nil {
		trainerCond = " AND EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = tp.user_id AND ur.branch_id IN @branches)"
	}
	q := `
WITH pt AS (
	SELECT ps.trainer_id,
		count(*) FILTER (WHERE ps.status = 'COMPLETED') AS pt_completed,
		count(*) FILTER (WHERE ps.status = 'SCHEDULED') AS pt_scheduled,
		count(*) FILTER (WHERE ps.status = 'NO_SHOW') AS pt_no_show,
		count(*) FILTER (WHERE ps.status = 'CANCELLED') AS pt_cancelled
	FROM pt_sessions ps
	WHERE ps.start_at >= @start AND ps.start_at < @end` + reportBranch("ps.branch_id", b) + `
	GROUP BY 1
), cls AS (
	SELECT cs.trainer_id, count(DISTINCT cs.id) AS classes_taught,
		count(bk.id) FILTER (WHERE bk.status = 'ATTENDED') AS class_attendees
	FROM class_schedules cs
	LEFT JOIN class_bookings bk ON bk.schedule_id = cs.id
	WHERE cs.status <> 'CANCELLED' AND cs.start_at >= @start AND cs.start_at < @end AND cs.start_at < @now` + reportBranch("cs.branch_id", b) + `
	GROUP BY 1
), ctr AS (
	SELECT c.trainer_id,
		count(DISTINCT c.member_id) FILTER (WHERE c.status = 'ACTIVE') AS active_pt_members,
		count(*) FILTER (WHERE c.created_at >= @start AND c.created_at < @end AND c.status NOT IN ('CANCELLED','PENDING')) AS pt_contracts_sold,
		COALESCE(SUM(c.price) FILTER (WHERE c.created_at >= @start AND c.created_at < @end AND c.status NOT IN ('CANCELLED','PENDING')), 0) AS pt_revenue
	FROM pt_contracts c
	WHERE TRUE` + reportBranch("c.branch_id", b) + `
	GROUP BY 1
), com AS (
	SELECT tc.trainer_id, COALESCE(SUM(tc.amount), 0) AS commission,
		COALESCE(SUM(tc.amount) FILTER (WHERE tc.status = 'PAID'), 0) AS commission_paid
	FROM trainer_commissions tc
	WHERE tc.created_at >= @start AND tc.created_at < @end` + reportBranch("tc.branch_id", b) + `
	GROUP BY 1
), ids AS (
	SELECT tp.user_id AS id FROM trainer_profiles tp WHERE TRUE` + trainerCond + `
	UNION SELECT trainer_id FROM pt
	UNION SELECT trainer_id FROM cls
	UNION SELECT trainer_id FROM ctr
	UNION SELECT trainer_id FROM com
)
SELECT u.id AS trainer_id, u.full_name AS trainer_name, COALESCE(tpr.level, '') AS level,
	COALESCE(pt.pt_completed, 0) AS pt_completed, COALESCE(pt.pt_scheduled, 0) AS pt_scheduled,
	COALESCE(pt.pt_no_show, 0) AS pt_no_show, COALESCE(pt.pt_cancelled, 0) AS pt_cancelled,
	COALESCE(cls.classes_taught, 0) AS classes_taught, COALESCE(cls.class_attendees, 0) AS class_attendees,
	COALESCE(ctr.active_pt_members, 0) AS active_pt_members, COALESCE(ctr.pt_contracts_sold, 0) AS pt_contracts_sold,
	COALESCE(ctr.pt_revenue, 0) AS pt_revenue,
	COALESCE(com.commission, 0) AS commission, COALESCE(com.commission_paid, 0) AS commission_paid
FROM ids
JOIN users u ON u.id = ids.id AND u.deleted_at IS NULL
LEFT JOIN trainer_profiles tpr ON tpr.user_id = u.id
LEFT JOIN pt ON pt.trainer_id = u.id
LEFT JOIN cls ON cls.trainer_id = u.id
LEFT JOIN ctr ON ctr.trainer_id = u.id
LEFT JOIN com ON com.trainer_id = u.id
ORDER BY pt_completed DESC, classes_taught DESC, u.full_name`
	var rows []ReportTrainerRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"now": now}))
	return rows, err
}

// ProductSales: dòng PRODUCT của hóa đơn đã thu (PAID/PARTIAL) lập trong kỳ. Doanh thu đã trừ giảm giá cấp hóa đơn
// (phân bổ theo total/subtotal). Giá vốn lấy từ thẻ kho SALE của chính hóa đơn (unit_cost), không có thì dùng products.cost_price.
func (r *reportRepository) ProductSales(ctx context.Context, s ReportScope) ([]ReportProductRow, error) {
	q := `
WITH items AS (
	SELECT i.id AS invoice_id, ii.ref_id AS product_id, SUM(ii.quantity) AS qty,
		SUM(ii.amount * CASE WHEN i.subtotal > 0 THEN i.total / i.subtotal ELSE 0 END) AS revenue
	FROM invoice_items ii
	JOIN invoices i ON i.id = ii.invoice_id
	WHERE ii.item_type = 'PRODUCT' AND ii.ref_id IS NOT NULL
	  AND i.status IN ('PAID','PARTIAL') AND i.issued_at >= @start AND i.issued_at < @end` + reportBranch("i.branch_id", s.BranchIDs) + `
	GROUP BY 1, 2
), costs AS (
	SELECT st.reference_id AS invoice_id, st.product_id, SUM(-st.quantity * st.unit_cost) AS cost
	FROM stock_transactions st
	WHERE st.type = 'SALE' AND st.unit_cost > 0 AND st.reference_id IN (SELECT invoice_id FROM items)
	GROUP BY 1, 2
)
SELECT p.id AS product_id, p.sku, p.name, p.category_id, COALESCE(pc.name, '') AS category_name,
	SUM(it.qty) AS quantity, SUM(it.revenue) AS revenue,
	SUM(COALESCE(c.cost, it.qty * p.cost_price)) AS cost
FROM items it
JOIN products p ON p.id = it.product_id
LEFT JOIN product_categories pc ON pc.id = p.category_id
LEFT JOIN costs c ON c.invoice_id = it.invoice_id AND c.product_id = it.product_id
GROUP BY p.id, pc.name
ORDER BY revenue DESC, p.name`
	var rows []ReportProductRow
	err := r.raw(ctx, &rows, q, s.args(nil))
	return rows, err
}

func (r *reportRepository) SubscriptionEvents(ctx context.Context, s ReportScope, unit string) ([]ReportSubEventRow, error) {
	q := `
SELECT to_char(date_trunc(@unit, e.created_at AT TIME ZONE {tz}), 'YYYY-MM-DD') AS bucket,
	CASE WHEN e.event_type = 'CREATED' THEN 'NEW' ELSE e.event_type END AS kind,
	count(*) AS count
FROM subscription_events e
JOIN subscriptions s ON s.id = e.subscription_id
WHERE e.created_at >= @start AND e.created_at < @end
  AND (e.event_type IN ('RENEWED','UPGRADED','TRANSFERRED','CANCELLED','EXPIRED','FROZEN')
       OR (e.event_type = 'CREATED' AND s.previous_id IS NULL))` + reportBranch("s.branch_id", s.BranchIDs) + `
GROUP BY 1, 2
ORDER BY 1`
	var rows []ReportSubEventRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"unit": unit}))
	return rows, err
}

func (r *reportRepository) SubscriptionPlans(ctx context.Context, s ReportScope) ([]ReportPlanRow, error) {
	q := `WITH ` + reportFlowsCTE + `,
sold AS (
	SELECT s.plan_id, count(*) AS sold,
		count(*) FILTER (WHERE s.previous_id IS NULL) AS new_sales,
		count(*) FILTER (WHERE s.previous_id IS NOT NULL) AS continued,
		COALESCE(SUM(s.price), 0) AS sold_amount
	FROM subscriptions s
	WHERE s.created_at >= @start AND s.created_at < @end AND s.status NOT IN ('PENDING','CANCELLED')` + reportBranch("s.branch_id", s.BranchIDs) + `
	GROUP BY 1
), collected AS (
	SELECT sub.plan_id, SUM(f.amount * a.ratio) AS collected
	FROM flows f
	JOIN invoices i ON i.id = f.invoice_id
	JOIN alloc a ON a.invoice_id = f.invoice_id AND a.item_type = 'MEMBERSHIP'
	JOIN subscriptions sub ON sub.id = a.ref_id
	WHERE TRUE` + reportBranch("i.branch_id", s.BranchIDs) + `
	GROUP BY 1
)
SELECT p.id AS plan_id, p.code AS plan_code, p.name AS plan_name, p.type AS plan_type,
	COALESCE(sold.sold, 0) AS sold, COALESCE(sold.new_sales, 0) AS new_sales, COALESCE(sold.continued, 0) AS continued,
	COALESCE(sold.sold_amount, 0) AS sold_amount, COALESCE(collected.collected, 0) AS collected
FROM membership_plans p
LEFT JOIN sold ON sold.plan_id = p.id
LEFT JOIN collected ON collected.plan_id = p.id
WHERE sold.plan_id IS NOT NULL OR collected.plan_id IS NOT NULL
ORDER BY collected DESC, sold DESC, p.sort_order`
	var rows []ReportPlanRow
	err := r.raw(ctx, &rows, q, s.args(nil))
	return rows, err
}

func (r *reportRepository) ExportMembers(ctx context.Context, s ReportScope, limit int) ([]ReportMemberExportRow, error) {
	q := `
SELECT m.member_code, m.full_name, m.gender, m.dob, m.phone, m.email, b.name AS branch_name, m.source, m.status,
	COALESCE(cur.plan_name, '') AS plan_name, cur.end_date, m.created_at
FROM members m
JOIN branches b ON b.id = m.home_branch_id
LEFT JOIN LATERAL (
	SELECT p.name AS plan_name, s.end_date FROM subscriptions s JOIN membership_plans p ON p.id = s.plan_id
	WHERE s.member_id = m.id AND s.status IN ('ACTIVE','FROZEN')
	ORDER BY s.end_date DESC LIMIT 1
) cur ON TRUE
WHERE m.deleted_at IS NULL AND m.created_at >= @start AND m.created_at < @end` + reportBranch("m.home_branch_id", s.BranchIDs) + `
ORDER BY m.created_at DESC
LIMIT @limit`
	var rows []ReportMemberExportRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"limit": limit}))
	return rows, err
}

func (r *reportRepository) ExportCheckins(ctx context.Context, s ReportScope, limit int) ([]ReportCheckinExportRow, error) {
	q := `
SELECT c.checked_in_at, c.checked_out_at, m.member_code, m.full_name AS member_name, b.name AS branch_name,
	c.method, c.result, c.deny_reason, COALESCE(u.full_name, '') AS staff_name
FROM checkins c
JOIN members m ON m.id = c.member_id
JOIN branches b ON b.id = c.branch_id
LEFT JOIN users u ON u.id = c.staff_id
WHERE c.checked_in_at >= @start AND c.checked_in_at < @end` + reportBranch("c.branch_id", s.BranchIDs) + `
ORDER BY c.checked_in_at DESC
LIMIT @limit`
	var rows []ReportCheckinExportRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"limit": limit}))
	return rows, err
}

func (r *reportRepository) ExportSubscriptions(ctx context.Context, s ReportScope, limit int) ([]ReportSubExportRow, error) {
	q := `
SELECT s.id, m.member_code, m.full_name AS member_name, p.name AS plan_name, b.name AS branch_name,
	s.start_date, s.end_date, s.price, s.status, COALESCE(u.full_name, '') AS sold_by, s.created_at
FROM subscriptions s
JOIN members m ON m.id = s.member_id
JOIN membership_plans p ON p.id = s.plan_id
JOIN branches b ON b.id = s.branch_id
LEFT JOIN users u ON u.id = s.sold_by
WHERE s.created_at >= @start AND s.created_at < @end` + reportBranch("s.branch_id", s.BranchIDs) + `
ORDER BY s.created_at DESC
LIMIT @limit`
	var rows []ReportSubExportRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"limit": limit}))
	return rows, err
}

func (r *reportRepository) ExportInvoices(ctx context.Context, s ReportScope, limit int) ([]ReportInvoiceExportRow, error) {
	q := `
SELECT i.invoice_no, i.issued_at, COALESCE(m.member_code, '') AS member_code, COALESCE(m.full_name, 'Khách lẻ') AS member_name,
	b.name AS branch_name, i.source, i.subtotal, i.discount, i.total, i.paid_amount, i.refunded_amount, i.status,
	COALESCE(u.full_name, '') AS issued_by
FROM invoices i
LEFT JOIN members m ON m.id = i.member_id
JOIN branches b ON b.id = i.branch_id
LEFT JOIN users u ON u.id = i.issued_by
WHERE i.issued_at >= @start AND i.issued_at < @end` + reportBranch("i.branch_id", s.BranchIDs) + `
ORDER BY i.issued_at DESC
LIMIT @limit`
	var rows []ReportInvoiceExportRow
	err := r.raw(ctx, &rows, q, s.args(map[string]any{"limit": limit}))
	return rows, err
}
