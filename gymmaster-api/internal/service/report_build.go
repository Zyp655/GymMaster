package service

import (
	"fmt"
	"math"
	"sort"
	"time"

	v1 "gymmaster-api/api/report/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// Phần dựng/ghép dữ liệu báo cáo – hàm thuần, không chạm DB (unit test trong report_service_test.go).

var reportLoc = func() *time.Location {
	if loc, err := time.LoadLocation(consts.TimeZone); err == nil {
		return loc
	}
	return time.FixedZone("ICT", 7*3600)
}()

const (
	reportMaxDays      = 366 * 3 // khoảng tối đa của một báo cáo
	reportMaxDaysByDay = 400     // group_by=day tối đa ~ 13 tháng điểm
)

// reportRange là khoảng ngày VN [From, To] (00:00 giờ VN) đã chuẩn hóa.
type reportRange struct {
	From time.Time
	To   time.Time
}

func (r reportRange) Start() time.Time { return r.From }
func (r reportRange) End() time.Time   { return r.To.AddDate(0, 0, 1) }
func (r reportRange) Days() int        { return reportDaysBetween(r.From, r.To) + 1 }

func (r reportRange) Period() v1.Period {
	return v1.Period{From: r.From.Format(consts.DateLayout), To: r.To.Format(consts.DateLayout), Days: r.Days()}
}

// Previous: kỳ liền trước cùng số ngày.
func (r reportRange) Previous() reportRange {
	n := r.Days()
	return reportRange{From: r.From.AddDate(0, 0, -n), To: r.From.AddDate(0, 0, -1)}
}

func (r reportRange) scope(branchIDs []uint) repository.ReportScope {
	return repository.ReportScope{
		BranchIDs: branchIDs, Start: r.Start(), End: r.End(),
		FromDate: r.From.Format(consts.DateLayout), ToDate: r.To.Format(consts.DateLayout),
	}
}

func reportDay(t time.Time) time.Time {
	t = t.In(reportLoc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, reportLoc)
}

func reportDaysBetween(a, b time.Time) int {
	// Dùng ngày lịch (tránh lệch do DST – VN không có DST nhưng vẫn an toàn).
	ua := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	ub := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(ub.Sub(ua).Hours() / 24)
}

// reportParseRange: from/to rỗng → từ ngày 1 tháng hiện tại đến hôm nay.
func reportParseRange(from, to string, now time.Time) (reportRange, error) {
	today := reportDay(now)
	r := reportRange{From: time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, reportLoc), To: today}
	var err error
	if to != "" {
		if r.To, err = time.ParseInLocation(consts.DateLayout, to, reportLoc); err != nil {
			return r, rescode.New(rescode.BadRequest, "Đến ngày không hợp lệ")
		}
		if from == "" {
			r.From = time.Date(r.To.Year(), r.To.Month(), 1, 0, 0, 0, 0, reportLoc)
		}
	}
	if from != "" {
		if r.From, err = time.ParseInLocation(consts.DateLayout, from, reportLoc); err != nil {
			return r, rescode.New(rescode.BadRequest, "Từ ngày không hợp lệ")
		}
	}
	if r.To.Before(r.From) {
		return r, rescode.New(rescode.BadRequest, "Từ ngày phải trước hoặc bằng đến ngày")
	}
	if r.Days() > reportMaxDays {
		return r, rescode.New(rescode.BadRequest, "Khoảng thời gian tối đa 3 năm")
	}
	return r, nil
}

func reportCheckGroupBy(r reportRange, groupBy string) (string, error) {
	switch groupBy {
	case "", "day":
		if r.Days() > reportMaxDaysByDay {
			return "", rescode.New(rescode.BadRequest, "Khoảng thời gian quá dài để nhóm theo ngày, hãy chọn theo tuần hoặc tháng")
		}
		return "day", nil
	case "week", "month":
		return groupBy, nil
	}
	return "", rescode.New(rescode.BadRequest, "Nhóm theo phải là day, week hoặc month")
}

// reportBucketStart trùng với date_trunc của Postgres: tuần bắt đầu thứ 2, tháng bắt đầu ngày 1.
func reportBucketStart(t time.Time, groupBy string) time.Time {
	switch groupBy {
	case "week":
		wd := (int(t.Weekday()) + 6) % 7 // thứ 2 = 0
		return time.Date(t.Year(), t.Month(), t.Day()-wd, 0, 0, 0, 0, t.Location())
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// reportBuckets sinh đủ các mốc trục thời gian (không thiếu ngày/tuần/tháng nào) trong [from, to].
func reportBuckets(r reportRange, groupBy string) []string {
	var out []string
	for b := reportBucketStart(r.From, groupBy); !b.After(r.To); {
		out = append(out, b.Format(consts.DateLayout))
		switch groupBy {
		case "week":
			b = b.AddDate(0, 0, 7)
		case "month":
			b = b.AddDate(0, 1, 0)
		default:
			b = b.AddDate(0, 0, 1)
		}
	}
	return out
}

// reportBucketLabel: nhãn trục X tiếng Việt.
func reportBucketLabel(period, groupBy string) string {
	t, err := time.Parse(consts.DateLayout, period)
	if err != nil {
		return period
	}
	switch groupBy {
	case "week":
		return "Tuần " + t.Format("02/01")
	case "month":
		return t.Format("01/2006")
	}
	return t.Format("02/01")
}

func reportRound(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}

// reportMoney làm tròn tiền đến đồng.
func reportMoney(v float64) float64 { return math.Round(v) }

func reportPct(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return reportRound(part*100/total, 1)
}

// reportChange: % thay đổi so với kỳ trước; nil khi kỳ trước = 0 (không xác định).
func reportChange(cur, prev float64) *float64 {
	if prev == 0 {
		return nil
	}
	v := reportRound((cur-prev)*100/math.Abs(prev), 1)
	return &v
}

// ---------------------------------------------------------------------------
// Doanh thu
// ---------------------------------------------------------------------------

var (
	reportItemTypeOrder = []string{"MEMBERSHIP", "PT", "PRODUCT", "FEE", "OTHER"}
	reportItemTypeLabel = map[string]string{
		"MEMBERSHIP": "Gói tập", "PT": "Huấn luyện cá nhân", "PRODUCT": "Sản phẩm", "FEE": "Phí", "OTHER": "Khác",
	}
	reportMethodOrder = []string{"CASH", "TRANSFER", "CARD"}
	reportMethodLabel = map[string]string{"CASH": "Tiền mặt", "TRANSFER": "Chuyển khoản", "CARD": "Thẻ"}
)

// reportBuildRevenue ghép các dòng thực thu thành chuỗi thời gian + các phân rã.
func reportBuildRevenue(rows []repository.ReportRevenueRow, buckets []string, groupBy string, branchNames map[uint]string) v1.RevenueRes {
	res := v1.RevenueRes{GroupBy: groupBy}
	idx := make(map[string]int, len(buckets))
	res.Series = make([]v1.RevenuePoint, len(buckets))
	for i, b := range buckets {
		idx[b] = i
		res.Series[i] = v1.RevenuePoint{Period: b, Label: reportBucketLabel(b, groupBy)}
	}
	byType := map[string]float64{}
	byMethod := map[string]float64{}
	byBranch := map[uint]float64{}
	for _, r := range rows {
		amt := r.Amount
		if r.Kind == "REFUND" {
			res.Totals.Refunds -= amt
		} else {
			res.Totals.Payments += amt
		}
		byType[r.ItemType] += amt
		byMethod[r.Method] += amt
		byBranch[r.BranchID] += amt
		i, ok := idx[r.Bucket]
		if !ok {
			continue
		}
		p := &res.Series[i]
		p.Total += amt
		if r.Kind == "REFUND" {
			p.Refunds -= amt
		} else {
			p.Payments += amt
		}
		switch r.ItemType {
		case "MEMBERSHIP":
			p.Membership += amt
		case "PT":
			p.PT += amt
		case "PRODUCT":
			p.Product += amt
		case "FEE":
			p.Fee += amt
		default:
			p.Other += amt
		}
	}
	for i := range res.Series {
		p := &res.Series[i]
		p.Total, p.Membership, p.PT, p.Product, p.Fee, p.Other = reportMoney(p.Total), reportMoney(p.Membership),
			reportMoney(p.PT), reportMoney(p.Product), reportMoney(p.Fee), reportMoney(p.Other)
		p.Payments, p.Refunds = reportMoney(p.Payments), reportMoney(p.Refunds)
	}
	res.Totals.Payments, res.Totals.Refunds = reportMoney(res.Totals.Payments), reportMoney(res.Totals.Refunds)
	res.Totals.Net = res.Totals.Payments - res.Totals.Refunds

	res.ByType = reportNameValues(byType, reportItemTypeOrder, reportItemTypeLabel, res.Totals.Net)
	res.ByMethod = reportNameValues(byMethod, reportMethodOrder, reportMethodLabel, res.Totals.Net)
	res.ByBranch = make([]v1.BranchAmount, 0, len(byBranch))
	for id, amt := range byBranch {
		res.ByBranch = append(res.ByBranch, v1.BranchAmount{
			BranchID: id, BranchName: branchNames[id], Amount: reportMoney(amt), Pct: reportPct(amt, res.Totals.Net),
		})
	}
	sort.Slice(res.ByBranch, func(i, j int) bool {
		if res.ByBranch[i].Amount != res.ByBranch[j].Amount {
			return res.ByBranch[i].Amount > res.ByBranch[j].Amount
		}
		return res.ByBranch[i].BranchID < res.ByBranch[j].BranchID
	})
	return res
}

// reportNameValues: luôn trả đủ các khóa trong order (giá trị 0 nếu không có) + các khóa lạ ở cuối.
func reportNameValues(m map[string]float64, order []string, labels map[string]string, total float64) []v1.NameValue {
	out := make([]v1.NameValue, 0, len(order))
	seen := map[string]bool{}
	for _, k := range order {
		seen[k] = true
		out = append(out, v1.NameValue{Key: k, Label: labels[k], Value: reportMoney(m[k]), Pct: reportPct(m[k], total)})
	}
	var extra []string
	for k := range m {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		out = append(out, v1.NameValue{Key: k, Label: k, Value: reportMoney(m[k]), Pct: reportPct(m[k], total)})
	}
	return out
}

func reportSumNet(rows []repository.ReportRevenueRow) float64 {
	var s float64
	for _, r := range rows {
		s += r.Amount
	}
	return reportMoney(s)
}

// ---------------------------------------------------------------------------
// Hội viên
// ---------------------------------------------------------------------------

var (
	reportSourceOrder = []string{"WALK_IN", "REFERRAL", "FACEBOOK", "WEBSITE", "OTHER"}
	reportSourceLabel = map[string]string{"WALK_IN": "Khách vãng lai", "REFERRAL": "Giới thiệu", "FACEBOOK": "Facebook", "WEBSITE": "Website", "OTHER": "Khác"}
	reportGenderOrder = []string{"MALE", "FEMALE", "OTHER"}
	reportGenderLabel = map[string]string{"MALE": "Nam", "FEMALE": "Nữ", "OTHER": "Khác"}
	reportAgeOrder    = []string{"UNDER_18", "18_24", "25_34", "35_44", "45_54", "55_PLUS", "UNKNOWN"}
	reportAgeLabel    = map[string]string{"UNDER_18": "Dưới 18", "18_24": "18–24", "25_34": "25–34", "35_44": "35–44", "45_54": "45–54", "55_PLUS": "Từ 55", "UNKNOWN": "Chưa rõ"}
	reportStatusOrder = []string{"ACTIVE", "FROZEN", "PENDING", "EXPIRED", "NONE"}
	reportStatusLabel = map[string]string{"ACTIVE": "Đang hoạt động", "FROZEN": "Bảo lưu", "PENDING": "Chờ thanh toán", "EXPIRED": "Hết hạn", "NONE": "Chưa có gói"}
)

func reportNameCounts(m map[string]int64, order []string, labels map[string]string) ([]v1.NameCount, int64) {
	var total int64
	for _, v := range m {
		total += v
	}
	out := make([]v1.NameCount, 0, len(order))
	seen := map[string]bool{}
	for _, k := range order {
		seen[k] = true
		out = append(out, v1.NameCount{Key: k, Label: labels[k], Count: m[k], Pct: reportPct(float64(m[k]), float64(total))})
	}
	var extra []string
	for k := range m {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		lb := labels[k]
		if lb == "" {
			lb = k
		}
		out = append(out, v1.NameCount{Key: k, Label: lb, Count: m[k], Pct: reportPct(float64(m[k]), float64(total))})
	}
	return out, total
}

func reportCountSeries(rows []repository.ReportCountRow, buckets []string, groupBy string) ([]v1.CountPoint, int64) {
	m := make(map[string]int64, len(rows))
	for _, r := range rows {
		m[r.Bucket] += r.Count
	}
	out := make([]v1.CountPoint, len(buckets))
	var total int64
	for i, b := range buckets {
		out[i] = v1.CountPoint{Period: b, Label: reportBucketLabel(b, groupBy), Count: m[b]}
		total += m[b]
	}
	return out, total
}

// reportBuildMemberDims tách các dòng phân bố (dim = source/gender/age/branch) vào response.
func reportBuildMemberDims(res *v1.MembersRes, rows []repository.ReportDimRow, branchNames map[uint]string) {
	maps := map[string]map[string]int64{"source": {}, "gender": {}, "age": {}, "branch": {}}
	for _, r := range rows {
		if m, ok := maps[r.Dim]; ok {
			m[r.Key] += r.Count
		}
	}
	res.BySource, _ = reportNameCounts(maps["source"], reportSourceOrder, reportSourceLabel)
	res.ByGender, _ = reportNameCounts(maps["gender"], reportGenderOrder, reportGenderLabel)
	res.ByAge, _ = reportNameCounts(maps["age"], reportAgeOrder, reportAgeLabel)

	labels := map[string]string{}
	keys := make([]string, 0, len(maps["branch"]))
	for k := range maps["branch"] {
		var id uint
		_, _ = fmt.Sscan(k, &id)
		labels[k] = branchNames[id]
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return maps["branch"][keys[i]] > maps["branch"][keys[j]] || (maps["branch"][keys[i]] == maps["branch"][keys[j]] && keys[i] < keys[j])
	})
	res.ByBranch, _ = reportNameCounts(maps["branch"], keys, labels)
}

// ---------------------------------------------------------------------------
// Retention
// ---------------------------------------------------------------------------

func reportRetentionPoint(month string, r repository.ReportRetentionRow) v1.RetentionPoint {
	p := v1.RetentionPoint{Month: month, Expired: r.Expired, Renewed: r.Renewed, Pending: r.Pending}
	p.Churned = max(r.Expired-r.Renewed-r.Pending, 0)
	p.RenewalRate = reportPct(float64(r.Renewed), float64(r.Expired))
	p.ChurnRate = reportPct(float64(p.Churned), float64(r.Expired))
	if month != "" {
		if t, err := time.Parse("2006-01", month); err == nil {
			p.Label = t.Format("01/2006")
		}
	} else {
		p.Label = "Tổng"
	}
	return p
}

// reportBuildRetention: đủ các tháng trong kỳ (tháng không có gói hết hạn = 0), dòng month NULL là tổng.
func reportBuildRetention(rows []repository.ReportRetentionRow, r reportRange) (v1.RetentionPoint, []v1.RetentionPoint) {
	byMonth := map[string]repository.ReportRetentionRow{}
	summary := reportRetentionPoint("", repository.ReportRetentionRow{})
	for _, row := range rows {
		if row.Month == nil {
			summary = reportRetentionPoint("", row)
			continue
		}
		byMonth[*row.Month] = row
	}
	var months []v1.RetentionPoint
	for _, b := range reportBuckets(r, "month") {
		m := b[:7]
		months = append(months, reportRetentionPoint(m, byMonth[m]))
	}
	return summary, months
}

// ---------------------------------------------------------------------------
// Heatmap check-in
// ---------------------------------------------------------------------------

var reportDowLabel = []string{"", "T2", "T3", "T4", "T5", "T6", "T7", "CN"}

// reportWeekdayCounts: số lần xuất hiện của từng thứ (ISODOW 1..7) trong khoảng.
func reportWeekdayCounts(r reportRange) [8]int {
	var out [8]int
	for d := r.From; !d.After(r.To); d = d.AddDate(0, 0, 1) {
		out[(int(d.Weekday())+6)%7+1]++
	}
	return out
}

func reportBuildHeatmap(rows []repository.ReportHeatmapRow, r reportRange) v1.CheckinHeatmapRes {
	res := v1.CheckinHeatmapRes{Period: r.Period(), ByHour: make([]int64, 24)}
	wd := reportWeekdayCounts(r)
	res.Matrix = make([]v1.HeatmapDay, 7)
	for i := range res.Matrix {
		res.Matrix[i] = v1.HeatmapDay{Dow: i + 1, Label: reportDowLabel[i+1], Hours: make([]int64, 24)}
	}
	for _, row := range rows {
		if row.Dow < 1 || row.Dow > 7 || row.Hour < 0 || row.Hour > 23 {
			continue
		}
		d := &res.Matrix[row.Dow-1]
		d.Hours[row.Hour] += row.Count
		d.Total += row.Count
		res.ByHour[row.Hour] += row.Count
		res.Total += row.Count
	}
	for i := range res.Matrix {
		d := &res.Matrix[i]
		if wd[d.Dow] > 0 {
			d.AvgDay = reportRound(float64(d.Total)/float64(wd[d.Dow]), 1)
		}
		for h, v := range d.Hours {
			if v > res.Max {
				res.Max = v
				res.Peak = &v1.PeakSlot{Dow: d.Dow, Hour: h, Count: v, Label: fmt.Sprintf("%s %02d:00", d.Label, h)}
			}
		}
	}
	if days := r.Days(); days > 0 {
		res.AvgPerDay = reportRound(float64(res.Total)/float64(days), 1)
	}
	hours := make([]int, 24)
	for i := range hours {
		hours[i] = i
	}
	sort.SliceStable(hours, func(i, j int) bool { return res.ByHour[hours[i]] > res.ByHour[hours[j]] })
	res.PeakHours = []int{}
	for _, h := range hours[:3] {
		if res.ByHour[h] > 0 {
			res.PeakHours = append(res.PeakHours, h)
		}
	}
	return res
}

// ---------------------------------------------------------------------------
// Lấp đầy lớp
// ---------------------------------------------------------------------------

type reportOccAcc struct {
	row   v1.OccupancyRow
	order int
}

func reportOccAdd(m map[string]*reportOccAcc, key, label string, id uint, color string, order int, s repository.ReportClassRow) {
	a, ok := m[key]
	if !ok {
		a = &reportOccAcc{row: v1.OccupancyRow{Key: key, Label: label, ID: id, Color: color}, order: order}
		m[key] = a
	}
	reportOccAccumulate(&a.row, s)
}

func reportOccAccumulate(o *v1.OccupancyRow, s repository.ReportClassRow) {
	o.Sessions++
	o.Capacity += s.Capacity
	o.Booked += s.Booked
	o.Attended += s.Attended
	o.NoShow += s.NoShow
	o.Cancelled += s.Cancelled
	o.Waitlist += s.Waitlist
}

func reportOccFinish(o *v1.OccupancyRow) {
	o.FillRate = reportPct(float64(o.Booked), float64(o.Capacity))
	o.AttendanceRate = reportPct(float64(o.Attended), float64(o.Attended+o.NoShow))
}

// reportOccList: sắp theo order (nếu > 0, ví dụ giờ/thứ) hoặc theo số buổi giảm dần.
func reportOccList(m map[string]*reportOccAcc, byOrder bool) []v1.OccupancyRow {
	list := make([]*reportOccAcc, 0, len(m))
	for _, a := range m {
		reportOccFinish(&a.row)
		list = append(list, a)
	}
	sort.Slice(list, func(i, j int) bool {
		if byOrder {
			return list[i].order < list[j].order
		}
		if list[i].row.Sessions != list[j].row.Sessions {
			return list[i].row.Sessions > list[j].row.Sessions
		}
		return list[i].row.Label < list[j].row.Label
	})
	out := make([]v1.OccupancyRow, len(list))
	for i, a := range list {
		out[i] = a.row
	}
	return out
}

func reportBuildOccupancy(rows []repository.ReportClassRow) (v1.OccupancyRow, []v1.OccupancyRow, []v1.OccupancyRow, []v1.OccupancyRow, []v1.OccupancyRow) {
	summary := v1.OccupancyRow{Key: "ALL", Label: "Tổng"}
	types, trainers, slots, days := map[string]*reportOccAcc{}, map[string]*reportOccAcc{}, map[string]*reportOccAcc{}, map[string]*reportOccAcc{}
	for _, s := range rows {
		reportOccAccumulate(&summary, s)
		reportOccAdd(types, fmt.Sprint(s.ClassTypeID), s.ClassTypeName, s.ClassTypeID, s.Color, 0, s)
		reportOccAdd(trainers, fmt.Sprint(s.TrainerID), s.TrainerName, s.TrainerID, "", 0, s)
		slot := fmt.Sprintf("%02d:00", s.Hour)
		reportOccAdd(slots, slot, slot, 0, "", s.Hour, s)
		if s.Dow >= 1 && s.Dow <= 7 {
			reportOccAdd(days, fmt.Sprint(s.Dow), reportDowLabel[s.Dow], 0, "", s.Dow, s)
		}
	}
	reportOccFinish(&summary)
	return summary, reportOccList(types, false), reportOccList(trainers, false), reportOccList(slots, true), reportOccList(days, true)
}

// ---------------------------------------------------------------------------
// HLV
// ---------------------------------------------------------------------------

func reportBuildTrainers(rows []repository.ReportTrainerRow) ([]v1.TrainerPerformance, v1.TrainerPerformance) {
	items := make([]v1.TrainerPerformance, 0, len(rows))
	total := v1.TrainerPerformance{TrainerName: "Tổng"}
	var weightedRating float64
	for _, r := range rows {
		it := v1.TrainerPerformance{
			TrainerID: r.TrainerID, TrainerName: r.TrainerName, Level: r.Level,
			PTCompleted: r.PTCompleted, PTScheduled: r.PTScheduled, PTNoShow: r.PTNoShow, PTCancelled: r.PTCancelled,
			ClassesTaught: r.ClassesTaught, ClassAttendees: r.ClassAttendees, ActivePTMembers: r.ActivePTMembers,
			PTContractsSold: r.PTContractsSold, PTRevenue: reportMoney(r.PTRevenue),
			Commission: reportMoney(r.Commission), CommissionPaid: reportMoney(r.CommissionPaid),
			RatingAvg: r.RatingAvg, RatingCount: r.RatingCount,
		}
		items = append(items, it)
		total.PTCompleted += it.PTCompleted
		total.PTScheduled += it.PTScheduled
		total.PTNoShow += it.PTNoShow
		total.PTCancelled += it.PTCancelled
		total.ClassesTaught += it.ClassesTaught
		total.ClassAttendees += it.ClassAttendees
		total.ActivePTMembers += it.ActivePTMembers
		total.PTContractsSold += it.PTContractsSold
		total.PTRevenue += it.PTRevenue
		total.Commission += it.Commission
		total.CommissionPaid += it.CommissionPaid
		total.RatingCount += it.RatingCount
		weightedRating += it.RatingAvg * float64(it.RatingCount)
	}
	if total.RatingCount > 0 {
		total.RatingAvg = math.Round((weightedRating/float64(total.RatingCount))*100) / 100
	}
	return items, total
}

// ---------------------------------------------------------------------------
// Sản phẩm
// ---------------------------------------------------------------------------

func reportBuildProducts(rows []repository.ReportProductRow, limit int) v1.SalesProductsRes {
	res := v1.SalesProductsRes{Totals: v1.ProductSales{Name: "Tổng"}, Products: len(rows)}
	cats := map[string]*v1.CategorySales{}
	var catOrder []string
	all := make([]v1.ProductSales, 0, len(rows))
	for _, r := range rows {
		p := v1.ProductSales{
			ProductID: r.ProductID, SKU: r.SKU, Name: r.Name, CategoryName: r.CategoryName,
			Quantity: r.Quantity, Revenue: reportMoney(r.Revenue), Cost: reportMoney(r.Cost),
		}
		p.GrossProfit = p.Revenue - p.Cost
		p.Margin = reportPct(p.GrossProfit, p.Revenue)
		all = append(all, p)
		res.Totals.Quantity += p.Quantity
		res.Totals.Revenue += p.Revenue
		res.Totals.Cost += p.Cost
		name := r.CategoryName
		if name == "" {
			name = "Chưa phân loại"
		}
		c, ok := cats[name]
		if !ok {
			c = &v1.CategorySales{CategoryName: name}
			cats[name] = c
			catOrder = append(catOrder, name)
		}
		c.Quantity += p.Quantity
		c.Revenue += p.Revenue
		c.GrossProfit += p.GrossProfit
	}
	res.Totals.GrossProfit = res.Totals.Revenue - res.Totals.Cost
	res.Totals.Margin = reportPct(res.Totals.GrossProfit, res.Totals.Revenue)
	sort.SliceStable(all, func(i, j int) bool { return all[i].Revenue > all[j].Revenue })
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	res.Top = all
	res.ByCategory = make([]v1.CategorySales, 0, len(cats))
	for _, n := range catOrder {
		c := cats[n]
		c.Pct = reportPct(c.Revenue, res.Totals.Revenue)
		res.ByCategory = append(res.ByCategory, *c)
	}
	sort.SliceStable(res.ByCategory, func(i, j int) bool { return res.ByCategory[i].Revenue > res.ByCategory[j].Revenue })
	return res
}

// ---------------------------------------------------------------------------
// Gói tập
// ---------------------------------------------------------------------------

func reportAddSubCount(c *v1.SubscriptionCounts, kind string, n int64) {
	switch kind {
	case "NEW":
		c.New += n
	case "RENEWED":
		c.Renewed += n
	case "UPGRADED":
		c.Upgraded += n
	case "TRANSFERRED":
		c.Transferred += n
	case "CANCELLED":
		c.Cancelled += n
	case "EXPIRED":
		c.Expired += n
	case "FROZEN":
		c.Frozen += n
	}
}

func reportBuildSubscriptions(events []repository.ReportSubEventRow, plans []repository.ReportPlanRow, buckets []string, groupBy string) v1.SubscriptionsRes {
	res := v1.SubscriptionsRes{GroupBy: groupBy}
	idx := make(map[string]int, len(buckets))
	res.Series = make([]v1.SubscriptionPoint, len(buckets))
	for i, b := range buckets {
		idx[b] = i
		res.Series[i] = v1.SubscriptionPoint{Period: b, Label: reportBucketLabel(b, groupBy)}
	}
	for _, e := range events {
		reportAddSubCount(&res.Counts, e.Kind, e.Count)
		if i, ok := idx[e.Bucket]; ok {
			reportAddSubCount(&res.Series[i].SubscriptionCounts, e.Kind, e.Count)
		}
	}
	var total float64
	for _, p := range plans {
		total += p.Collected
	}
	res.ByPlan = make([]v1.PlanSales, 0, len(plans))
	for _, p := range plans {
		res.ByPlan = append(res.ByPlan, v1.PlanSales{
			PlanID: p.PlanID, PlanCode: p.PlanCode, PlanName: p.PlanName, PlanType: p.PlanType,
			Sold: p.Sold, NewSales: p.NewSales, Continued: p.Continued,
			SoldAmount: reportMoney(p.SoldAmount), Collected: reportMoney(p.Collected), Pct: reportPct(p.Collected, total),
		})
	}
	return res
}
