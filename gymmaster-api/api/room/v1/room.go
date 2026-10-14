package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type Room struct {
	ID          uint      `json:"id"`
	BranchID    uint      `json:"branch_id"`
	BranchName  string    `json:"branch_name"`
	Name        string    `json:"name"`
	Capacity    int       `json:"capacity"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RoomInput struct {
	BranchID    uint   `json:"branch_id"   v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	Name        string `json:"name"        v:"required|max-length:100#Vui lòng nhập tên phòng|Tên phòng tối đa 100 ký tự"`
	Capacity    int    `json:"capacity"    v:"required|between:1,1000#Vui lòng nhập sức chứa|Sức chứa từ 1 đến 1000 người"`
	Description string `json:"description" v:"max-length:255#Mô tả tối đa 255 ký tự"`
	Status      string `json:"status"      v:"required|in:ACTIVE,INACTIVE#Vui lòng chọn trạng thái|Trạng thái không hợp lệ" d:"ACTIVE"`
}

type ListReq struct {
	g.Meta `path:"/rooms" method:"get" tags:"Phòng tập" summary:"Danh sách phòng tập" perm:"room.view"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh"`
	Status   string `json:"status"    in:"query" v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ"`
}
type ListRes = dto.PageRes[*Room]

type GetReq struct {
	g.Meta `path:"/rooms/{id}" method:"get" tags:"Phòng tập" summary:"Chi tiết phòng tập" perm:"room.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = Room

type CreateReq struct {
	g.Meta `path:"/rooms" method:"post" tags:"Phòng tập" summary:"Thêm phòng tập" perm:"room.manage"`
	RoomInput
}
type CreateRes = Room

type UpdateReq struct {
	g.Meta `path:"/rooms/{id}" method:"put" tags:"Phòng tập" summary:"Cập nhật phòng tập" perm:"room.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	RoomInput
}
type UpdateRes = Room

type DeleteReq struct {
	g.Meta `path:"/rooms/{id}" method:"delete" tags:"Phòng tập" summary:"Xóa phòng tập (xóa mềm)" perm:"room.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}

type RoomOption struct {
	ID       uint   `json:"id"`
	BranchID uint   `json:"branch_id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

type OptionsReq struct {
	g.Meta   `path:"/rooms/options" method:"get" tags:"Phòng tập" summary:"Phòng đang hoạt động (dropdown)" perm:"room.view,class.view"`
	BranchID uint `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh"`
}
type OptionsRes []*RoomOption
