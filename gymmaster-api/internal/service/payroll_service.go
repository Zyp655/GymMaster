package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/payroll/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityPayroll = "payroll"
	defaultBaseSalary  = 10000000.0
)

type (
	PayrollService interface {
		ScanAttendance(ctx context.Context, req *v1.ScanAttendanceReq) (*v1.AttendanceItem, error)
		ListAttendances(ctx context.Context, req *v1.ListAttendancesReq) (v1.ListAttendancesRes, error)
		GeneratePayroll(ctx context.Context, req *v1.GeneratePayrollReq) (*v1.PayrollDetail, error)
		ListPayrolls(ctx context.Context, req *v1.ListPayrollsReq) (v1.ListPayrollsRes, error)
		GetPayroll(ctx context.Context, id uint) (*v1.PayrollDetail, error)
		UpdatePayroll(ctx context.Context, req *v1.UpdatePayrollReq) (*v1.PayrollDetail, error)
		LockPayroll(ctx context.Context, id uint) (*v1.PayrollDetail, error)
		PayPayroll(ctx context.Context, id uint) (*v1.PayrollDetail, error)
		ExportPayroll(ctx context.Context, id uint) (*v1.ExportPayrollRes, error)
		MyPayslips(ctx context.Context) (v1.MyPayslipsRes, error)
	}

	payrollService struct {
		repo  repository.PayrollRepository
		tx    postgres.TxManager
		audit AuditService
		now   func() time.Time
	}
)

func NewPayrollService(repo repository.PayrollRepository, tx postgres.TxManager, audit AuditService) PayrollService {
	return &payrollService{
		repo:  repo,
		tx:    tx,
		audit: audit,
		now:   time.Now,
	}
}

// payrollStaffBranchChecker là năng lực tùy chọn của repository: kiểm tra nhân viên chấm công.
type payrollStaffBranchChecker interface {
	StaffBranchStatus(ctx context.Context, staffID, branchID uint) (exists, inBranch bool, err error)
}

func payrollDayRange(t time.Time) (time.Time, time.Time) {
	loc := t.Location()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 1)
}

func payrollPeriodRange(period string) (time.Time, time.Time, error) {
	period = strings.TrimSpace(period)
	t, err := time.ParseInLocation("2006-01", period, memberVNLoc)
	if err != nil {
		return time.Time{}, time.Time{}, rescode.New(rescode.BadRequest, "Kỳ lương phải có định dạng YYYY-MM")
	}
	return t, t.AddDate(0, 1, 0), nil
}

func (s *payrollService) ScanAttendance(ctx context.Context, req *v1.ScanAttendanceReq) (*v1.AttendanceItem, error) {
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền chấm công tại chi nhánh này")
	}
	staffID := req.StaffID
	if staffID == 0 {
		if uid := reqctx.UserID(ctx); uid != nil {
			staffID = *uid
		}
	}
	if staffID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Không xác định được nhân viên chấm công")
	}
	// Nhân viên phải tồn tại, đang hoạt động và thuộc chi nhánh chấm công (hoặc có vai trò toàn hệ thống).
	if checker, ok := s.repo.(payrollStaffBranchChecker); ok {
		exists, inBranch, err := checker.StaffBranchStatus(ctx, staffID, req.BranchID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy nhân viên hoặc tài khoản đã bị khóa")
		}
		if !inBranch {
			return nil, rescode.New(rescode.BadRequest, "Nhân viên không thuộc chi nhánh chấm công")
		}
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method != entity.AttendanceMethodQR && method != entity.AttendanceMethodManual {
		method = entity.AttendanceMethodQR
	}

	now := s.now()
	dayStart, dayEnd := payrollDayRange(now)

	openAtt, err := s.repo.FindOpenAttendanceToday(ctx, staffID, dayStart, dayEnd)
	if err == nil && openAtt != nil {
		if !reqctx.CanAccessBranch(ctx, openAtt.BranchID) {
			return nil, rescode.New(rescode.PermissionDenied, "Nhân viên đang có lượt chấm công tại chi nhánh khác mà bạn không quản lý")
		}
		// Checkout
		openAtt.CheckOutAt = &now
		if openAtt.ShiftID != nil {
			if sh, sErr := s.repo.FindShiftByID(ctx, *openAtt.ShiftID); sErr == nil && sh != nil && now.Before(sh.EndAt) {
				openAtt.EarlyMinutes = int(sh.EndAt.Sub(now).Minutes())
			}
		} else if sh, sErr := s.repo.FindShiftInDay(ctx, staffID, openAtt.BranchID, dayStart, dayEnd); sErr == nil && sh != nil && now.Before(sh.EndAt) {
			openAtt.EarlyMinutes = int(sh.EndAt.Sub(now).Minutes())
		}
		if note := strings.TrimSpace(req.Note); note != "" {
			if openAtt.Note != "" {
				openAtt.Note = openAtt.Note + " | " + note
			} else {
				openAtt.Note = note
			}
		}
		openAtt.UpdatedAt = now
		if err := s.repo.UpdateAttendance(ctx, openAtt); err != nil {
			return nil, err
		}
		res := toAttendanceItem(repository.AttendanceRow{StaffAttendance: *openAtt})
		res.Action = "CHECK_OUT"
		return res, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Check-in mới
	att := &entity.StaffAttendance{
		StaffID:   staffID,
		BranchID:  req.BranchID,
		CheckInAt: now,
		Method:    method,
		Note:      strings.TrimSpace(req.Note),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if sh, sErr := s.repo.FindShiftInDay(ctx, staffID, req.BranchID, dayStart, dayEnd); sErr == nil && sh != nil {
		sid := sh.ID
		att.ShiftID = &sid
		if now.After(sh.StartAt) {
			att.LateMinutes = int(now.Sub(sh.StartAt).Minutes())
		}
	}
	if err := s.repo.CreateAttendance(ctx, att); err != nil {
		return nil, err
	}
	res := toAttendanceItem(repository.AttendanceRow{StaffAttendance: *att})
	res.Action = "CHECK_IN"
	return res, nil
}

func (s *payrollService) ListAttendances(ctx context.Context, req *v1.ListAttendancesReq) (v1.ListAttendancesRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem chấm công tại chi nhánh này")
	}
	var from, to *time.Time
	if sFrom := strings.TrimSpace(req.From); sFrom != "" {
		t, err := time.ParseInLocation(consts.DateLayout, sFrom, memberVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Từ ngày phải có định dạng YYYY-MM-DD")
		}
		from = &t
	}
	if sTo := strings.TrimSpace(req.To); sTo != "" {
		t, err := time.ParseInLocation(consts.DateLayout, sTo, memberVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Đến ngày phải có định dạng YYYY-MM-DD")
		}
		end := t.AddDate(0, 0, 1)
		to = &end
	}
	rows, err := s.repo.ListAttendances(ctx, repository.AttendanceFilter{
		BranchIDs: branchIDs,
		StaffID:   req.StaffID,
		From:      from,
		To:        to,
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toAttendanceItem), nil
}

func (s *payrollService) GeneratePayroll(ctx context.Context, req *v1.GeneratePayrollReq) (*v1.PayrollDetail, error) {
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được tạo bảng lương tại chi nhánh này")
	}
	period := strings.TrimSpace(req.Period)
	pStart, pEnd, err := payrollPeriodRange(period)
	if err != nil {
		return nil, err
	}
	// Không tạo bảng lương cho kỳ tương lai (sau tháng hiện tại).
	cur := s.now().In(memberVNLoc)
	if pStart.After(time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, memberVNLoc)) {
		return nil, rescode.New(rescode.BadRequest, "Không thể tạo bảng lương cho kỳ trong tương lai")
	}

	now := s.now()
	var payrollID uint

	existing, err := s.repo.FindPayrollByBranchPeriod(ctx, req.BranchID, period)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		if existing.Status != entity.PayrollStatusDraft {
			return nil, rescode.New(rescode.PayrollLocked, "Bảng lương kỳ này đã khóa hoặc đã chi trả, không thể tổng hợp lại")
		}
		payrollID = existing.ID
	} else {
		p := &entity.Payroll{
			BranchID:  req.BranchID,
			Period:    period,
			Status:    entity.PayrollStatusDraft,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.repo.CreatePayroll(ctx, p); err != nil {
			return nil, err
		}
		payrollID = p.ID
	}

	sources, err := s.repo.CollectStaffPayrollSources(ctx, req.BranchID, period, pStart, pEnd)
	if err != nil {
		return nil, err
	}
	for _, src := range sources {
		baseSalary := src.BaseSalary
		if baseSalary <= 0 {
			baseSalary = defaultBaseSalary
		}
		total := baseSalary + src.PTCommission + src.SalesCommission
		item := &entity.PayrollItem{
			PayrollID:       payrollID,
			StaffID:         src.StaffID,
			BaseSalary:      baseSalary,
			WorkDays:        src.WorkDays,
			PTCommission:    src.PTCommission,
			SalesCommission: src.SalesCommission,
			Total:           total,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := s.repo.UpsertPayrollItem(ctx, item); err != nil {
			return nil, err
		}
	}

	res, err := s.GetPayroll(ctx, payrollID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityPayroll, payrollID, nil, res)
	}
	return res, nil
}

func (s *payrollService) ListPayrolls(ctx context.Context, req *v1.ListPayrollsReq) (v1.ListPayrollsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem bảng lương tại chi nhánh này")
	}
	rows, err := s.repo.ListPayrolls(ctx, repository.PayrollFilter{
		BranchIDs: branchIDs,
		Period:    strings.TrimSpace(req.Period),
		Status:    strings.ToUpper(strings.TrimSpace(req.Status)),
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toPayrollSummary), nil
}

func (s *payrollService) GetPayroll(ctx context.Context, id uint) (*v1.PayrollDetail, error) {
	row, err := s.repo.FindPayrollByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bảng lương")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem bảng lương tại chi nhánh này")
	}
	items, err := s.repo.ListPayrollItems(ctx, id)
	if err != nil {
		return nil, err
	}
	lineItems := dto.Map(items, toPayrollLineItem)
	var total float64
	for _, it := range lineItems {
		total += it.Total
	}
	summary := toPayrollSummary(*row)
	summary.StaffCount = len(lineItems)
	summary.TotalAmount = total
	return &v1.PayrollDetail{
		PayrollSummary: *summary,
		Items:          lineItems,
	}, nil
}

func (s *payrollService) UpdatePayroll(ctx context.Context, req *v1.UpdatePayrollReq) (*v1.PayrollDetail, error) {
	row, err := s.repo.FindPayrollByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bảng lương")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác bảng lương tại chi nhánh này")
	}
	if row.Status != entity.PayrollStatusDraft {
		return nil, rescode.New(rescode.PayrollLocked, "Bảng lương đã khóa hoặc đã chi trả, không thể chỉnh sửa")
	}

	items, err := s.repo.ListPayrollItems(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	byID := map[uint]*entity.PayrollItem{}
	byStaff := map[uint]*entity.PayrollItem{}
	for i := range items {
		it := &items[i].PayrollItem
		byID[it.ID] = it
		byStaff[it.StaffID] = it
	}

	now := s.now()
	for _, in := range req.Items {
		var target *entity.PayrollItem
		if in.ID > 0 {
			target = byID[in.ID]
		}
		if target == nil && in.StaffID > 0 {
			target = byStaff[in.StaffID]
		}
		if target == nil {
			continue
		}
		if in.BaseSalary != nil {
			target.BaseSalary = *in.BaseSalary
		}
		if in.WorkDays != nil {
			target.WorkDays = *in.WorkDays
		}
		if in.OTHours != nil {
			target.OTHours = *in.OTHours
		}
		if in.Bonus != nil {
			target.Bonus = *in.Bonus
		}
		if in.Penalty != nil {
			target.Penalty = *in.Penalty
		}
		if in.Note != nil {
			target.Note = strings.TrimSpace(*in.Note)
		}
		target.Total = target.BaseSalary + target.PTCommission + target.SalesCommission + target.Bonus - target.Penalty
		target.UpdatedAt = now
		if err := s.repo.UpdatePayrollItem(ctx, target); err != nil {
			return nil, err
		}
	}
	return s.GetPayroll(ctx, req.ID)
}

func (s *payrollService) LockPayroll(ctx context.Context, id uint) (*v1.PayrollDetail, error) {
	row, err := s.repo.FindPayrollByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bảng lương")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được khóa bảng lương tại chi nhánh này")
	}
	if row.Status != entity.PayrollStatusDraft {
		return nil, rescode.New(rescode.PayrollLocked, "Bảng lương đã được khóa trước đó")
	}
	now := s.now()
	p := row.Payroll
	p.Status = entity.PayrollStatusLocked
	p.LockedBy = reqctx.UserID(ctx)
	p.LockedAt = &now
	p.UpdatedAt = now
	if err := s.repo.UpdatePayroll(ctx, &p); err != nil {
		return nil, err
	}
	return s.GetPayroll(ctx, id)
}

func (s *payrollService) PayPayroll(ctx context.Context, id uint) (*v1.PayrollDetail, error) {
	row, err := s.repo.FindPayrollByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy bảng lương")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được chi lương tại chi nhánh này")
	}
	if row.Status == entity.PayrollStatusPaid {
		return nil, rescode.New(rescode.PayrollLocked, "Bảng lương đã được chi trả trước đó")
	}
	now := s.now()
	uid := reqctx.UserID(ctx)
	p := row.Payroll
	p.Status = entity.PayrollStatusPaid
	if p.LockedAt == nil {
		p.LockedAt = &now
		p.LockedBy = uid
	}
	p.PaidBy = uid
	p.PaidAt = &now
	p.UpdatedAt = now

	run := func(txCtx context.Context) error {
		if err := s.repo.UpdatePayroll(txCtx, &p); err != nil {
			return err
		}
		return s.repo.MarkTrainerCommissionsPaid(txCtx, p.BranchID, p.Period, uid, now)
	}
	if s.tx != nil {
		err = s.tx.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}
	if err != nil {
		return nil, err
	}
	return s.GetPayroll(ctx, id)
}

func (s *payrollService) ExportPayroll(ctx context.Context, id uint) (*v1.ExportPayrollRes, error) {
	detail, err := s.GetPayroll(ctx, id)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"Mã NV", "Họ tên", "Vị trí", "Ngày công", "Lương cơ bản",
		"Hoa hồng PT", "Hoa hồng Sales", "Thưởng", "Phạt", "Thực lĩnh", "Ghi chú",
	})
	for _, it := range detail.Items {
		_ = w.Write([]string{
			it.StaffCode,
			it.StaffName,
			it.Position,
			fmt.Sprintf("%.1f", it.WorkDays),
			strconv.FormatFloat(it.BaseSalary, 'f', 0, 64),
			strconv.FormatFloat(it.PTCommission, 'f', 0, 64),
			strconv.FormatFloat(it.SalesCommission, 'f', 0, 64),
			strconv.FormatFloat(it.Bonus, 'f', 0, 64),
			strconv.FormatFloat(it.Penalty, 'f', 0, 64),
			strconv.FormatFloat(it.Total, 'f', 0, 64),
			it.Note,
		})
	}
	w.Flush()
	return &v1.ExportPayrollRes{
		PayrollID:   detail.ID,
		BranchID:    detail.BranchID,
		BranchName:  detail.BranchName,
		Period:      detail.Period,
		Status:      detail.Status,
		TotalAmount: detail.TotalAmount,
		CSV:         buf.String(),
		Items:       detail.Items,
	}, nil
}

func (s *payrollService) MyPayslips(ctx context.Context) (v1.MyPayslipsRes, error) {
	uid := reqctx.UserID(ctx)
	if uid == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	rows, err := s.repo.ListPayslipsByStaff(ctx, *uid)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, func(r repository.PayslipRow) *v1.PayslipItem {
		return &v1.PayslipItem{
			PayrollLineItem: *toPayrollLineItem(r.PayrollItemRow),
			BranchID:        r.BranchID,
			BranchName:      r.BranchName,
			Period:          r.Period,
			Status:          r.Status,
			PaidAt:          r.PaidAt,
		}
	}), nil
}

func toAttendanceItem(r repository.AttendanceRow) *v1.AttendanceItem {
	return &v1.AttendanceItem{
		ID:           r.ID,
		StaffID:      r.StaffID,
		StaffCode:    r.StaffCode,
		StaffName:    r.StaffName,
		Position:     r.Position,
		BranchID:     r.BranchID,
		BranchName:   r.BranchName,
		ShiftID:      r.ShiftID,
		CheckInAt:    r.CheckInAt,
		CheckOutAt:   r.CheckOutAt,
		LateMinutes:  r.LateMinutes,
		EarlyMinutes: r.EarlyMinutes,
		Method:       r.Method,
		Note:         r.Note,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func toPayrollSummary(r repository.PayrollRow) *v1.PayrollSummary {
	return &v1.PayrollSummary{
		ID:          r.ID,
		BranchID:    r.BranchID,
		BranchName:  r.BranchName,
		Period:      r.Period,
		Status:      r.Status,
		StaffCount:  r.StaffCount,
		TotalAmount: r.TotalAmount,
		LockedBy:    r.LockedBy,
		LockedAt:    r.LockedAt,
		PaidBy:      r.PaidBy,
		PaidAt:      r.PaidAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toPayrollLineItem(r repository.PayrollItemRow) *v1.PayrollLineItem {
	return &v1.PayrollLineItem{
		ID:              r.ID,
		PayrollID:       r.PayrollID,
		StaffID:         r.StaffID,
		StaffCode:       r.StaffCode,
		StaffName:       r.StaffName,
		Position:        r.Position,
		BaseSalary:      r.BaseSalary,
		WorkDays:        r.WorkDays,
		OTHours:         r.OTHours,
		PTCommission:    r.PTCommission,
		SalesCommission: r.SalesCommission,
		Bonus:           r.Bonus,
		Penalty:         r.Penalty,
		Total:           r.Total,
		Note:            r.Note,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}
