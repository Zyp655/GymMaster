package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type Shift struct {
	ID         uint      `json:"id"`
	BranchID   uint      `json:"branch_id"`
	BranchName string    `json:"branch_name"`
	StaffID    uint      `json:"staff_id"`
	StaffName  string    `json:"staff_name"`
	StaffCode  string    `json:"staff_code"`
	Position   string    `json:"position"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Note       string    `json:"note"`
}

type ShiftInput struct {
	BranchID uint   `json:"branch_id" v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	StaffID  uint   `json:"staff_id"  v:"required|min:1#Vui lòng chọn nhân viên|Vui lòng chọn nhân viên"`
	StartAt  string `json:"start_at"  v:"required#Vui lòng nhập giờ bắt đầu" dc:"RFC3339, VD: 2026-10-12T06:00:00+07:00"`
	EndAt    string `json:"end_at"    v:"required#Vui lòng nhập giờ kết thúc" dc:"RFC3339"`
	Note     string `json:"note"      v:"max-length:255#Ghi chú tối đa 255 ký tự"`
}

type ListReq struct {
	g.Meta   `path:"/shifts" method:"get" tags:"Ca làm" summary:"Lịch ca làm trong khoảng ngày" perm:"shift.view"`
	BranchID uint   `json:"branch_id" in:"query" dc:"Lọc theo chi nhánh"`
	StaffID  uint   `json:"staff_id"  in:"query" dc:"Lọc theo nhân viên"`
	From     string `json:"from"      in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn ngày bắt đầu|Ngày phải có dạng YYYY-MM-DD"`
	To       string `json:"to"        in:"query" v:"required|date-format:Y-m-d#Vui lòng chọn ngày kết thúc|Ngày phải có dạng YYYY-MM-DD"`
}
type ListRes []*Shift

type CreateReq struct {
	g.Meta `path:"/shifts" method:"post" tags:"Ca làm" summary:"Xếp ca làm" perm:"shift.manage"`
	ShiftInput
}
type CreateRes = Shift

type UpdateReq struct {
	g.Meta `path:"/shifts/{id}" method:"put" tags:"Ca làm" summary:"Sửa ca làm" perm:"shift.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	ShiftInput
}
type UpdateRes = Shift

type DeleteReq struct {
	g.Meta `path:"/shifts/{id}" method:"delete" tags:"Ca làm" summary:"Xóa ca làm" perm:"shift.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}

type CopyWeekReq struct {
	g.Meta        `path:"/shifts/copy-week" method:"post" tags:"Ca làm" summary:"Sao chép lịch ca của một tuần sang tuần khác" perm:"shift.manage"`
	BranchID      uint   `json:"branch_id"       v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
	FromWeekStart string `json:"from_week_start" v:"required|date-format:Y-m-d#Vui lòng chọn tuần nguồn|Ngày phải có dạng YYYY-MM-DD" dc:"Thứ Hai của tuần nguồn"`
	ToWeekStart   string `json:"to_week_start"   v:"required|date-format:Y-m-d#Vui lòng chọn tuần đích|Ngày phải có dạng YYYY-MM-DD" dc:"Thứ Hai của tuần đích"`
}
type CopyWeekRes struct {
	Created int `json:"created" dc:"Số ca đã tạo"`
	Skipped int `json:"skipped" dc:"Số ca bỏ qua (trùng giờ hoặc nhân viên không còn thuộc chi nhánh)"`
}
