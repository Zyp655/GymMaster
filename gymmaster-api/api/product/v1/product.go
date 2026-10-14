package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// ---- Danh mục sản phẩm ----------------------------------------------------------

type Category struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	SortOrder    int       `json:"sort_order"`
	ProductCount int64     `json:"product_count" dc:"Số sản phẩm (chưa xóa) thuộc danh mục"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CategoryInput struct {
	Name      string `json:"name"       v:"required|max-length:100#Vui lòng nhập tên danh mục|Tên danh mục tối đa 100 ký tự"`
	SortOrder int    `json:"sort_order" v:"min:0#Thứ tự không được âm"`
}

type CategoryListReq struct {
	g.Meta  `path:"/product-categories" method:"get" tags:"Sản phẩm" summary:"Danh sách danh mục sản phẩm (mảng, sắp theo sort_order)" perm:"product.view,pos.sell"`
	Keyword string `json:"keyword" in:"query"`
}
type CategoryListRes []*Category

type CategoryCreateReq struct {
	g.Meta `path:"/product-categories" method:"post" tags:"Sản phẩm" summary:"Thêm danh mục" perm:"product.manage"`
	CategoryInput
}
type CategoryCreateRes = Category

type CategoryUpdateReq struct {
	g.Meta `path:"/product-categories/{id}" method:"put" tags:"Sản phẩm" summary:"Sửa danh mục" perm:"product.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	CategoryInput
}
type CategoryUpdateRes = Category

type CategoryDeleteReq struct {
	g.Meta `path:"/product-categories/{id}" method:"delete" tags:"Sản phẩm" summary:"Xóa danh mục (chặn nếu còn sản phẩm)" perm:"product.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type CategoryDeleteRes struct{}

// ---- Nhà cung cấp ---------------------------------------------------------------

type Supplier struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SupplierInput struct {
	Name    string `json:"name"    v:"required|max-length:150#Vui lòng nhập tên nhà cung cấp|Tên tối đa 150 ký tự"`
	Phone   string `json:"phone"   v:"regex:^[0-9+ ]{8,20}$#Số điện thoại không hợp lệ"`
	Email   string `json:"email"   v:"email|max-length:150#Email không hợp lệ|Email tối đa 150 ký tự"`
	Address string `json:"address" v:"max-length:255#Địa chỉ tối đa 255 ký tự"`
	Note    string `json:"note"    v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}

type SupplierListReq struct {
	g.Meta `path:"/suppliers" method:"get" tags:"Sản phẩm" summary:"Danh sách nhà cung cấp" perm:"product.view"`
	dto.PageReq
}
type SupplierListRes = dto.PageRes[*Supplier]

type SupplierOption struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type SupplierOptionsReq struct {
	g.Meta `path:"/suppliers/options" method:"get" tags:"Sản phẩm" summary:"Nhà cung cấp (dropdown)" perm:"product.view,inventory.import"`
}
type SupplierOptionsRes []*SupplierOption

type SupplierGetReq struct {
	g.Meta `path:"/suppliers/{id}" method:"get" tags:"Sản phẩm" summary:"Chi tiết nhà cung cấp" perm:"product.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type SupplierGetRes = Supplier

type SupplierCreateReq struct {
	g.Meta `path:"/suppliers" method:"post" tags:"Sản phẩm" summary:"Thêm nhà cung cấp" perm:"product.manage"`
	SupplierInput
}
type SupplierCreateRes = Supplier

type SupplierUpdateReq struct {
	g.Meta `path:"/suppliers/{id}" method:"put" tags:"Sản phẩm" summary:"Sửa nhà cung cấp" perm:"product.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	SupplierInput
}
type SupplierUpdateRes = Supplier

type SupplierDeleteReq struct {
	g.Meta `path:"/suppliers/{id}" method:"delete" tags:"Sản phẩm" summary:"Xóa nhà cung cấp (xóa mềm)" perm:"product.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type SupplierDeleteRes struct{}

// ---- Sản phẩm -------------------------------------------------------------------

type Product struct {
	ID           uint      `json:"id"`
	SKU          string    `json:"sku"`
	Barcode      string    `json:"barcode"`
	Name         string    `json:"name"`
	CategoryID   *uint     `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Unit         string    `json:"unit"`
	CostPrice    float64   `json:"cost_price"   dc:"Giá vốn bình quân gia quyền"`
	SellPrice    float64   `json:"sell_price"`
	MemberPrice  *float64  `json:"member_price" dc:"null = hội viên mua theo giá bán"`
	ImageURL     string    `json:"image_url"`
	MinStock     int       `json:"min_stock"    dc:"Ngưỡng cảnh báo tồn thấp (0 = không cảnh báo)"`
	Status       string    `json:"status"`
	TotalStock   int       `json:"total_stock"  dc:"Tổng tồn mọi chi nhánh"`
	Stock        *int      `json:"stock"        dc:"Tồn tại branch_id (chỉ có khi lọc branch_id)"`
	LowStock     *bool     `json:"low_stock"    dc:"stock < min_stock (chỉ có khi lọc branch_id)"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProductStock là tồn của sản phẩm tại từng chi nhánh (chi tiết sản phẩm).
type ProductStock struct {
	BranchID   uint   `json:"branch_id"`
	BranchName string `json:"branch_name"`
	Quantity   int    `json:"quantity"`
	LowStock   bool   `json:"low_stock"`
}

type ProductDetail struct {
	Product
	Stocks []*ProductStock `json:"stocks" dc:"Tồn theo chi nhánh (chỉ chi nhánh trong phạm vi, đã từng có tồn)"`
}

type ProductInput struct {
	SKU         string   `json:"sku"          v:"required|regex:^[A-Za-z0-9_.-]{2,30}$#Vui lòng nhập mã SKU|SKU gồm 2-30 ký tự chữ, số, '.', '-', '_'"`
	Barcode     string   `json:"barcode"      v:"max-length:50|regex:^[0-9A-Za-z-]*$#Mã vạch tối đa 50 ký tự|Mã vạch chỉ gồm chữ, số, '-'"`
	Name        string   `json:"name"         v:"required|max-length:150#Vui lòng nhập tên sản phẩm|Tên tối đa 150 ký tự"`
	CategoryID  *uint    `json:"category_id"`
	Unit        string   `json:"unit"         v:"max-length:20#Đơn vị tính tối đa 20 ký tự" d:"cái"`
	CostPrice   float64  `json:"cost_price"   v:"min:0#Giá vốn không được âm"`
	SellPrice   float64  `json:"sell_price"   v:"required|min:0#Vui lòng nhập giá bán|Giá bán không được âm"`
	MemberPrice *float64 `json:"member_price" dc:"Giá hội viên, ≤ giá bán; null = như giá bán"`
	ImageURL    string   `json:"image_url"    v:"max-length:500#URL ảnh tối đa 500 ký tự" dc:"URL từ POST /admin/uploads"`
	MinStock    int      `json:"min_stock"    v:"min:0#Tồn tối thiểu không được âm"`
	Status      string   `json:"status"       v:"required|in:ACTIVE,INACTIVE#Vui lòng chọn trạng thái|Trạng thái không hợp lệ" d:"ACTIVE"`
}

type ProductListReq struct {
	g.Meta `path:"/products" method:"get" tags:"Sản phẩm" summary:"Danh sách sản phẩm" perm:"product.view"`
	dto.PageReq
	CategoryID uint   `json:"category_id" in:"query"`
	Status     string `json:"status"      in:"query" v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ"`
	BranchID   uint   `json:"branch_id"   in:"query" dc:"Có → kèm stock và low_stock tại chi nhánh"`
	LowStock   bool   `json:"low_stock"   in:"query" dc:"true (cần branch_id) → chỉ sản phẩm tồn thấp"`
}
type ProductListRes = dto.PageRes[*Product]

type ProductGetReq struct {
	g.Meta `path:"/products/{id}" method:"get" tags:"Sản phẩm" summary:"Chi tiết sản phẩm + tồn theo chi nhánh" perm:"product.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ProductGetRes = ProductDetail

type ProductCreateReq struct {
	g.Meta `path:"/products" method:"post" tags:"Sản phẩm" summary:"Thêm sản phẩm" perm:"product.manage"`
	ProductInput
}
type ProductCreateRes = ProductDetail

type ProductUpdateReq struct {
	g.Meta `path:"/products/{id}" method:"put" tags:"Sản phẩm" summary:"Sửa sản phẩm" perm:"product.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ProductInput
}
type ProductUpdateRes = ProductDetail

type ProductDeleteReq struct {
	g.Meta `path:"/products/{id}" method:"delete" tags:"Sản phẩm" summary:"Xóa sản phẩm (chặn nếu còn tồn)" perm:"product.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ProductDeleteRes struct{}
