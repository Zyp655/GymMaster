package service

import (
	"context"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

// financeMaxAmount: giới hạn số tiền vừa cột NUMERIC(14,2).
const financeMaxAmount = 999_999_999_999

var (
	validExpenseCategoryTypes = []string{entity.ExpenseTypeFixed, entity.ExpenseTypeVariable}

	errCashSessionAlreadyOpen   = rescode.New(rescode.Conflict, "Chi nhánh đang có ca thu ngân chưa đóng")
	errCashSessionAlreadyClosed = rescode.New(rescode.Conflict, "Ca thu ngân đã được đóng trước đó")
	errInstallmentPlanExists    = rescode.New(rescode.Conflict, "Hóa đơn này đã có kế hoạch trả góp")
	errInstallmentAlreadyPaid   = rescode.New(rescode.Conflict, "Kỳ trả góp này đã được thanh toán")
	errInstallmentClosed        = rescode.New(rescode.BadRequest, "Kỳ trả góp đã tất toán hoặc đã hủy (hóa đơn đã thu đủ/đã hủy), không thể thu tiền")
	errFinanceNoteTooLong       = rescode.New(rescode.BadRequest, "Ghi chú tối đa 500 ký tự")
)

// financeNoteMaxLen: độ dài tối đa ghi chú ca thu ngân (cột VARCHAR(500)).
const financeNoteMaxLen = 500

// withTx chạy fn trong transaction (tx nil – trong unit test – thì chạy trực tiếp).
func (s *financeService) withTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}
	return s.tx.WithTx(ctx, fn)
}

func financeValidateExpenseAmount(amount float64) error {
	if amount <= 0 {
		return rescode.New(rescode.BadRequest, "Số tiền chi phí phải lớn hơn 0")
	}
	if amount > financeMaxAmount {
		return rescode.New(rescode.BadRequest, "Số tiền chi phí vượt quá giới hạn cho phép")
	}
	return nil
}

// financeExpenseMaxAgeMonths: ngày chi không được cũ hơn 12 tháng so với hôm nay (chặn nhập nhầm năm, vd 1999).
const financeExpenseMaxAgeMonths = 12

// financeParseSpentAt đọc ngày chi (YYYY-MM-DD). Quy tắc: phiếu chi ghi nhận khoản ĐÃ chi nên ngày chi không được
// sau hôm nay (giờ VN) – chặn lỗi nhập nhầm năm (vd 2030) làm lệch báo cáo lãi/lỗ; khoản trả trước ghi theo ngày chi thực tế.
// Đồng thời không được cũ hơn financeExpenseMaxAgeMonths tháng (vd 1999) – kỳ báo cáo cũ đã chốt.
func financeParseSpentAt(raw string, now time.Time) (*time.Time, error) {
	parsed, err := equipmentParseDateOpt(raw)
	if err != nil || parsed == nil {
		return parsed, err
	}
	today := equipmentStartOfDay(now)
	if parsed.After(today) {
		return nil, rescode.New(rescode.BadRequest, "Ngày chi không được sau ngày hôm nay")
	}
	if minDate := today.AddDate(0, -financeExpenseMaxAgeMonths, 0); parsed.Before(minDate) {
		return nil, rescode.New(rescode.BadRequest,
			"Ngày chi không được trước "+minDate.Format("02/01/2006")+" (tối đa 12 tháng trước hôm nay)")
	}
	return parsed, nil
}

// checkBranch: chi nhánh phải tồn tại (404); requireActive = chặn thao tác ghi tại chi nhánh ngừng hoạt động (400).
func (s *financeService) checkBranch(ctx context.Context, branchID uint, requireActive bool) error {
	status, err := s.repo.FindBranchStatus(ctx, branchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rescode.New(rescode.NotFound, "Không tìm thấy chi nhánh")
		}
		return err
	}
	if requireActive && status != consts.StatusActive {
		return rescode.New(rescode.BadRequest, "Chi nhánh đang ngừng hoạt động")
	}
	return nil
}

// auditLog ghi nhật ký thao tác tài chính (audit nil trong unit test).
func (s *financeService) auditLog(ctx context.Context, action, entityName string, id uint, before, after any) {
	if s.audit != nil {
		s.audit.Log(ctx, action, entityName, id, before, after)
	}
}

// installmentSplitAmounts chia total thành n kỳ (làm tròn đồng), kỳ cuối nhận phần dư để tổng khớp total.
func installmentSplitAmounts(total float64, n int) []float64 {
	out := make([]float64, n)
	per := math.Floor(total / float64(n))
	var allocated float64
	for i := 0; i < n; i++ {
		amt := per
		if i == n-1 {
			amt = roundMoneyVND(total - allocated)
		}
		allocated += amt
		out[i] = amt
	}
	return out
}

// installmentDueDate trả ngày đến hạn kỳ thứ i (0-based) = firstDue + i tháng, kẹp về ngày cuối tháng
// nếu tháng đích ngắn hơn (31/01 → 28|29/02 → 31/03 …) thay vì tràn sang tháng sau như time.AddDate.
func installmentDueDate(firstDue time.Time, i int) time.Time {
	y, m, d := firstDue.Date()
	target := time.Date(y, m+time.Month(i), 1, 0, 0, 0, 0, firstDue.Location())
	last := target.AddDate(0, 1, -1).Day()
	if d > last {
		d = last
	}
	return time.Date(target.Year(), target.Month(), d, 0, 0, 0, 0, firstDue.Location())
}

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
	if err := financeValidateExpenseAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := s.checkBranch(ctx, req.BranchID, true); err != nil {
		return nil, err
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
		parsed, pErr := financeParseSpentAt(req.SpentAt, now)
		if pErr != nil {
			return nil, pErr
		}
		spentAt = *parsed
	}

	if !validAttachmentURL(strings.TrimSpace(req.AttachmentURL)) {
		return nil, rescode.New(rescode.BadRequest, "Đường dẫn chứng từ không hợp lệ (chỉ nhận tệp đã tải lên hệ thống hoặc liên kết http/https)")
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
	// Trạng thái chỉ đổi qua POST /expenses/{id}/approve (ghi nhận người duyệt) – PUT không được đổi trạng thái.
	if status := strings.ToUpper(strings.TrimSpace(req.Status)); status != "" && status != row.Status {
		return nil, rescode.New(rescode.BadRequest, "Không thể đổi trạng thái khi cập nhật chi phí, vui lòng dùng chức năng duyệt / từ chối")
	}
	exp := row.Expense
	updatedRow := *row
	if req.CategoryID > 0 && req.CategoryID != exp.CategoryID {
		cat, cErr := s.repo.FindExpenseCategoryByID(ctx, req.CategoryID)
		if cErr != nil {
			if errors.Is(cErr, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.NotFound, "Không tìm thấy danh mục chi phí")
			}
			return nil, cErr
		}
		exp.CategoryID = cat.ID
		updatedRow.CategoryName, updatedRow.CategoryType = cat.Name, cat.Type
	}
	if req.Amount != nil {
		if err := financeValidateExpenseAmount(*req.Amount); err != nil {
			return nil, err
		}
		exp.Amount = *req.Amount
	}
	if strings.TrimSpace(req.SpentAt) != "" {
		parsed, pErr := financeParseSpentAt(req.SpentAt, s.now())
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
		if !validAttachmentURL(exp.AttachmentURL) {
			return nil, rescode.New(rescode.BadRequest, "Đường dẫn chứng từ không hợp lệ (chỉ nhận tệp đã tải lên hệ thống hoặc liên kết http/https)")
		}
	}
	// Khoản đã duyệt đã vào báo cáo lãi/lỗ: không được sửa số tiền/danh mục/ngày chi (chỉ sửa mô tả, chứng từ).
	// Cần điều chỉnh → ghi khoản chi bù trừ mới. (Trước đây sửa được rồi tự quay về chờ duyệt → xóa hẳn.)
	if row.Status == entity.ExpenseStatusApproved &&
		(exp.Amount != row.Amount || exp.CategoryID != row.CategoryID || !exp.SpentAt.Equal(row.SpentAt)) {
		return nil, rescode.New(rescode.Conflict,
			"Khoản chi đã duyệt không thể sửa số tiền, danh mục hoặc ngày chi; vui lòng ghi khoản chi điều chỉnh mới")
	}
	// Khoản bị từ chối mà sửa số tiền/danh mục → quay về chờ duyệt để duyệt lại.
	if row.Status == entity.ExpenseStatusRejected && (exp.Amount != row.Amount || exp.CategoryID != row.CategoryID) {
		exp.Status = entity.ExpenseStatusPending
		exp.ApprovedBy = nil
		updatedRow.ApproverName = ""
	}
	exp.UpdatedAt = s.now()
	if err := s.repo.UpdateExpense(ctx, &exp); err != nil {
		return nil, err
	}
	updatedRow.Expense = exp
	res := toExpenseItem(updatedRow)
	s.auditLog(ctx, AuditUpdate, auditEntityExpense, exp.ID, toExpenseItem(*row), res)
	return res, nil
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
	// Khoản đã duyệt là chứng từ kế toán đã vào P&L → không xóa (chỉ xóa khoản chờ duyệt/bị từ chối).
	if row.Status == entity.ExpenseStatusApproved {
		return rescode.New(rescode.Conflict, "Khoản chi đã duyệt không thể xóa")
	}
	if err := s.repo.DeleteExpense(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, AuditDelete, auditEntityExpense, id, toExpenseItem(*row), nil)
	return nil
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
	// Chỉ duyệt/từ chối khoản đang chờ duyệt – không đổi thẳng APPROVED ↔ REJECTED.
	if row.Status != entity.ExpenseStatusPending {
		return nil, errExpenseNotPending(row.Status)
	}
	exp := row.Expense
	exp.Status = status
	exp.ApprovedBy = reqctx.UserID(ctx)
	exp.UpdatedAt = s.now()
	// UPDATE có điều kiện status = 'PENDING': duyệt và từ chối đồng thời chỉ một yêu cầu thắng.
	ok, err := s.repo.SetExpenseStatus(ctx, exp.ID, entity.ExpenseStatusPending, status, exp.ApprovedBy, exp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errExpenseNotPending("")
	}
	updatedRow := *row
	updatedRow.Expense = exp
	res := toExpenseItem(updatedRow)
	s.auditLog(ctx, AuditUpdate, auditEntityExpense, exp.ID, toExpenseItem(*row), res)
	return res, nil
}

func errExpenseNotPending(status string) error {
	switch status {
	case entity.ExpenseStatusApproved:
		return rescode.New(rescode.Conflict, "Khoản chi đã được duyệt trước đó, không thể duyệt/từ chối lại")
	case entity.ExpenseStatusRejected:
		return rescode.New(rescode.Conflict, "Khoản chi đã bị từ chối; vui lòng sửa khoản chi để gửi duyệt lại")
	}
	return rescode.New(rescode.Conflict, "Khoản chi không còn ở trạng thái chờ duyệt")
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
	if utf8.RuneCountInString(strings.TrimSpace(req.Note)) > financeNoteMaxLen {
		return nil, errFinanceNoteTooLong
	}
	if err := s.checkBranch(ctx, req.BranchID, true); err != nil {
		return nil, err
	}
	if open, err := s.repo.FindOpenCashSession(ctx, req.BranchID); err == nil && open != nil {
		return nil, errCashSessionAlreadyOpen
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
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
	// Unique index một phần uq_cash_sessions_branch_open chặn hai yêu cầu mở ca đồng thời.
	if err := s.repo.CreateCashSession(ctx, cs); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errCashSessionAlreadyOpen
		}
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
		return nil, errCashSessionAlreadyClosed
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Note)) > financeNoteMaxLen {
		return nil, errFinanceNoteTooLong
	}
	now := s.now()
	cashCollected, err := s.repo.SumCashPaymentsBetween(ctx, row.BranchID, row.OpenedAt, now)
	if err != nil {
		return nil, err
	}
	cashRefunded, err := s.repo.SumCashRefundsBetween(ctx, row.BranchID, row.OpenedAt, now)
	if err != nil {
		return nil, err
	}
	cs := row.CashSession
	cs.ClosedAt = &now
	// Tiền mặt kỳ vọng = đầu ca + phiếu thu CASH − phiếu hoàn CASH trong ca.
	cs.ExpectedCash = roundMoneyVND(cs.OpeningCash + cashCollected - cashRefunded)
	cs.CountedCash = req.CountedCash
	cs.Difference = cs.CountedCash - cs.ExpectedCash
	if note := strings.TrimSpace(req.Note); note != "" {
		cs.Note = note
	}
	cs.Status = entity.CashSessionClosed
	cs.UpdatedAt = now
	// UPDATE có điều kiện status = 'OPEN': nhiều yêu cầu đóng cùng lúc chỉ một yêu cầu thắng, còn lại 409
	// (trước đây Save() vô điều kiện → yêu cầu cuối ghi đè số tiền đếm được của yêu cầu trước).
	closed, err := s.repo.CloseCashSession(ctx, &cs)
	if err != nil {
		return nil, err
	}
	if !closed {
		return nil, errCashSessionAlreadyClosed
	}
	updatedRow := *row
	updatedRow.CashSession = cs
	res := toCashSessionItem(updatedRow)
	if s.audit != nil {
		s.audit.Log(ctx, AuditUpdate, auditEntityCashSession, cs.ID, toCashSessionItem(*row), res)
	}
	return res, nil
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
	// Chi nhánh không tồn tại → 404 (thống nhất với KPI/cohort), không trả báo cáo toàn số 0.
	if req.BranchID > 0 {
		if err := s.checkBranch(ctx, req.BranchID, false); err != nil {
			return nil, err
		}
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
	if to.Before(from) {
		return nil, rescode.New(rescode.BadRequest, "Ngày kết thúc phải sau hoặc bằng ngày bắt đầu")
	}
	// Mốc ngày theo giờ Việt Nam (00:00 +07) – các cột timestamptz (issued_at, created_at) được so theo ngày lịch VN.
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, memberVNLoc)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, memberVNLoc)
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
		ExpensesByCategory: dto.Map(data.ExpensesByCategory, func(c repository.ExpenseCategoryTotal) *v1.ExpenseByCategoryItem {
			return &v1.ExpenseByCategoryItem{
				CategoryID:   c.CategoryID,
				CategoryName: c.CategoryName,
				CategoryType: c.CategoryType,
				Amount:       c.Amount,
			}
		}),
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
	// Chỉ chia kỳ phần còn nợ của hóa đơn đang chờ thu (UNPAID/PARTIAL).
	switch inv.Status {
	case entity.InvoiceStatusVoid:
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn đã bị hủy, không thể tạo kế hoạch trả góp")
	case entity.InvoiceStatusRefunded:
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn đã được hoàn tiền, không thể tạo kế hoạch trả góp")
	case entity.InvoiceStatusPaid:
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn đã được thanh toán đủ, không cần trả góp")
	}
	totalToSplit := roundMoneyVND(inv.Total - inv.PaidAmount)
	if totalToSplit <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn không còn số tiền phải thu để trả góp")
	}
	if existing, err := s.repo.FindInstallmentPlanByInvoice(ctx, inv.ID); err == nil && existing != nil {
		return nil, errInstallmentPlanExists
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := s.now()
	today := equipmentStartOfDay(now)
	firstDue := today
	if strings.TrimSpace(req.FirstDueDate) != "" {
		parsed, pErr := equipmentParseDateOpt(req.FirstDueDate)
		if pErr != nil {
			return nil, pErr
		}
		if parsed.Before(today) {
			return nil, rescode.New(rescode.BadRequest, "Ngày đến hạn kỳ đầu không được ở quá khứ")
		}
		firstDue = *parsed
	}

	amounts := installmentSplitAmounts(totalToSplit, req.Installments)
	schedules := make([]entity.InstallmentSchedule, req.Installments)
	for i := range schedules {
		schedules[i] = entity.InstallmentSchedule{
			Seq:       i + 1,
			DueDate:   installmentDueDate(firstDue, i),
			Amount:    amounts[i],
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
	if err := s.withTx(ctx, func(ctx context.Context) error {
		return s.repo.CreateInstallmentPlan(ctx, plan, schedules)
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errInstallmentPlanExists
		}
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, AuditCreate, auditEntityInstallment, plan.ID, nil, map[string]any{
			"invoice_id": plan.InvoiceID, "total": plan.Total, "installments": plan.Installments,
		})
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
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = entity.PaymentCash
	}
	if !slices.Contains(invoicePayMethods, method) {
		return nil, rescode.New(rescode.BadRequest, "Phương thức thanh toán không hợp lệ")
	}
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
		return nil, errInstallmentAlreadyPaid
	}
	if !installmentIsOpen(row.Status) {
		return nil, errInstallmentClosed
	}
	if s.invoices == nil || row.InvoiceID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Kỳ trả góp không gắn với hóa đơn để thu tiền")
	}

	var sc entity.InstallmentSchedule
	// Một transaction: khóa kỳ trả góp → đánh dấu kỳ đã thu → ghi phiếu thu qua InvoiceService (khóa hóa đơn,
	// kiểm tra số còn nợ, đồng bộ các kỳ còn lại). Thu thất bại → rollback cả việc đánh dấu kỳ.
	err = s.withTx(ctx, func(ctx context.Context) error {
		// Thứ tự khóa: hóa đơn → kỳ trả góp (giống AddPayment → đồng bộ trả góp) để không deadlock.
		inv, err := s.invoices.LockInvoice(ctx, row.InvoiceID)
		if err != nil {
			return err
		}
		locked, err := s.repo.LockInstallmentSchedule(ctx, row.ID)
		if err != nil {
			return err
		}
		if locked.Status == entity.InstallmentPaid {
			return errInstallmentAlreadyPaid
		}
		if !installmentIsOpen(locked.Status) {
			return errInstallmentClosed
		}
		if inv.Status != entity.InvoiceStatusUnpaid && inv.Status != entity.InvoiceStatusPartial {
			return rescode.New(rescode.BadRequest, "Hóa đơn của kỳ trả góp đã thanh toán đủ hoặc đã hủy, không cần thu thêm")
		}
		amount := roundMoneyVND(locked.Amount - locked.PaidAmount)
		// Hóa đơn còn nợ ít hơn số của kỳ (đã thu trực tiếp một phần) → chỉ thu đúng số còn nợ và tất toán kỳ.
		if due := roundMoneyVND(inv.Total - inv.PaidAmount); due < amount {
			amount = due
		}
		if amount <= 0 {
			return errInstallmentAlreadyPaid
		}
		note := strings.TrimSpace(req.Note)
		if note == "" {
			note = "Thu trả góp kỳ " + strconv.Itoa(locked.Seq)
		}
		now := s.now()
		sc = *locked
		sc.PaidAmount = roundMoneyVND(locked.PaidAmount + amount)
		sc.Amount = sc.PaidAmount
		sc.Status = entity.InstallmentPaid
		sc.PaidAt = &now
		sc.UpdatedAt = now
		// Đánh dấu kỳ TRƯỚC khi ghi thu để bước đồng bộ trả góp trong AddPayment thấy đúng các kỳ còn mở.
		if err := s.repo.UpdateInstallmentSchedule(ctx, &sc); err != nil {
			return err
		}
		if _, err := s.invoices.AddPayment(ctx, row.InvoiceID, PaymentInput{
			Amount: amount,
			Method: method,
			Note:   note,
		}); err != nil {
			// Transaction sẽ rollback; khôi phục tường minh để an toàn cả khi chạy không có transaction.
			_ = s.repo.UpdateInstallmentSchedule(ctx, locked)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
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
