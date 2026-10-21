package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type LockerItem struct {
	ID         uint       `json:"id"`
	BranchID   uint       `json:"branch_id"`
	BranchName string     `json:"branch_name,omitempty"`
	Code       string     `json:"code"`
	Zone       string     `json:"zone"`
	Size       string     `json:"size"`
	MonthlyFee float64    `json:"monthly_fee"`
	Status     string     `json:"status"`
	Note       string     `json:"note"`
	RentalID   *uint      `json:"rental_id,omitempty"`
	MemberID   *uint      `json:"member_id,omitempty"`
	MemberName string     `json:"member_name,omitempty"`
	MemberCode string     `json:"member_code,omitempty"`
	RentalType string     `json:"rental_type,omitempty"`
	StartAt    *time.Time `json:"start_at,omitempty"`
	EndAt      *time.Time `json:"end_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type LockerRentalItem struct {
	ID         uint       `json:"id"`
	LockerID   uint       `json:"locker_id"`
	LockerCode string     `json:"locker_code,omitempty"`
	Zone       string     `json:"zone,omitempty"`
	Size       string     `json:"size,omitempty"`
	BranchID   uint       `json:"branch_id,omitempty"`
	BranchName string     `json:"branch_name,omitempty"`
	MemberID   uint       `json:"member_id"`
	MemberCode string     `json:"member_code,omitempty"`
	MemberName string     `json:"member_name,omitempty"`
	Type       string     `json:"type"`
	StartAt    time.Time  `json:"start_at"`
	EndAt      *time.Time `json:"end_at"`
	InvoiceID  *uint      `json:"invoice_id"`
	Price      float64    `json:"price"`
	Status     string     `json:"status"`
	Note       string     `json:"note"`
	CreatedBy  *uint      `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type LockerBoardStats struct {
	Available int `json:"available"`
	Occupied  int `json:"occupied"`
	Rented    int `json:"rented"`
	Broken    int `json:"broken"`
	Total     int `json:"total"`
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListLockersReq struct {
	g.Meta   `path:"/lockers" method:"get" tags:"Tủ đồ" summary:"Danh sách tủ đồ" perm:"locker.view,locker.manage"`
	BranchID uint   `json:"branch_id" in:"query"`
	Zone     string `json:"zone"      in:"query"`
	Size     string `json:"size"      in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListLockersRes []*LockerItem

type CreateLockerReq struct {
	g.Meta     `path:"/lockers" method:"post" tags:"Tủ đồ" summary:"Tạo tủ đồ mới" perm:"locker.manage"`
	BranchID   uint     `json:"branch_id"   v:"required|min:1"`
	Code       string   `json:"code"        v:"required"`
	Zone       string   `json:"zone"        d:"Khu A"`
	Size       string   `json:"size"        d:"STANDARD" v:"in:SMALL,STANDARD,LARGE"`
	MonthlyFee *float64 `json:"monthly_fee" v:"min:0" dc:"Bỏ trống = 150.000đ; 0 = tủ miễn phí"`
	Status     string   `json:"status"      d:"AVAILABLE" v:"in:AVAILABLE,OCCUPIED,RENTED,BROKEN"`
	Note       string   `json:"note"`
}
type CreateLockerRes = LockerItem

type UpdateLockerReq struct {
	g.Meta     `path:"/lockers/{id}" method:"put" tags:"Tủ đồ" summary:"Cập nhật tủ đồ" perm:"locker.manage"`
	ID         uint     `json:"id"          in:"path" v:"required|min:1"`
	Code       string   `json:"code"`
	Zone       string   `json:"zone"`
	Size       string   `json:"size"`
	MonthlyFee *float64 `json:"monthly_fee"`
	Status     string   `json:"status"`
	Note       *string  `json:"note"`
}
type UpdateLockerRes = LockerItem

type DeleteLockerReq struct {
	g.Meta `path:"/lockers/{id}" method:"delete" tags:"Tủ đồ" summary:"Xóa tủ đồ" perm:"locker.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteLockerRes struct{}

type GetLockerBoardReq struct {
	g.Meta   `path:"/lockers/board" method:"get" tags:"Tủ đồ" summary:"Sơ đồ tủ đồ & thống kê" perm:"locker.view,locker.manage"`
	BranchID uint `json:"branch_id" in:"query"`
}
type GetLockerBoardRes struct {
	Available int               `json:"available"`
	Occupied  int               `json:"occupied"`
	Rented    int               `json:"rented"`
	Broken    int               `json:"broken"`
	Total     int               `json:"total"`
	Stats     *LockerBoardStats `json:"stats,omitempty"`
	Lockers   []*LockerItem     `json:"lockers"`
	Items     []*LockerItem     `json:"items,omitempty"`
}

type AssignLockerReq struct {
	g.Meta   `path:"/lockers/{id}/assign" method:"post" tags:"Tủ đồ" summary:"Gán tủ theo lượt hoặc thuê theo tháng" perm:"locker.manage"`
	ID       uint   `json:"id"        in:"path" v:"required|min:1"`
	MemberID uint   `json:"member_id" v:"required|min:1"`
	Type     string `json:"type"      d:"DAILY" v:"in:DAILY,MONTHLY"`
	Months   int    `json:"months"    d:"1"`
	Note     string `json:"note"`
}
type AssignLockerRes = LockerRentalItem

type ReleaseLockerReq struct {
	g.Meta `path:"/lockers/{id}/release" method:"post" tags:"Tủ đồ" summary:"Giải phóng / trả tủ đồ" perm:"locker.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type ReleaseLockerRes = LockerItem

type CreateLockerRentalReq struct {
	g.Meta   `path:"/locker-rentals" method:"post" tags:"Tủ đồ" summary:"Tạo lượt thuê tủ đồ" perm:"locker.manage"`
	LockerID uint   `json:"locker_id" v:"required|min:1"`
	MemberID uint   `json:"member_id" v:"required|min:1"`
	Type     string `json:"type"      d:"MONTHLY" v:"in:DAILY,MONTHLY"`
	Months   int    `json:"months"    d:"1"`
	Note     string `json:"note"`
}
type CreateLockerRentalRes = LockerRentalItem

type ListLockerRentalsReq struct {
	g.Meta   `path:"/locker-rentals" method:"get" tags:"Tủ đồ" summary:"Danh sách lượt sử dụng & thuê tủ" perm:"locker.view,locker.manage"`
	BranchID uint   `json:"branch_id" in:"query"`
	LockerID uint   `json:"locker_id" in:"query"`
	MemberID uint   `json:"member_id" in:"query"`
	Type     string `json:"type"      in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListLockerRentalsRes []*LockerRentalItem

// ---- Member (/v1/member) --------------------------------------------------------

type MemberLockersReq struct {
	g.Meta `path:"/locker" method:"get" tags:"Cổng hội viên" summary:"Danh sách tủ đồ đang sử dụng của tôi" role:"MEMBER"`
}
type MemberLockersRes []*LockerRentalItem
