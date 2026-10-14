package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type Plan struct {
	ID              uint      `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"            dc:"TIME | SESSION"`
	DurationDays    int       `json:"duration_days"`
	SessionCount    *int      `json:"session_count"   dc:"Chỉ có với gói SESSION"`
	Price           float64   `json:"price"`
	AccessScope     string    `json:"access_scope"    dc:"SINGLE_BRANCH | ALL_BRANCHES"`
	MaxFreezeDays   int       `json:"max_freeze_days"`
	MaxFreezeTimes  int       `json:"max_freeze_times"`
	IncludesClasses bool      `json:"includes_classes"`
	TransferFee     float64   `json:"transfer_fee"`
	Status          string    `json:"status"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PlanInput là body dùng chung cho tạo/sửa gói tập.
type PlanInput struct {
	Code            string  `json:"code"             v:"required|regex:^[A-Z0-9_-]{2,30}$#Vui lòng nhập mã gói|Mã gồm 2-30 ký tự in hoa, số, '-' hoặc '_'"`
	Name            string  `json:"name"             v:"required|max-length:150#Vui lòng nhập tên gói|Tên tối đa 150 ký tự"`
	Description     string  `json:"description"      v:"max-length:2000#Mô tả tối đa 2000 ký tự"`
	Type            string  `json:"type"             v:"required|in:TIME,SESSION#Vui lòng chọn loại gói|Loại gói không hợp lệ"`
	DurationDays    int     `json:"duration_days"    v:"required|between:1,3650#Vui lòng nhập thời hạn (ngày)|Thời hạn từ 1 đến 3650 ngày"`
	SessionCount    *int    `json:"session_count"    v:"between:1,10000#Số buổi từ 1 đến 10000"`
	Price           float64 `json:"price"            v:"min:0#Giá không được âm"`
	AccessScope     string  `json:"access_scope"     v:"in:SINGLE_BRANCH,ALL_BRANCHES#Phạm vi chi nhánh không hợp lệ" d:"SINGLE_BRANCH"`
	MaxFreezeDays   int     `json:"max_freeze_days"  v:"min:0#Số ngày bảo lưu không được âm"`
	MaxFreezeTimes  int     `json:"max_freeze_times" v:"min:0#Số lần bảo lưu không được âm"`
	IncludesClasses bool    `json:"includes_classes"`
	TransferFee     float64 `json:"transfer_fee"     v:"min:0#Phí chuyển nhượng không được âm"`
	Status          string  `json:"status"           v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ" d:"ACTIVE"`
	SortOrder       int     `json:"sort_order"`
}

type ListReq struct {
	g.Meta `path:"/membership-plans" method:"get" tags:"Gói tập" summary:"Danh sách gói tập" perm:"plan.view"`
	dto.PageReq
	Status string `json:"status" in:"query" v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ" dc:"Lọc theo trạng thái"`
	Type   string `json:"type"   in:"query" v:"in:TIME,SESSION#Loại gói không hợp lệ"     dc:"Lọc theo loại gói"`
}
type ListRes = dto.PageRes[*Plan]

type OptionsReq struct {
	g.Meta `path:"/membership-plans/options" method:"get" tags:"Gói tập" summary:"Gói tập đang bán (dropdown)" perm:"plan.view,subscription.sell"`
}
type OptionsRes []*Plan

type GetReq struct {
	g.Meta `path:"/membership-plans/{id}" method:"get" tags:"Gói tập" summary:"Chi tiết gói tập" perm:"plan.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = Plan

type CreateReq struct {
	g.Meta `path:"/membership-plans" method:"post" tags:"Gói tập" summary:"Thêm gói tập" perm:"plan.manage"`
	PlanInput
}
type CreateRes = Plan

type UpdateReq struct {
	g.Meta `path:"/membership-plans/{id}" method:"put" tags:"Gói tập" summary:"Cập nhật gói tập" perm:"plan.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	PlanInput
}
type UpdateRes = Plan

type UpdateStatusReq struct {
	g.Meta `path:"/membership-plans/{id}/status" method:"put" tags:"Gói tập" summary:"Đổi trạng thái gói tập (đang bán / ngừng bán)" perm:"plan.manage"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Status string `json:"status" v:"required|in:ACTIVE,INACTIVE#Vui lòng chọn trạng thái|Trạng thái không hợp lệ"`
}
type UpdateStatusRes = Plan

type DeleteReq struct {
	g.Meta `path:"/membership-plans/{id}" method:"delete" tags:"Gói tập" summary:"Xóa gói tập (chỉ khi chưa có đăng ký)" perm:"plan.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}
