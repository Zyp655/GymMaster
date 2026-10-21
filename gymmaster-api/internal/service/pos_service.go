package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	invoicev1 "gymmaster-api/api/invoice/v1"
	v1 "gymmaster-api/api/pos/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

const (
	auditEntityPosOrder = "pos_order"
	AuditPosSale        = "SALE"
	AuditPosReturn      = "RETURN"

	posMaxProducts = 500
)

type (
	// PosService: bán hàng tại quầy – sản phẩm kèm tồn, lập đơn (hóa đơn + trừ kho + thu đủ tiền),
	// trả hàng (hoàn kho + hoàn tiền), danh sách đơn, tổng kết ngày.
	PosService interface {
		Products(ctx context.Context, req *v1.ProductsReq) ([]*v1.PosProduct, error)
		ByBarcode(ctx context.Context, code string, branchID uint) (*v1.PosProduct, error)
		CreateOrder(ctx context.Context, req *v1.CreateOrderReq) (*v1.OrderRes, error)
		ListOrders(ctx context.Context, req *v1.ListOrdersReq) (*v1.ListOrdersRes, error)
		GetOrder(ctx context.Context, invoiceID uint) (*v1.GetOrderRes, error)
		Return(ctx context.Context, req *v1.ReturnReq) (*v1.ReturnRes, error)
		Summary(ctx context.Context, req *v1.SummaryReq) (*v1.SummaryRes, error)
	}

	posService struct {
		repo     repository.PosRepository
		products repository.ProductRepository
		stock    *stockService
		invoices InvoiceService
		tx       postgres.TxManager
		audit    AuditService
		now      func() time.Time
	}

	// posLine là một dòng đơn đã gộp theo sản phẩm.
	posLine struct {
		ProductID uint
		Quantity  int
	}

	// posReturnState: tình trạng trả hàng của một sản phẩm trong hóa đơn.
	posReturnState struct {
		ProductID   uint
		Description string
		Sold        int     // số lượng đã xuất bán theo hóa đơn
		Returned    int     // đã nhập lại (trả hàng / hủy)
		Amount      float64 // thành tiền (sau giảm giá) của sản phẩm trên hóa đơn
		SaleCost    float64
	}
)

// NewPosService tạo service POS và đăng ký VoidHandler "PRODUCT" với InvoiceService (hoàn kho khi hủy hóa đơn).
// Gọi MỘT lần khi khởi tạo module.
func NewPosService(repo repository.PosRepository, products repository.ProductRepository, stockRepo repository.StockRepository,
	invoices InvoiceService, tx postgres.TxManager, audit AuditService, notifier Notifier) PosService {
	s := &posService{
		repo: repo, products: products, invoices: invoices, tx: tx, audit: audit, now: time.Now,
		stock: newStockService(stockRepo, products, tx, audit, notifier),
	}
	invoices.RegisterVoidHandler(entity.InvoiceItemProduct, s.onInvoiceVoid)
	return s
}

// ---- Hàm thuần ------------------------------------------------------------------

// posUnitPrice: hội viên (đang ACTIVE) mua theo member_price nếu sản phẩm có, ngược lại theo sell_price.
func posUnitPrice(p *entity.Product, memberActive bool) float64 {
	if memberActive && p.MemberPrice != nil {
		return *p.MemberPrice
	}
	return p.SellPrice
}

// posPlanPayments kiểm tra khách trả đủ và tính các lần thu thực ghi nhận.
//   - Σ amount < total → lỗi (POS bắt buộc thanh toán đủ).
//   - Phần thừa (change) chỉ được phép từ tiền mặt: change ≤ Σ tiền mặt. Các khoản tiền mặt được giảm
//     (từ khoản cuối lên) đúng bằng change → Σ ghi nhận = total.
//
// Trả các lần thu (bỏ khoản 0đ), tiền khách đưa và tiền thừa.
func posPlanPayments(total float64, pays []*v1.OrderPayment) ([]PaymentInput, float64, float64, error) {
	total = roundMoneyVND(total)
	var received, cash float64
	for _, p := range pays {
		if p == nil {
			continue
		}
		amt := roundMoneyVND(p.Amount)
		if amt <= 0 {
			return nil, 0, 0, rescode.New(rescode.BadRequest, "Số tiền thanh toán phải lớn hơn 0")
		}
		if err := invoiceValidMethod(p.Method); err != nil {
			return nil, 0, 0, err
		}
		received += amt
		if p.Method == entity.PaymentCash {
			cash += amt
		}
	}
	if received < total {
		return nil, 0, 0, rescode.New(rescode.BadRequest, fmt.Sprintf("Khách thanh toán chưa đủ: tổng hóa đơn %s, đã nhận %s (còn thiếu %s)",
			formatMoneyVND(total), formatMoneyVND(received), formatMoneyVND(total-received)))
	}
	change := received - total
	if change > cash {
		return nil, 0, 0, rescode.New(rescode.BadRequest,
			"Chỉ tiền mặt mới được đưa thừa: số tiền chuyển khoản/thẻ vượt quá tổng hóa đơn")
	}
	amounts := make([]float64, len(pays))
	for i, p := range pays {
		if p != nil {
			amounts[i] = roundMoneyVND(p.Amount)
		}
	}
	rest := change
	for i := len(pays) - 1; i >= 0 && rest > 0; i-- {
		if pays[i] == nil || pays[i].Method != entity.PaymentCash {
			continue
		}
		cut := math.Min(rest, amounts[i])
		amounts[i] -= cut
		rest -= cut
	}
	out := make([]PaymentInput, 0, len(pays))
	for i, p := range pays {
		if p == nil || amounts[i] <= 0 {
			continue
		}
		out = append(out, PaymentInput{Amount: amounts[i], Method: p.Method, Reference: strings.TrimSpace(p.Reference)})
	}
	return out, received, change, nil
}

// posReturnRefund: số tiền hoàn khi trả thêm qty đơn vị của một sản phẩm đã bán sold đơn vị với thành tiền amount,
// trong đó đã trả returned đơn vị. Tính lũy kế để trả hết thì tổng hoàn đúng bằng amount (không lệch do làm tròn):
//
//	refund = round(amount × (returned + qty) / sold) − round(amount × returned / sold)
func posReturnRefund(amount float64, sold, returned, qty int) float64 {
	if sold <= 0 || qty <= 0 {
		return 0
	}
	cum := func(n int) float64 { return roundMoneyVND(amount * float64(n) / float64(sold)) }
	return cum(returned+qty) - cum(returned)
}

// posMergeLines gộp các dòng trùng sản phẩm (cộng số lượng).
func posMergeLines(items []*v1.OrderItem) ([]posLine, error) {
	if len(items) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Đơn hàng phải có ít nhất một sản phẩm")
	}
	ids := make([]uint, 0, len(items))
	qtys := make([]int, 0, len(items))
	for _, it := range items {
		if it == nil || it.ProductID == 0 {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn sản phẩm")
		}
		if it.Quantity <= 0 {
			return nil, rescode.New(rescode.BadRequest, "Số lượng phải lớn hơn 0")
		}
		ids, qtys = append(ids, it.ProductID), append(qtys, it.Quantity)
	}
	order, sum := stockMergeQty(ids, qtys)
	out := make([]posLine, 0, len(order))
	for _, id := range order {
		out = append(out, posLine{ProductID: id, Quantity: sum[id]})
	}
	return out, nil
}

// ---- Sản phẩm cho màn hình POS ---------------------------------------------------

func toPosProduct(r repository.ProductRow) *v1.PosProduct {
	return &v1.PosProduct{ID: r.ID, SKU: r.SKU, Barcode: r.Barcode, Name: r.Name, CategoryID: r.CategoryID,
		CategoryName: r.CategoryName, Unit: r.Unit, SellPrice: r.SellPrice, MemberPrice: r.MemberPrice,
		ImageURL: r.ImageURL, Stock: r.Stock, MinStock: r.MinStock, LowStock: r.LowStock}
}

func (s *posService) checkBranchScope(ctx context.Context, branchID uint) error {
	if branchID == 0 {
		return rescode.New(rescode.BadRequest, "Vui lòng chọn chi nhánh")
	}
	if !reqctx.CanAccessBranch(ctx, branchID) {
		return rescode.New(rescode.PermissionDenied, "Bạn không có quyền bán hàng tại chi nhánh này")
	}
	return nil
}

func (s *posService) Products(ctx context.Context, req *v1.ProductsReq) ([]*v1.PosProduct, error) {
	if err := s.checkBranchScope(ctx, req.BranchID); err != nil {
		return nil, err
	}
	q := dto.Query{Page: 1, Limit: posMaxProducts, Keyword: strings.TrimSpace(req.Keyword), OrderBy: "p.name", OrderDir: "ASC"}
	rows, _, err := s.products.ListProducts(ctx, q, repository.ProductFilter{
		CategoryID: req.CategoryID, Status: entity.ProductStatusActive, BranchID: req.BranchID, InStock: req.InStock,
	})
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toPosProduct), nil
}

func (s *posService) ByBarcode(ctx context.Context, code string, branchID uint) (*v1.PosProduct, error) {
	if err := s.checkBranchScope(ctx, branchID); err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	p, err := s.products.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy sản phẩm có mã "+code)
		}
		return nil, err
	}
	if p.Status != entity.ProductStatusActive {
		return nil, rescode.New(rescode.BadRequest, "Sản phẩm \""+p.Name+"\" đã ngừng kinh doanh")
	}
	row, err := s.products.FindProductRow(ctx, p.ID, branchID)
	if err != nil {
		return nil, err
	}
	return toPosProduct(*row), nil
}

// ---- Bán hàng --------------------------------------------------------------------

func (s *posService) CreateOrder(ctx context.Context, req *v1.CreateOrderReq) (*v1.OrderRes, error) {
	if err := s.checkBranchScope(ctx, req.BranchID); err != nil {
		return nil, err
	}
	lines, err := posMergeLines(req.Items)
	if err != nil {
		return nil, err
	}
	if _, err := s.stock.checkBranch(ctx, req.BranchID, true, "chi nhánh"); err != nil {
		return nil, err
	}
	var (
		invoiceID        uint
		received, change float64
	)
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		memberActive := false
		if req.MemberID != nil {
			m, err := s.repo.FindMember(ctx, *req.MemberID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return rescode.New(rescode.NotFound, "Không tìm thấy hội viên")
				}
				return err
			}
			memberActive = m.Status == entity.MemberStatusActive
		}
		products := map[uint]*entity.Product{}
		keys := make([]stockKey, 0, len(lines))
		for _, l := range lines {
			p, err := s.stock.findProduct(ctx, l.ProductID, false)
			if err != nil {
				return err
			}
			if p.Status != entity.ProductStatusActive {
				return rescode.New(rescode.BadRequest, "Sản phẩm \""+p.Name+"\" đã ngừng kinh doanh")
			}
			products[p.ID] = p
			keys = append(keys, stockKey{p.ID, req.BranchID})
		}
		// Khóa tồn theo thứ tự (product_id, branch_id) và kiểm tra đủ hàng TRƯỚC khi lập hóa đơn.
		levels, err := s.stock.lockLevels(ctx, keys)
		if err != nil {
			return err
		}
		drafts := make([]InvoiceItemDraft, 0, len(lines))
		for _, l := range lines {
			p := products[l.ProductID]
			if have := levels[stockKey{p.ID, req.BranchID}]; have < l.Quantity {
				return stockShortage(p, have, l.Quantity)
			}
			id := p.ID
			drafts = append(drafts, InvoiceItemDraft{ItemType: entity.InvoiceItemProduct, RefID: &id,
				Description: p.Name, Quantity: l.Quantity, UnitPrice: posUnitPrice(p, memberActive)})
		}
		draft := InvoiceDraft{MemberID: req.MemberID, BranchID: req.BranchID, Source: entity.InvoiceSourcePOS, Items: drafts,
			PromotionCode: strings.TrimSpace(req.PromotionCode), AppliesTo: entity.AppliesProduct, Note: req.Note}
		// Kiểm tra thanh toán đủ trên bản xem trước để không lập hóa đơn (tốn số HĐ) khi khách trả thiếu.
		preview, err := s.invoices.Preview(ctx, draft)
		if err != nil {
			return err
		}
		if _, _, _, err := posPlanPayments(preview.Total, req.Payments); err != nil {
			return err
		}
		inv, err := s.invoices.CreateInvoice(ctx, draft)
		if err != nil {
			return err
		}
		pays, rcv, chg, err := posPlanPayments(inv.Total, req.Payments)
		if err != nil {
			return err
		}
		invID := inv.ID
		for _, l := range lines {
			if _, err := s.stock.applyStockChange(ctx, l.ProductID, req.BranchID, -l.Quantity, entity.StockSale, stockRef{
				RefType: entity.StockRefInvoice, RefID: &invID, VoucherNo: inv.InvoiceNo, Note: "Bán hàng " + inv.InvoiceNo,
			}); err != nil {
				return err
			}
		}
		for _, p := range pays {
			if _, err := s.invoices.AddPayment(ctx, inv.ID, p); err != nil {
				return err
			}
		}
		auditCatalog(ctx, s.audit, AuditPosSale, auditEntityPosOrder, inv.ID, nil, map[string]any{
			"invoice_no": inv.InvoiceNo, "branch_id": req.BranchID, "member_id": req.MemberID, "member_price": memberActive,
			"items": lines, "total": inv.Total, "received": rcv, "change": chg,
		})
		invoiceID, received, change = inv.ID, rcv, chg
		return nil
	})
	if err != nil {
		return nil, err
	}
	pr, err := s.invoices.Print(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	return &v1.OrderRes{Invoice: pr.Invoice, ReceivedAmount: received, ChangeAmount: change, Print: pr}, nil
}

func (s *posService) ListOrders(ctx context.Context, req *v1.ListOrdersReq) (*v1.ListOrdersRes, error) {
	return s.invoices.List(ctx, &invoicev1.ListReq{PageReq: req.PageReq, BranchID: req.BranchID, MemberID: req.MemberID,
		Status: req.Status, Source: entity.InvoiceSourcePOS, From: req.From, To: req.To})
}

// returnStates tính tình trạng trả hàng theo sản phẩm của hóa đơn (thứ tự theo dòng hóa đơn).
func (s *posService) returnStates(ctx context.Context, invoiceID uint) ([]*posReturnState, error) {
	items, err := s.repo.InvoiceItems(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	moves, err := s.repo.InvoiceMoves(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	byID := map[uint]*posReturnState{}
	var out []*posReturnState
	for _, it := range items {
		if it.ItemType != entity.InvoiceItemProduct || it.RefID == nil {
			continue
		}
		st, ok := byID[*it.RefID]
		if !ok {
			st = &posReturnState{ProductID: *it.RefID, Description: it.Description}
			byID[*it.RefID] = st
			out = append(out, st)
		}
		st.Sold += it.Quantity
		st.Amount += it.Amount
	}
	for _, st := range out {
		var mv repository.PosInvoiceMove
		for _, m := range moves {
			if m.ProductID == st.ProductID {
				mv = m
			}
		}
		// Chỉ trả được phần thực sự đã xuất kho theo hóa đơn.
		st.Sold = min(st.Sold, mv.SoldOut)
		st.Returned, st.SaleCost = mv.Returned, mv.SaleCost
	}
	return out, nil
}

func toReturnableLines(states []*posReturnState) []*v1.ReturnableLine {
	return dto.Map(states, func(st *posReturnState) *v1.ReturnableLine {
		return &v1.ReturnableLine{ProductID: st.ProductID, Description: st.Description, SoldQuantity: st.Sold,
			ReturnedQty: st.Returned, ReturnableQty: max(0, st.Sold-st.Returned), Amount: st.Amount,
			RefundedValue: posReturnRefund(st.Amount, st.Sold, 0, min(st.Returned, st.Sold))}
	})
}

var errPosOrderNotFound = rescode.New(rescode.NotFound, "Không tìm thấy đơn bán hàng POS")

func (s *posService) GetOrder(ctx context.Context, invoiceID uint) (*v1.GetOrderRes, error) {
	detail, err := s.invoices.Get(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if detail.Source != entity.InvoiceSourcePOS {
		return nil, errPosOrderNotFound
	}
	states, err := s.returnStates(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	return &v1.GetOrderRes{Invoice: detail, Lines: toReturnableLines(states)}, nil
}

// posValidateReturn kiểm tra số lượng trả không vượt (đã bán − đã trả) và tính số tiền hoàn (hàm thuần).
func posValidateReturn(states []*posReturnState, items []posLine) (float64, error) {
	byID := map[uint]*posReturnState{}
	for _, st := range states {
		byID[st.ProductID] = st
	}
	var refund float64
	for _, l := range items {
		st, ok := byID[l.ProductID]
		if !ok || st.Sold == 0 {
			return 0, rescode.New(rescode.BadRequest, fmt.Sprintf("Sản phẩm #%d không có trong hóa đơn", l.ProductID))
		}
		left := st.Sold - st.Returned
		if l.Quantity > left {
			return 0, rescode.New(rescode.BadRequest, fmt.Sprintf("Sản phẩm \"%s\" chỉ còn có thể trả %d (đã bán %d, đã trả %d)",
				st.Description, max(0, left), st.Sold, st.Returned))
		}
		refund += posReturnRefund(st.Amount, st.Sold, st.Returned, l.Quantity)
	}
	return roundMoneyVND(refund), nil
}

func (s *posService) Return(ctx context.Context, req *v1.ReturnReq) (*v1.ReturnRes, error) {
	items := make([]*v1.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		if it == nil {
			continue
		}
		items = append(items, &v1.OrderItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	lines, err := posMergeLines(items)
	if err != nil {
		return nil, err
	}
	if err := invoiceValidMethod(req.RefundMethod); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập lý do trả hàng")
	}
	var (
		voucherNo string
		refund    float64
	)
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Khóa hóa đơn: các lần trả hàng của cùng hóa đơn chạy tuần tự → không trả quá số đã bán.
		inv, err := s.repo.LockInvoice(ctx, req.InvoiceID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errPosOrderNotFound
			}
			return err
		}
		if !reqctx.CanAccessBranch(ctx, inv.BranchID) {
			return errInvoiceOutOfScope
		}
		if inv.Source != entity.InvoiceSourcePOS {
			return rescode.New(rescode.BadRequest, "Chỉ trả hàng cho hóa đơn bán tại quầy (POS)")
		}
		switch inv.Status {
		case entity.InvoiceStatusVoid:
			return rescode.New(rescode.BadRequest, "Hóa đơn đã bị hủy")
		case entity.InvoiceStatusUnpaid, entity.InvoiceStatusPartial:
			return rescode.New(rescode.BadRequest, "Hóa đơn chưa thanh toán đủ, không thể trả hàng")
		}
		states, err := s.returnStates(ctx, inv.ID)
		if err != nil {
			return err
		}
		if refund, err = posValidateReturn(states, lines); err != nil {
			return err
		}
		if voucherNo, err = s.stock.repo.NextVoucherNo(ctx, s.stock.voucherPrefix(stockPrefixReturn)); err != nil {
			return err
		}
		keys := make([]stockKey, 0, len(lines))
		for _, l := range lines {
			keys = append(keys, stockKey{l.ProductID, inv.BranchID})
		}
		if _, err := s.stock.lockLevels(ctx, keys); err != nil {
			return err
		}
		invID := inv.ID
		for _, l := range lines {
			var cost *float64
			for _, st := range states {
				if st.ProductID == l.ProductID {
					c := st.SaleCost
					cost = &c
				}
			}
			if _, err := s.stock.applyStockChange(ctx, l.ProductID, inv.BranchID, l.Quantity, entity.StockSaleReturn, stockRef{
				UnitCost: cost, RefType: entity.StockRefInvoice, RefID: &invID, VoucherNo: voucherNo,
				Note: "Trả hàng HĐ " + inv.InvoiceNo + ": " + reason,
			}); err != nil {
				return err
			}
		}
		// Không hoàn quá số tiền còn có thể hoàn (hóa đơn có thể đã được hoàn tay một phần).
		refund = math.Min(refund, roundMoneyVND(inv.PaidAmount-inv.RefundedAmount))
		if refund > 0 {
			if _, err := s.invoices.Refund(ctx, inv.ID, RefundInput{Amount: refund, Method: req.RefundMethod,
				Reason: invoiceTruncate("Trả hàng "+voucherNo+": "+reason, 255)}); err != nil {
				return err
			}
		} else {
			refund = 0
		}
		auditCatalog(ctx, s.audit, AuditPosReturn, auditEntityPosOrder, inv.ID, nil, map[string]any{
			"invoice_no": inv.InvoiceNo, "voucher_no": voucherNo, "items": lines, "refund_amount": refund,
			"refund_method": req.RefundMethod, "reason": reason,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	order, err := s.GetOrder(ctx, req.InvoiceID)
	if err != nil {
		return nil, err
	}
	return &v1.ReturnRes{VoucherNo: voucherNo, RefundAmount: refund, Invoice: order.Invoice, Lines: order.Lines}, nil
}

// onInvoiceVoid (VoidHandler "PRODUCT"): hủy hóa đơn → nhập lại kho phần đã xuất mà chưa trả (SALE_RETURN).
// Idempotent: tính theo thẻ kho của hóa đơn nên gọi lại không nhập trùng.
func (s *posService) onInvoiceVoid(ctx context.Context, inv *entity.Invoice, it *entity.InvoiceItem) error {
	if it.RefID == nil {
		return nil
	}
	moves, err := s.repo.InvoiceMoves(ctx, inv.ID)
	if err != nil {
		return err
	}
	for _, m := range moves {
		if m.ProductID != *it.RefID {
			continue
		}
		remain := m.SoldOut - m.Returned
		if remain <= 0 {
			return nil
		}
		invID, cost := inv.ID, m.SaleCost
		_, err := s.stock.applyStockChange(ctx, m.ProductID, inv.BranchID, remain, entity.StockSaleReturn, stockRef{
			UnitCost: &cost, RefType: entity.StockRefInvoice, RefID: &invID, VoucherNo: inv.InvoiceNo,
			Note: "Hủy hóa đơn " + inv.InvoiceNo,
		})
		return err
	}
	return nil
}

// ---- Tổng kết ngày ---------------------------------------------------------------

func (s *posService) Summary(ctx context.Context, req *v1.SummaryReq) (*v1.SummaryRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	day := s.now().In(stockVNLoc)
	if strings.TrimSpace(req.Date) != "" {
		d, err := time.ParseInLocation(consts.DateLayout, strings.TrimSpace(req.Date), stockVNLoc)
		if err != nil {
			return nil, rescode.New(rescode.BadRequest, "Ngày phải có dạng YYYY-MM-DD")
		}
		day = d
	}
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, stockVNLoc)
	f := repository.PosSummaryFilter{BranchIDs: branchIDs, From: from, To: from.AddDate(0, 0, 1)}
	totals, err := s.repo.SalesTotals(ctx, f)
	if err != nil {
		return nil, err
	}
	itemsSold, err := s.repo.ItemsSold(ctx, f)
	if err != nil {
		return nil, err
	}
	refunds, err := s.repo.RefundTotal(ctx, f)
	if err != nil {
		return nil, err
	}
	cogs, err := s.repo.CostOfGoods(ctx, f)
	if err != nil {
		return nil, err
	}
	methods, err := s.repo.PaymentsByMethod(ctx, f)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopProducts(ctx, f, 5)
	if err != nil {
		return nil, err
	}
	res := &v1.SummaryRes{
		Date: from.Format(consts.DateLayout), BranchID: req.BranchID,
		OrderCount: totals.OrderCount, ItemsSold: itemsSold,
		GrossRevenue: roundMoneyVND(totals.GrossRevenue), Discount: roundMoneyVND(totals.Discount),
		RefundAmount: roundMoneyVND(refunds), CostOfGoods: roundMoneyVND(cogs),
		TopProducts: dto.Map(top, func(t repository.PosTopProduct) *v1.TopProduct {
			return &v1.TopProduct{ProductID: t.ProductID, SKU: t.SKU, Name: t.Name, Quantity: t.Quantity, Revenue: roundMoneyVND(t.Revenue)}
		}),
	}
	res.NetRevenue = res.GrossRevenue - res.RefundAmount
	res.GrossProfit = res.NetRevenue - res.CostOfGoods
	sum := invoiceSummarize(methods)
	res.ByMethod = dto.Map(sum.ByMethod, func(m *invoicev1.MethodTotal) *v1.MethodTotal {
		return &v1.MethodTotal{Method: m.Method, Amount: m.Amount, Count: m.Count}
	})
	return res, nil
}
