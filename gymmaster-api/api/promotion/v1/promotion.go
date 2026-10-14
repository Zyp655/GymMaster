package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

// Trạng thái hiệu lực (tính theo thời điểm hiện tại) trả về trong trường `validity`.
const (
	ValidityActive    = "active"    // đang áp dụng được
	ValidityUpcoming  = "upcoming"  // chưa đến ngày bắt đầu
	ValidityExpired   = "expired"   // đã quá ngày kết thúc
	ValidityExhausted = "exhausted" // đã hết lượt sử dụng
	ValidityInactive  = "inactive"  // đã tắt (status = INACTIVE)
)

type Promotion struct {
	ID          uint      `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Type        string    `json:"type"         dc:"PERCENT | FIXED"`
	Value       float64   `json:"value"`
	MaxDiscount *float64  `json:"max_discount" dc:"Trần giảm cho loại PERCENT"`
	MinAmount   float64   `json:"min_amount"`
	AppliesTo   string    `json:"applies_to"   dc:"ALL | MEMBERSHIP | PT | PRODUCT"`
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	UsageLimit  *int      `json:"usage_limit"  dc:"null = không giới hạn"`
	UsedCount   int       `json:"used_count"`
	Status      string    `json:"status"`
	Validity    string    `json:"validity"     dc:"active | upcoming | expired | exhausted | inactive"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PromotionInput là body dùng chung cho tạo/sửa khuyến mãi.
type PromotionInput struct {
	Code        string    `json:"code"         v:"required|regex:^[A-Za-z0-9_-]{3,30}$#Vui lòng nhập mã khuyến mãi|Mã gồm 3-30 ký tự chữ, số, '-' hoặc '_'"`
	Name        string    `json:"name"         v:"required|max-length:150#Vui lòng nhập tên chương trình|Tên tối đa 150 ký tự"`
	Type        string    `json:"type"         v:"required|in:PERCENT,FIXED#Vui lòng chọn loại giảm giá|Loại giảm giá không hợp lệ"`
	Value       float64   `json:"value"        v:"required#Vui lòng nhập giá trị giảm"`
	MaxDiscount *float64  `json:"max_discount"`
	MinAmount   float64   `json:"min_amount"   v:"min:0#Giá trị đơn tối thiểu không được âm"`
	AppliesTo   string    `json:"applies_to"   v:"in:ALL,MEMBERSHIP,PT,PRODUCT#Phạm vi áp dụng không hợp lệ" d:"ALL"`
	StartAt     time.Time `json:"start_at"     v:"required#Vui lòng chọn thời gian bắt đầu"`
	EndAt       time.Time `json:"end_at"       v:"required#Vui lòng chọn thời gian kết thúc"`
	UsageLimit  *int      `json:"usage_limit"  v:"min:1#Giới hạn lượt dùng phải lớn hơn 0"`
	Status      string    `json:"status"       v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ" d:"ACTIVE"`
}

type ListReq struct {
	g.Meta `path:"/promotions" method:"get" tags:"Khuyến mãi" summary:"Danh sách khuyến mãi" perm:"promotion.view"`
	dto.PageReq
	Status    string `json:"status"     in:"query" v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ"                       dc:"Lọc theo trạng thái"`
	Validity  string `json:"validity"   in:"query" v:"in:active,upcoming,expired,inactive#Trạng thái hiệu lực không hợp lệ" dc:"active | upcoming | expired (gồm hết lượt) | inactive"`
	AppliesTo string `json:"applies_to" in:"query" v:"in:ALL,MEMBERSHIP,PT,PRODUCT#Phạm vi áp dụng không hợp lệ"           dc:"Lọc theo phạm vi áp dụng"`
}
type ListRes = dto.PageRes[*Promotion]

type GetReq struct {
	g.Meta `path:"/promotions/{id}" method:"get" tags:"Khuyến mãi" summary:"Chi tiết khuyến mãi" perm:"promotion.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = Promotion

type CreateReq struct {
	g.Meta `path:"/promotions" method:"post" tags:"Khuyến mãi" summary:"Thêm khuyến mãi" perm:"promotion.manage"`
	PromotionInput
}
type CreateRes = Promotion

type UpdateReq struct {
	g.Meta `path:"/promotions/{id}" method:"put" tags:"Khuyến mãi" summary:"Cập nhật khuyến mãi" perm:"promotion.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	PromotionInput
}
type UpdateRes = Promotion

type DeleteReq struct {
	g.Meta `path:"/promotions/{id}" method:"delete" tags:"Khuyến mãi" summary:"Xóa khuyến mãi (chỉ khi chưa được dùng)" perm:"promotion.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}

type CheckReq struct {
	g.Meta    `path:"/promotions/check" method:"post" tags:"Khuyến mãi" summary:"Kiểm tra mã khuyến mãi và tính số tiền giảm" perm:"subscription.sell,pt.sell,pos.sell,promotion.view"`
	Code      string  `json:"code"       v:"required#Vui lòng nhập mã khuyến mãi"`
	AppliesTo string  `json:"applies_to" v:"required|in:MEMBERSHIP,PT,PRODUCT#Vui lòng chọn loại đơn|Loại đơn không hợp lệ"`
	Amount    float64 `json:"amount"     v:"min:0#Giá trị đơn không được âm"`
}
type CheckRes struct {
	Promotion   *Promotion `json:"promotion"`
	Discount    float64    `json:"discount"`
	FinalAmount float64    `json:"final_amount"`
}
