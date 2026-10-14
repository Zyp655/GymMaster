package service

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/repository"
)

// ---- Fake dùng cho test POS & kho (tiền tố pos/stock) -----------------------------
// Các fake nhúng interface repository: phương thức không dùng tới sẽ panic nếu bị gọi.

type stockFakeProductRepo struct {
	repository.ProductRepository
	products  map[uint]*entity.Product
	suppliers map[uint]*entity.Supplier
	stock     *stockFakeRepo
}

func (f *stockFakeProductRepo) FindProduct(_ context.Context, id uint, _ bool) (*entity.Product, error) {
	if p, ok := f.products[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *stockFakeProductRepo) FindSupplier(_ context.Context, id uint) (*entity.Supplier, error) {
	if s, ok := f.suppliers[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *stockFakeProductRepo) TotalStock(_ context.Context, productID uint) (int, error) {
	n := 0
	for k, q := range f.stock.levels {
		if k.ProductID == productID {
			n += q
		}
	}
	return n, nil
}

type stockFakeRepo struct {
	repository.StockRepository
	levels   map[stockKey]int
	txs      []entity.StockTransaction
	products *stockFakeProductRepo
	branches map[uint]*entity.Branch
	vouchers map[string]int
	lockLog  []stockKey
}

func (f *stockFakeRepo) LockLevel(_ context.Context, productID, branchID uint) (int, error) {
	k := stockKey{productID, branchID}
	f.lockLog = append(f.lockLog, k)
	return f.levels[k], nil
}

func (f *stockFakeRepo) SetLevel(_ context.Context, productID, branchID uint, q int) error {
	if q < 0 {
		return fmt.Errorf("CHECK quantity >= 0")
	}
	f.levels[stockKey{productID, branchID}] = q
	return nil
}

func (f *stockFakeRepo) CreateTransaction(_ context.Context, t *entity.StockTransaction) error {
	t.ID = uint(len(f.txs) + 1)
	f.txs = append(f.txs, *t)
	return nil
}

func (f *stockFakeRepo) UpdateProductCost(_ context.Context, id uint, cost float64) error {
	f.products.products[id].CostPrice = cost
	return nil
}

func (f *stockFakeRepo) NextVoucherNo(_ context.Context, prefix string) (string, error) {
	f.vouchers[prefix]++
	return fmt.Sprintf("%s%04d", prefix, f.vouchers[prefix]), nil
}

func (f *stockFakeRepo) FindBranch(_ context.Context, id uint) (*entity.Branch, error) {
	if b, ok := f.branches[id]; ok {
		return b, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *stockFakeRepo) VoucherLines(_ context.Context, no string) ([]repository.StockTransactionRow, error) {
	var out []repository.StockTransactionRow
	for _, t := range f.txs {
		if t.VoucherNo == no {
			out = append(out, repository.StockTransactionRow{StockTransaction: t, ProductName: f.products.products[t.ProductID].Name,
				BranchName: f.branches[t.BranchID].Name})
		}
	}
	return out, nil
}

// txsOf trả các dòng thẻ kho theo loại.
func (f *stockFakeRepo) txsOf(typ string) []entity.StockTransaction {
	var out []entity.StockTransaction
	for _, t := range f.txs {
		if t.Type == typ {
			out = append(out, t)
		}
	}
	return out
}

type posFakeRepo struct {
	repository.PosRepository
	inv     *invoiceFakeRepo
	stock   *stockFakeRepo
	members map[uint]*entity.Member
}

func (f *posFakeRepo) LockInvoice(ctx context.Context, id uint) (*entity.Invoice, error) {
	return f.inv.FindByID(ctx, id, true)
}

func (f *posFakeRepo) InvoiceItems(ctx context.Context, id uint) ([]entity.InvoiceItem, error) {
	return f.inv.Items(ctx, id)
}

func (f *posFakeRepo) InvoiceMoves(_ context.Context, invoiceID uint) ([]repository.PosInvoiceMove, error) {
	byID := map[uint]*repository.PosInvoiceMove{}
	for _, t := range f.stock.txs {
		if t.ReferenceType != entity.StockRefInvoice || t.ReferenceID == nil || *t.ReferenceID != invoiceID {
			continue
		}
		m, ok := byID[t.ProductID]
		if !ok {
			m = &repository.PosInvoiceMove{ProductID: t.ProductID}
			byID[t.ProductID] = m
		}
		switch t.Type {
		case entity.StockSale:
			m.SoldOut += -t.Quantity
			m.SaleCost = t.UnitCost
		case entity.StockSaleReturn:
			m.Returned += t.Quantity
		}
	}
	out := make([]repository.PosInvoiceMove, 0, len(byID))
	for _, m := range byID {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out, nil
}

func (f *posFakeRepo) FindMember(_ context.Context, id uint) (*entity.Member, error) {
	if m, ok := f.members[id]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}

type stockFakeNotifier struct {
	calls []string
}

func (n *stockFakeNotifier) Notify(context.Context, []uint, string, string, string, string) error {
	return nil
}
func (n *stockFakeNotifier) NotifyPermission(_ context.Context, perm string, branchID *uint, typ, title, _, _ string) error {
	n.calls = append(n.calls, fmt.Sprintf("%s|%d|%s|%s", perm, *branchID, typ, title))
	return nil
}

// posFixture: 2 chi nhánh, 3 sản phẩm, 1 hội viên ACTIVE (id 1) và 1 hội viên INACTIVE (id 2).
type posFixture struct {
	products *stockFakeProductRepo
	stock    *stockFakeRepo
	inv      *invoiceFakeRepo
	pos      *posFakeRepo
	notifier *stockFakeNotifier
	invoices InvoiceService
	svc      *posService
	stockSvc *stockService
}

func posPtrF(v float64) *float64 { return &v }

func newPosFixture() *posFixture {
	fx := &posFixture{notifier: &stockFakeNotifier{}}
	fx.products = &stockFakeProductRepo{
		products: map[uint]*entity.Product{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, SKU: "AQUA500", Name: "Nước suối Aquafina 500ml", Unit: "chai",
				CostPrice: 5000, SellPrice: 10000, MemberPrice: posPtrF(8000), MinStock: 5, Status: entity.ProductStatusActive},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, SKU: "WHEY-ON", Name: "Whey ON Gold Standard 5lbs", Unit: "hộp",
				CostPrice: 1500000, SellPrice: 1950000, MinStock: 2, Status: entity.ProductStatusActive},
			3: {BaseEntity: entity.BaseEntity{ID: 3}, SKU: "OLD", Name: "Hàng ngừng bán", Unit: "cái",
				SellPrice: 1000, Status: entity.ProductStatusInactive},
		},
		suppliers: map[uint]*entity.Supplier{1: {ID: 1, Name: "NCC"}},
	}
	fx.stock = &stockFakeRepo{
		levels:   map[stockKey]int{{1, 1}: 20, {2, 1}: 3, {1, 2}: 10, {3, 1}: 5},
		products: fx.products,
		branches: map[uint]*entity.Branch{
			1: {BaseEntity: entity.BaseEntity{ID: 1}, Name: "Cầu Giấy", Status: "ACTIVE"},
			2: {BaseEntity: entity.BaseEntity{ID: 2}, Name: "Hai Bà Trưng", Status: "ACTIVE"},
		},
		vouchers: map[string]int{},
	}
	fx.products.stock = fx.stock
	promos := &invoiceFakePromoRepo{items: map[uint]*entity.Promotion{}}
	fx.inv = newInvoiceFakeRepo(promos)
	fx.invoices = NewInvoiceService(fx.inv, NewPromotionService(promos, nil), subNoTx{}, nil)
	fx.pos = &posFakeRepo{inv: fx.inv, stock: fx.stock, members: map[uint]*entity.Member{
		1: {BaseEntity: entity.BaseEntity{ID: 1}, FullName: "Nguyễn Văn An", Status: entity.MemberStatusActive},
		2: {BaseEntity: entity.BaseEntity{ID: 2}, FullName: "Trần Thị Bình", Status: entity.MemberStatusInactive},
	}}
	fx.svc = NewPosService(fx.pos, fx.products, fx.stock, fx.invoices, subNoTx{}, nil, fx.notifier).(*posService)
	fx.stockSvc = fx.svc.stock
	return fx
}
