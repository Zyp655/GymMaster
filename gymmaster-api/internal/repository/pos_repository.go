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

// posInvoices: hóa đơn POS (không tính hóa đơn hủy) lập trong khoảng thời gian.
func (r *posRepository) posInvoices(ctx context.Context, f PosSummaryFilter) *gorm.DB {
	tx := postgres.Conn(ctx, r.db).Table("invoices i").
		Where("i.source = ? AND i.status <> ? AND i.issued_at >= ? AND i.issued_at < ?",
			entity.InvoiceSourcePOS, entity.InvoiceStatusVoid, f.From, f.To)
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

func (r *posRepository) ItemsSold(ctx context.Context, f PosSummaryFilter) (int64, error) {
	var n int64
	err := r.posInvoices(ctx, f).Joins("JOIN invoice_items it ON it.invoice_id = i.id AND it.item_type = ?", entity.InvoiceItemProduct).
		Select("COALESCE(SUM(it.quantity), 0)").Scan(&n).Error
	return n, err
}

func (r *posRepository) RefundTotal(ctx context.Context, f PosSummaryFilter) (float64, error) {
	var v float64
	tx := postgres.Conn(ctx, r.db).Table("refunds rf").
		Joins("JOIN invoices i ON i.id = rf.invoice_id").
		Where("i.source = ? AND rf.created_at >= ? AND rf.created_at < ?", entity.InvoiceSourcePOS, f.From, f.To)
	if f.BranchIDs != nil {
		tx = tx.Where("i.branch_id IN ?", f.BranchIDs)
	}
	err := tx.Select("COALESCE(SUM(rf.amount), 0)").Scan(&v).Error
	return v, err
}

func (r *posRepository) PaymentsByMethod(ctx context.Context, f PosSummaryFilter) ([]PaymentMethodTotal, error) {
	tx := postgres.Conn(ctx, r.db).Table("payments pm").
		Joins("JOIN invoices i ON i.id = pm.invoice_id").
		Where("i.source = ? AND pm.paid_at >= ? AND pm.paid_at < ?", entity.InvoiceSourcePOS, f.From, f.To)
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

func (r *posRepository) TopProducts(ctx context.Context, f PosSummaryFilter, limit int) ([]PosTopProduct, error) {
	var rows []PosTopProduct
	err := r.posInvoices(ctx, f).
		Joins("JOIN invoice_items it ON it.invoice_id = i.id AND it.item_type = ?", entity.InvoiceItemProduct).
		Joins("JOIN products p ON p.id = it.ref_id").
		Select("p.id AS product_id, p.sku, p.name, SUM(it.quantity) AS quantity, SUM(it.amount) AS revenue").
		Group("p.id, p.sku, p.name").Order("quantity DESC, revenue DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}
