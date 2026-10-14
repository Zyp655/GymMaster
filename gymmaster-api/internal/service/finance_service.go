package service

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/finance/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/einvoice"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityExpense     = "expense"
	auditEntityCashSession = "cash_session"
	auditEntityInstallment = "installment_plan"
)

var (
	validExpenseCategoryTypes = []string{entity.ExpenseTypeFixed, entity.ExpenseTypeVariable}
	validExpenseStatuses      = []string{entity.ExpenseStatusPending, entity.ExpenseStatusApproved, entity.ExpenseStatusRejected}
)

type (
	FinanceService interface {
		ListExpenseCategories(ctx context.Context) (v1.ListExpenseCategoriesRes, error)
		CreateExpenseCategory(ctx context.Context, req *v1.CreateExpenseCategoryReq) (*v1.ExpenseCategoryItem, error)

		ListExpenses(ctx context.Context, req *v1.ListExpensesReq) (v1.ListExpensesRes, error)
		CreateExpense(ctx context.Context, req *v1.CreateExpenseReq) (*v1.ExpenseItem, error)
		UpdateExpense(ctx context.Context, req *v1.UpdateExpenseReq) (*v1.ExpenseItem, error)
		DeleteExpense(ctx context.Context, id uint) error
		ApproveExpense(ctx context.Context, req *v1.ApproveExpenseReq) (*v1.ExpenseItem, error)

		OpenCashSession(ctx context.Context, req *v1.OpenCashSessionReq) (*v1.CashSessionItem, error)
		CloseCashSession(ctx context.Context, req *v1.CloseCashSessionReq) (*v1.CashSessionItem, error)
		ListCashSessions(ctx context.Context, req *v1.ListCashSessionsReq) (v1.ListCashSessionsRes, error)

		ProfitLossReport(ctx context.Context, req *v1.ProfitLossReportReq) (*v1.ProfitLossReportRes, error)

		CreateInstallmentPlan(ctx context.Context, req *v1.CreateInstallmentPlanReq) (*v1.InstallmentPlanItem, error)
		ListInstallments(ctx context.Context, req *v1.ListInstallmentsReq) (v1.ListInstallmentsRes, error)
		PayInstallmentSchedule(ctx context.Context, req *v1.PayInstallmentScheduleReq) (*v1.InstallmentScheduleItem, error)
		MemberInstallments(ctx context.Context) (v1.MemberInstallmentsRes, error)

		IssueEInvoice(ctx context.Context, invoiceID uint) (*v1.IssueEInvoiceRes, error)
	}

	financeService struct {
		repo             repository.FinanceRepository
		invoices         InvoiceService
		einvoiceProvider einvoice.Provider
		tx               postgres.TxManager
		audit            AuditService
		now              func() time.Time
	}
)

func NewFinanceService(
	repo repository.FinanceRepository,
	invoices InvoiceService,
	einvoiceProvider einvoice.Provider,
	tx postgres.TxManager,
	audit AuditService,
) FinanceService {
	if einvoiceProvider == nil {
		einvoiceProvider = einvoice.NewMockProvider()
	}
	return &financeService{
		repo:             repo,
		invoices:         invoices,
		einvoiceProvider: einvoiceProvider,
		tx:               tx,
		audit:            audit,
		now:              time.Now,
	}
}

func (s *financeService) ListExpenseCategories(ctx context.Context) (v1.ListExpenseCategoriesRes, error) {
	rows, err := s.repo.ListExpenseCategories(ctx)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, func(c entity.ExpenseCategory) *v1.ExpenseCategoryItem {
		return &v1.ExpenseCategoryItem{
			ID:        c.ID,
			Name:      c.Name,
			Type:      c.Type,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}), nil
}

func (s *financeService) CreateExpenseCategory(ctx context.Context, req *v1.CreateExpenseCategoryReq) (*v1.ExpenseCategoryItem, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập tên danh mục chi phí")
	}
	typ := strings.ToUpper(strings.TrimSpace(req.Type))
	if typ == "" {
		typ = entity.ExpenseTypeVariable
	}
	if !slices.Contains(validExpenseCategoryTypes, typ) {
		return nil, rescode.New(rescode.BadRequest, "Loại danh mục chi phí không hợp lệ")
	}
	dup, err := s.repo.ExistsExpenseCategoryName(ctx, name)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, rescode.New(rescode.Conflict, "Danh mục chi phí đã tồn tại")
	}
	now := s.now()
	cat := &entity.ExpenseCategory{
		Name:      name,
		Type:      typ,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateExpenseCategory(ctx, cat); err != nil {
		return nil, mapDuplicate(err, "Danh mục chi phí đã tồn tại")
	}
	return &v1.ExpenseCategoryItem{
		ID:        cat.ID,
		Name:      cat.Name,
		Type:      cat.Type,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}, nil
}

func (s *financeService) ListExpenses(ctx context.Context, req *v1.ListExpensesReq) (v1.ListExpensesRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem chi phí tại chi nhánh này")
	}
	from, err := equipmentParseDateOpt(req.From)
	if err != nil {
		return nil, err
	}
	to, err := equipmentParseDateOpt(req.To)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListExpenses(ctx, repository.ExpenseFilter{
		BranchIDs:  branchIDs,
		CategoryID: req.CategoryID,
		Status:     strings.ToUpper(strings.TrimSpace(req.Status)),
		From:       from,
		To:         to,
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toExpenseItem), nil
}

func (s *financeService) CreateExpense(ctx context.Context, req *v1.CreateExpenseReq) (*v1.ExpenseItem, error) {
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được tạo chi phí tại chi nhánh này")
	}
	if req.CategoryID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn danh mục chi phí")
	}
	if req.Amount < 0 {
		return nil, rescode.New(rescode.BadRequest, "Số tiền chi phí không được âm")
	}
	cat, err := s.repo.FindExpenseCategoryByID(ctx, req.CategoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy danh mục chi phí")
		}
		return nil, err
	}

	now := s.now()
	spentAt := equipmentStartOfDay(now)
	if strings.TrimSpace(req.SpentAt) != "" {
		parsed, pErr := equipmentParseDateOpt(req.SpentAt)
		if pErr != nil {
			return nil, pErr
		}
		spentAt = *parsed
	}

	exp := &entity.Expense{
		BranchID:         req.BranchID,
		CategoryID:       req.CategoryID,
		Amount:           req.Amount,
		SpentAt:          spentAt,
		Description:      strings.TrimSpace(req.Description),
		AttachmentURL:    strings.TrimSpace(req.AttachmentURL),
		MaintenanceLogID: req.MaintenanceLogID,
		CreatedBy:        reqctx.UserID(ctx),
		Status:           entity.ExpenseStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.CreateExpense(ctx, exp); err != nil {
		return nil, err
	}
	res := toExpenseItem(repository.ExpenseRow{
		Expense:      *exp,
		CategoryName: cat.Name,
		CategoryType: cat.Type,
	})
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityExpense, exp.ID, nil, res)
	}
	return res, nil
}

func (s *financeService) UpdateExpense(ctx context.Context, req *v1.UpdateExpenseReq) (*v1.ExpenseItem, error) {
	row, err := s.repo.FindExpenseByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khoản chi phí")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thao tác chi phí tại chi nhánh này")
	}
	exp := row.Expense
	if req.CategoryID > 0 {
		exp.CategoryID = req.CategoryID
	}
	if req.Amount != nil {
		if *req.Amount < 0 {
			return nil, rescode.New(rescode.BadRequest, "Số tiền chi phí không được âm")
		}
		exp.Amount = *req.Amount
	}
	if strings.TrimSpace(req.SpentAt) != "" {
		parsed, pErr := equipmentParseDateOpt(req.SpentAt)
		if pErr != nil {
			return nil, pErr
		}
		exp.SpentAt = *parsed
	}
	if req.Description != nil {
		exp.Description = strings.TrimSpace(*req.Description)
	}
	if req.AttachmentURL != nil {
		exp.AttachmentURL = strings.TrimSpace(*req.AttachmentURL)
	}
	if status := strings.ToUpper(strings.TrimSpace(req.Status)); status != "" {
		if !slices.Contains(validExpenseStatuses, status) {
			return nil, rescode.New(rescode.BadRequest, "Trạng thái chi phí không hợp lệ")
		}
		exp.Status = status
	}
	exp.UpdatedAt = s.now()
	if err := s.repo.UpdateExpense(ctx, &exp); err != nil {
		return nil, err
	}
	updatedRow := *row
	updatedRow.Expense = exp
	return toExpenseItem(updatedRow), nil
}

func (s *financeService) DeleteExpense(ctx context.Context, id uint) error {
	row, err := s.repo.FindExpenseByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy khoản chi phí")
		}
		return err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không được xóa chi phí tại chi nhánh này")
	}
	return s.repo.DeleteExpense(ctx, id)
}

func (s *financeService) ApproveExpense(ctx context.Context, req *v1.ApproveExpenseReq) (*v1.ExpenseItem, error) {
	row, err := s.repo.FindExpenseByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy khoản chi phí")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được duyệt chi phí tại chi nhánh này")
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status == "" {
		status = entity.ExpenseStatusApproved
	}
	if status != entity.ExpenseStatusApproved && status != entity.ExpenseStatusRejected {
		return nil, rescode.New(rescode.BadRequest, "Trạng thái duyệt không hợp lệ")
	}
	exp := row.Expense
	exp.Status = status
	exp.ApprovedBy = reqctx.UserID(ctx)
	exp.UpdatedAt = s.now()
	if err := s.repo.UpdateExpense(ctx, &exp); err != nil {
		return nil, err
	}
	updatedRow := *row
	updatedRow.Expense = exp
	return toExpenseItem(updatedRow), nil
}

func (s *financeService) OpenCashSession(ctx context.Context, req *v1.OpenCashSessionReq) (*v1.CashSessionItem, error) {
	if req.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, req.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được mở ca thu ngân tại chi nhánh này")
	}
	if req.OpeningCash < 0 {
		return nil, rescode.New(rescode.BadRequest, "Tiền mặt đầu ca không được âm")
	}
	if open, err := s.repo.FindOpenCashSession(ctx, req.BranchID); err == nil && open != nil {
		return nil, rescode.New(rescode.Conflict, "Chi nhánh đang có ca thu ngân chưa đóng")
	}
	var staffID uint
	if uid := reqctx.UserID(ctx); uid != nil {
		staffID = *uid
	}
	now := s.now()
	cs := &entity.CashSession{
		BranchID:     req.BranchID,
		StaffID:      staffID,
		OpenedAt:     now,
		OpeningCash:  req.OpeningCash,
		ExpectedCash: req.OpeningCash,
		Note:         strings.TrimSpace(req.Note),
		Status:       entity.CashSessionOpen,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateCashSession(ctx, cs); err != nil {
		return nil, err
	}
	res := toCashSessionItem(repository.CashSessionRow{CashSession: *cs})
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityCashSession, cs.ID, nil, res)
	}
	return res, nil
}

func (s *financeService) CloseCashSession(ctx context.Context, req *v1.CloseCashSessionReq) (*v1.CashSessionItem, error) {
	row, err := s.repo.FindCashSessionByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy ca thu ngân")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được đóng ca thu ngân tại chi nhánh này")
	}
	if row.Status == entity.CashSessionClosed {
		return nil, rescode.New(rescode.Conflict, "Ca thu ngân đã được đóng trước đó")
	}
	now := s.now()
	cashCollected, err := s.repo.SumCashPaymentsBetween(ctx, row.BranchID, row.OpenedAt, now)
	if err != nil {
		return nil, err
	}
	cs := row.CashSession
	cs.ClosedAt = &now
	cs.ExpectedCash = cs.OpeningCash + cashCollected
	cs.CountedCash = req.CountedCash
	cs.Difference = cs.CountedCash - cs.ExpectedCash
	if note := strings.TrimSpace(req.Note); note != "" {
		cs.Note = note
	}
	cs.Status = entity.CashSessionClosed
	cs.UpdatedAt = now
	if err := s.repo.UpdateCashSession(ctx, &cs); err != nil {
		return nil, err
	}
	updatedRow := *row
	updatedRow.CashSession = cs
	return toCashSessionItem(updatedRow), nil
}

func (s *financeService) ListCashSessions(ctx context.Context, req *v1.ListCashSessionsReq) (v1.ListCashSessionsRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem ca thu ngân tại chi nhánh này")
	}
	rows, err := s.repo.ListCashSessions(ctx, repository.CashSessionFilter{
		BranchIDs: branchIDs,
		Status:    strings.ToUpper(strings.TrimSpace(req.Status)),
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toCashSessionItem), nil
}

func (s *financeService) ProfitLossReport(ctx context.Context, req *v1.ProfitLossReportReq) (*v1.ProfitLossReportRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được xem báo cáo P&L của chi nhánh này")
	}
	now := s.now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, memberVNLoc)
	to := equipmentStartOfDay(now)
	if strings.TrimSpace(req.From) != "" {
		parsed, err := equipmentParseDateOpt(req.From)
		if err != nil {
			return nil, err
		}
		from = *parsed
	}
	if strings.TrimSpace(req.To) != "" {
		parsed, err := equipmentParseDateOpt(req.To)
		if err != nil {
			return nil, err
		}
		to = *parsed
	}
	data, err := s.repo.CalcProfitLoss(ctx, branchIDs, from, to)
	if err != nil {
		return nil, err
	}
	gross := data.MembershipRevenue + data.PTRevenue + data.POSRevenue + data.LockerRevenue + data.OtherRevenue
	netRev := gross - data.Refunds
	totalExp := data.POSCost + data.OperatingExpenses + data.PayrollExpenses
	netProfit := netRev - totalExp

	return &v1.ProfitLossReportRes{
		From:              from.Format(consts.DateLayout),
		To:                to.Format(consts.DateLayout),
		MembershipRevenue: data.MembershipRevenue,
		PTRevenue:         data.PTRevenue,
		POSRevenue:        data.POSRevenue,
		LockerRevenue:     data.LockerRevenue,
		OtherRevenue:      data.OtherRevenue,
		GrossRevenue:      gross,
		Refunds:           data.Refunds,
		NetRevenue:        netRev,
		POSCost:           data.POSCost,
		OperatingExpenses: data.OperatingExpenses,
		PayrollExpenses:   data.PayrollExpenses,
		TotalExpenses:     totalExp,
		NetProfit:         netProfit,
	}, nil
}

func (s *financeService) CreateInstallmentPlan(ctx context.Context, req *v1.CreateInstallmentPlanReq) (*v1.InstallmentPlanItem, error) {
	if req.Installments < 2 || req.Installments > 12 {
		return nil, rescode.New(rescode.BadRequest, "Số kỳ trả góp phải từ 2 đến 12 kỳ")
	}
	inv, err := s.repo.FindInvoiceByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hóa đơn")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, inv.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền thao tác hóa đơn của chi nhánh này")
	}
	if inv.MemberID == nil {
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn trả góp phải gắn với hội viên")
	}
	if existing, err := s.repo.FindInstallmentPlanByInvoice(ctx, inv.ID); err == nil && existing != nil {
		return nil, rescode.New(rescode.Conflict, "Hóa đơn này đã có kế hoạch trả góp")
	}

	now := s.now()
	firstDue := equipmentStartOfDay(now)
	if strings.TrimSpace(req.FirstDueDate) != "" {
		parsed, pErr := equipmentParseDateOpt(req.FirstDueDate)
		if pErr != nil {
			return nil, pErr
		}
		firstDue = *parsed
	}

	totalToSplit := inv.Total - inv.PaidAmount
	if totalToSplit <= 0 {
		totalToSplit = inv.Total
	}
	perInstallment := math.Floor(totalToSplit / float64(req.Installments))
	schedules := make([]entity.InstallmentSchedule, req.Installments)
	var allocated float64
	for i := 0; i < req.Installments; i++ {
		amt := perInstallment
		if i == req.Installments-1 {
			amt = totalToSplit - allocated
		}
		allocated += amt
		schedules[i] = entity.InstallmentSchedule{
			Seq:       i + 1,
			DueDate:   firstDue.AddDate(0, i, 0),
			Amount:    amt,
			Status:    entity.InstallmentPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	plan := &entity.InstallmentPlan{
		InvoiceID:    inv.ID,
		MemberID:     *inv.MemberID,
		Total:        totalToSplit,
		Installments: req.Installments,
		CreatedBy:    reqctx.UserID(ctx),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateInstallmentPlan(ctx, plan, schedules); err != nil {
		return nil, err
	}

	schedItems := make([]*v1.InstallmentScheduleItem, 0, len(schedules))
	for _, sc := range schedules {
		schedItems = append(schedItems, toInstallmentScheduleItem(repository.InstallmentScheduleRow{
			InstallmentSchedule: sc,
			InvoiceID:           inv.ID,
			InvoiceNo:           inv.InvoiceNo,
			MemberID:            *inv.MemberID,
		}))
	}
	return &v1.InstallmentPlanItem{
		ID:           plan.ID,
		InvoiceID:    plan.InvoiceID,
		InvoiceNo:    inv.InvoiceNo,
		MemberID:     plan.MemberID,
		Total:        plan.Total,
		Installments: plan.Installments,
		CreatedBy:    plan.CreatedBy,
		Schedules:    schedItems,
		CreatedAt:    plan.CreatedAt,
		UpdatedAt:    plan.UpdatedAt,
	}, nil
}

func (s *financeService) ListInstallments(ctx context.Context, req *v1.ListInstallmentsReq) (v1.ListInstallmentsRes, error) {
	branchIDs := reqctx.AllowedBranches(ctx)
	rows, err := s.repo.ListInstallmentSchedules(ctx, repository.InstallmentFilter{
		BranchIDs: branchIDs,
		Status:    strings.ToUpper(strings.TrimSpace(req.Status)),
		MemberID:  req.MemberID,
		InvoiceID: req.InvoiceID,
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toInstallmentScheduleItem), nil
}

func (s *financeService) PayInstallmentSchedule(ctx context.Context, req *v1.PayInstallmentScheduleReq) (*v1.InstallmentScheduleItem, error) {
	row, err := s.repo.FindInstallmentScheduleByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy kỳ trả góp")
		}
		return nil, err
	}
	if row.BranchID > 0 && !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không được thu tiền trả góp của chi nhánh này")
	}
	if row.Status == entity.InstallmentPaid {
		return nil, rescode.New(rescode.Conflict, "Kỳ trả góp này đã được thanh toán")
	}

	now := s.now()
	sc := row.InstallmentSchedule
	sc.PaidAmount = sc.Amount
	sc.Status = entity.InstallmentPaid
	sc.PaidAt = &now
	sc.UpdatedAt = now

	if err := s.repo.UpdateInstallmentSchedule(ctx, &sc); err != nil {
		return nil, err
	}
	if s.invoices != nil && row.InvoiceID > 0 {
		method := strings.ToUpper(strings.TrimSpace(req.Method))
		if method == "" {
			method = entity.PaymentCash
		}
		_, _ = s.invoices.AddPayment(ctx, row.InvoiceID, PaymentInput{
			Amount: sc.Amount,
			Method: method,
			Note:   strings.TrimSpace(req.Note),
		})
	}
	updatedRow := *row
	updatedRow.InstallmentSchedule = sc
	return toInstallmentScheduleItem(updatedRow), nil
}

func (s *financeService) MemberInstallments(ctx context.Context) (v1.MemberInstallmentsRes, error) {
	uid := reqctx.UserID(ctx)
	if uid == nil {
		return nil, rescode.New(rescode.Unauthorized, "Bạn chưa đăng nhập")
	}
	m, err := s.repo.FindMemberByUserID(ctx, *uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Tài khoản chưa được liên kết với hội viên")
		}
		return nil, err
	}
	plans, err := s.repo.ListInstallmentPlansByMember(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if len(plans) == 0 {
		return v1.MemberInstallmentsRes{}, nil
	}
	planIDs := make([]uint, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}
	schedRows, err := s.repo.ListSchedulesByPlanIDs(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	byPlan := map[uint][]*v1.InstallmentScheduleItem{}
	for _, sr := range schedRows {
		byPlan[sr.PlanID] = append(byPlan[sr.PlanID], toInstallmentScheduleItem(sr))
	}
	out := make(v1.MemberInstallmentsRes, 0, len(plans))
	for _, p := range plans {
		out = append(out, &v1.InstallmentPlanItem{
			ID:           p.ID,
			InvoiceID:    p.InvoiceID,
			InvoiceNo:    p.InvoiceNo,
			MemberID:     p.MemberID,
			MemberCode:   p.MemberCode,
			MemberName:   p.MemberName,
			Total:        p.Total,
			Installments: p.Installments,
			CreatedBy:    p.CreatedBy,
			Schedules:    byPlan[p.ID],
			CreatedAt:    p.CreatedAt,
			UpdatedAt:    p.UpdatedAt,
		})
	}
	return out, nil
}

func (s *financeService) IssueEInvoice(ctx context.Context, invoiceID uint) (*v1.IssueEInvoiceRes, error) {
	inv, err := s.repo.FindInvoiceByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy hóa đơn")
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, inv.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xuất hóa đơn điện tử của chi nhánh này")
	}
	return s.einvoiceProvider.IssueInvoice(ctx, inv)
}

func toExpenseItem(r repository.ExpenseRow) *v1.ExpenseItem {
	return &v1.ExpenseItem{
		ID:               r.ID,
		BranchID:         r.BranchID,
		BranchName:       r.BranchName,
		CategoryID:       r.CategoryID,
		CategoryName:     r.CategoryName,
		CategoryType:     r.CategoryType,
		Amount:           r.Amount,
		SpentAt:          r.SpentAt.Format(consts.DateLayout),
		Description:      r.Description,
		AttachmentURL:    r.AttachmentURL,
		MaintenanceLogID: r.MaintenanceLogID,
		CreatedBy:        r.CreatedBy,
		CreatorName:      r.CreatorName,
		ApprovedBy:       r.ApprovedBy,
		ApproverName:     r.ApproverName,
		Status:           r.Status,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func toCashSessionItem(r repository.CashSessionRow) *v1.CashSessionItem {
	return &v1.CashSessionItem{
		ID:           r.ID,
		BranchID:     r.BranchID,
		BranchName:   r.BranchName,
		StaffID:      r.StaffID,
		StaffName:    r.StaffName,
		OpenedAt:     r.OpenedAt,
		ClosedAt:     r.ClosedAt,
		OpeningCash:  r.OpeningCash,
		ExpectedCash: r.ExpectedCash,
		CountedCash:  r.CountedCash,
		Difference:   r.Difference,
		Note:         r.Note,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func toInstallmentScheduleItem(r repository.InstallmentScheduleRow) *v1.InstallmentScheduleItem {
	return &v1.InstallmentScheduleItem{
		ID:         r.ID,
		PlanID:     r.PlanID,
		InvoiceID:  r.InvoiceID,
		InvoiceNo:  r.InvoiceNo,
		MemberID:   r.MemberID,
		MemberCode: r.MemberCode,
		MemberName: r.MemberName,
		Seq:        r.Seq,
		DueDate:    r.DueDate.Format(consts.DateLayout),
		Amount:     r.Amount,
		PaidAmount: r.PaidAmount,
		Status:     r.Status,
		PaidAt:     r.PaidAt,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}
