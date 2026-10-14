package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gymmaster-api/internal/dto"
	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/storage/postgres"
)

type (
	CategoryRow struct {
		entity.ProductCategory
		ProductCount int64
	}

	// ProductRow là sản phẩm kèm tên danh mục, tổng tồn mọi chi nhánh và tồn tại chi nhánh lọc (nếu có).
	ProductRow struct {
		entity.Product
		CategoryName string
		TotalStock   int
		Stock        int // tồn tại ProductFilter.BranchID (0 nếu không lọc)
	}

	ProductFilter struct {
		CategoryID uint
		Status     string
		BranchID   uint // > 0 → tính Stock tại chi nhánh
		LowStock   bool // cần BranchID: stock < min_stock
		InStock    bool // cần BranchID: stock > 0
	}

	ProductStockRow struct {
		BranchID   uint
		BranchName string
		Quantity   int
		MinStock   int
	}

	ProductRepository interface {
		ListCategories(ctx context.Context, keyword string) ([]CategoryRow, error)
		FindCategory(ctx context.Context, id uint) (*entity.ProductCategory, error)
		ExistsCategoryName(ctx context.Context, name string, excludeID uint) (bool, error)
		CreateCategory(ctx context.Context, c *entity.ProductCategory) error
		UpdateCategory(ctx context.Context, c *entity.ProductCategory) error
		DeleteCategory(ctx context.Context, id uint) error
		CountProductsInCategory(ctx context.Context, id uint) (int64, error)

		ListSuppliers(ctx context.Context, q dto.Query) ([]entity.Supplier, int64, error)
		SupplierOptions(ctx context.Context) ([]entity.Supplier, error)
		FindSupplier(ctx context.Context, id uint) (*entity.Supplier, error)
		CreateSupplier(ctx context.Context, s *entity.Supplier) error
		UpdateSupplier(ctx context.Context, s *entity.Supplier) error
		DeleteSupplier(ctx context.Context, id uint) error

		ListProducts(ctx context.Context, q dto.Query, f ProductFilter) ([]ProductRow, int64, error)
		// FindProduct đọc sản phẩm chưa xóa; lock = true → SELECT ... FOR NO KEY UPDATE (trong transaction).
		// Dùng NO KEY UPDATE (không phải FOR UPDATE) để không chặn khóa KEY SHARE mà FK của stock_levels/
		// stock_transactions lấy trên products – tránh deadlock giữa nhập hàng và bán hàng.
		FindProduct(ctx context.Context, id uint, lock bool) (*entity.Product, error)
		FindProductRow(ctx context.Context, id uint, branchID uint) (*ProductRow, error)
		// FindByCode tìm sản phẩm theo mã vạch, nếu không có thì theo SKU.
		FindByCode(ctx context.Context, code string) (*entity.Product, error)
		ExistsSKU(ctx context.Context, sku string, excludeID uint) (bool, error)
		CreateProduct(ctx context.Context, p *entity.Product) error
		UpdateProduct(ctx context.Context, p *entity.Product) error
		DeleteProduct(ctx context.Context, id uint, deletedBy *uint) error
		// ProductStocks: tồn của sản phẩm theo chi nhánh (branchIDs nil = mọi chi nhánh).
		ProductStocks(ctx context.Context, productID uint, branchIDs []uint) ([]ProductStockRow, error)
		TotalStock(ctx context.Context, productID uint) (int, error)
	}

	productRepository struct {
		db *gorm.DB
	}
)

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

// ---- Danh mục -------------------------------------------------------------------

func (r *productRepository) ListCategories(ctx context.Context, keyword string) ([]CategoryRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("product_categories c").
		Select(`c.*, (SELECT COUNT(*) FROM products p WHERE p.category_id = c.id AND p.deleted_at IS NULL) AS product_count`).
		Where("c.deleted_at IS NULL")
	if kw := strings.TrimSpace(keyword); kw != "" {
		tx = tx.Where("c.name ILIKE ?", "%"+kw+"%")
	}
	var rows []CategoryRow
	err := tx.Order("c.sort_order, c.name").Scan(&rows).Error
	return rows, err
}

func (r *productRepository) FindCategory(ctx context.Context, id uint) (*entity.ProductCategory, error) {
	var c entity.ProductCategory
	if err := postgres.Conn(ctx, r.db).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *productRepository) ExistsCategoryName(ctx context.Context, name string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.ProductCategory{}).
		Where("lower(name) = lower(?) AND id <> ?", name, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *productRepository) CreateCategory(ctx context.Context, c *entity.ProductCategory) error {
	return postgres.Conn(ctx, r.db).Create(c).Error
}

func (r *productRepository) UpdateCategory(ctx context.Context, c *entity.ProductCategory) error {
	return postgres.Conn(ctx, r.db).Model(c).Select("name", "sort_order").Updates(c).Error
}

func (r *productRepository) DeleteCategory(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.ProductCategory{}, id).Error
}

func (r *productRepository) CountProductsInCategory(ctx context.Context, id uint) (int64, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Product{}).Where("category_id = ?", id).Count(&n).Error
	return n, err
}

// ---- Nhà cung cấp ---------------------------------------------------------------

func (r *productRepository) ListSuppliers(ctx context.Context, q dto.Query) ([]entity.Supplier, int64, error) {
	tx := postgres.Conn(ctx, r.db).Model(&entity.Supplier{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR phone LIKE ? OR email ILIKE ?", kw, kw, kw)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entity.Supplier
	if err := tx.Scopes(q.Paginate).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *productRepository) SupplierOptions(ctx context.Context) ([]entity.Supplier, error) {
	var items []entity.Supplier
	err := postgres.Conn(ctx, r.db).Order("name").Find(&items).Error
	return items, err
}

func (r *productRepository) FindSupplier(ctx context.Context, id uint) (*entity.Supplier, error) {
	var s entity.Supplier
	if err := postgres.Conn(ctx, r.db).First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *productRepository) CreateSupplier(ctx context.Context, s *entity.Supplier) error {
	return postgres.Conn(ctx, r.db).Create(s).Error
}

func (r *productRepository) UpdateSupplier(ctx context.Context, s *entity.Supplier) error {
	return postgres.Conn(ctx, r.db).Model(s).Select("name", "phone", "email", "address", "note").Updates(s).Error
}

func (r *productRepository) DeleteSupplier(ctx context.Context, id uint) error {
	return postgres.Conn(ctx, r.db).Delete(&entity.Supplier{}, id).Error
}

// ---- Sản phẩm -------------------------------------------------------------------

// productQuery trả truy vấn (chưa Select, để Count được) và biểu thức select tương ứng.
func (r *productRepository) productQuery(ctx context.Context, branchID uint) (*gorm.DB, string) {
	tx := postgres.Conn(ctx, r.db).Table("products p").
		Joins("LEFT JOIN product_categories c ON c.id = p.category_id").
		Where("p.deleted_at IS NULL")
	sel := `p.*, COALESCE(c.name, '') AS category_name,
		COALESCE((SELECT SUM(sl.quantity) FROM stock_levels sl WHERE sl.product_id = p.id), 0) AS total_stock`
	if branchID > 0 {
		tx = tx.Joins("LEFT JOIN stock_levels st ON st.product_id = p.id AND st.branch_id = ?", branchID)
		sel += ", COALESCE(st.quantity, 0) AS stock"
	} else {
		sel += ", 0 AS stock"
	}
	return tx, sel
}

func (r *productRepository) ListProducts(ctx context.Context, q dto.Query, f ProductFilter) ([]ProductRow, int64, error) {
	tx, sel := r.productQuery(ctx, f.BranchID)
	if f.CategoryID > 0 {
		tx = tx.Where("p.category_id = ?", f.CategoryID)
	}
	if f.Status != "" {
		tx = tx.Where("p.status = ?", f.Status)
	}
	if f.BranchID > 0 && f.LowStock {
		tx = tx.Where("p.min_stock > 0 AND COALESCE(st.quantity, 0) < p.min_stock")
	}
	if f.BranchID > 0 && f.InStock {
		tx = tx.Where("COALESCE(st.quantity, 0) > 0")
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("p.name ILIKE ? OR p.sku ILIKE ? OR p.barcode ILIKE ?", kw, kw, kw)
	}
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ProductRow
	if err := tx.Select(sel).Scopes(q.Paginate).Order("p.id DESC").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *productRepository) FindProduct(ctx context.Context, id uint, lock bool) (*entity.Product, error) {
	db := postgres.Conn(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "NO KEY UPDATE"})
	}
	var p entity.Product
	if err := db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *productRepository) FindProductRow(ctx context.Context, id uint, branchID uint) (*ProductRow, error) {
	var rows []ProductRow
	tx, sel := r.productQuery(ctx, branchID)
	if err := tx.Select(sel).Where("p.id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *productRepository) FindByCode(ctx context.Context, code string) (*entity.Product, error) {
	var items []entity.Product
	err := postgres.Conn(ctx, r.db).Where("barcode = ? AND barcode <> ''", code).Order("id").Limit(1).Find(&items).Error
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		err = postgres.Conn(ctx, r.db).Where("upper(sku) = upper(?)", code).Limit(1).Find(&items).Error
		if err != nil {
			return nil, err
		}
	}
	if len(items) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &items[0], nil
}

func (r *productRepository) ExistsSKU(ctx context.Context, sku string, excludeID uint) (bool, error) {
	var n int64
	err := postgres.Conn(ctx, r.db).Model(&entity.Product{}).Where("sku = ? AND id <> ?", sku, excludeID).Count(&n).Error
	return n > 0, err
}

func (r *productRepository) CreateProduct(ctx context.Context, p *entity.Product) error {
	return postgres.Conn(ctx, r.db).Create(p).Error
}

func (r *productRepository) UpdateProduct(ctx context.Context, p *entity.Product) error {
	return postgres.Conn(ctx, r.db).Model(p).
		Select("sku", "barcode", "name", "category_id", "unit", "cost_price", "sell_price", "member_price",
			"image_url", "min_stock", "status", "updated_by").
		Updates(p).Error
}

func (r *productRepository) DeleteProduct(ctx context.Context, id uint, deletedBy *uint) error {
	db := postgres.Conn(ctx, r.db)
	if err := db.Model(&entity.Product{}).Where("id = ?", id).UpdateColumn("updated_by", deletedBy).Error; err != nil {
		return err
	}
	return db.Delete(&entity.Product{}, id).Error
}

func (r *productRepository) ProductStocks(ctx context.Context, productID uint, branchIDs []uint) ([]ProductStockRow, error) {
	tx := postgres.Conn(ctx, r.db).Table("stock_levels sl").
		Select("sl.branch_id, b.name AS branch_name, sl.quantity, p.min_stock").
		Joins("JOIN branches b ON b.id = sl.branch_id").
		Joins("JOIN products p ON p.id = sl.product_id").
		Where("sl.product_id = ?", productID)
	if branchIDs != nil {
		tx = tx.Where("sl.branch_id IN ?", branchIDs)
	}
	var rows []ProductStockRow
	err := tx.Order("sl.branch_id").Scan(&rows).Error
	return rows, err
}

func (r *productRepository) TotalStock(ctx context.Context, productID uint) (int, error) {
	var n int
	err := postgres.Conn(ctx, r.db).Raw(`SELECT COALESCE(SUM(quantity), 0) FROM stock_levels WHERE product_id = ?`, productID).Scan(&n).Error
	return n, err
}
