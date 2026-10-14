package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type ClassType struct {
	ID                 uint      `json:"id"`
	Code               string    `json:"code"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Color              string    `json:"color"`
	DefaultDurationMin int       `json:"default_duration_min"`
	DefaultCapacity    int       `json:"default_capacity"`
	Level              string    `json:"level"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ClassTypeInput struct {
	Code               string `json:"code"                 v:"required|max-length:30|regex:^[A-Za-z0-9_-]+$#Vui lòng nhập mã loại lớp|Mã tối đa 30 ký tự|Mã chỉ gồm chữ, số, '-' và '_'"`
	Name               string `json:"name"                 v:"required|max-length:100#Vui lòng nhập tên loại lớp|Tên tối đa 100 ký tự"`
	Description        string `json:"description"`
	Color              string `json:"color"                v:"regex:^#[0-9A-Fa-f]{6}$#Màu phải có dạng #RRGGBB" dc:"Mặc định #F97316"`
	DefaultDurationMin int    `json:"default_duration_min" v:"between:15,240#Thời lượng mặc định từ 15 đến 240 phút" dc:"Mặc định 60"`
	DefaultCapacity    int    `json:"default_capacity"     v:"between:1,500#Sức chứa mặc định từ 1 đến 500" dc:"Mặc định 20"`
	Level              string `json:"level"                v:"in:ALL,BEGINNER,INTERMEDIATE,ADVANCED#Cấp độ không hợp lệ" dc:"ALL | BEGINNER | INTERMEDIATE | ADVANCED (mặc định ALL)"`
	Status             string `json:"status"               v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ" dc:"ACTIVE | INACTIVE (mặc định ACTIVE)"`
}

type ListTypeReq struct {
	g.Meta `path:"/class-types" method:"get" tags:"Lớp nhóm" summary:"Danh sách loại lớp" perm:"class.view"`
	dto.PageReq
	Status string `json:"status" in:"query" v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ"`
	Level  string `json:"level"  in:"query" v:"in:ALL,BEGINNER,INTERMEDIATE,ADVANCED#Cấp độ không hợp lệ"`
}
type ListTypeRes = dto.PageRes[*ClassType]

type TypeOptionsReq struct {
	g.Meta `path:"/class-types/options" method:"get" tags:"Lớp nhóm" summary:"Loại lớp đang hoạt động (dropdown)" perm:"class.view"`
}
type ClassTypeOption struct {
	ID                 uint   `json:"id"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	Color              string `json:"color"`
	Level              string `json:"level"`
	DefaultDurationMin int    `json:"default_duration_min"`
	DefaultCapacity    int    `json:"default_capacity"`
}
type TypeOptionsRes []*ClassTypeOption

type GetTypeReq struct {
	g.Meta `path:"/class-types/{id}" method:"get" tags:"Lớp nhóm" summary:"Chi tiết loại lớp" perm:"class.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetTypeRes = ClassType

type CreateTypeReq struct {
	g.Meta `path:"/class-types" method:"post" tags:"Lớp nhóm" summary:"Thêm loại lớp" perm:"class.manage"`
	ClassTypeInput
}
type CreateTypeRes = ClassType

type UpdateTypeReq struct {
	g.Meta `path:"/class-types/{id}" method:"put" tags:"Lớp nhóm" summary:"Sửa loại lớp" perm:"class.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ClassTypeInput
}
type UpdateTypeRes = ClassType

type DeleteTypeReq struct {
	g.Meta `path:"/class-types/{id}" method:"delete" tags:"Lớp nhóm" summary:"Xóa loại lớp (chặn nếu còn lịch sắp diễn ra)" perm:"class.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteTypeRes struct{}
