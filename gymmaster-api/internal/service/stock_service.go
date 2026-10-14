package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"

	v1 "gymmaster-api/api/inventory/v1"
	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/storage/postgres"
)

// Audit & mã phiếu kho.
const (
	auditEntityStockVoucher = "stock_voucher"
	AuditStockImport        = "IMPORT"
	AuditStockTransfer      = "TRANSFER"
	AuditStockAdjust        = "ADJUST"

	stockPrefixImport   = "NK" // nhập kho
	stockPrefixTransfer = "DC" // điều chuyển
	stockPrefixAdjust   = "KK" // kiểm kê
	stockPrefixReturn   = "TH" // trả hàng
)

type (
	// StockService: nhập / điều chuyển / kiểm kê, tồn kho, thẻ kho, phiếu kho, cảnh báo tồn thấp.
	// Mọi biến động tồn đi qua MỘT hàm nội bộ applyStockChange (cả bán hàng POS).
	StockService interface {
		Import(ctx context.Context, req *v1.ImportReq) (*v1.Voucher, error)
		Transfer(ctx context.Context, req *v1.TransferReq) (*v1.Voucher, error)
		Adjust(ctx context.Context, req *v1.AdjustReq) (*v1.Voucher, error)
		Stock(ctx context.Context, req *v1.StockReq) (*v1.StockRes, error)
		Transactions(ctx context.Context, req *v1.TransactionListReq) (*v1.TransactionListRes, error)
		Voucher(ctx context.Context, voucherNo string) (*v1.Voucher, error)
		// LowStock: API (kiểm tra phạm vi chi nhánh; branchID 0 = mọi chi nhánh trong phạm vi).
		LowStock(ctx context.Context, branchID uint) ([]*v1.LowStockItem, error)
		// ListLowStock dành cho cron: các dòng tồn (sản phẩm ACTIVE, chi nhánh ACTIVE) có quantity < min_stock.
		// branchID nil = mọi chi nhánh. KHÔNG kiểm tra phạm vi người dùng.
		ListLowStock(ctx context.Context, branchID *uint) ([]*v1.LowStockItem, error)
	}

	stockService struct {
		repo     repository.StockRepository
		products repository.ProductRepository
		tx       postgres.TxManager
		audit    AuditService
		notifier Notifier
		now      func() time.Time
	}

	// stockRef là thông tin tham chiếu ghi kèm dòng thẻ kho.
	stockRef struct {
		UnitCost   *float64 // nil = giá vốn hiện tại của sản phẩm
		RefType    string
		RefID      *uint
		SupplierID *uint
		VoucherNo  string
		Note       string
	}

	stockKey struct{ ProductID, BranchID uint }
)

// NewStockService: audit/notifier có thể nil (test).
func NewStockService(repo repository.StockRepository, products repository.ProductRepository, tx postgres.TxManager,
	audit AuditService, notifier Notifier) StockService {
	return newStockService(repo, products, tx, audit, notifier)
}

func newStockService(repo repository.StockRepository, products repository.ProductRepository, tx postgres.TxManager,
	audit AuditService, notifier Notifier) *stockService {
	return &stockService{repo: repo, products: products, tx: tx, audit: audit, notifier: notifier, now: time.Now}
}

var stockVNLoc = func() *time.Location {
	loc, err := time.LoadLocation(consts.TimeZone)
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}()

// ---- Hàm thuần ------------------------------------------------------------------

// stockRoundCost làm tròn giá vốn đến 2 chữ số thập phân (cột NUMERIC(14,2)).
func stockRoundCost(v float64) float64 { return math.Round(v*100) / 100 }

// stockWeightedCost tính giá vốn bình quân gia quyền sau khi nhập addQty đơn vị giá addCost,
// khi tổng tồn (mọi chi nhánh) trước khi nhập là oldQty với giá vốn oldCost:
//
//	giá vốn mới = (oldQty × oldCost + addQty × addCost) / (oldQty + addQty)
//
// oldQty ≤ 0 (hết hàng) → giá vốn mới = addCost.
func stockWeightedCost(oldQty int, oldCost float64, addQty int, addCost float64) float64 {
	if addQty <= 0 {
		return oldCost
	}
	if oldQty <= 0 {
		return stockRoundCost(addCost)
	}
	return stockRoundCost((float64(oldQty)*oldCost + float64(addQty)*addCost) / float64(oldQty+addQty))
}

// stockCrossedLow: cảnh báo chỉ khi tồn VỪA vượt xuống dưới ngưỡng (cũ ≥ min, mới < min).
func stockCrossedLow(oldQty, newQty, minStock int) bool {
	return minStock > 0 && oldQty >= minStock && newQty < minStock
}

func stockIsLow(qty, minStock int) bool { return minStock > 0 && qty < minStock }

func stockShortage(p *entity.Product, available, need int) error {
	return rescode.New(rescode.Conflict, fmt.Sprintf("Sản phẩm \"%s\" không đủ hàng tại chi nhánh (còn %d %s, cần %d)",
		p.Name, available, p.Unit, need))
}

// stockSortKeys loại trùng và sắp (product_id, branch_id) tăng dần – thứ tự khóa cố định để tránh deadlock.
func stockSortKeys(keys []stockKey) []stockKey {
	seen := map[stockKey]bool{}
	out := make([]stockKey, 0, len(keys))
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProductID != out[j].ProductID {
			return out[i].ProductID < out[j].ProductID
		}
		return out[i].BranchID < out[j].BranchID
	})
	return out
}

func (s *stockService) voucherPrefix(code string) string {
	return code + s.now().In(stockVNLoc).Format("060102")
}

// ---- Lõi biến động kho -----------------------------------------------------------

// lockLevels khóa (upsert + SELECT ... FOR UPDATE) các dòng tồn theo thứ tự (product_id, branch_id) tăng dần
// và trả tồn hiện tại. Gọi trong transaction, TRƯỚC khi applyStockChange cho nhiều dòng.
func (s *stockService) lockLevels(ctx context.Context, keys []stockKey) (map[stockKey]int, error) {
	out := map[stockKey]int{}
	for _, k := range stockSortKeys(keys) {
		q, err := s.repo.LockLevel(ctx, k.ProductID, k.BranchID)
		if err != nil {
			return nil, err
		}
		out[k] = q
	}
	return out, nil
}

// applyStockChange là cửa DUY NHẤT thay đổi tồn kho. Trong transaction: khóa dòng stock_levels
// (upsert + FOR UPDATE), cộng qty (có dấu), kiểm tra không âm (CHECK quantity >= 0 của DB là chốt chặn
// cuối – lỗi 23514 cũng trả 409), ghi thẻ kho với balance_after; khi tồn vừa xuống dưới min_stock thì
// gửi thông báo LOW_STOCK cho người có quyền inventory.view của chi nhánh.
func (s *stockService) applyStockChange(ctx context.Context, productID, branchID uint, qty int, typ string, ref stockRef) (*entity.StockTransaction, error) {
	if qty == 0 {
		return nil, rescode.New(rescode.BadRequest, "Số lượng biến động phải khác 0")
	}
	var out *entity.StockTransaction
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		p, err := s.products.FindProduct(ctx, productID, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return rescode.New(rescode.BadRequest, fmt.Sprintf("Sản phẩm #%d không tồn tại", productID))
			}
			return err
		}
		old, err := s.repo.LockLevel(ctx, productID, branchID)
		if err != nil {
			return err
		}
		newQty := old + qty
		if newQty < 0 {
			return stockShortage(p, old, -qty)
		}
		if err := s.repo.SetLevel(ctx, productID, branchID, newQty); err != nil {
			if isCheckViolation(err) {
				return stockShortage(p, old, -qty)
			}
			return err
		}
		cost := p.CostPrice
		if ref.UnitCost != nil {
			cost = *ref.UnitCost
		}
		t := &entity.StockTransaction{
			ProductID: productID, BranchID: branchID, Type: typ, Quantity: qty, BalanceAfter: newQty,
			UnitCost: stockRoundCost(cost), ReferenceType: ref.RefType, ReferenceID: ref.RefID, SupplierID: ref.SupplierID,
			VoucherNo: ref.VoucherNo, Note: invoiceTruncate(strings.TrimSpace(ref.Note), 255), CreatedBy: reqctx.UserID(ctx),
			CreatedAt: s.now(),
		}
		if err := s.repo.CreateTransaction(ctx, t); err != nil {
			if isCheckViolation(err) {
				return stockShortage(p, old, -qty)
			}
			return err
		}
		if qty < 0 && stockCrossedLow(old, newQty, p.MinStock) {
			s.notifyLowStock(ctx, p, branchID, newQty)
		}
		out = t
		return nil
	})
	return out, err
}

func (s *stockService) notifyLowStock(ctx context.Context, p *entity.Product, branchID uint, qty int) {
	if s.notifier == nil {
		return
	}
	branchName := fmt.Sprintf("#%d", branchID)
	if b, err := s.repo.FindBranch(ctx, branchID); err == nil {
		branchName = b.Name
	}
	title := "Tồn kho thấp: " + p.Name
	body := fmt.Sprintf("%s (%s) tại %s chỉ còn %d %s, dưới mức tối thiểu %d.", p.Name, p.SKU, branchName, qty, p.Unit, p.MinStock)
	link := fmt.Sprintf("/inventory/stock?branch_id=%d&low_stock=true", branchID)
	if err := s.notifier.NotifyPermission(ctx, consts.PermInventoryView, &branchID, NotifyLowStock, title, body, link); err != nil {
		g.Log().Warningf(ctx, "low stock notify: %v", err)
	}
}

// ---- Kiểm tra đầu vào ------------------------------------------------------------

// checkBranch: chi nhánh tồn tại, đang hoạt động và (nếu needScope) thuộc phạm vi người dùng.
func (s *stockService) checkBranch(ctx context.Context, id uint, needScope bool, label string) (*entity.Branch, error) {
	if id == 0 {
		return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn "+label)
	}
	if needScope && !reqctx.CanAccessBranch(ctx, id) {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền thao tác kho tại chi nhánh này")
	}
	b, err := s.repo.FindBranch(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.NotFound, "Không tìm thấy "+label)
		}
		return nil, err
	}
	if b.Status != "ACTIVE" {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh "+b.Name+" đang ngừng hoạt động")
	}
	return b, nil
}

func (s *stockService) findProduct(ctx context.Context, id uint, lock bool) (*entity.Product, error) {
	p, err := s.products.FindProduct(ctx, id, lock)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Sản phẩm #%d không tồn tại hoặc đã bị xóa", id))
		}
		return nil, err
	}
	return p, nil
}

// ---- Nghiệp vụ phiếu kho ---------------------------------------------------------

func (s *stockService) Import(ctx context.Context, req *v1.ImportReq) (*v1.Voucher, error) {
	if len(req.Items) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Phiếu nhập phải có ít nhất một dòng")
	}
	for _, it := range req.Items {
		if it == nil || it.ProductID == 0 || it.Quantity <= 0 {
			return nil, rescode.New(rescode.BadRequest, "Số lượng nhập phải lớn hơn 0")
		}
		if it.UnitCost < 0 {
			return nil, rescode.New(rescode.BadRequest, "Đơn giá nhập không được âm")
		}
	}
	if _, err := s.checkBranch(ctx, req.BranchID, true, "chi nhánh nhập"); err != nil {
		return nil, err
	}
	if req.SupplierID != nil && *req.SupplierID > 0 {
		if _, err := s.products.FindSupplier(ctx, *req.SupplierID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, rescode.New(rescode.BadRequest, "Nhà cung cấp không tồn tại")
			}
			return nil, err
		}
	} else {
		req.SupplierID = nil
	}
	var voucherNo string
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Khóa sản phẩm (tăng dần) để tính giá vốn bình quân tuần tự giữa các phiếu nhập đồng thời.
		ids := make([]uint, 0, len(req.Items))
		keys := make([]stockKey, 0, len(req.Items))
		for _, it := range req.Items {
			ids = append(ids, it.ProductID)
			keys = append(keys, stockKey{it.ProductID, req.BranchID})
		}
		products, err := s.lockProducts(ctx, ids)
		if err != nil {
			return err
		}
		if voucherNo, err = s.repo.NextVoucherNo(ctx, s.voucherPrefix(stockPrefixImport)); err != nil {
			return err
		}
		if _, err := s.lockLevels(ctx, keys); err != nil {
			return err
		}
		lines := make([]map[string]any, 0, len(req.Items))
		for _, it := range req.Items {
			p := products[it.ProductID]
			total, err := s.products.TotalStock(ctx, p.ID)
			if err != nil {
				return err
			}
			unit := stockRoundCost(it.UnitCost)
			newCost := stockWeightedCost(total, p.CostPrice, it.Quantity, unit)
			if _, err := s.applyStockChange(ctx, p.ID, req.BranchID, it.Quantity, entity.StockImport, stockRef{
				UnitCost: &unit, SupplierID: req.SupplierID, VoucherNo: voucherNo, Note: req.Note,
			}); err != nil {
				return err
			}
			if newCost != p.CostPrice {
				if err := s.repo.UpdateProductCost(ctx, p.ID, newCost); err != nil {
					return err
				}
			}
			lines = append(lines, map[string]any{"product_id": p.ID, "quantity": it.Quantity, "unit_cost": unit,
				"old_cost": p.CostPrice, "new_cost": newCost, "stock_before": total})
			p.CostPrice = newCost
		}
		auditCatalog(ctx, s.audit, AuditStockImport, auditEntityStockVoucher, 0, nil, map[string]any{
			"voucher_no": voucherNo, "branch_id": req.BranchID, "supplier_id": req.SupplierID, "note": req.Note, "items": lines,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Voucher(ctx, voucherNo)
}

// lockProducts khóa các sản phẩm theo id tăng dần (FOR NO KEY UPDATE).
func (s *stockService) lockProducts(ctx context.Context, ids []uint) (map[uint]*entity.Product, error) {
	uniq := append([]uint(nil), ids...)
	sort.Slice(uniq, func(i, j int) bool { return uniq[i] < uniq[j] })
	out := map[uint]*entity.Product{}
	for _, id := range uniq {
		if _, ok := out[id]; ok {
			continue
		}
		p, err := s.findProduct(ctx, id, true)
		if err != nil {
			return nil, err
		}
		out[id] = p
	}
	return out, nil
}

// stockMergeQty gộp các dòng trùng sản phẩm (cộng số lượng), giữ thứ tự xuất hiện đầu tiên.
func stockMergeQty(ids []uint, qtys []int) ([]uint, map[uint]int) {
	order := make([]uint, 0, len(ids))
	sum := map[uint]int{}
	for i, id := range ids {
		if _, ok := sum[id]; !ok {
			order = append(order, id)
		}
		sum[id] += qtys[i]
	}
	return order, sum
}

func (s *stockService) Transfer(ctx context.Context, req *v1.TransferReq) (*v1.Voucher, error) {
	if len(req.Items) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Phiếu điều chuyển phải có ít nhất một dòng")
	}
	if req.FromBranchID == req.ToBranchID {
		return nil, rescode.New(rescode.BadRequest, "Chi nhánh nhận phải khác chi nhánh xuất")
	}
	ids := make([]uint, 0, len(req.Items))
	qtys := make([]int, 0, len(req.Items))
	for _, it := range req.Items {
		if it == nil || it.ProductID == 0 || it.Quantity <= 0 {
			return nil, rescode.New(rescode.BadRequest, "Số lượng điều chuyển phải lớn hơn 0")
		}
		ids, qtys = append(ids, it.ProductID), append(qtys, it.Quantity)
	}
	from, err := s.checkBranch(ctx, req.FromBranchID, true, "chi nhánh xuất")
	if err != nil {
		return nil, err
	}
	to, err := s.checkBranch(ctx, req.ToBranchID, false, "chi nhánh nhận")
	if err != nil {
		return nil, err
	}
	order, sum := stockMergeQty(ids, qtys)
	var voucherNo string
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		products := map[uint]*entity.Product{}
		keys := make([]stockKey, 0, 2*len(order))
		for _, id := range order {
			p, err := s.findProduct(ctx, id, false)
			if err != nil {
				return err
			}
			products[id] = p
			keys = append(keys, stockKey{id, from.ID}, stockKey{id, to.ID})
		}
		if voucherNo, err = s.repo.NextVoucherNo(ctx, s.voucherPrefix(stockPrefixTransfer)); err != nil {
			return err
		}
		// Khóa TẤT CẢ dòng tồn liên quan theo (product_id, branch_id) tăng dần trước khi đổi:
		// hai phiếu điều chuyển ngược chiều đồng thời luôn khóa cùng thứ tự → không deadlock.
		levels, err := s.lockLevels(ctx, keys)
		if err != nil {
			return err
		}
		for _, id := range order {
			if have := levels[stockKey{id, from.ID}]; have < sum[id] {
				return stockShortage(products[id], have, sum[id])
			}
		}
		lines := make([]map[string]any, 0, len(order))
		for _, id := range order {
			p, qty := products[id], sum[id]
			cost := p.CostPrice
			if _, err := s.applyStockChange(ctx, id, from.ID, -qty, entity.StockTransferOut, stockRef{
				UnitCost: &cost, VoucherNo: voucherNo, Note: stockJoinNote("Chuyển đến "+to.Name, req.Note),
			}); err != nil {
				return err
			}
			if _, err := s.applyStockChange(ctx, id, to.ID, qty, entity.StockTransferIn, stockRef{
				UnitCost: &cost, VoucherNo: voucherNo, Note: stockJoinNote("Nhận từ "+from.Name, req.Note),
			}); err != nil {
				return err
			}
			lines = append(lines, map[string]any{"product_id": id, "quantity": qty})
		}
		auditCatalog(ctx, s.audit, AuditStockTransfer, auditEntityStockVoucher, 0, nil, map[string]any{
			"voucher_no": voucherNo, "from_branch_id": from.ID, "to_branch_id": to.ID, "note": req.Note, "items": lines,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Voucher(ctx, voucherNo)
}

func stockJoinNote(a, b string) string {
	if b = strings.TrimSpace(b); b != "" {
		return a + " – " + b
	}
	return a
}

func (s *stockService) Adjust(ctx context.Context, req *v1.AdjustReq) (*v1.Voucher, error) {
	if len(req.Items) == 0 {
		return nil, rescode.New(rescode.BadRequest, "Phiếu kiểm kê phải có ít nhất một dòng")
	}
	seen := map[uint]bool{}
	for _, it := range req.Items {
		if it == nil || it.ProductID == 0 {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng chọn sản phẩm")
		}
		if it.ActualQuantity == nil {
			return nil, rescode.New(rescode.BadRequest, "Vui lòng nhập số lượng thực tế")
		}
		if *it.ActualQuantity < 0 {
			return nil, rescode.New(rescode.BadRequest, "Số lượng thực tế không được âm")
		}
		if seen[it.ProductID] {
			return nil, rescode.New(rescode.BadRequest, fmt.Sprintf("Sản phẩm #%d bị lặp trong phiếu kiểm kê", it.ProductID))
		}
		seen[it.ProductID] = true
	}
	if _, err := s.checkBranch(ctx, req.BranchID, true, "chi nhánh"); err != nil {
		return nil, err
	}
	var voucherNo string
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		keys := make([]stockKey, 0, len(req.Items))
		for _, it := range req.Items {
			if _, err := s.findProduct(ctx, it.ProductID, false); err != nil {
				return err
			}
			keys = append(keys, stockKey{it.ProductID, req.BranchID})
		}
		var err error
		if voucherNo, err = s.repo.NextVoucherNo(ctx, s.voucherPrefix(stockPrefixAdjust)); err != nil {
			return err
		}
		levels, err := s.lockLevels(ctx, keys)
		if err != nil {
			return err
		}
		lines := make([]map[string]any, 0, len(req.Items))
		for _, it := range req.Items {
			current := levels[stockKey{it.ProductID, req.BranchID}]
			diff := *it.ActualQuantity - current
			if diff == 0 {
				continue // khớp sổ sách – không ghi thẻ kho
			}
			if _, err := s.applyStockChange(ctx, it.ProductID, req.BranchID, diff, entity.StockAdjust, stockRef{
				VoucherNo: voucherNo, Note: stockJoinNote(fmt.Sprintf("Kiểm kê: sổ sách %d, thực tế %d", current, *it.ActualQuantity), req.Note),
			}); err != nil {
				return err
			}
			lines = append(lines, map[string]any{"product_id": it.ProductID, "book_quantity": current,
				"actual_quantity": *it.ActualQuantity, "difference": diff})
		}
		if len(lines) == 0 {
			return rescode.New(rescode.BadRequest, "Số lượng thực tế khớp với tồn hiện tại, không có chênh lệch cần điều chỉnh")
		}
		auditCatalog(ctx, s.audit, AuditStockAdjust, auditEntityStockVoucher, 0, nil, map[string]any{
			"voucher_no": voucherNo, "branch_id": req.BranchID, "note": req.Note, "items": lines,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Voucher(ctx, voucherNo)
}

// ---- Truy vấn -------------------------------------------------------------------

var stockOrderable = map[string]string{
	"product_id": "p.id", "sku": "p.sku", "name": "p.name", "quantity": "COALESCE(sl.quantity, 0)",
	"branch_id": "b.id", "min_stock": "p.min_stock",
}

func (s *stockService) Stock(ctx context.Context, req *v1.StockReq) (*v1.StockRes, error) {
	f := repository.StockFilter{CategoryID: req.CategoryID, LowStock: req.LowStock}
	if req.BranchID > 0 {
		if !reqctx.CanAccessBranch(ctx, req.BranchID) {
			return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
		}
		f.BranchID = req.BranchID
	} else {
		f.BranchIDs = reqctx.AllowedBranches(ctx)
	}
	keys := make([]string, 0, len(stockOrderable))
	for k := range stockOrderable {
		keys = append(keys, k)
	}
	q := req.PageReq.ToQuery(keys, "product_id")
	q.OrderBy = stockOrderable[q.OrderBy]
	if req.PageReq.OrderBy == "" {
		q.OrderDir = "ASC"
	}
	rows, total, err := s.repo.ListStock(ctx, q, f)
	if err != nil {
		return nil, err
	}
	sum, err := s.repo.SumStock(ctx, q, f)
	if err != nil {
		return nil, err
	}
	items := dto.Map(rows, func(r repository.StockRow) *v1.StockRow {
		return &v1.StockRow{ProductID: r.ProductID, SKU: r.SKU, Barcode: r.Barcode, ProductName: r.ProductName,
			CategoryID: r.CategoryID, CategoryName: r.CategoryName, Unit: r.Unit, ImageURL: r.ImageURL,
			BranchID: r.BranchID, BranchName: r.BranchName, Quantity: r.Quantity, MinStock: r.MinStock,
			LowStock: stockIsLow(r.Quantity, r.MinStock), CostPrice: r.CostPrice,
			StockValue: roundMoneyVND(float64(r.Quantity) * r.CostPrice), SellPrice: r.SellPrice}
	})
	return &v1.StockRes{
		PageRes: *dto.NewPageRes(items, total, q),
		Summary: v1.StockSummary{TotalQuantity: sum.TotalQuantity, TotalValue: roundMoneyVND(sum.TotalValue), LowStockCount: sum.LowStockCount},
	}, nil
}

func toStockTxRes(r repository.StockTransactionRow) *v1.Transaction {
	return &v1.Transaction{ID: r.ID, ProductID: r.ProductID, ProductSKU: r.ProductSKU, ProductName: r.ProductName, Unit: r.Unit,
		BranchID: r.BranchID, BranchName: r.BranchName, Type: r.Type, Quantity: r.Quantity, BalanceAfter: r.BalanceAfter,
		UnitCost: r.UnitCost, ReferenceType: r.ReferenceType, ReferenceID: r.ReferenceID, SupplierID: r.SupplierID,
		SupplierName: r.SupplierName, VoucherNo: r.VoucherNo, Note: r.Note, CreatedBy: r.CreatedBy,
		CreatedByName: r.CreatedByName, CreatedAt: r.CreatedAt}
}

func (s *stockService) Transactions(ctx context.Context, req *v1.TransactionListReq) (*v1.TransactionListRes, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, req.BranchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	f := repository.StockTransactionFilter{BranchIDs: branchIDs, ProductID: req.ProductID, Type: req.Type,
		VoucherNo: strings.TrimSpace(req.VoucherNo)}
	var err error
	if f.From, f.To, err = subInstantRange(req.From, req.To); err != nil {
		return nil, err
	}
	q := req.PageReq.ToQuery([]string{"id", "created_at", "quantity"}, "id")
	q.OrderBy = "st." + q.OrderBy
	rows, total, err := s.repo.ListTransactions(ctx, q, f)
	if err != nil {
		return nil, err
	}
	return dto.NewPageRes(dto.Map(rows, toStockTxRes), total, q), nil
}

func (s *stockService) Voucher(ctx context.Context, voucherNo string) (*v1.Voucher, error) {
	voucherNo = strings.TrimSpace(voucherNo)
	rows, err := s.repo.VoucherLines(ctx, voucherNo)
	if err != nil {
		return nil, err
	}
	if voucherNo == "" || len(rows) == 0 {
		return nil, rescode.New(rescode.NotFound, "Không tìm thấy phiếu kho")
	}
	allowed := false
	for _, r := range rows {
		if reqctx.CanAccessBranch(ctx, r.BranchID) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, rescode.New(rescode.PermissionDenied, "Phiếu kho thuộc chi nhánh ngoài phạm vi của bạn")
	}
	return stockBuildVoucher(voucherNo, rows), nil
}

// stockBuildVoucher dựng phiếu kho từ các dòng thẻ kho cùng voucher_no (hàm thuần).
func stockBuildVoucher(voucherNo string, rows []repository.StockTransactionRow) *v1.Voucher {
	first := rows[0]
	v := &v1.Voucher{VoucherNo: voucherNo, BranchID: first.BranchID, BranchName: first.BranchName,
		SupplierID: first.SupplierID, SupplierName: first.SupplierName, ReferenceType: first.ReferenceType,
		ReferenceID: first.ReferenceID, Note: first.Note, CreatedBy: first.CreatedBy, CreatedByName: first.CreatedByName,
		CreatedAt: first.CreatedAt, Lines: dto.Map(rows, toStockTxRes)}
	switch first.Type {
	case entity.StockTransferIn, entity.StockTransferOut:
		v.Kind = "TRANSFER"
	default:
		v.Kind = first.Type
	}
	for _, r := range rows {
		if r.Type == entity.StockSale {
			v.Kind = entity.StockSale // hóa đơn POS: dòng bán (+ dòng hoàn kho khi hủy)
		}
		switch r.Type {
		case entity.StockTransferIn:
			id := r.BranchID
			v.ToBranchID, v.ToBranchName = &id, r.BranchName
			continue
		case entity.StockTransferOut:
			v.BranchID, v.BranchName = r.BranchID, r.BranchName
		}
		q := r.Quantity
		if q < 0 {
			q = -q
		}
		v.TotalQuantity += q
		v.TotalValue += float64(q) * r.UnitCost
	}
	v.TotalValue = roundMoneyVND(v.TotalValue)
	if v.Kind == "TRANSFER" || v.Kind == entity.StockAdjust {
		v.Note = stockStripAutoNote(v.Note)
	}
	return v
}

// stockStripAutoNote bỏ phần ghi chú tự sinh ("Chuyển đến …", "Kiểm kê: …") để lấy ghi chú người nhập.
func stockStripAutoNote(note string) string {
	if i := strings.Index(note, " – "); i >= 0 {
		return note[i+len(" – "):]
	}
	if strings.HasPrefix(note, "Chuyển đến ") || strings.HasPrefix(note, "Nhận từ ") || strings.HasPrefix(note, "Kiểm kê:") {
		return ""
	}
	return note
}

func toLowStockRes(r repository.StockLowRow) *v1.LowStockItem {
	return &v1.LowStockItem{ProductID: r.ProductID, SKU: r.SKU, ProductName: r.ProductName, Unit: r.Unit,
		BranchID: r.BranchID, BranchName: r.BranchName, Quantity: r.Quantity, MinStock: r.MinStock}
}

func (s *stockService) LowStock(ctx context.Context, branchID uint) ([]*v1.LowStockItem, error) {
	branchIDs, ok := reqctx.ResolveBranchFilter(ctx, branchID)
	if !ok {
		return nil, rescode.New(rescode.PermissionDenied, "Bạn không có quyền xem dữ liệu chi nhánh này")
	}
	rows, err := s.repo.ListLowStock(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toLowStockRes), nil
}

func (s *stockService) ListLowStock(ctx context.Context, branchID *uint) ([]*v1.LowStockItem, error) {
	var ids []uint
	if branchID != nil {
		ids = []uint{*branchID}
	}
	rows, err := s.repo.ListLowStock(ctx, ids)
	if err != nil {
		return nil, err
	}
	return dto.Map(rows, toLowStockRes), nil
}
