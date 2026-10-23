package service

import (
	"context"
	"time"

	v1 "gymmaster-api/api/report/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

// ReportService: báo cáo & dashboard. Mọi số liệu tính bằng SQL gộp (GROUP BY) – không N+1;
// phần ghép chuỗi thời gian/phân rã nằm trong report_build.go.
type ReportService interface {
	Overview(ctx context.Context, req *v1.OverviewReq) (*v1.OverviewRes, error)
	Revenue(ctx context.Context, req *v1.RevenueReq) (*v1.RevenueRes, error)
	Members(ctx context.Context, req *v1.MembersReq) (*v1.MembersRes, error)
	Retention(ctx context.Context, req *v1.RetentionReq) (*v1.RetentionRes, error)
	CheckinHeatmap(ctx context.Context, req *v1.CheckinHeatmapReq) (*v1.CheckinHeatmapRes, error)
	ClassOccupancy(ctx context.Context, req *v1.ClassOccupancyReq) (*v1.ClassOccupancyRes, error)
	TrainerPerformance(ctx context.Context, req *v1.TrainerPerformanceReq) (*v1.TrainerPerformanceRes, error)
	SalesProducts(ctx context.Context, req *v1.SalesProductsReq) (*v1.SalesProductsRes, error)
	Subscriptions(ctx context.Context, req *v1.SubscriptionsReq) (*v1.SubscriptionsRes, error)
	// Export trả nội dung file .xlsx và tên file (không dấu).
	Export(ctx context.Context, req *v1.ExportReq) ([]byte, string, error)
}

type reportService struct {
	repo repository.ReportRepository
	now  func() time.Time
}

func NewReportService(repo repository.ReportRepository) ReportService {
	return &reportService{repo: repo, now: time.Now}
}

func (s *reportService) branches(ctx context.Context, requested uint) ([]uint, error) {
	ids, ok := reqctx.ResolveBranchFilter(ctx, requested)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem báo cáo của chi nhánh này")
	}
	// Chi nhánh không tồn tại → 404 (thống nhất với KPI/cohort/P&L) thay vì báo cáo toàn số 0.
	if requested > 0 {
		exists, err := s.repo.BranchExists(ctx, requested)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy chi nhánh")
		}
	}
	return ids, nil
}

// prepare: phạm vi chi nhánh + khoảng ngày.
func (s *reportService) prepare(ctx context.Context, f v1.Filter) ([]uint, reportRange, error) {
	ids, err := s.branches(ctx, f.BranchID)
	if err != nil {
		return nil, reportRange{}, err
	}
	r, err := reportParseRange(f.From, f.To, s.now())
	return ids, r, err
}

func (s *reportService) Overview(ctx context.Context, req *v1.OverviewReq) (*v1.OverviewRes, error) {
	ids, err := s.branches(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	today := reportDay(now)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, reportLoc)
	prevMonthStart := monthStart.AddDate(0, -1, 0)
	// So sánh month-to-date: cùng số ngày đã qua của tháng trước (không vượt quá cuối tháng trước).
	prevSame := prevMonthStart.AddDate(0, 0, today.Day())
	if prevSame.After(monthStart) {
		prevSame = monthStart
	}
	row, err := s.repo.Overview(ctx, repository.ReportOverviewParams{
		BranchIDs: ids, Now: now, TodayStart: today, TomorrowStart: today.AddDate(0, 0, 1),
		MonthStart: monthStart, PrevMonthStart: prevMonthStart, PrevMonthSame: prevSame,
		Today: today.Format(consts.DateLayout), Today7: today.AddDate(0, 0, 7).Format(consts.DateLayout),
	})
	if err != nil {
		return nil, err
	}
	low, err := s.repo.LowStock(ctx, ids, 5)
	if err != nil {
		return nil, err
	}
	res := &v1.Overview{
		Date: today.Format(consts.DateLayout),
		Revenue: v1.OverviewRevenue{
			Today: reportMoney(row.RevenueToday), Month: reportMoney(row.RevenueMonth),
			PrevMonthToDate: reportMoney(row.RevenuePrevMonthToDate), PrevMonth: reportMoney(row.RevenuePrevMonth),
			ChangePct: reportChange(row.RevenueMonth, row.RevenuePrevMonthToDate),
		},
		Members: v1.OverviewMembers{
			Active: row.ActiveMembers, NewThisMonth: row.NewMembersMonth, NewPrevMonth: row.NewMembersPrevMonth,
			NewChangePct: reportChange(float64(row.NewMembersMonth), float64(row.NewMembersPrevMonth)),
			ExpiringIn7d: row.Expiring7d,
		},
		Checkin: v1.OverviewCheckin{
			Today: row.CheckinsToday, Present: row.PresentNow,
			PresentSince: repository.PresentSince(now), PresentWindowHours: int(repository.PresentWindow / time.Hour),
		},
		Classes: v1.OverviewClasses{
			Today: row.ClassesToday, Capacity: row.ClassCapacityToday, Booked: row.ClassBookedToday,
			FillRate: reportPct(float64(row.ClassBookedToday), float64(row.ClassCapacityToday)),
		},
		PT:             v1.OverviewPT{Today: row.PTSessionsToday, Completed: row.PTCompletedToday},
		PendingFreezes: row.PendingFreezes,
		LowStock:       row.LowStock,
		LowStockItems:  make([]v1.LowStockItem, 0, len(low)),
	}
	for _, l := range low {
		res.LowStockItems = append(res.LowStockItems, v1.LowStockItem{
			ProductID: l.ProductID, SKU: l.SKU, Name: l.Name, BranchID: l.BranchID, BranchName: l.BranchName,
			Quantity: l.Quantity, MinStock: l.MinStock,
		})
	}
	return res, nil
}

func (s *reportService) Revenue(ctx context.Context, req *v1.RevenueReq) (*v1.RevenueRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	groupBy, err := reportCheckGroupBy(r, req.GroupBy.GroupBy)
	if err != nil {
		return nil, err
	}
	return s.revenue(ctx, ids, r, groupBy)
}

func (s *reportService) revenue(ctx context.Context, ids []uint, r reportRange, groupBy string) (*v1.RevenueRes, error) {
	rows, err := s.repo.RevenueFlows(ctx, r.scope(ids), groupBy)
	if err != nil {
		return nil, err
	}
	prev := r.Previous()
	prevRows, err := s.repo.RevenueFlows(ctx, prev.scope(ids), "month")
	if err != nil {
		return nil, err
	}
	names, err := s.repo.BranchNames(ctx)
	if err != nil {
		return nil, err
	}
	res := reportBuildRevenue(rows, reportBuckets(r, groupBy), groupBy, names)
	res.Period = r.Period()
	prevNet := reportSumNet(prevRows)
	res.Previous = v1.RevenueCompare{Period: prev.Period(), Net: prevNet, ChangePct: reportChange(res.Totals.Net, prevNet)}
	return &res, nil
}

func (s *reportService) Members(ctx context.Context, req *v1.MembersReq) (*v1.MembersRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	groupBy, err := reportCheckGroupBy(r, req.GroupBy.GroupBy)
	if err != nil {
		return nil, err
	}
	pop := req.Population
	if pop != "all" {
		pop = "new"
	}
	sc := r.scope(ids)
	series, err := s.repo.NewMemberSeries(ctx, sc, groupBy)
	if err != nil {
		return nil, err
	}
	dims, err := s.repo.MemberDistribution(ctx, sc, pop == "new")
	if err != nil {
		return nil, err
	}
	status, err := s.repo.MemberStatusCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	names, err := s.repo.BranchNames(ctx)
	if err != nil {
		return nil, err
	}
	res := &v1.MembersRes{Period: r.Period(), GroupBy: groupBy, Population: pop}
	res.NewSeries, res.NewTotal = reportCountSeries(series, reportBuckets(r, groupBy), groupBy)
	reportBuildMemberDims(res, dims, names)
	sm := map[string]int64{}
	for _, row := range status {
		sm[row.Key] += row.Count
	}
	res.Status, res.TotalMembers = reportNameCounts(sm, reportStatusOrder, reportStatusLabel)
	return res, nil
}

func (s *reportService) Retention(ctx context.Context, req *v1.RetentionReq) (*v1.RetentionRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	window := req.WindowDays
	if window <= 0 {
		window = 30
	}
	today := reportDay(s.now()).Format(consts.DateLayout)
	rows, err := s.repo.Retention(ctx, r.scope(ids), window, today)
	if err != nil {
		return nil, err
	}
	res := &v1.RetentionRes{Period: r.Period(), WindowDays: window}
	res.Summary, res.Months = reportBuildRetention(rows, r)
	return res, nil
}

func (s *reportService) CheckinHeatmap(ctx context.Context, req *v1.CheckinHeatmapReq) (*v1.CheckinHeatmapRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	sc := r.scope(ids)
	rows, err := s.repo.CheckinHeatmap(ctx, sc)
	if err != nil {
		return nil, err
	}
	totals, err := s.repo.CheckinTotals(ctx, sc)
	if err != nil {
		return nil, err
	}
	res := reportBuildHeatmap(rows, r)
	res.Denied, res.UniqueMembers = totals.Denied, totals.UniqueMembers
	res.Daily = []v1.CountPoint{}
	if r.Days() <= reportMaxDaysByDay {
		daily, err := s.repo.CheckinDaily(ctx, sc)
		if err != nil {
			return nil, err
		}
		res.Daily, _ = reportCountSeries(daily, reportBuckets(r, "day"), "day")
	}
	return &res, nil
}

func (s *reportService) ClassOccupancy(ctx context.Context, req *v1.ClassOccupancyReq) (*v1.ClassOccupancyRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ClassScheduleStats(ctx, r.scope(ids))
	if err != nil {
		return nil, err
	}
	res := &v1.ClassOccupancyRes{Period: r.Period()}
	res.Summary, res.ByClassType, res.ByTrainer, res.ByTimeSlot, res.ByWeekday = reportBuildOccupancy(rows)
	return res, nil
}

func (s *reportService) TrainerPerformance(ctx context.Context, req *v1.TrainerPerformanceReq) (*v1.TrainerPerformanceRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.TrainerPerformance(ctx, r.scope(ids), s.now())
	if err != nil {
		return nil, err
	}
	res := &v1.TrainerPerformanceRes{Period: r.Period()}
	res.Items, res.Totals = reportBuildTrainers(rows)
	return res, nil
}

func (s *reportService) SalesProducts(ctx context.Context, req *v1.SalesProductsReq) (*v1.SalesProductsRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ProductSales(ctx, r.scope(ids))
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	res := reportBuildProducts(rows, limit)
	res.Period = r.Period()
	return &res, nil
}

func (s *reportService) Subscriptions(ctx context.Context, req *v1.SubscriptionsReq) (*v1.SubscriptionsRes, error) {
	ids, r, err := s.prepare(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	groupBy, err := reportCheckGroupBy(r, req.GroupBy.GroupBy)
	if err != nil {
		return nil, err
	}
	return s.subscriptions(ctx, ids, r, groupBy)
}

func (s *reportService) subscriptions(ctx context.Context, ids []uint, r reportRange, groupBy string) (*v1.SubscriptionsRes, error) {
	sc := r.scope(ids)
	events, err := s.repo.SubscriptionEvents(ctx, sc, groupBy)
	if err != nil {
		return nil, err
	}
	plans, err := s.repo.SubscriptionPlans(ctx, sc)
	if err != nil {
		return nil, err
	}
	res := reportBuildSubscriptions(events, plans, reportBuckets(r, groupBy), groupBy)
	res.Period = r.Period()
	return &res, nil
}
