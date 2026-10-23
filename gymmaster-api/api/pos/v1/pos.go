package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	invoicev1 "gymmaster-api/api/invoice/v1"
	"gymmaster-api/internal/dto"
)

// PosProduct là sản phẩm đang bán kèm tồn tại chi nhánh (màn hình POS).
type PosProduct struct {
	ID           uint     `json:"id"`
	SKU          string   `json:"sku"`
	Barcode      string   `json:"barcode"`
	Name         string   `json:"name"`
	CategoryID   *uint    `json:"category_id"`
	CategoryName string   `json:"category_name"`
	Unit         string   `json:"unit"`
	SellPrice    float64  `json:"sell_price"`
	MemberPrice  *float64 `json:"member_price" dc:"null = như giá bán"`
	ImageURL     string   `json:"image_url"`
	Stock        int      `json:"stock"        dc:"Tồn tại chi nhánh"`
	MinStock     int      `json:"min_stock"`
	LowStock     bool     `json:"low_stock"`
}

type ProductsReq struct {
	g.Meta     `path:"/pos/products" method:"get" tags:"POS" summary:"Sản phẩm đang bán kèm tồn tại chi nhánh" perm:"pos.sell"`
	BranchID   uint   `json:"branch_id"   in:"query" v:"required|min:1#Vui lòng chọn chi nhánh|Chi nhánh không hợp lệ"`
	Keyword    string `json:"keyword"     in:"query" dc:"Tên / SKU / mã vạch"`
	CategoryID uint   `json:"category_id" in:"query"`
	InStock    bool   `json:"in_stock"    in:"query" dc:"true → chỉ sản phẩm còn hàng"`
}
type ProductsRes []*PosProduct

type ByBarcodeReq struct {
	g.Meta   `path:"/pos/products/by-barcode/{code}" method:"get" tags:"POS" summary:"Tìm sản phẩm theo mã vạch (hoặc SKU)" perm:"pos.sell"`
	Code     string `json:"code"      in:"path"  v:"required"`
	BranchID uint   `json:"branch_id" in:"query" v:"required|min:1#Vui lòng chọn chi nhánh|Chi nhánh không hợp lệ"`
}
type ByBarcodeRes = PosProduct

type OrderItem struct {
	ProductID uint `json:"product_id" v:"required|min:1#Vui lòng chọn sản phẩm|Sản phẩm không hợp lệ"`
	Quantity  int  `json:"quantity"   v:"required|integer|min:1#Vui lòng nhập số lượng|Số lượng phải là số nguyên|Số lượng phải lớn hơn 0"`
}

type OrderPayment struct {
	Amount    float64 `json:"amount"    v:"required|min:1#Vui lòng nhập số tiền|Số tiền phải lớn hơn 0"`
	Method    string  `json:"method"    v:"required|in:CASH,TRANSFER,CARD#Vui lòng chọn phương thức thanh toán|Phương thức thanh toán không hợp lệ"`
	Reference string  `json:"reference" v:"max-length:100#Mã giao dịch tối đa 100 ký tự"`
}

type CreateOrderReq struct {
	g.Meta        `path:"/pos/orders" method:"post" tags:"POS" summary:"Bán hàng tại quầy (lập hóa đơn + trừ kho + thu tiền đủ)" perm:"pos.sell"`
	BranchID      uint            `json:"branch_id"      v:"required|min:1#Vui lòng chọn chi nhánh|Chi nhánh không hợp lệ"`
	MemberID      *uint           `json:"member_id"      dc:"null = khách vãng lai"`
	Items         []*OrderItem    `json:"items"          v:"required#Đơn hàng phải có ít nhất một sản phẩm"`
	PromotionCode string          `json:"promotion_code" v:"max-length:30#Mã khuyến mãi tối đa 30 ký tự"`
	Payments      []*OrderPayment `json:"payments"       dc:"Có thể nhiều phương thức. Σ amount ≥ total; phần thừa chỉ được từ tiền mặt (trả lại change_amount)"`
	Note          string          `json:"note"           v:"max-length:500#Ghi chú tối đa 500 ký tự"`
}

type OrderRes struct {
	Invoice        *invoicev1.InvoiceDetail `json:"invoice"`
	ReceivedAmount float64                  `json:"received_amount" dc:"Tổng tiền khách đưa"`
	ChangeAmount   float64                  `json:"change_amount"   dc:"Tiền thừa trả khách (= received_amount − total)"`
	Print          *invoicev1.PrintRes      `json:"print"           dc:"Dữ liệu in hóa đơn"`
}
type CreateOrderRes = OrderRes

type ListOrdersReq struct {
	g.Meta `path:"/pos/orders" method:"get" tags:"POS" summary:"Danh sách hóa đơn POS" perm:"pos.sell,invoice.view"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query"`
	MemberID uint   `json:"member_id" in:"query"`
	Status   string `json:"status"    in:"query" v:"in:UNPAID,PARTIAL,PAID,VOID,REFUNDED#Trạng thái không hợp lệ"`
	From     string `json:"from"      in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD"`
	To       string `json:"to"        in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD"`
}
type ListOrdersRes = invoicev1.ListRes

type GetOrderReq struct {
	g.Meta    `path:"/pos/orders/{invoice_id}" method:"get" tags:"POS" summary:"Chi tiết đơn POS (kèm số lượng đã trả theo sản phẩm)" perm:"pos.sell,invoice.view"`
	InvoiceID uint `json:"invoice_id" in:"path" v:"required|min:1"`
}

// ReturnableLine là tình trạng trả hàng của một sản phẩm trong đơn.
type ReturnableLine struct {
	ProductID     uint    `json:"product_id"`
	Description   string  `json:"description"`
	SoldQuantity  int     `json:"sold_quantity"`
	ReturnedQty   int     `json:"returned_quantity"`
	ReturnableQty int     `json:"returnable_quantity"`
	Amount        float64 `json:"amount"        dc:"Thành tiền sau giảm giá của sản phẩm trên hóa đơn"`
	RefundedValue float64 `json:"refunded_value" dc:"Giá trị hàng đã trả (đã hoàn)"`
}

type GetOrderRes struct {
	Invoice *invoicev1.InvoiceDetail `json:"invoice"`
	Lines   []*ReturnableLine        `json:"lines"`
}

type ReturnItem struct {
	ProductID uint `json:"product_id" v:"required|min:1#Vui lòng chọn sản phẩm|Sản phẩm không hợp lệ"`
	Quantity  int  `json:"quantity"   v:"required|integer|min:1#Vui lòng nhập số lượng trả|Số lượng trả phải là số nguyên|Số lượng trả phải lớn hơn 0"`
}

type ReturnReq struct {
	g.Meta       `path:"/pos/orders/{invoice_id}/returns" method:"post" tags:"POS" summary:"Trả hàng (hoàn kho + hoàn tiền tương ứng)" perm:"pos.sell"`
	InvoiceID    uint          `json:"invoice_id"    in:"path" v:"required|min:1"`
	Items        []*ReturnItem `json:"items"         v:"required#Vui lòng chọn sản phẩm trả"`
	RefundMethod string        `json:"refund_method" v:"required|in:CASH,TRANSFER,CARD#Vui lòng chọn phương thức hoàn tiền|Phương thức hoàn tiền không hợp lệ"`
	Reason       string        `json:"reason"        v:"required|max-length:255#Vui lòng nhập lý do trả hàng|Lý do tối đa 255 ký tự"`
}

type ReturnRes struct {
	VoucherNo    string                   `json:"voucher_no"    dc:"Phiếu trả hàng TH…"`
	RefundAmount float64                  `json:"refund_amount"`
	Invoice      *invoicev1.InvoiceDetail `json:"invoice"`
	Lines        []*ReturnableLine        `json:"lines"`
}

type SummaryReq struct {
	g.Meta   `path:"/pos/summary" method:"get" tags:"POS" summary:"Tổng kết bán hàng trong ngày" perm:"pos.sell,invoice.view"`
	BranchID uint   `json:"branch_id" in:"query" dc:"Bỏ trống = mọi chi nhánh trong phạm vi"`
	Date     string `json:"date"      in:"query" v:"date-format:Y-m-d#Ngày phải có dạng YYYY-MM-DD" dc:"Mặc định hôm nay (giờ VN)"`
}

type TopProduct struct {
	ProductID uint    `json:"product_id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int64   `json:"quantity"`
	Revenue   float64 `json:"revenue"`
}

type MethodTotal struct {
	Method string  `json:"method"`
	Amount float64 `json:"amount"`
	Count  int64   `json:"count"`
}

type SummaryRes struct {
	Date         string         `json:"date"`
	BranchID     uint           `json:"branch_id"`
	OrderCount   int64          `json:"order_count"    dc:"Số hóa đơn POS (trừ hóa đơn hủy)"`
	ItemsSold    int64          `json:"items_sold"     dc:"Số lượng sản phẩm bán THUẦN = SL trên HĐ POS lập trong ngày − SL trả hàng phát sinh trong ngày (cùng cơ sở net_revenue)"`
	GrossRevenue float64        `json:"gross_revenue"  dc:"Σ total hóa đơn POS trong ngày"`
	Discount     float64        `json:"discount"`
	RefundAmount float64        `json:"refund_amount"  dc:"Σ tiền hoàn (trả hàng) phát sinh trong ngày cho hóa đơn POS"`
	NetRevenue   float64        `json:"net_revenue"    dc:"= gross_revenue − refund_amount"`
	CostOfGoods  float64        `json:"cost_of_goods"  dc:"Giá vốn hàng bán (SALE − SALE_RETURN trong ngày)"`
	GrossProfit  float64        `json:"gross_profit"   dc:"= net_revenue − cost_of_goods"`
	ByMethod     []*MethodTotal `json:"by_method"      dc:"Tiền thu theo phương thức (CASH, TRANSFER, CARD)"`
	TopProducts  []*TopProduct  `json:"top_products"   dc:"Top 5 sản phẩm theo số lượng thuần (đã trừ trả hàng); revenue là doanh thu thuần"`
}
