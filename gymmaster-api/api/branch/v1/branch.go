package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type Branch struct {
	ID        uint      `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	OpenTime  string    `json:"open_time"`
	CloseTime string    `json:"close_time"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BranchInput là phần body dùng chung cho tạo/sửa.
type BranchInput struct {
	Code      string `json:"code"       v:"required|regex:^[A-Z0-9_-]{2,20}$#Vui lòng nhập mã chi nhánh|Mã gồm 2-20 ký tự in hoa, số, '-' hoặc '_'"`
	Name      string `json:"name"       v:"required|max-length:150#Vui lòng nhập tên chi nhánh|Tên tối đa 150 ký tự"`
	Address   string `json:"address"    v:"max-length:255#Địa chỉ tối đa 255 ký tự"`
	Phone     string `json:"phone"      v:"regex:^[0-9+ ]{8,20}$#Số điện thoại không hợp lệ"`
	Email     string `json:"email"      v:"email#Email không hợp lệ"`
	OpenTime  string `json:"open_time"  v:"required|regex:^([01][0-9]|2[0-3]):[0-5][0-9]$#Vui lòng nhập giờ mở cửa|Giờ mở cửa phải có dạng HH:mm"`
	CloseTime string `json:"close_time" v:"required|regex:^([01][0-9]|2[0-3]):[0-5][0-9]$#Vui lòng nhập giờ đóng cửa|Giờ đóng cửa phải có dạng HH:mm"`
	Status    string `json:"status"     v:"required|in:ACTIVE,INACTIVE#Vui lòng chọn trạng thái|Trạng thái không hợp lệ" d:"ACTIVE"`
	// Force: vẫn lưu giờ mở cửa mới dù còn lớp/buổi PT tương lai nằm ngoài khung giờ (chỉ dùng khi cập nhật).
	Force bool `json:"force" dc:"true = bỏ qua cảnh báo lớp/PT tương lai nằm ngoài giờ mở cửa mới"`
}

type ListReq struct {
	g.Meta `path:"/branches" method:"get" tags:"Chi nhánh" summary:"Danh sách chi nhánh" perm:"branch.view"`
	dto.PageReq
	Status string `json:"status" in:"query" v:"in:ACTIVE,INACTIVE#Trạng thái không hợp lệ" dc:"Lọc theo trạng thái"`
}
type ListRes = dto.PageRes[*Branch]

type GetReq struct {
	g.Meta `path:"/branches/{id}" method:"get" tags:"Chi nhánh" summary:"Chi tiết chi nhánh" perm:"branch.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = Branch

type CreateReq struct {
	g.Meta `path:"/branches" method:"post" tags:"Chi nhánh" summary:"Thêm chi nhánh" perm:"branch.create"`
	BranchInput
}
type CreateRes = Branch

type UpdateReq struct {
	g.Meta `path:"/branches/{id}" method:"put" tags:"Chi nhánh" summary:"Cập nhật chi nhánh" perm:"branch.update"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	BranchInput
}
type UpdateRes = Branch

type DeleteReq struct {
	g.Meta `path:"/branches/{id}" method:"delete" tags:"Chi nhánh" summary:"Xóa chi nhánh (xóa mềm)" perm:"branch.delete"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}
