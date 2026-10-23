package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// PosInvoiceMove: tổng số lượng đã xuất bán (SALE) và đã nhập lại (SALE_RETURN) của một sản phẩm
	// theo một hóa đơn, kèm giá vốn lúc bán.
	PosInvoiceMove struct {
		ProductID uint
		SoldOut   int     // Σ |quantity| của SALE
		Returned  int     // Σ quantity của SALE_RETURN
		SaleCost  float64 // unit_cost của dòng SALE (giá vốn lúc bán)
	}

	PosSummaryFilter struct {
		BranchIDs []uint
		From      time.Time
		To        time.Time
	}

	PosSalesTotals struct {
		OrderCount   int64
		GrossRevenue float64
		Discount     float64
	}

	PosTopProduct struct {
		ProductID uint
		SKU       string `gorm:"column:sku"`
		Name      string
		Quantity  int64
		Revenue   float64
	}

	PosRepository interface {
		// LockInvoice khóa hóa đơn (SELECT ... FOR UPDATE) – tuần tự hóa các lần trả hàng của cùng hóa đơn.
		LockInvoice(ctx context.Context, id uint) (*entity.Invoice, error)
		InvoiceItems(ctx context.Context, invoiceID uint) ([]entity.InvoiceItem, error)
		InvoiceMoves(ctx context.Context, invoiceID uint) ([]PosInvoiceMove, error)
		FindMember(ctx context.Context, id uint) (*entity.Member, error)

		SalesTotals(ctx context.Context, f PosSummaryFilter) (*PosSalesTotals, error)
		ItemsSold(ctx context.Context, f PosSummaryFilter) (int64, error)
		RefundTotal(ctx context.Context, f PosSummaryFilter) (float64, error)
		PaymentsByMethod(ctx context.Context, f PosSummaryFilter) ([]PaymentMethodTotal, error)
		CostOfGoods(ctx context.Context, f PosSummaryFilter) (float64, error)
		TopProducts(ctx context.Context, f PosSummaryFilter, limit int) ([]PosTopProduct, error)
	}

	posRepository struct {
		db *gorm.DB
	}
)

func NewPosRepository(db *gorm.DB) PosRepository {
	return &posRepository{db: db}
}

func (r *posRepository) LockInvoice(ctx context.Context, id uint) (*entity.Invoice, error) {
	var inv entity.Invoice
	if err := postgres.Conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).First(&inv, id).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *posRepository) InvoiceItems(ctx context.Context, invoiceID uint) ([]entity.InvoiceItem, error) {
	var items []entity.InvoiceItem
	err := postgres.Conn(ctx, r.db).Where("invoice_id = ?", invoiceID).Order("id").Find(&items).Error
	return items, err
}

func (r *posRepository) InvoiceMoves(ctx context.Context, invoiceID uint) ([]PosInvoiceMove, error) {
	var rows []PosInvoiceMove
	err := postgres.Conn(ctx, r.db).Table("stock_transactions").
		Select(`product_id,
			COALESCE(-SUM(quantity) FILTER (WHERE type = ?), 0) AS sold_out,
			COALESCE(SUM(quantity) FILTER (WHERE type = ?), 0) AS returned,
			COALESCE(MAX(unit_cost) FILTER (WHERE type = ?), 0) AS sale_cost`,
			entity.StockSale, entity.StockSaleReturn, entity.StockSale).
		Where("reference_type = ? AND reference_id = ?", entity.StockRefInvoice, invoiceID).
		Group("product_id").Scan(&rows).Error
	return rows, err
}

func (r *posRepository) FindMember(ctx context.Context, id uint) (*entity.Member, error) {
	var m entity.Member
	if err := postgres.Conn(ctx, r.db).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// posProductInvoiceCond: hóa đơn POS có ít nhất một dòng sản phẩm. Hóa đơn source POS không có dòng sản phẩm
// (vd vé tập theo ngày lập trước khi chuyển sang COUNTER) không phải đơn bán hàng → không tính vào tổng kết POS.
const posProductInvoiceCond = "i.source = ? AND EXISTS (SELECT 1 FROM invoice_items pit WHERE pit.invoice_id = i.id AND pit.item_type = ?)"

// posInvoices: hóa đơn bán hàng POS (có dòng sản phẩm, không tính hóa đơn hủy) lập trong khoảng thời gian.
func (r *posRepository) posInvoices(ctx context.Context, f PosSummaryFilter) *gorm.DB {
	tx := postgres.Conn(ctx, r.db).Table("invoices i").
		Where(posProductInvoiceCond, entity.InvoiceSourcePOS, entity.InvoiceItemProduct).
		Where("i.status <> ? AND i.issued_at >= ? AND i.issued_at < ?", entity.InvoiceStatusVoid, f.From, f.To)
	if f.BranchIDs != nil {
		tx = tx.Where("i.branch_id IN ?", f.BranchIDs)
	}
	return tx
}

func (r *posRepository) SalesTotals(ctx context.Context, f PosSummaryFilter) (*PosSalesTotals, error) {
	var out PosSalesTotals
	err := r.posInvoices(ctx, f).
		Select("COUNT(*) AS order_count, COALESCE(SUM(i.total), 0) AS gross_revenue, COALESCE(SUM(i.discount), 0) AS discount").
		Scan(&out).Error
	return &out, err
}

// posNetItemsCTE: số lượng & doanh thu sản phẩm THUẦN theo sản phẩm trong khoảng thời gian, cùng cơ sở với
// net_revenue/cost_of_goods của tổng kết ngày:
//   - bán: dòng PRODUCT của hóa đơn POS (không hủy) LẬP trong khoảng;
//   - trừ trả hàng: SALE_RETURN phát sinh trong khoảng của hóa đơn POS không hủy (hóa đơn hủy đã bị loại khỏi
//     phần bán nên phiếu nhập lại kho khi hủy không bị trừ hai lần); giá trị trả = SL trả × đơn giá thực bán
//     của sản phẩm trên hóa đơn đó (Σ amount / Σ quantity).
func posNetItemsCTE(f PosSummaryFilter) (string, map[string]any) {
	branch := ""
	if f.BranchIDs != nil {
		branch = " AND i.branch_id IN @branches"
	}
	q := `WITH sold AS (
	SELECT it.ref_id AS product_id, SUM(it.quantity) AS qty, SUM(it.amount) AS amount
	FROM invoices i JOIN invoice_items it ON it.invoice_id = i.id AND it.item_type = @item_type
	WHERE i.source = @source AND i.status <> @void AND i.issued_at >= @from AND i.issued_at < @to` + branch + `
	GROUP BY it.ref_id
), ret AS (
	SELECT st.product_id, SUM(st.quantity) AS qty,
		SUM(st.quantity * COALESCE((SELECT SUM(x.amount) / NULLIF(SUM(x.quantity), 0) FROM invoice_items x
			WHERE x.invoice_id = i.id AND x.item_type = @item_type AND x.ref_id = st.product_id), 0)) AS amount
	FROM stock_transactions st JOIN invoices i ON i.id = st.reference_id
	WHERE st.type = @sale_return AND st.reference_type = @ref_invoice AND st.created_at >= @from AND st.created_at < @to
		AND i.source = @source AND i.status <> @void` + branch + `
	GROUP BY st.product_id
), net AS (
	SELECT COALESCE(s.product_id, r.product_id) AS product_id,
		COALESCE(s.qty, 0) - COALESCE(r.qty, 0) AS quantity,
		COALESCE(s.amount, 0) - COALESCE(r.amount, 0) AS revenue
	FROM sold s FULL JOIN ret r ON r.product_id = s.product_id
)`
	args := map[string]any{"item_type": entity.InvoiceItemProduct, "source": entity.InvoiceSourcePOS,
		"void": entity.InvoiceStatusVoid, "from": f.From, "to": f.To, "sale_return": entity.StockSaleReturn,
		"ref_invoice": entity.StockRefInvoice}
	if f.BranchIDs != nil {
		args["branches"] = f.BranchIDs
	}
	return q, args
}

// ItemsSold: số lượng sản phẩm bán THUẦN (đã trừ trả hàng trong ngày) – xem posNetItemsCTE.
func (r *posRepository) ItemsSold(ctx context.Context, f PosSummaryFilter) (int64, error) {
	var n int64
	q, args := posNetItemsCTE(f)
	err := postgres.Conn(ctx, r.db).Raw(q+` SELECT COALESCE(SUM(quantity), 0) FROM net`, args).Scan(&n).Error
	return n, err
}

func (r *posRepository) RefundTotal(ctx context.Context, f PosSummaryFilter) (float64, error) {
	var v float64
	tx := postgres.Conn(ctx, r.db).Table("refunds rf").
		Joins("JOIN invoices i ON i.id = rf.invoice_id").
		Where(posProductInvoiceCond, entity.InvoiceSourcePOS, entity.InvoiceItemProduct).
		Where("rf.created_at >= ? AND rf.created_at < ?", f.From, f.To)
	if f.BranchIDs != nil {
		tx = tx.Where("i.branch_id IN ?", f.BranchIDs)
	}
	err := tx.Select("COALESCE(SUM(rf.amount), 0)").Scan(&v).Error
	return v, err
}

func (r *posRepository) PaymentsByMethod(ctx context.Context, f PosSummaryFilter) ([]PaymentMethodTotal, error) {
	tx := postgres.Conn(ctx, r.db).Table("payments pm").
		Joins("JOIN invoices i ON i.id = pm.invoice_id").
		Where(posProductInvoiceCond, entity.InvoiceSourcePOS, entity.InvoiceItemProduct).
		Where("pm.paid_at >= ? AND pm.paid_at < ?", f.From, f.To)
	if f.BranchIDs != nil {
		tx = tx.Where("i.branch_id IN ?", f.BranchIDs)
	}
	var rows []PaymentMethodTotal
	err := tx.Select("pm.method AS method, COALESCE(SUM(pm.amount), 0) AS amount, COUNT(*) AS count").
		Group("pm.method").Scan(&rows).Error
	return rows, err
}

func (r *posRepository) CostOfGoods(ctx context.Context, f PosSummaryFilter) (float64, error) {
	var v float64
	tx := postgres.Conn(ctx, r.db).Table("stock_transactions st").
		Where("st.type IN ? AND st.created_at >= ? AND st.created_at < ?",
			[]string{entity.StockSale, entity.StockSaleReturn}, f.From, f.To)
	if f.BranchIDs != nil {
		tx = tx.Where("st.branch_id IN ?", f.BranchIDs)
	}
	// SALE có quantity âm, SALE_RETURN dương → giá vốn hàng bán = −Σ quantity × unit_cost.
	err := tx.Select("COALESCE(-SUM(st.quantity * st.unit_cost), 0)").Scan(&v).Error
	return v, err
}

// TopProducts: top sản phẩm theo số lượng THUẦN (đã trừ trả hàng), doanh thu thuần – cùng cơ sở với ItemsSold.
func (r *posRepository) TopProducts(ctx context.Context, f PosSummaryFilter, limit int) ([]PosTopProduct, error) {
	var rows []PosTopProduct
	q, args := posNetItemsCTE(f)
	args["limit"] = limit
	err := postgres.Conn(ctx, r.db).Raw(q+`
SELECT p.id AS product_id, p.sku, p.name, n.quantity, n.revenue
FROM net n JOIN products p ON p.id = n.product_id
WHERE n.quantity > 0
ORDER BY n.quantity DESC, n.revenue DESC, p.id
LIMIT @limit`, args).Scan(&rows).Error
	return rows, err
}
