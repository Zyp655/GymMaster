package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// ---- Phiếu kho ------------------------------------------------------------------

type ImportItem struct {
	ProductID uint    `json:"product_id" v:"required|min:1#Vui lòng chọn sản phẩm|Sản phẩm không hợp lệ"`
	Quantity  int     `json:"quantity"   v:"required|min:1#Vui lòng nhập số lượng|Số lượng phải lớn hơn 0"`
	UnitCost  float64 `json:"unit_cost"  v:"min:0#Đơn giá nhập không được âm"`
}

type ImportReq struct {
	g.Meta     `path:"/inventory/imports" method:"post" tags:"Kho" summary:"Nhập hàng (phiếu NK)" perm:"inventory.import"`
	BranchID   uint          `json:"branch_id"   v:"required|min:1#Vui lòng chọn chi nhánh nhập|Chi nhánh không hợp lệ"`
	SupplierID *uint         `json:"supplier_id"`
	Items      []*ImportItem `json:"items"       v:"required#Phiếu nhập phải có ít nhất một dòng"`
	Note       string        `json:"note"        v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type ImportRes = Voucher

type TransferItem struct {
	ProductID uint `json:"product_id" v:"required|min:1#Vui lòng chọn sản phẩm|Sản phẩm không hợp lệ"`
	Quantity  int  `json:"quantity"   v:"required|min:1#Vui lòng nhập số lượng|Số lượng phải lớn hơn 0"`
}

type TransferReq struct {
	g.Meta       `path:"/inventory/transfers" method:"post" tags:"Kho" summary:"Điều chuyển hàng giữa chi nhánh (phiếu DC)" perm:"inventory.transfer"`
	FromBranchID uint            `json:"from_branch_id" v:"required|min:1#Vui lòng chọn chi nhánh xuất|Chi nhánh không hợp lệ"`
	ToBranchID   uint            `json:"to_branch_id"   v:"required|min:1#Vui lòng chọn chi nhánh nhận|Chi nhánh không hợp lệ"`
	Items        []*TransferItem `json:"items"          v:"required#Phiếu điều chuyển phải có ít nhất một dòng"`
	Note         string          `json:"note"           v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type TransferRes = Voucher

type AdjustItem struct {
	ProductID      uint `json:"product_id"      v:"required|min:1#Vui lòng chọn sản phẩm|Sản phẩm không hợp lệ"`
	ActualQuantity *int `json:"actual_quantity" v:"min:0#Số lượng thực tế không được âm" dc:"Bắt buộc"`
}

type AdjustReq struct {
	g.Meta   `path:"/inventory/adjustments" method:"post" tags:"Kho" summary:"Kiểm kê – điều chỉnh tồn (phiếu KK)" perm:"inventory.adjust"`
	BranchID uint          `json:"branch_id" v:"required|min:1#Vui lòng chọn chi nhánh|Chi nhánh không hợp lệ"`
	Items    []*AdjustItem `json:"items"     v:"required#Phiếu kiểm kê phải có ít nhất một dòng"`
	Note     string        `json:"note"      v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}
type AdjustRes = Voucher

// Transaction là một dòng thẻ kho.
type Transaction struct {
	ID            uint      `json:"id"`
	ProductID     uint      `json:"product_id"`
	ProductSKU    string    `json:"product_sku"`
	ProductName   string    `json:"product_name"`
	Unit          string    `json:"unit"`
	BranchID      uint      `json:"branch_id"`
	BranchName    string    `json:"branch_name"`
	Type          string    `json:"type"           dc:"IMPORT | SALE | SALE_RETURN | TRANSFER_IN | TRANSFER_OUT | ADJUST"`
	Quantity      int       `json:"quantity"       dc:"Có dấu: + nhập, − xuất"`
	BalanceAfter  int       `json:"balance_after"  dc:"Tồn tại chi nhánh sau biến động"`
	UnitCost      float64   `json:"unit_cost"`
	ReferenceType string    `json:"reference_type" dc:"INVOICE với bán/trả hàng"`
	ReferenceID   *uint     `json:"reference_id"`
	SupplierID    *uint     `json:"supplier_id"`
	SupplierName  string    `json:"supplier_name"`
	VoucherNo     string    `json:"voucher_no"`
	Note          string    `json:"note"`
	CreatedBy     *uint     `json:"created_by"`
	CreatedByName string    `json:"created_by_name"`
	CreatedAt     time.Time `json:"created_at"`
}

// Voucher là một phiếu kho (nhóm các dòng thẻ kho cùng voucher_no).
type Voucher struct {
	VoucherNo     string         `json:"voucher_no"`
	Kind          string         `json:"kind"            dc:"IMPORT | TRANSFER | ADJUST | SALE | SALE_RETURN"`
	BranchID      uint           `json:"branch_id"       dc:"Chi nhánh (phiếu điều chuyển: chi nhánh xuất)"`
	BranchName    string         `json:"branch_name"`
	ToBranchID    *uint          `json:"to_branch_id"    dc:"Chỉ phiếu điều chuyển"`
	ToBranchName  string         `json:"to_branch_name"`
	SupplierID    *uint          `json:"supplier_id"`
	SupplierName  string         `json:"supplier_name"`
	ReferenceType string         `json:"reference_type"`
	ReferenceID   *uint          `json:"reference_id"`
	Note          string         `json:"note"`
	TotalQuantity int            `json:"total_quantity"  dc:"Σ |quantity| (phiếu điều chuyển chỉ tính chiều xuất)"`
	TotalValue    float64        `json:"total_value"     dc:"Σ |quantity| × unit_cost (cùng quy tắc)"`
	CreatedBy     *uint          `json:"created_by"`
	CreatedByName string         `json:"created_by_name"`
	CreatedAt     time.Time      `json:"created_at"`
	Lines         []*Transaction `json:"lines"`
}

// ---- Truy vấn -------------------------------------------------------------------

type StockReq struct {
	g.Meta `path:"/inventory/stock" method:"get" tags:"Kho" summary:"Tồn kho hiện tại + giá trị tồn" perm:"inventory.view"`
	dto.PageReq
	BranchID   uint `json:"branch_id"   in:"query" dc:"Có → mọi sản phẩm đang kinh doanh tại chi nhánh (kể cả tồn 0); không → các dòng tồn của chi nhánh trong phạm vi"`
	CategoryID uint `json:"category_id" in:"query"`
	LowStock   bool `json:"low_stock"   in:"query" dc:"true → chỉ dòng quantity < min_stock"`
}

type StockRow struct {
	ProductID    uint    `json:"product_id"`
	SKU          string  `json:"sku"`
	Barcode      string  `json:"barcode"`
	ProductName  string  `json:"product_name"`
	CategoryID   *uint   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Unit         string  `json:"unit"`
	ImageURL     string  `json:"image_url"`
	BranchID     uint    `json:"branch_id"`
	BranchName   string  `json:"branch_name"`
	Quantity     int     `json:"quantity"`
	MinStock     int     `json:"min_stock"`
	LowStock     bool    `json:"low_stock"`
	CostPrice    float64 `json:"cost_price"`
	StockValue   float64 `json:"stock_value" dc:"= quantity × cost_price"`
	SellPrice    float64 `json:"sell_price"`
}

type StockSummary struct {
	TotalQuantity int64   `json:"total_quantity"`
	TotalValue    float64 `json:"total_value"`
	LowStockCount int64   `json:"low_stock_count"`
}

type StockRes struct {
	dto.PageRes[*StockRow]
	Summary StockSummary `json:"summary" dc:"Tính trên toàn bộ bộ lọc"`
}

type TransactionListReq struct {
	g.Meta `path:"/inventory/transactions" method:"get" tags:"Kho" summary:"Thẻ kho" perm:"inventory.view"`
	dto.PageReq
	ProductID uint   `json:"product_id" in:"query"`
	BranchID  uint   `json:"branch_id"  in:"query"`
	Type      string `json:"type"       in:"query" v:"in:IMPORT,SALE,SALE_RETURN,TRANSFER_IN,TRANSFER_OUT,ADJUST#Loại biến động không hợp lệ"`
	VoucherNo string `json:"voucher_no" in:"query"`
	From      string `json:"from"       in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD"`
	To        string `json:"to"         in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD"`
}
type TransactionListRes = dto.PageRes[*Transaction]

type VoucherGetReq struct {
	g.Meta    `path:"/inventory/vouchers/{voucher_no}" method:"get" tags:"Kho" summary:"Chi tiết phiếu kho" perm:"inventory.view"`
	VoucherNo string `json:"voucher_no" in:"path" v:"required"`
}
type VoucherGetRes = Voucher

// LowStockItem là một dòng tồn thấp (cron cảnh báo, dashboard).
type LowStockItem struct {
	ProductID   uint   `json:"product_id"`
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Unit        string `json:"unit"`
	BranchID    uint   `json:"branch_id"`
	BranchName  string `json:"branch_name"`
	Quantity    int    `json:"quantity"`
	MinStock    int    `json:"min_stock"`
}

type LowStockReq struct {
	g.Meta   `path:"/inventory/low-stock" method:"get" tags:"Kho" summary:"Sản phẩm tồn dưới mức tối thiểu" perm:"inventory.view"`
	BranchID uint `json:"branch_id" in:"query"`
}
type LowStockRes []*LowStockItem
