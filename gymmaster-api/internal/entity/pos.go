package entity

import (
	"time"

	"gorm.io/gorm"
)

// Hằng số module POS & tồn kho (migration 000009).
const (
	ProductStatusActive   = "ACTIVE"
	ProductStatusInactive = "INACTIVE"

	// Loại biến động kho (stock_transactions.type). Quantity có dấu: + nhập, − xuất.
	StockImport      = "IMPORT"
	StockSale        = "SALE"
	StockSaleReturn  = "SALE_RETURN"
	StockTransferIn  = "TRANSFER_IN"
	StockTransferOut = "TRANSFER_OUT"
	StockAdjust      = "ADJUST"

	// stock_transactions.reference_type
	StockRefInvoice = "INVOICE"
)

// ProductCategory là danh mục sản phẩm (đồ uống, thực phẩm bổ sung, phụ kiện…).
type ProductCategory struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"type:varchar(100);not null"`
	SortOrder int    `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (ProductCategory) TableName() string { return "product_categories" }

// Supplier là nhà cung cấp hàng.
type Supplier struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"type:varchar(150);not null"`
	Phone     string `gorm:"type:varchar(20);not null;default:''"`
	Email     string `gorm:"type:varchar(150);not null;default:''"`
	Address   string `gorm:"type:varchar(255);not null;default:''"`
	Note      string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (Supplier) TableName() string { return "suppliers" }

// Product là sản phẩm bán tại quầy. CostPrice là giá vốn bình quân gia quyền (cập nhật khi nhập hàng).
// MemberPrice nil = hội viên mua theo giá bán.
type Product struct {
	BaseEntity
	SKU         string   `gorm:"column:sku;type:varchar(30);not null"`
	Barcode     string   `gorm:"type:varchar(50);not null;default:''"`
	Name        string   `gorm:"type:varchar(150);not null"`
	CategoryID  *uint    //
	Unit        string   `gorm:"type:varchar(20);not null;default:'cái'"`
	CostPrice   float64  `gorm:"type:numeric(14,2);not null;default:0"`
	SellPrice   float64  `gorm:"type:numeric(14,2);not null"`
	MemberPrice *float64 `gorm:"type:numeric(14,2)"`
	ImageURL    string   `gorm:"column:image_url;type:varchar(500);not null;default:''"`
	MinStock    int      `gorm:"not null;default:0"`
	Status      string   `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
}

func (Product) TableName() string { return "products" }

// StockLevel là tồn hiện tại của một sản phẩm tại một chi nhánh (CHECK quantity >= 0).
type StockLevel struct {
	ProductID uint `gorm:"primaryKey"`
	BranchID  uint `gorm:"primaryKey"`
	Quantity  int  `gorm:"not null;default:0"`
	UpdatedAt time.Time
}

func (StockLevel) TableName() string { return "stock_levels" }

// StockTransaction là một dòng thẻ kho.
type StockTransaction struct {
	ID            uint    `gorm:"primaryKey;autoIncrement"`
	ProductID     uint    `gorm:"not null"`
	BranchID      uint    `gorm:"not null"`
	Type          string  `gorm:"type:varchar(20);not null"`
	Quantity      int     `gorm:"not null"`
	BalanceAfter  int     `gorm:"not null"`
	UnitCost      float64 `gorm:"type:numeric(14,2);not null;default:0"`
	ReferenceType string  `gorm:"type:varchar(20);not null;default:''"`
	ReferenceID   *uint
	SupplierID    *uint
	VoucherNo     string `gorm:"type:varchar(30);not null;default:''"`
	Note          string `gorm:"type:varchar(255);not null;default:''"`
	CreatedBy     *uint
	CreatedAt     time.Time
}

func (StockTransaction) TableName() string { return "stock_transactions" }
