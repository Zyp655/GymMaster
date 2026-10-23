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
	"gymmaster-api/internal/pkg/reqctx"
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

func resolveKPIBranchScope(ctx context.Context, requested uint) (uint, error) {
	if requested > 0 {
		if !reqctx.CanAccessBranch(ctx, requested) {
			return 0, rescode.New(rescode.PermissionDenied, "Bạn không có quyền truy cập dữ liệu KPI của chi nhánh này")
		}
		return requested, nil
	}
	if allowed := reqctx.AllowedBranches(ctx); len(allowed) > 0 {
		return allowed[0], nil
	}
	return 0, nil
}

// resolveKPIBranch = resolveKPIBranchScope + chi nhánh được yêu cầu phải tồn tại (chưa xóa) → 404 thay vì báo cáo rỗng.
func (s *kpiService) resolveKPIBranch(ctx context.Context, requested uint) (uint, error) {
	branchID, err := resolveKPIBranchScope(ctx, requested)
	if err != nil || requested == 0 {
		return branchID, err
	}
	ok, err := s.repo.BranchExists(ctx, requested)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, rescode.New(rescode.NotFound, "Không tìm thấy chi nhánh")
	}
	return branchID, nil
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
	since := repository.PresentSince(s.now()) // định nghĩa chung "đang trong phòng tập"
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
	branchID, err := s.resolveKPIBranch(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	q := req.PageReq.ToQuery(salesTargetOrderable, "id")
	q.OrderBy = "st." + q.OrderBy
	rows, total, err := s.repo.ListSalesTargets(ctx, q, branchID, strings.TrimSpace(req.Period), strings.TrimSpace(req.Metric))
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toSalesTargetItem), total, q), nil
}

func (s *kpiService) CreateSalesTarget(ctx context.Context, req *v1.CreateSalesTargetReq) (*v1.SalesTargetItem, error) {
	if req.BranchID > 0 {
		if !reqctx.CanAccessBranch(ctx, req.BranchID) {
			return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền đặt chỉ tiêu cho chi nhánh này")
		}
	} else if allowed := reqctx.AllowedBranches(ctx); len(allowed) > 0 {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn chỉ được đặt chỉ tiêu trong phạm vi chi nhánh được phân công")
	}
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
	if req.TargetValue > financeMaxAmount {
		return nil, rescode.New(rescode.BadRequest, "Giá trị mục tiêu vượt quá giới hạn cho phép")
	}
	staffID := req.StaffID
	if staffID != nil && *staffID == 0 {
		staffID = nil
	}
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if ok, err := s.repo.BranchExists(ctx, req.BranchID); err != nil {
		return nil, err
	} else if !ok {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại hoặc đã bị xóa")
	}
	// Không đặt chỉ tiêu cho chi nhánh ngừng hoạt động.
	if ok, err := s.repo.BranchActive(ctx, req.BranchID); err != nil {
		return nil, err
	} else if !ok {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động, không thể đặt chỉ tiêu")
	}
	if staffID != nil {
		exists, inBranch, err := s.repo.StaffBranchStatus(ctx, *staffID, req.BranchID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, rescode.New(rescode.BadRequest, "Nhân viên không tồn tại, đã nghỉ hoặc không phải tài khoản nhân viên")
		}
		if !inBranch {
			return nil, rescode.New(rescode.BadRequest, "Nhân viên không thuộc chi nhánh được đặt chỉ tiêu")
		}
	}
	st := &entity.SalesTarget{
		BranchID:    req.BranchID,
		StaffID:     staffID,
		Period:      period,
		Metric:      metric,
		TargetValue: req.TargetValue,
	}
	if err := s.repo.UpsertSalesTarget(ctx, st); err != nil {
		// branch_id / staff_id tham chiếu bản ghi không tồn tại → lỗi khóa ngoại → 400 thay vì 500.
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			if staffID != nil {
				return nil, rescode.New(rescode.BadRequest, "Nhân viên hoặc chi nhánh không tồn tại")
			}
			return nil, rescode.New(rescode.BadRequest, "Chi nhánh không tồn tại")
		}
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
	if existing.BranchID > 0 {
		if !reqctx.CanAccessBranch(ctx, existing.BranchID) {
			return rescode.New(rescode.PermissionDenied, "Bạn không có quyền xóa chỉ tiêu của chi nhánh này")
		}
	} else if allowed := reqctx.AllowedBranches(ctx); len(allowed) > 0 {
		return rescode.New(rescode.PermissionDenied, "Bạn không có quyền xóa chỉ tiêu cấp toàn hệ thống")
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
	branchID, err := s.resolveKPIBranch(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
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

	targets, err := s.repo.ListTargetsForPeriod(ctx, branchID, period)
	if err != nil {
		return nil, err
	}
	// Chỉ tiêu cấp chi nhánh = tổng các chỉ tiêu KHÔNG gắn nhân viên (staff_id IS NULL);
	// chỉ tiêu cá nhân là phần chia nhỏ của chỉ tiêu chi nhánh nên không cộng dồn thêm.
	// targetByBranch[metric][branch] – để khi xem toàn hệ thống chỉ so thực tế của các chi nhánh CÓ chỉ tiêu.
	targetByBranch := map[string]map[uint]float64{}
	for _, t := range targets {
		if t.StaffID != nil {
			continue
		}
		if targetByBranch[t.Metric] == nil {
			targetByBranch[t.Metric] = map[uint]float64{}
		}
		targetByBranch[t.Metric][t.BranchID] += t.TargetValue
	}

	actualCache := map[uint]repository.KPIActuals{}
	actualsOf := func(b uint) (repository.KPIActuals, error) {
		if a, ok := actualCache[b]; ok {
			return a, nil
		}
		a, err := s.repo.ComputeKPIActuals(ctx, b, monthStart, monthEnd)
		if err != nil {
			return a, err
		}
		actualCache[b] = a
		return a, nil
	}
	// kpiActualOf: thực tế của một chỉ số tại phạm vi b. PT_SALES: chỉ tiêu < 1000 được hiểu là số hợp đồng,
	// ngược lại là doanh số (đ) – xét theo chỉ tiêu của chính phạm vi đó.
	kpiActualOf := func(a repository.KPIActuals, metric string, tgt float64) float64 {
		switch metric {
		case entity.KPIMetricRevenue:
			return a.Revenue
		case entity.KPIMetricNewMembers:
			return a.NewMembers
		case entity.KPIMetricRenewals:
			return a.Renewals
		case entity.KPIMetricPTSales:
			if tgt > 0 && tgt < 1000 {
				return a.PTSalesCount
			}
			return a.PTSalesAmount
		}
		return 0
	}

	metrics := make([]*v1.KPIMetricComparison, 0, len(validKPIMetrics))
	for _, m := range validKPIMetrics {
		perBranch := targetByBranch[m]
		var act, tgt float64
		if len(perBranch) == 0 {
			// Chưa đặt chỉ tiêu: vẫn hiện thực tế của cả phạm vi xem, không tính % hoàn thành.
			a, err := actualsOf(branchID)
			if err != nil {
				return nil, err
			}
			act = kpiActualOf(a, m, 0)
		} else {
			branchIDs := make([]uint, 0, len(perBranch))
			for b := range perBranch {
				branchIDs = append(branchIDs, b)
			}
			slices.Sort(branchIDs)
			for _, b := range branchIDs {
				a, err := actualsOf(b)
				if err != nil {
					return nil, err
				}
				act += kpiActualOf(a, m, perBranch[b])
				tgt += perBranch[b]
			}
		}
		var pct float64
		if tgt > 0 {
			pct = math.Round((act/tgt)*1000) / 10
		}
		metrics = append(metrics, &v1.KPIMetricComparison{
			Metric:            m,
			Label:             kpiMetricLabels[m],
			Actual:            act,
			TargetValue:       tgt,
			AchievementPct:    pct,
			HasTarget:         len(perBranch) > 0,
			TargetBranchCount: len(perBranch),
		})
	}

	return &v1.KPIReportRes{
		Period:   period,
		BranchID: branchID,
		Metrics:  metrics,
		Items:    metrics,
	}, nil
}

func (s *kpiService) CohortReport(ctx context.Context, req *v1.CohortReportReq) (*v1.CohortReportRes, error) {
	branchID, err := s.resolveKPIBranch(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}
	months := req.Months
	if months <= 0 {
		months = 6
	} else if months > 12 {
		months = 12
	}
	now := s.now().In(memberVNLoc)
	curMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, memberVNLoc)
	startMonth := curMonth.AddDate(0, -(months - 1), 0)

	members, err := s.repo.ListCohortMembers(ctx, branchID, startMonth)
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
