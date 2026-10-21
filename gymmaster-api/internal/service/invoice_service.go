package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	v1 "gymmaster-api/api/invoice/v1"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Hành động audit của hóa đơn.
const (
	auditEntityInvoice = "invoice"
	AuditPayment       = "PAYMENT"
	AuditVoid          = "VOID"
	AuditRefund        = "REFUND"
)

type (
	// InvoiceDraft mô tả hóa đơn cần lập. Dùng chung cho mọi module bán hàng.
	InvoiceDraft struct {
		MemberID *uint  // nil = khách vãng lai (chỉ hợp lệ với POS)
		BranchID uint   // chi nhánh lập hóa đơn (phải thuộc phạm vi người dùng)
		Source   string // COUNTER (mặc định) | POS | PORTAL
		Items    []InvoiceItemDraft
		// PromotionCode: mã khuyến mãi (tùy chọn). Khi có thì AppliesTo bắt buộc.
		PromotionCode string
		AppliesTo     string // MEMBERSHIP | PT | PRODUCT – loại đơn để kiểm tra mã
		Note          string
	}

	// InvoiceItemDraft là một dòng hóa đơn. Discount là giảm giá riêng của dòng (ví dụ khấu trừ khi nâng cấp),
	// không gồm khuyến mãi – phần khuyến mãi được InvoiceService tự phân bổ.
	InvoiceItemDraft struct {
		ItemType    string // MEMBERSHIP | PT | PRODUCT | FEE | OTHER
		RefID       *uint
		Description string
		Quantity    int // 0 = 1
		UnitPrice   float64
		Discount    float64
	}

	PaymentInput struct {
		Amount    float64
		Method    string // CASH | TRANSFER | CARD
		Reference string
		Note      string
	}

	RefundInput struct {
		Amount float64
		Method string
		Reason string
	}

	// InvoicePreview là kết quả tính tiền (không ghi DB).
	InvoicePreview struct {
		Items             []entity.InvoiceItem // đã có Discount (gồm khuyến mãi phân bổ) và Amount
		Subtotal          float64              // Σ quantity × unit_price
		Discount          float64              // Σ giảm theo dòng + khuyến mãi
		Total             float64
		Promotion         *entity.Promotion
		PromotionDiscount float64
	}

	// InvoiceItemHandler được gọi cho TỪNG dòng có item_type tương ứng khi hóa đơn chuyển trạng thái.
	// Handler chạy trong transaction của thao tác (thu tiền / hủy); trả lỗi → rollback toàn bộ.
	// inv là trạng thái hóa đơn SAU thay đổi.
	InvoiceItemHandler func(ctx context.Context, inv *entity.Invoice, item *entity.InvoiceItem) error

	// InvoiceService là dịch vụ hóa đơn – thanh toán – hoàn tiền dùng chung cho bán gói tập, PT, POS.
	//
	// Module khác dùng qua module.Deps.Invoices:
	//
	//	// 1) Khi khởi tạo module: đăng ký xử lý khi hóa đơn được thanh toán đủ / bị hủy.
	//	d.Invoices.RegisterPaidHandler(entity.InvoiceItemPT, func(ctx context.Context, inv *entity.Invoice, it *entity.InvoiceItem) error {
	//	    return contractRepo.Activate(ctx, *it.RefID)
	//	})
	//	// 2) Trong transaction nghiệp vụ: tạo bản ghi nghiệp vụ trước, rồi lập hóa đơn trỏ tới nó.
	//	err := d.Tx.WithTx(ctx, func(ctx context.Context) error {
	//	    ... tạo pt_contract ...
	//	    inv, err := d.Invoices.CreateInvoice(ctx, service.InvoiceDraft{MemberID: &memberID, BranchID: b,
	//	        Items: []service.InvoiceItemDraft{{ItemType: entity.InvoiceItemPT, RefID: &contract.ID, Description: "PT 10 buổi", UnitPrice: 3500000}},
	//	        PromotionCode: code, AppliesTo: entity.AppliesPT})
	//	    ... lưu inv.ID vào bản ghi nghiệp vụ ...
	//	    if pay != nil { _, err = d.Invoices.AddPayment(ctx, inv.ID, *pay) } // đủ tiền → PaidHandler chạy
	//	    return err
	//	})
	InvoiceService interface {
		// Preview tính tiền (dòng, giảm giá, khuyến mãi) mà không ghi DB – dùng cho màn hình báo giá.
		Preview(ctx context.Context, draft InvoiceDraft) (*InvoicePreview, error)
		// CreateInvoice lập hóa đơn UNPAID: tính subtotal/discount/total, kiểm tra + tiêu thụ mã khuyến mãi
		// (Validate + Consume) và phân bổ giảm giá vào từng dòng. Nên gọi TRONG transaction của nghiệp vụ
		// (tự mở transaction nếu chưa có). Hóa đơn tổng 0đ được đánh dấu PAID ngay và gọi PaidHandler.
		// Trả hóa đơn kèm Items.
		CreateInvoice(ctx context.Context, draft InvoiceDraft) (*entity.Invoice, error)
		// AddPayment thu tiền: khóa hóa đơn (FOR UPDATE), không cho thu quá số còn nợ, cập nhật paid_amount/status.
		// Khi hóa đơn chuyển sang PAID → gọi PaidHandler cho từng dòng.
		AddPayment(ctx context.Context, invoiceID uint, in PaymentInput) (*entity.Invoice, error)
		// Void hủy hóa đơn – chỉ khi CHƯA thu đồng nào (UNPAID). Trả lại lượt khuyến mãi, gọi VoidHandler.
		Void(ctx context.Context, invoiceID uint, reason string) error
		// Refund hoàn tiền ≤ paid_amount − refunded_amount; hoàn hết số đã thu → REFUNDED.
		Refund(ctx context.Context, invoiceID uint, in RefundInput) (*entity.Invoice, error)
		// RegisterPaidHandler / RegisterVoidHandler: đăng ký xử lý theo item_type (gọi khi khởi tạo module,
		// có thể đăng ký nhiều handler cho một loại).
		RegisterPaidHandler(itemType string, h InvoiceItemHandler)
		RegisterVoidHandler(itemType string, h InvoiceItemHandler)
		// RegisterPayGuard: kiểm tra trước khi thu tiền theo item_type (vd. gói tập đã hủy thì không thu nữa).
		// Guard trả lỗi → AddPayment bị từ chối.
		RegisterPayGuard(itemType string, h InvoiceItemHandler)
		// WriteOffRemaining khóa sổ phần còn nợ của hóa đơn (vd. khi hủy gói tập đã thu một phần):
		// chưa thu đồng nào → VOID; đã thu một phần → giảm total = paid_amount (phần nợ chuyển thành giảm trừ)
		// để hóa đơn không còn nợ và không thu tiếp. Không gọi Paid/VoidHandler.
		WriteOffRemaining(ctx context.Context, invoiceID uint, reason string) (*entity.Invoice, error)
		// FindInvoice đọc hóa đơn (không kiểm tra phạm vi) – cho module khác.
		FindInvoice(ctx context.Context, id uint) (*entity.Invoice, error)
		// LockInvoice khóa hóa đơn (FOR UPDATE, kiểm tra phạm vi chi nhánh) – gọi TRONG transaction khi cần
		// khóa hóa đơn trước các bản ghi phụ (vd kỳ trả góp) để giữ thứ tự khóa thống nhất, tránh deadlock.
		LockInvoice(ctx context.Context, id uint) (*entity.Invoice, error)

		// API quản trị (có kiểm tra phạm vi chi nhánh).
		Get(ctx context.Context, id uint) (*v1.InvoiceDetail, error)
		// Detail giống Get nhưng KHÔNG kiểm tra phạm vi chi nhánh (dùng khi bản ghi nghiệp vụ đã được kiểm tra).
		Detail(ctx context.Context, id uint) (*v1.InvoiceDetail, error)
		List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error)
		ListPayments(ctx context.Context, req *v1.PaymentListReq) (*v1.PaymentListRes, error)
		Print(ctx context.Context, id uint) (*v1.PrintRes, error)
		// ListByMember: hóa đơn của một hội viên (cổng hội viên, không kiểm tra phạm vi chi nhánh).
		ListByMember(ctx context.Context, memberID uint, page dto.PageReq) (*v1.ListRes, error)
	}

	invoiceService struct {
		repo       repository.InvoiceRepository
		promotions PromotionService
		tx         postgres.TxManager
		audit      AuditService
		now        func() time.Time

		mu           sync.RWMutex
		paidHandlers map[string][]InvoiceItemHandler
		voidHandlers map[string][]InvoiceItemHandler
		payGuards    map[string][]InvoiceItemHandler
	}
)

// NewInvoiceService tạo dịch vụ hóa đơn. Tạo MỘT lần trong cmd và chia sẻ qua module.Deps.Invoices
// để handler của các module cùng đăng ký vào một chỗ. promotions/audit có thể nil (test).
func NewInvoiceService(repo repository.InvoiceRepository, promotions PromotionService, tx postgres.TxManager, audit AuditService) InvoiceService {
	return &invoiceService{
		repo:         repo,
		promotions:   promotions,
		tx:           tx,
		audit:        audit,
		now:          time.Now,
		paidHandlers: map[string][]InvoiceItemHandler{},
		voidHandlers: map[string][]InvoiceItemHandler{},
		payGuards:    map[string][]InvoiceItemHandler{},
	}
}

// invoiceActorID: người thao tác cho các cột FK users(id). Tác vụ hệ thống (IPN, cron) không có người dùng
// hoặc UserID = 0 → NULL (tránh vi phạm FK received_by/issued_by...).
func invoiceActorID(ctx context.Context) *uint {
	if uid := reqctx.UserID(ctx); uid != nil && *uid != 0 {
		return uid
	}
	return nil
}

var (
	invoiceItemTypes     = []string{entity.InvoiceItemMembership, entity.InvoiceItemPT, entity.InvoiceItemProduct, entity.InvoiceItemFee, entity.InvoiceItemLocker, entity.InvoiceItemOther}
	invoiceSources       = []string{entity.InvoiceSourceCounter, entity.InvoiceSourcePOS, entity.InvoiceSourcePortal}
	invoicePayMethods    = []string{entity.PaymentCash, entity.PaymentTransfer, entity.PaymentCard}
	invoiceOrderable     = []string{"id", "issued_at", "total", "paid_amount", "invoice_no", "status"}
	paymentOrderable     = []string{"id", "paid_at", "amount", "method"}
	errInvoiceNotFound   = rescode.New(rescode.NotFound, "Không tìm thấy hóa đơn")
	errInvoiceOutOfScope = rescode.New(rescode.PermissionDenied, "Hóa đơn thuộc chi nhánh ngoài phạm vi của bạn")
)

func (s *invoiceService) RegisterPaidHandler(itemType string, h InvoiceItemHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paidHandlers[itemType] = append(s.paidHandlers[itemType], h)
}

func (s *invoiceService) RegisterVoidHandler(itemType string, h InvoiceItemHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.voidHandlers[itemType] = append(s.voidHandlers[itemType], h)
}

func (s *invoiceService) RegisterPayGuard(itemType string, h InvoiceItemHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payGuards[itemType] = append(s.payGuards[itemType], h)
}

func (s *invoiceService) runHandlers(ctx context.Context, handlers map[string][]InvoiceItemHandler, inv *entity.Invoice, items []entity.InvoiceItem) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range items {
		for _, h := range handlers[items[i].ItemType] {
			if err := h(ctx, inv, &items[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---- Tính tiền (hàm thuần) ------------------------------------------------------

// invoiceBuildItems chuẩn hóa các dòng: làm tròn VND, amount = qty × unit − discount.
// Trả các dòng, subtotal (Σ qty × unit) và tổng giảm theo dòng.
func invoiceBuildItems(drafts []InvoiceItemDraft) ([]entity.InvoiceItem, float64, float64, error) {
	if len(drafts) == 0 {
		return nil, 0, 0, rescode.New(rescode.BadRequest, "Hóa đơn phải có ít nhất một dòng")
	}
	items := make([]entity.InvoiceItem, 0, len(drafts))
	var subtotal, lineDiscount float64
	for _, d := range drafts {
		if !slices.Contains(invoiceItemTypes, d.ItemType) {
			return nil, 0, 0, rescode.New(rescode.BadRequest, "Loại dòng hóa đơn không hợp lệ")
		}
		desc := strings.TrimSpace(d.Description)
		if desc == "" {
			return nil, 0, 0, rescode.New(rescode.BadRequest, "Dòng hóa đơn phải có mô tả")
		}
		if len([]rune(desc)) > 255 {
			desc = string([]rune(desc)[:255])
		}
		qty := d.Quantity
		if qty == 0 {
			qty = 1
		}
		if qty < 0 {
			return nil, 0, 0, rescode.New(rescode.BadRequest, "Số lượng phải lớn hơn 0")
		}
		unit := roundMoneyVND(d.UnitPrice)
		disc := roundMoneyVND(d.Discount)
		if unit < 0 || disc < 0 {
			return nil, 0, 0, rescode.New(rescode.BadRequest, "Đơn giá và giảm giá không được âm")
		}
		gross := roundMoneyVND(float64(qty) * unit)
		if disc > gross {
			disc = gross
		}
		items = append(items, entity.InvoiceItem{
			ItemType: d.ItemType, RefID: d.RefID, Description: desc,
			Quantity: qty, UnitPrice: unit, Discount: disc, Amount: gross - disc,
		})
		subtotal += gross
		lineDiscount += disc
	}
	return items, subtotal, lineDiscount, nil
}

// invoiceAllocateDiscount phân bổ số tiền giảm (khuyến mãi) vào các dòng theo tỷ lệ thành tiền,
// làm tròn đến đồng, phần lẻ dồn vào các dòng còn dư. Bảo đảm Σ phần phân bổ = discount và không dòng nào âm.
func invoiceAllocateDiscount(items []entity.InvoiceItem, discount float64) {
	discount = roundMoneyVND(discount)
	var net float64
	for _, it := range items {
		net += it.Amount
	}
	if discount <= 0 || net <= 0 {
		return
	}
	if discount > net {
		discount = net
	}
	shares := make([]float64, len(items))
	var allocated float64
	for i, it := range items {
		share := math.Min(math.Floor(discount*it.Amount/net), it.Amount)
		shares[i] = share
		allocated += share
	}
	// Phần còn lại (do làm tròn xuống) dồn lần lượt vào dòng có thành tiền lớn nhất còn dư.
	rest := discount - allocated
	for rest > 0 {
		best := -1
		for i, it := range items {
			if it.Amount-shares[i] > 0 && (best < 0 || it.Amount > items[best].Amount) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		add := math.Min(rest, items[best].Amount-shares[best])
		shares[best] += add
		rest -= add
	}
	for i := range items {
		items[i].Discount += shares[i]
		items[i].Amount -= shares[i]
	}
}

func (s *invoiceService) Preview(ctx context.Context, draft InvoiceDraft) (*InvoicePreview, error) {
	items, subtotal, lineDiscount, err := invoiceBuildItems(draft.Items)
	if err != nil {
		return nil, err
	}
	p := &InvoicePreview{Items: items, Subtotal: subtotal, Discount: lineDiscount}
	if code := strings.TrimSpace(draft.PromotionCode); code != "" {
		if draft.AppliesTo == "" {
			return nil, rescode.New(rescode.BadRequest, "Thiếu loại đơn để áp mã khuyến mãi")
		}
		if s.promotions == nil {
			return nil, rescode.New(rescode.BadRequest, "Không áp dụng được mã khuyến mãi")
		}
		promo, disc, err := s.promotions.Validate(ctx, code, draft.AppliesTo, subtotal-lineDiscount, s.now())
		if err != nil {
			return nil, err
		}
		invoiceAllocateDiscount(p.Items, disc)
		p.Promotion, p.PromotionDiscount = promo, disc
		p.Discount += disc
	}
	p.Total = p.Subtotal - p.Discount
	return p, nil
}

// ---- Nghiệp vụ ------------------------------------------------------------------

func (s *invoiceService) CreateInvoice(ctx context.Context, draft InvoiceDraft) (*entity.Invoice, error) {
	if draft.BranchID == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh lập hóa đơn")
	}
	if !reqctx.CanAccessBranch(ctx, draft.BranchID) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền lập hóa đơn tại chi nhánh này")
	}
	if draft.Source == "" {
		draft.Source = entity.InvoiceSourceCounter
	}
	if !slices.Contains(invoiceSources, draft.Source) {
		return nil, rescode.New(rescode.BadRequest, "Kênh lập hóa đơn không hợp lệ")
	}
	if draft.MemberID == nil && draft.Source != entity.InvoiceSourcePOS {
		return nil, rescode.New(rescode.BadRequest, "Hóa đơn phải gắn với hội viên")
	}
	preview, err := s.Preview(ctx, draft)
	if err != nil {
		return nil, err
	}
	now := s.now()
	inv := &entity.Invoice{
		MemberID: draft.MemberID, BranchID: draft.BranchID,
		Subtotal: preview.Subtotal, Discount: preview.Discount, Total: preview.Total,
		Status: entity.InvoiceStatusUnpaid, Source: draft.Source,
		Note: invoiceTruncate(strings.TrimSpace(draft.Note), 500), IssuedBy: invoiceActorID(ctx), IssuedAt: now,
		Items: preview.Items,
	}
	if preview.Promotion != nil {
		inv.PromotionID = &preview.Promotion.ID
	}
	if inv.Total == 0 {
		inv.Status = entity.InvoiceStatusPaid
	}
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if preview.Promotion != nil {
			if err := s.promotions.Consume(ctx, preview.Promotion.ID); err != nil {
				return err
			}
		}
		if err := s.repo.Create(ctx, inv); err != nil {
			return err
		}
		saved, err := s.repo.FindByID(ctx, inv.ID, false)
		if err != nil {
			return err
		}
		inv.InvoiceNo = saved.InvoiceNo
		auditCatalog(ctx, s.audit, AuditCreate, auditEntityInvoice, inv.ID, nil, inv)
		if inv.Status == entity.InvoiceStatusPaid {
			return s.runHandlers(ctx, s.paidHandlers, inv, inv.Items)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *invoiceService) lock(ctx context.Context, id uint) (*entity.Invoice, error) {
	inv, err := s.repo.FindByID(ctx, id, true)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvoiceNotFound
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, inv.BranchID) {
		return nil, errInvoiceOutOfScope
	}
	return inv, nil
}

func invoiceValidMethod(m string) error {
	if m != entity.PaymentVNPay && !slices.Contains(invoicePayMethods, m) {
		return rescode.New(rescode.BadRequest, "Phương thức thanh toán không hợp lệ")
	}
	return nil
}

// invoiceApplyPayment cập nhật số đã thu và trạng thái khi thu thêm amount (hàm thuần, đã kiểm tra hợp lệ).
func invoiceApplyPayment(inv *entity.Invoice, amount float64) error {
	switch inv.Status {
	case entity.InvoiceStatusVoid:
		return rescode.New(rescode.BadRequest, "Hóa đơn đã bị hủy, không thể thu tiền")
	case entity.InvoiceStatusPaid, entity.InvoiceStatusRefunded:
		return rescode.New(rescode.BadRequest, "Hóa đơn đã được thanh toán đủ")
	}
	due := roundMoneyVND(inv.Total - inv.PaidAmount)
	if amount > due {
		return rescode.New(rescode.BadRequest, "Số tiền thu vượt quá số còn nợ ("+formatMoneyVND(due)+")")
	}
	inv.PaidAmount = roundMoneyVND(inv.PaidAmount + amount)
	if inv.PaidAmount >= inv.Total {
		inv.Status = entity.InvoiceStatusPaid
	} else {
		inv.Status = entity.InvoiceStatusPartial
	}
	return nil
}

func (s *invoiceService) AddPayment(ctx context.Context, invoiceID uint, in PaymentInput) (*entity.Invoice, error) {
	amount := roundMoneyVND(in.Amount)
	if amount <= 0 {
		return nil, rescode.New(rescode.BadRequest, "Số tiền thu phải lớn hơn 0")
	}
	if err := invoiceValidMethod(in.Method); err != nil {
		return nil, err
	}
	var out *entity.Invoice
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		inv, err := s.lock(ctx, invoiceID)
		if err != nil {
			return err
		}
		before := *inv
		if err := invoiceApplyPayment(inv, amount); err != nil {
			return err
		}
		// Hội viên đang thanh toán VNPay cho hóa đơn này → không thu tại quầy song song (tránh thu trùng/thu vượt).
		// Kiểm tra SAU khi khóa hóa đơn; IPN VNPay (method VNPAY) không bị chặn.
		if in.Method != entity.PaymentVNPay {
			pending, err := s.repo.PendingOnlineTxn(ctx, inv.ID, s.now().Add(-onlineTxnTTL))
			if err != nil {
				return err
			}
			if pending != nil {
				return rescode.New(rescode.Conflict, fmt.Sprintf(
					"Hóa đơn đang có giao dịch VNPay (%s) chờ hội viên thanh toán, hết hạn lúc %s. Vui lòng đợi giao dịch hoàn tất hoặc hết hạn rồi thu tại quầy",
					pending.TxnRef, pending.CreatedAt.Add(onlineTxnTTL).In(subVNLoc).Format("15:04 02/01/2006")))
			}
		}
		if err := s.checkPayGuards(ctx, inv); err != nil {
			return err
		}
		pay := &entity.Payment{
			InvoiceID: inv.ID, Amount: amount, Method: in.Method,
			Reference: invoiceTruncate(strings.TrimSpace(in.Reference), 100), Note: invoiceTruncate(strings.TrimSpace(in.Note), 255),
			PaidAt: s.now(), ReceivedBy: invoiceActorID(ctx),
		}
		if err := s.repo.CreatePayment(ctx, pay); err != nil {
			return err
		}
		if err := s.repo.UpdateFields(ctx, inv.ID, map[string]any{"paid_amount": inv.PaidAmount, "status": inv.Status}); err != nil {
			return err
		}
		auditCatalog(ctx, s.audit, AuditPayment, auditEntityInvoice, inv.ID,
			map[string]any{"paid_amount": before.PaidAmount, "status": before.Status},
			map[string]any{"payment_id": pay.ID, "amount": amount, "method": pay.Method, "paid_amount": inv.PaidAmount, "status": inv.Status})
		if err := s.syncInstallments(ctx, inv, installmentSettled); err != nil {
			return err
		}
		if inv.Status == entity.InvoiceStatusPaid {
			items, err := s.repo.Items(ctx, inv.ID)
			if err != nil {
				return err
			}
			inv.Items = items
			if err := s.runHandlers(ctx, s.paidHandlers, inv, items); err != nil {
				return err
			}
		}
		out = inv
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// checkPayGuards gọi các guard đã đăng ký cho từng dòng hóa đơn trước khi ghi nhận thu tiền.
func (s *invoiceService) checkPayGuards(ctx context.Context, inv *entity.Invoice) error {
	s.mu.RLock()
	hasGuards := len(s.payGuards) > 0
	s.mu.RUnlock()
	if !hasGuards {
		return nil
	}
	items, err := s.repo.Items(ctx, inv.ID)
	if err != nil {
		return err
	}
	return s.runHandlers(ctx, s.payGuards, inv, items)
}

// invoiceApplyWriteOff khóa sổ phần còn nợ (hàm thuần). Trả false nếu không còn gì để khóa sổ.
func invoiceApplyWriteOff(inv *entity.Invoice) bool {
	if inv.Status == entity.InvoiceStatusVoid {
		return false
	}
	due := roundMoneyVND(inv.Total - inv.PaidAmount)
	if due <= 0 {
		return false
	}
	if inv.PaidAmount <= 0 {
		inv.Status = entity.InvoiceStatusVoid
		return true
	}
	// Phần nợ được xóa ghi riêng (write_off_amount), KHÔNG cộng vào giảm giá.
	inv.WriteOffAmount = roundMoneyVND(inv.WriteOffAmount + due)
	inv.Total = roundMoneyVND(inv.PaidAmount)
	if inv.RefundedAmount >= inv.PaidAmount {
		inv.Status = entity.InvoiceStatusRefunded
	} else {
		inv.Status = entity.InvoiceStatusPaid
	}
	return true
}

func (s *invoiceService) WriteOffRemaining(ctx context.Context, invoiceID uint, reason string) (*entity.Invoice, error) {
	reason = invoiceTruncate(strings.TrimSpace(reason), 255)
	var out *entity.Invoice
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		inv, err := s.lock(ctx, invoiceID)
		if err != nil {
			return err
		}
		before := map[string]any{"total": inv.Total, "write_off_amount": inv.WriteOffAmount, "status": inv.Status}
		if !invoiceApplyWriteOff(inv) {
			out = inv
			return nil
		}
		fields := map[string]any{"status": inv.Status}
		if inv.Status == entity.InvoiceStatusVoid {
			now := s.now()
			inv.VoidedAt, inv.VoidedBy, inv.VoidReason = &now, invoiceActorID(ctx), reason
			fields["voided_at"], fields["voided_by"], fields["void_reason"] = now, inv.VoidedBy, reason
			if inv.PromotionID != nil {
				if err := s.repo.ReleasePromotion(ctx, *inv.PromotionID); err != nil {
					return err
				}
			}
		} else {
			if reason == "" {
				reason = "Xóa nợ"
			}
			inv.WriteOffReason = reason
			fields["total"], fields["write_off_amount"], fields["write_off_reason"] = inv.Total, inv.WriteOffAmount, reason
		}
		if err := s.repo.UpdateFields(ctx, inv.ID, fields); err != nil {
			return err
		}
		auditCatalog(ctx, s.audit, AuditUpdate, auditEntityInvoice, inv.ID, before,
			map[string]any{"total": inv.Total, "write_off_amount": inv.WriteOffAmount, "status": inv.Status, "write_off_reason": reason})
		// Hóa đơn không còn nợ do xóa nợ/hủy → các kỳ trả góp chưa thu bị hủy.
		if err := s.syncInstallments(ctx, inv, installmentCancelled); err != nil {
			return err
		}
		out = inv
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *invoiceService) Void(ctx context.Context, invoiceID uint, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return rescode.New(rescode.BadRequest, "Vui lòng nhập lý do hủy hóa đơn")
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		inv, err := s.lock(ctx, invoiceID)
		if err != nil {
			return err
		}
		if inv.Status == entity.InvoiceStatusVoid {
			return rescode.New(rescode.Conflict, "Hóa đơn đã bị hủy trước đó")
		}
		if inv.Status != entity.InvoiceStatusUnpaid || inv.PaidAmount > 0 {
			return rescode.New(rescode.BadRequest, "Hóa đơn đã thu tiền nên không thể hủy, hãy dùng chức năng hoàn tiền")
		}
		now := s.now()
		inv.Status, inv.VoidedAt, inv.VoidedBy, inv.VoidReason = entity.InvoiceStatusVoid, &now, invoiceActorID(ctx), invoiceTruncate(reason, 255)
		if err := s.repo.UpdateFields(ctx, inv.ID, map[string]any{
			"status": inv.Status, "voided_at": now, "voided_by": inv.VoidedBy, "void_reason": inv.VoidReason,
		}); err != nil {
			return err
		}
		if inv.PromotionID != nil {
			if err := s.repo.ReleasePromotion(ctx, *inv.PromotionID); err != nil {
				return err
			}
		}
		auditCatalog(ctx, s.audit, AuditVoid, auditEntityInvoice, inv.ID, map[string]any{"status": entity.InvoiceStatusUnpaid},
			map[string]any{"status": inv.Status, "reason": inv.VoidReason})
		if err := s.syncInstallments(ctx, inv, installmentCancelled); err != nil {
			return err
		}
		items, err := s.repo.Items(ctx, inv.ID)
		if err != nil {
			return err
		}
		inv.Items = items
		return s.runHandlers(ctx, s.voidHandlers, inv, items)
	})
}

// invoiceApplyRefund cập nhật số đã hoàn và trạng thái (hàm thuần).
func invoiceApplyRefund(inv *entity.Invoice, amount float64) error {
	switch inv.Status {
	case entity.InvoiceStatusVoid:
		return rescode.New(rescode.BadRequest, "Hóa đơn đã bị hủy")
	case entity.InvoiceStatusRefunded:
		return rescode.New(rescode.BadRequest, "Hóa đơn đã được hoàn tiền toàn bộ")
	case entity.InvoiceStatusUnpaid:
		return rescode.New(rescode.BadRequest, "Hóa đơn chưa thu tiền nên không có gì để hoàn")
	}
	refundable := roundMoneyVND(inv.PaidAmount - inv.RefundedAmount)
	if amount > refundable {
		return rescode.New(rescode.BadRequest, "Số tiền hoàn vượt quá số tiền có thể hoàn ("+formatMoneyVND(refundable)+")")
	}
	inv.RefundedAmount = roundMoneyVND(inv.RefundedAmount + amount)
	if inv.RefundedAmount >= inv.PaidAmount {
		inv.Status = entity.InvoiceStatusRefunded
	}
	return nil
}

func (s *invoiceService) Refund(ctx context.Context, invoiceID uint, in RefundInput) (*entity.Invoice, error) {
	inv, _, err := s.refund(ctx, invoiceID, in, nil, "")
	return inv, err
}

// refund là logic hoàn tiền chuẩn của hóa đơn (khóa hóa đơn, kiểm tra số có thể hoàn, cập nhật trạng thái,
// ghi phiếu hoàn, audit, đồng bộ trả góp). onlineTxnID/gatewayRef: phiếu hoàn của giao dịch VNPay (tùy chọn).
func (s *invoiceService) refund(ctx context.Context, invoiceID uint, in RefundInput, onlineTxnID *uint, gatewayRef string) (*entity.Invoice, *entity.Refund, error) {
	amount := roundMoneyVND(in.Amount)
	if amount <= 0 {
		return nil, nil, rescode.New(rescode.BadRequest, "Số tiền hoàn phải lớn hơn 0")
	}
	if err := invoiceValidMethod(in.Method); err != nil {
		return nil, nil, err
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, nil, rescode.New(rescode.BadRequest, "Vui lòng nhập lý do hoàn tiền")
	}
	var out *entity.Invoice
	var outRefund *entity.Refund
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		inv, err := s.lock(ctx, invoiceID)
		if err != nil {
			return err
		}
		before := *inv
		if err := invoiceApplyRefund(inv, amount); err != nil {
			return err
		}
		rf := &entity.Refund{InvoiceID: inv.ID, Amount: amount, Method: in.Method, Reason: invoiceTruncate(reason, 255), RefundedBy: invoiceActorID(ctx),
			OnlineTxnID: onlineTxnID, GatewayRef: invoiceTruncate(gatewayRef, 100), Status: "SUCCESS"}
		if err := s.repo.CreateRefund(ctx, rf); err != nil {
			return err
		}
		if err := s.repo.UpdateFields(ctx, inv.ID, map[string]any{"refunded_amount": inv.RefundedAmount, "status": inv.Status}); err != nil {
			return err
		}
		auditCatalog(ctx, s.audit, AuditRefund, auditEntityInvoice, inv.ID,
			map[string]any{"refunded_amount": before.RefundedAmount, "status": before.Status},
			map[string]any{"refund_id": rf.ID, "amount": amount, "method": rf.Method, "reason": rf.Reason, "refunded_amount": inv.RefundedAmount, "status": inv.Status, "online_txn_id": onlineTxnID})
		if err := s.syncInstallments(ctx, inv, installmentCancelled); err != nil {
			return err
		}
		out, outRefund = inv, rf
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out, outRefund, nil
}

func (s *invoiceService) LockInvoice(ctx context.Context, id uint) (*entity.Invoice, error) {
	return s.lock(ctx, id)
}

func (s *invoiceService) FindInvoice(ctx context.Context, id uint) (*entity.Invoice, error) {
	inv, err := s.repo.FindByID(ctx, id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvoiceNotFound
		}
		return nil, err
	}
	return inv, nil
}

// ---- Truy vấn quản trị ----------------------------------------------------------

func (s *invoiceService) Get(ctx context.Context, id uint) (*v1.InvoiceDetail, error) {
	row, err := s.repo.FindRow(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvoiceNotFound
		}
		return nil, err
	}
	if !reqctx.CanAccessBranch(ctx, row.BranchID) {
		return nil, errInvoiceOutOfScope
	}
	return s.detail(ctx, row)
}

// detail dựng chi tiết hóa đơn (không kiểm tra phạm vi).
func (s *invoiceService) detail(ctx context.Context, row *repository.InvoiceRow) (*v1.InvoiceDetail, error) {
	items, err := s.repo.Items(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	pays, err := s.repo.Payments(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	refunds, err := s.repo.Refunds(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &v1.InvoiceDetail{
		Invoice:      *toInvoiceRes(*row),
		VoidedByName: row.VoidedByName,
		Items: dto.Map(items, func(it entity.InvoiceItem) *v1.InvoiceItem {
			return &v1.InvoiceItem{ID: it.ID, ItemType: it.ItemType, RefID: it.RefID, Description: it.Description,
				Quantity: it.Quantity, UnitPrice: it.UnitPrice, Discount: it.Discount, Amount: it.Amount}
		}),
		Payments: dto.Map(pays, func(p repository.PaymentRow) *v1.Payment { return toPaymentRes(p) }),
		Refunds: dto.Map(refunds, func(r repository.RefundRow) *v1.Refund {
			return &v1.Refund{ID: r.ID, InvoiceID: r.InvoiceID, Amount: r.Amount, Method: r.Method, Reason: r.Reason,
				RefundedBy: r.RefundedBy, RefundedByName: r.RefundedByName, CreatedAt: r.CreatedAt}
		}),
	}, nil
}

func (s *invoiceService) Detail(ctx context.Context, id uint) (*v1.InvoiceDetail, error) {
	row, err := s.repo.FindRow(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvoiceNotFound
		}
		return nil, err
	}
	return s.detail(ctx, row)
}

func (s *invoiceService) List(ctx context.Context, req *v1.ListReq) (*v1.ListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	f := repository.InvoiceListFilter{BranchIDs: branchIDs, MemberID: req.MemberID, Status: req.Status, Source: req.Source}
	var err error
	if f.From, f.To, err = subInstantRange(req.From, req.To); err != nil {
		return nil, err
	}
	return s.list(ctx, req.PageReq, f)
}

func (s *invoiceService) ListByMember(ctx context.Context, memberID uint, page dto.PageReq) (*v1.ListRes, error) {
	return s.list(ctx, page, repository.InvoiceListFilter{MemberID: memberID})
}

func (s *invoiceService) list(ctx context.Context, page dto.PageReq, f repository.InvoiceListFilter) (*v1.ListRes, error) {
	q := page.ToQuery(invoiceOrderable, "id")
	q.OrderBy = "i." + q.OrderBy
	rows, total, err := s.repo.List(ctx, q, f)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, func(r repository.InvoiceRow) *v1.Invoice { return toInvoiceRes(r) }), total, q), nil
}

func (s *invoiceService) ListPayments(ctx context.Context, req *v1.PaymentListReq) (*v1.PaymentListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	f := repository.PaymentLedgerFilter{BranchIDs: branchIDs, Method: req.Method, ReceivedBy: req.ReceivedBy}
	var err error
	if f.From, f.To, err = subInstantRange(req.From, req.To); err != nil {
		return nil, err
	}
	q := req.PageReq.ToQuery(paymentOrderable, "id")
	q.OrderBy = "pm." + q.OrderBy
	rows, total, err := s.repo.ListPayments(ctx, q, f)
	if err != nil {
		return nil, err
	}
	sums, err := s.repo.SumPayments(ctx, f)
	if err != nil {
		return nil, err
	}
	res := &v1.PaymentListRes{
		PageRes: *dto.NewPageRes(dto.Map(rows, func(p repository.PaymentRow) *v1.PaymentItem {
			return &v1.PaymentItem{Payment: *toPaymentRes(p), InvoiceNo: p.InvoiceNo, BranchID: p.BranchID, BranchName: p.BranchName,
				MemberID: p.MemberID, MemberCode: p.MemberCode, MemberName: p.MemberName}
		}), total, q),
	}
	res.Summary = invoiceSummarize(sums)
	return res, nil
}

// invoiceSummarize gộp tổng thu theo phương thức (luôn đủ CASH, TRANSFER, CARD).
func invoiceSummarize(sums []repository.PaymentMethodTotal) v1.PaymentSummary {
	out := v1.PaymentSummary{ByMethod: make([]*v1.MethodTotal, 0, len(invoicePayMethods))}
	for _, m := range invoicePayMethods {
		mt := &v1.MethodTotal{Method: m}
		for _, s := range sums {
			if s.Method == m {
				mt.Amount, mt.Count = roundMoneyVND(s.Amount), s.Count
			}
		}
		out.TotalAmount += mt.Amount
		out.Count += mt.Count
		out.ByMethod = append(out.ByMethod, mt)
	}
	return out
}

func (s *invoiceService) Print(ctx context.Context, id uint) (*v1.PrintRes, error) {
	detail, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	branch, err := s.repo.FindBranch(ctx, detail.BranchID)
	if err != nil {
		return nil, err
	}
	res := &v1.PrintRes{
		Invoice:      detail,
		Branch:       v1.PrintBranch{ID: branch.ID, Code: branch.Code, Name: branch.Name, Address: branch.Address, Phone: branch.Phone, Email: branch.Email},
		Member:       detail.Member,
		TotalInWords: VNDInWords(detail.Total),
		DueInWords:   VNDInWords(detail.Due),
		PrintedAt:    s.now(),
	}
	if uid := reqctx.UserID(ctx); uid != nil {
		res.PrintedByName, _ = s.repo.UserName(ctx, *uid)
	}
	return res, nil
}

// ---- Đồng bộ kỳ trả góp ----------------------------------------------------------

// Trạng thái kỳ trả góp đóng do hóa đơn thay đổi (CHECK của migration 000037).
const (
	installmentSettled   = "SETTLED"   // hóa đơn đã thu đủ (thu trực tiếp) → kỳ chưa thu được tất toán
	installmentCancelled = "CANCELLED" // hóa đơn hủy / hoàn toàn bộ / xóa nợ → kỳ chưa thu bị hủy
)

// installmentIsOpen: kỳ còn phải thu.
func installmentIsOpen(status string) bool {
	return status == entity.InstallmentPending || status == entity.InstallmentOverdue
}

// invoiceResyncInstallments (hàm thuần) đưa các kỳ trả góp chưa thu về khớp với hóa đơn, trả các kỳ đã đổi:
//   - hóa đơn VOID/REFUNDED → kỳ chưa thu: CANCELLED;
//   - hóa đơn hết nợ (PAID) → kỳ chưa thu: closeStatus (SETTLED khi thu đủ, CANCELLED khi xóa nợ);
//   - còn nợ D nhưng tổng phần chưa thu của các kỳ ≠ D (vd thu trực tiếp một phần) → chia lại D cho các kỳ chưa thu
//     (kỳ cuối nhận phần lẻ; kỳ được chia 0đ → SETTLED).
func invoiceResyncInstallments(scheds []entity.InstallmentSchedule, inv *entity.Invoice, closeStatus string, now time.Time) []entity.InstallmentSchedule {
	var open []int
	var openSum float64
	for i := range scheds {
		if installmentIsOpen(scheds[i].Status) {
			open = append(open, i)
			openSum += roundMoneyVND(scheds[i].Amount - scheds[i].PaidAmount)
		}
	}
	if len(open) == 0 {
		return nil
	}
	var changed []entity.InstallmentSchedule
	closeAll := func(status string) {
		for _, i := range open {
			sc := scheds[i]
			sc.Status, sc.UpdatedAt = status, now
			changed = append(changed, sc)
		}
	}
	switch inv.Status {
	case entity.InvoiceStatusVoid, entity.InvoiceStatusRefunded:
		closeAll(installmentCancelled)
		return changed
	}
	due := roundMoneyVND(inv.Total - inv.PaidAmount)
	if due <= 0 {
		closeAll(closeStatus)
		return changed
	}
	if roundMoneyVND(openSum) == due {
		return nil
	}
	shares := installmentSplitAmounts(due, len(open))
	for k, i := range open {
		sc := scheds[i]
		sc.Amount = roundMoneyVND(sc.PaidAmount + shares[k])
		if shares[k] <= 0 {
			sc.Status = installmentSettled
		}
		sc.UpdatedAt = now
		changed = append(changed, sc)
	}
	return changed
}

// syncInstallments khóa và cập nhật các kỳ trả góp của hóa đơn theo trạng thái mới (trong transaction của thao tác).
func (s *invoiceService) syncInstallments(ctx context.Context, inv *entity.Invoice, closeStatus string) error {
	scheds, err := s.repo.InstallmentSchedules(ctx, inv.ID, true)
	if err != nil || len(scheds) == 0 {
		return err
	}
	for _, sc := range invoiceResyncInstallments(scheds, inv, closeStatus, s.now()) {
		if err := s.repo.SaveInstallmentSchedule(ctx, &sc); err != nil {
			return err
		}
	}
	return nil
}

// ---- Mapping --------------------------------------------------------------------

func toInvoiceRes(r repository.InvoiceRow) *v1.Invoice {
	out := &v1.Invoice{
		ID: r.ID, InvoiceNo: r.InvoiceNo, MemberID: r.MemberID, BranchID: r.BranchID, BranchName: r.BranchName,
		Subtotal: r.Subtotal, Discount: r.Discount, Total: r.Total, PaidAmount: r.PaidAmount, RefundedAmount: r.RefundedAmount,
		Due: roundMoneyVND(r.Total - r.PaidAmount), Status: r.Status, Source: r.Source,
		PromotionID: r.PromotionID, PromotionCode: r.PromotionCode, Note: r.Note,
		IssuedBy: r.IssuedBy, IssuedByName: r.IssuedByName, IssuedAt: r.IssuedAt,
		VoidedAt: r.VoidedAt, VoidReason: r.VoidReason, CreatedAt: r.CreatedAt,
		WriteOffAmount: r.WriteOffAmount, WriteOffReason: r.WriteOffReason,
	}
	if r.Status == entity.InvoiceStatusVoid {
		out.Due = 0
	}
	if r.MemberID != nil {
		out.Member = &v1.MemberRef{ID: *r.MemberID, MemberCode: r.MemberCode, FullName: r.MemberName, Phone: r.MemberPhone}
	}
	return out
}

func toPaymentRes(p repository.PaymentRow) *v1.Payment {
	return &v1.Payment{ID: p.ID, InvoiceID: p.InvoiceID, Amount: p.Amount, Method: p.Method, Reference: p.Reference,
		PaidAt: p.PaidAt, ReceivedBy: p.ReceivedBy, ReceivedByName: p.ReceivedByName, Note: p.Note}
}

func invoiceTruncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// invoiceDescDate định dạng ngày dd/mm/yyyy cho mô tả dòng hóa đơn.
func invoiceDescDate(d time.Time) string {
	return fmt.Sprintf("%02d/%02d/%04d", d.Day(), int(d.Month()), d.Year())
}
