package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type PTPackage struct {
	ID              uint      `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	SessionCount    int       `json:"session_count"`
	SessionMinutes  int       `json:"session_minutes"`
	ValidityDays    int       `json:"validity_days"`
	Price           float64   `json:"price"`
	PricePerSession float64   `json:"price_per_session" dc:"Giá mỗi buổi (làm tròn VND)"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PTPackageInput là body dùng chung cho tạo/sửa gói PT.
type PTPackageInput struct {
	Code           string  `json:"code"            v:"required|regex:^[A-Z0-9_-]{2,30}$#Vui lòng nhập mã gói|Mã gồm 2-30 ký tự in hoa, số, '-' hoặc '_'"`
	Name           string  `json:"name"            v:"required|max-length:150#Vui lòng nhập tên gói|Tên tối đa 150 ký tự"`
	Description    string  `json:"description"     v:"max-length:2000#Mô tả tối đa 2000 ký tự"`
	SessionCount   int     `json:"session_count"   v:"required|between:1,1000#Vui lòng nhập số buổi|Số buổi từ 1 đến 1000"`
	SessionMinutes int     `json:"session_minutes" v:"between:15,240#Thời lượng mỗi buổi từ 15 đến 240 phút" d:"60"`
	ValidityDays   int     `json:"validity_days"   v:"required|between:1,3650#Vui lòng nhập thời hạn (ngày)|Thời hạn từ 1 đến 3650 ngày"`
	Price          float64 `json:"price"           v:"min:0#Giá không được âm"`
	Status         string  `json:"status"          v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ" d:"ACTIVE"`
}

type ListReq struct {
	g.Meta `path:"/pt-packages" method:"get" tags:"Gói PT" summary:"Danh sách gói PT" perm:"plan.view"`
	dto.PageReq
	Status string `json:"status" in:"query" v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ" dc:"Lọc theo trạng thái"`
}
type ListRes = dto.PageRes[*PTPackage]

type OptionsReq struct {
	g.Meta `path:"/pt-packages/options" method:"get" tags:"Gói PT" summary:"Gói PT đang bán (dropdown)" perm:"plan.view,pt.sell"`
}
type OptionsRes []*PTPackage

type GetReq struct {
	g.Meta `path:"/pt-packages/{id}" method:"get" tags:"Gói PT" summary:"Chi tiết gói PT" perm:"plan.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = PTPackage

type CreateReq struct {
	g.Meta `path:"/pt-packages" method:"post" tags:"Gói PT" summary:"Thêm gói PT" perm:"plan.manage"`
	PTPackageInput
}
type CreateRes = PTPackage

type UpdateReq struct {
	g.Meta `path:"/pt-packages/{id}" method:"put" tags:"Gói PT" summary:"Cập nhật gói PT" perm:"plan.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	PTPackageInput
}
type UpdateRes = PTPackage

type UpdateStatusReq struct {
	g.Meta `path:"/pt-packages/{id}/status" method:"put" tags:"Gói PT" summary:"Đổi trạng thái gói PT" perm:"plan.manage"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Status string `json:"status" v:"required|in:ACTIVE,INACTIVE#Vui lòng chọn trạng thái|Trạng thái không hợp lệ"`
}
type UpdateStatusRes = PTPackage

type DeleteReq struct {
	g.Meta `path:"/pt-packages/{id}" method:"delete" tags:"Gói PT" summary:"Xóa gói PT (chỉ khi chưa có hợp đồng)" perm:"plan.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}
