package service

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/kpi/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
)

const auditEntitySalesTarget = "sales_target"

var (
	salesTargetOrderable = []string{"id", "period", "metric", "target_value", "created_at"}
	validKPIMetrics      = []string{
		entity.KPIMetricRevenue,
		entity.KPIMetricNewMembers,
		entity.KPIMetricRenewals,
		entity.KPIMetricPTSales,
	}
	kpiMetricLabels = map[string]string{
		entity.KPIMetricRevenue:    "Doanh thu thuần",
		entity.KPIMetricNewMembers: "Hội viên mới",
		entity.KPIMetricRenewals:   "Gia hạn gói tập",
		entity.KPIMetricPTSales:    "Doanh số PT",
	}
)

type (
	KPIService interface {
		BranchOccupancy(ctx context.Context) ([]*v1.BranchOccupancyItem, error)

		ListSalesTargets(ctx context.Context, req *v1.ListSalesTargetsReq) (*v1.ListSalesTargetsRes, error)
		CreateSalesTarget(ctx context.Context, req *v1.CreateSalesTargetReq) (*v1.SalesTargetItem, error)
		DeleteSalesTarget(ctx context.Context, id uint) error

		KPIReport(ctx context.Context, req *v1.KPIReportReq) (*v1.KPIReportRes, error)
		CohortReport(ctx context.Context, req *v1.CohortReportReq) (*v1.CohortReportRes, error)
	}

	kpiService struct {
		repo  repository.KPIRepository
		audit AuditService
		now   func() time.Time
	}
)

func NewKPIService(repo repository.KPIRepository, audit AuditService) KPIService {
	return &kpiService{repo: repo, audit: audit, now: time.Now}
}

func classifyOccupancyLevel(pct float64) string {
	switch {
	case pct >= 90:
		return "FULL"
	case pct >= 65:
		return "BUSY"
	case pct >= 30:
		return "NORMAL"
	default:
		return "QUIET"
	}
}

func (s *kpiService) BranchOccupancy(ctx context.Context) ([]*v1.BranchOccupancyItem, error) {
	since := s.now().Add(-3 * time.Hour)
	rows, err := s.repo.ListBranchOccupancy(ctx, since)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.BranchOccupancyItem, 0, len(rows))
	for _, r := range rows {
		capVal := r.Capacity
		if capVal <= 0 {
			capVal = 120
		}
		pct := math.Round((float64(r.PresentCount)/float64(capVal))*1000) / 10
		if pct > 100 {
			pct = 100
		}
		out = append(out, &v1.BranchOccupancyItem{
			BranchID:     r.BranchID,
			BranchCode:   r.BranchCode,
			BranchName:   r.BranchName,
			PresentCount: r.PresentCount,
			Capacity:     capVal,
			OccupancyPct: pct,
			Level:        classifyOccupancyLevel(pct),
		})
	}
	return out, nil
}

func (s *kpiService) ListSalesTargets(ctx context.Context, req *v1.ListSalesTargetsReq) (*v1.ListSalesTargetsRes, error) {
	q := req.PageReq.ToQuery(salesTargetOrderable, "id")
	q.OrderBy = "st." + q.OrderBy
	rows, total, err := s.repo.ListSalesTargets(ctx, q, req.BranchID, strings.TrimSpace(req.Period), strings.TrimSpace(req.Metric))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toSalesTargetItem), total, q), nil
}

func (s *kpiService) CreateSalesTarget(ctx context.Context, req *v1.CreateSalesTargetReq) (*v1.SalesTargetItem, error) {
	period := strings.TrimSpace(req.Period)
	if _, err := time.Parse("2006-01", period); err != nil {
		return nil, rescode.New(rescode.BadRequest, "Kỳ KPI phải có định dạng YYYY-MM")
	}
	metric := strings.ToUpper(strings.TrimSpace(req.Metric))
	if !slices.Contains(validKPIMetrics, metric) {
		return nil, rescode.New(rescode.BadRequest, "Chỉ số KPI không hợp lệ")
	}
	if req.TargetValue < 0 {
		return nil, rescode.New(rescode.BadRequest, "Giá trị mục tiêu không được âm")
	}
	st := &entity.SalesTarget{
		BranchID:    req.BranchID,
		StaffID:     req.StaffID,
		Period:      period,
		Metric:      metric,
		TargetValue: req.TargetValue,
	}
	if err := s.repo.UpsertSalesTarget(ctx, st); err != nil {
		return nil, err
	}
	row, err := s.repo.FindSalesTargetByID(ctx, st.ID)
	if err != nil {
		row = &repository.SalesTargetRow{SalesTarget: *st}
	}
	res := toSalesTargetItem(*row)
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntitySalesTarget, st.ID, nil, res)
	}
	return res, nil
}

func (s *kpiService) DeleteSalesTarget(ctx context.Context, id uint) error {
	existing, err := s.repo.FindSalesTargetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy chỉ tiêu KPI")
		}
		return err
	}
	if err := s.repo.DeleteSalesTarget(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditDelete, auditEntitySalesTarget, id, toSalesTargetItem(*existing), nil)
	}
	return nil
}

func (s *kpiService) KPIReport(ctx context.Context, req *v1.KPIReportReq) (*v1.KPIReportRes, error) {
	now := s.now().In(memberVNLoc)
	period := strings.TrimSpace(req.Period)
	if period == "" {
		period = now.Format("2006-01")
	}
	monthStart, err := time.ParseInLocation("2006-01", period, memberVNLoc)
	if err != nil {
		return nil, rescode.New(rescode.BadRequest, "Kỳ báo cáo phải có định dạng YYYY-MM")
	}
	monthEnd := monthStart.AddDate(0, 1, 0)

	targets, err := s.repo.ListTargetsForPeriod(ctx, req.BranchID, period)
	if err != nil {
		return nil, err
	}
	targetMap := map[string]float64{}
	for _, t := range targets {
		targetMap[t.Metric] += t.TargetValue
	}

	actuals, err := s.repo.ComputeKPIActuals(ctx, req.BranchID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	ptActual := actuals.PTSalesAmount
	if tVal := targetMap[entity.KPIMetricPTSales]; tVal > 0 && tVal < 1000 {
		ptActual = actuals.PTSalesCount
	}

	actualMap := map[string]float64{
		entity.KPIMetricRevenue:    actuals.Revenue,
		entity.KPIMetricNewMembers: actuals.NewMembers,
		entity.KPIMetricRenewals:   actuals.Renewals,
		entity.KPIMetricPTSales:    ptActual,
	}

	metrics := make([]*v1.KPIMetricComparison, 0, len(validKPIMetrics))
	for _, m := range validKPIMetrics {
		act := actualMap[m]
		tgt := targetMap[m]
		var pct float64
		if tgt > 0 {
			pct = math.Round((act/tgt)*1000) / 10
		}
		metrics = append(metrics, &v1.KPIMetricComparison{
			Metric:         m,
			Label:          kpiMetricLabels[m],
			Actual:         act,
			TargetValue:    tgt,
			AchievementPct: pct,
		})
	}

	return &v1.KPIReportRes{
		Period:   period,
		BranchID: req.BranchID,
		Metrics:  metrics,
		Items:    metrics,
	}, nil
}

func (s *kpiService) CohortReport(ctx context.Context, req *v1.CohortReportReq) (*v1.CohortReportRes, error) {
	months := req.Months
	if months <= 0 {
		months = 6
	} else if months > 12 {
		months = 12
	}
	now := s.now().In(memberVNLoc)
	curMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, memberVNLoc)
	startMonth := curMonth.AddDate(0, -(months - 1), 0)

	members, err := s.repo.ListCohortMembers(ctx, req.BranchID, startMonth)
	if err != nil {
		return nil, err
	}
	memberIDs := make([]uint, 0, len(members))
	byCohort := map[string][]uint{}
	for _, m := range members {
		memberIDs = append(memberIDs, m.ID)
		byCohort[m.JoinedMonth] = append(byCohort[m.JoinedMonth], m.ID)
	}

	activeMap, err := s.repo.ListActiveMemberMonths(ctx, memberIDs, startMonth)
	if err != nil {
		return nil, err
	}

	cohorts := make([]*v1.CohortRow, 0, months)
	for i := 0; i < months; i++ {
		cMonthTime := startMonth.AddDate(0, i, 0)
		cMonthStr := cMonthTime.Format("2006-01")
		cMembers := byCohort[cMonthStr]
		size := len(cMembers)

		maxOffset := months - i
		retPcts := make([]float64, 0, maxOffset)
		actCounts := make([]int, 0, maxOffset)

		for offset := 0; offset < maxOffset; offset++ {
			targetMonthStr := cMonthTime.AddDate(0, offset, 0).Format("2006-01")
			if offset == 0 {
				actCounts = append(actCounts, size)
				if size > 0 {
					retPcts = append(retPcts, 100)
				} else {
					retPcts = append(retPcts, 0)
				}
				continue
			}
			cnt := 0
			for _, mid := range cMembers {
				if activeMap[mid] != nil && activeMap[mid][targetMonthStr] {
					cnt++
				}
			}
			actCounts = append(actCounts, cnt)
			var pct float64
			if size > 0 {
				pct = math.Round((float64(cnt)/float64(size))*1000) / 10
			}
			retPcts = append(retPcts, pct)
		}

		cohorts = append(cohorts, &v1.CohortRow{
			CohortMonth:      cMonthStr,
			Size:             size,
			RetentionByMonth: retPcts,
			ActiveCounts:     actCounts,
		})
	}

	return &v1.CohortReportRes{
		Months:  months,
		Cohorts: cohorts,
		Items:   cohorts,
	}, nil
}

func toSalesTargetItem(r repository.SalesTargetRow) *v1.SalesTargetItem {
	return &v1.SalesTargetItem{
		ID:          r.ID,
		BranchID:    r.BranchID,
		BranchName:  r.BranchName,
		StaffID:     r.StaffID,
		StaffName:   r.StaffName,
		Period:      r.Period,
		Metric:      r.Metric,
		TargetValue: r.TargetValue,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
