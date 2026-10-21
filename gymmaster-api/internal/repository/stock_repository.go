package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	// StockRow là tồn của một sản phẩm tại một chi nhánh kèm thông tin hiển thị.
	StockRow struct {
		ProductID    uint
		SKU          string `gorm:"column:sku"`
		Barcode      string
		ProductName  string
		CategoryID   *uint
		CategoryName string
		Unit         string
		ImageURL     string `gorm:"column:image_url"`
		BranchID     uint
		BranchName   string
		Quantity     int
		MinStock     int
		CostPrice    float64
		SellPrice    float64
		LowStock     bool // theo LowStockCond
	}

	StockFilter struct {
		BranchID   uint   // > 0 → mọi sản phẩm (chưa xóa, ACTIVE hoặc còn tồn) tại chi nhánh, kể cả tồn 0
		BranchIDs  []uint // khi BranchID = 0: giới hạn các dòng stock_levels theo phạm vi (nil = mọi chi nhánh)
		CategoryID uint
		LowStock   bool
	}

	StockSummaryRow struct {
		TotalQuantity int64
		TotalValue    float64
		LowStockCount int64
	}

	// StockTransactionRow là dòng thẻ kho kèm tên sản phẩm/chi nhánh/NCC/người tạo.
	StockTransactionRow struct {
		entity.StockTransaction
		ProductSKU    string
		ProductName   string
		Unit          string
		BranchName    string
		SupplierName  string
		CreatedByName string
	}

	StockTransactionFilter struct {
		BranchIDs []uint
		ProductID uint
		Type      string
		VoucherNo string
		From      *time.Time
		To        *time.Time
	}

	StockLowRow struct {
		ProductID   uint
		SKU         string `gorm:"column:sku"`
		ProductName string
		Unit        string
		BranchID    uint
		BranchName  string
		Quantity    int
		MinStock    int
	}

	StockRepository interface {
		// LockLevel bảo đảm có dòng stock_levels (upsert quantity 0) rồi khóa nó bằng SELECT ... FOR UPDATE,
		// trả tồn hiện tại. Phải gọi trong transaction.
		LockLevel(ctx context.Context, productID, branchID uint) (int, error)
		SetLevel(ctx context.Context, productID, branchID uint, quantity int) error
		CreateTransaction(ctx context.Context, t *entity.StockTransaction) error
		// UpdateProductCost ghi giá vốn bình quân mới của sản phẩm.
		UpdateProductCost(ctx context.Context, productID uint, cost float64) error
		// NextVoucherNo sinh mã phiếu prefix + số thứ tự 4 chữ số (prefix đã gồm ngày, ví dụ NK261014 → NK2610140001).
		// Dùng advisory lock theo prefix trong transaction để hai phiếu đồng thời không trùng số.
		NextVoucherNo(ctx context.Context, prefix string) (string, error)
		// FindBranch đọc chi nhánh chưa xóa (gorm.ErrRecordNotFound nếu không có).
		FindBranch(ctx context.Context, id uint) (*entity.Branch, error)

		ListStock(ctx context.Context, q dto.Query, f StockFilter) ([]StockRow, int64, error)
		SumStock(ctx context.Context, q dto.Query, f StockFilter) (*StockSummaryRow, error)
		ListTransactions(ctx context.Context, q dto.Query, f StockTransactionFilter) ([]StockTransactionRow, int64, error)
		VoucherLines(ctx context.Context, voucherNo string) ([]StockTransactionRow, error)
		ListLowStock(ctx context.Context, branchIDs []uint) ([]StockLowRow, error)
	}

	stockRepository struct {
		db *gorm.DB
	}
)

func NewStockRepository(db *gorm.DB) StockRepository {
	return &stockRepository{db: db}
}

func (r *stockRepository) LockLevel(ctx context.Context, productID, branchID uint) (int, error) {
	db := postgres.Conn(ctx, r.db)
	if err := db.Exec(`INSERT INTO stock_levels (product_id, branch_id, quantity) VALUES (?, ?, 0)
		ON CONFLICT (product_id, branch_id) DO NOTHING`, productID, branchID).Error; err != nil {
		return 0, err
	}
	var lvl entity.StockLevel
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("product_id = ? AND branch_id = ?", productID, branchID).First(&lvl).Error
	return lvl.Quantity, err
}

func (r *stockRepository) SetLevel(ctx context.Context, productID, branchID uint, quantity int) error {
	return postgres.Conn(ctx, r.db).Exec(`UPDATE stock_levels SET quantity = ?, updated_at = now()
		WHERE product_id = ? AND branch_id = ?`, quantity, productID, branchID).Error
}

func (r *stockRepository) CreateTransaction(ctx context.Context, t *entity.StockTransaction) error {
	return postgres.Conn(ctx, r.db).Create(t).Error
}

func (r *stockRepository) UpdateProductCost(ctx context.Context, productID uint, cost float64) error {
	return postgres.Conn(ctx, r.db).Model(&entity.Product{}).Where("id = ?", productID).UpdateColumn("cost_price", cost).Error
}

func (r *stockRepository) NextVoucherNo(ctx context.Context, prefix string) (string, error) {
	db := postgres.Conn(ctx, r.db)
	if err := db.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, "voucher:"+prefix).Error; err != nil {
		return "", err
	}
	var last []string
	if err := db.Raw(`SELECT voucher_no FROM stock_transactions WHERE voucher_no LIKE ? AND length(voucher_no) = ?
		ORDER BY voucher_no DESC LIMIT 1`, prefix+"%", len(prefix)+4).Scan(&last).Error; err != nil {
		return "", err
	}
	n := 0
	if len(last) > 0 {
		n, _ = strconv.Atoi(last[0][len(prefix):])
	}
	return fmt.Sprintf("%s%04d", prefix, n+1), nil
}

func (r *stockRepository) FindBranch(ctx context.Context, id uint) (*entity.Branch, error) {
	var b entity.Branch
	if err := postgres.Conn(ctx, r.db).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// stockQuery dựng truy vấn tồn theo bộ lọc (chưa Select).
func (r *stockRepository) stockQuery(ctx context.Context, q dto.Query, f StockFilter) *gorm.DB {
	var tx *gorm.DB
	if f.BranchID > 0 {
		// Mọi sản phẩm tại một chi nhánh: sản phẩm đang kinh doanh hoặc còn tồn.
		tx = postgres.Conn(ctx, r.db).Table("products p").
			Joins("JOIN branches b ON b.id = ?", f.BranchID).
			Joins("LEFT JOIN stock_levels sl ON sl.product_id = p.id AND sl.branch_id = b.id").
			Where("p.deleted_at IS NULL AND (p.status = ? OR COALESCE(sl.quantity, 0) > 0)", entity.ProductStatusActive)
	} else {
		tx = postgres.Conn(ctx, r.db).Table("stock_levels sl").
			Joins("JOIN products p ON p.id = sl.product_id AND p.deleted_at IS NULL").
			Joins("JOIN branches b ON b.id = sl.branch_id")
		if f.BranchIDs != nil {
			tx = tx.Where("sl.branch_id IN ?", f.BranchIDs)
		}
	}
	tx = tx.Joins("LEFT JOIN product_categories c ON c.id = p.category_id")
	if f.CategoryID > 0 {
		tx = tx.Where("p.category_id = ?", f.CategoryID)
	}
	if f.LowStock {
		tx = tx.Where(lowStockSQL)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("p.name ILIKE ? OR p.sku ILIKE ? OR p.barcode ILIKE ?", kw, kw, kw)
	}
	return tx
}

func (r *stockRepository) ListStock(ctx context.Context, q dto.Query, f StockFilter) ([]StockRow, int64, error) {
	tx := r.stockQuery(ctx, q, f)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []StockRow
	err := tx.Select(`p.id AS product_id, p.sku, p.barcode, p.name AS product_name, p.category_id,
		COALESCE(c.name, '') AS category_name, p.unit, p.image_url, b.id AS branch_id, b.name AS branch_name,
		COALESCE(sl.quantity, 0) AS quantity, p.min_stock, p.cost_price, p.sell_price, `+lowStockSQL+` AS low_stock`).
		Scopes(q.Paginate).Order("p.id, b.id").Scan(&rows).Error
	return rows, total, err
}

func (r *stockRepository) SumStock(ctx context.Context, q dto.Query, f StockFilter) (*StockSummaryRow, error) {
	var out StockSummaryRow
	err := r.stockQuery(ctx, q, f).Select(`COALESCE(SUM(COALESCE(sl.quantity, 0)), 0) AS total_quantity,
		COALESCE(SUM(COALESCE(sl.quantity, 0) * p.cost_price), 0) AS total_value,
		COUNT(*) FILTER (WHERE `+lowStockSQL+`) AS low_stock_count`).
		Scan(&out).Error
	return &out, err
}

const stockTxSelect = `st.*, p.sku AS product_sku, p.name AS product_name, p.unit, b.name AS branch_name,
	COALESCE(s.name, '') AS supplier_name, COALESCE(u.full_name, '') AS created_by_name`

func (r *stockRepository) txQuery(ctx context.Context) *gorm.DB {
	return postgres.Conn(ctx, r.db).Table("stock_transactions st").
		Joins("JOIN products p ON p.id = st.product_id").
		Joins("JOIN branches b ON b.id = st.branch_id").
		Joins("LEFT JOIN suppliers s ON s.id = st.supplier_id").
		Joins("LEFT JOIN users u ON u.id = st.created_by")
}

func (r *stockRepository) ListTransactions(ctx context.Context, q dto.Query, f StockTransactionFilter) ([]StockTransactionRow, int64, error) {
	tx := r.txQuery(ctx)
	if f.BranchIDs != nil {
		tx = tx.Where("st.branch_id IN ?", f.BranchIDs)
	}
	if f.ProductID > 0 {
		tx = tx.Where("st.product_id = ?", f.ProductID)
	}
	if f.Type != "" {
		tx = tx.Where("st.type = ?", f.Type)
	}
	if f.VoucherNo != "" {
		tx = tx.Where("st.voucher_no ILIKE ?", "%"+f.VoucherNo+"%")
	}
	if f.From != nil {
		tx = tx.Where("st.created_at >= ?", *f.From)
	}
	if f.To != nil {
		tx = tx.Where("st.created_at < ?", *f.To)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("p.name ILIKE ? OR p.sku ILIKE ? OR st.voucher_no ILIKE ? OR st.note ILIKE ?", kw, kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []StockTransactionRow
	err := tx.Select(stockTxSelect).Scopes(q.Paginate).Order("st.id DESC").Scan(&rows).Error
	return rows, total, err
}

func (r *stockRepository) VoucherLines(ctx context.Context, voucherNo string) ([]StockTransactionRow, error) {
	var rows []StockTransactionRow
	err := r.txQuery(ctx).Select(stockTxSelect).Where("st.voucher_no = ?", voucherNo).Order("st.id").Scan(&rows).Error
	return rows, err
}

func (r *stockRepository) ListLowStock(ctx context.Context, branchIDs []uint) ([]StockLowRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("stock_levels sl").
		Select(`p.id AS product_id, p.sku, p.name AS product_name, p.unit, b.id AS branch_id, b.name AS branch_name,
			sl.quantity, p.min_stock`).
		Joins("JOIN products p ON p.id = sl.product_id").
		Joins("JOIN branches b ON b.id = sl.branch_id").
		Where(lowStockSQL)
	if branchIDs != nil {
		tx = tx.Where("sl.branch_id IN ?", branchIDs)
	}
	var rows []StockLowRow
	err := tx.Order("b.id, sl.quantity, p.name").Scan(&rows).Error
	return rows, err
}
