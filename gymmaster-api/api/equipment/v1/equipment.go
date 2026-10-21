package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type Equipment struct {
	ID                      uint              `json:"id"`
	BranchID                uint              `json:"branch_id"`
	BranchName              string            `json:"branch_name,omitempty"`
	RoomID                  *uint             `json:"room_id"`
	RoomName                string            `json:"room_name,omitempty"`
	Code                    string            `json:"code"`
	Name                    string            `json:"name"`
	Category                string            `json:"category"`
	PurchaseDate            *string           `json:"purchase_date"`
	Price                   float64           `json:"price"`
	WarrantyUntil           *string           `json:"warranty_until"`
	Status                  string            `json:"status"`
	MaintenanceIntervalDays int               `json:"maintenance_interval_days"`
	NextMaintenanceDate     *string           `json:"next_maintenance_date"`
	ImageURL                string            `json:"image_url"`
	Note                    string            `json:"note"`
	MaintenanceLogs         []*MaintenanceLog `json:"maintenance_logs,omitempty"`
	CreatedAt               time.Time         `json:"created_at"`
	UpdatedAt               time.Time         `json:"updated_at"`
}

type EquipmentInput struct {
	BranchID                uint    `json:"branch_id"                 v:"required|min:1#Vui lòng chọn chi nhánh"`
	RoomID                  *uint   `json:"room_id"`
	Code                    string  `json:"code"                      v:"required|max-length:30#Vui lòng nhập mã thiết bị|Mã thiết bị tối đa 30 ký tự"`
	Name                    string  `json:"name"                      v:"required|max-length:150#Vui lòng nhập tên thiết bị|Tên thiết bị tối đa 150 ký tự"`
	Category                string  `json:"category"                  d:"CARDIO" v:"in:CARDIO,STRENGTH,FREE_WEIGHT,FUNCTIONAL,ACCESSORY,OTHER#Phân loại thiết bị không hợp lệ"`
	PurchaseDate            string  `json:"purchase_date"`
	Price                   float64 `json:"price"                     v:"min:0#Giá mua không được âm"`
	WarrantyUntil           string  `json:"warranty_until"`
	Status                  string  `json:"status"                    d:"ACTIVE" v:"in:ACTIVE,BROKEN,MAINTENANCE,RETIRED#Trạng thái thiết bị không hợp lệ"`
	MaintenanceIntervalDays int     `json:"maintenance_interval_days" d:"90"     v:"min:0#Chu kỳ bảo trì không được âm"`
	NextMaintenanceDate     string  `json:"next_maintenance_date"`
	ImageURL                string  `json:"image_url"                 v:"max-length:500"`
	Note                    string  `json:"note"`
}

type MaintenanceLog struct {
	ID            uint       `json:"id"`
	EquipmentID   uint       `json:"equipment_id"`
	EquipmentCode string     `json:"equipment_code,omitempty"`
	EquipmentName string     `json:"equipment_name,omitempty"`
	BranchID      uint       `json:"branch_id,omitempty"`
	BranchName    string     `json:"branch_name,omitempty"`
	Type          string     `json:"type"`
	ReportedBy    *uint      `json:"reported_by"`
	ReporterName  string     `json:"reporter_name,omitempty"`
	Description   string     `json:"description"`
	Cost          float64    `json:"cost"`
	Status        string     `json:"status"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type MaintenanceLogInput struct {
	EquipmentID uint    `json:"equipment_id"`
	Type        string  `json:"type"        d:"PERIODIC" v:"in:PERIODIC,REPAIR#Loại bảo trì không hợp lệ"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"        v:"min:0#Chi phí không được âm"`
	Status      string  `json:"status"      d:"OPEN"     v:"in:OPEN,IN_PROGRESS,DONE#Trạng thái không hợp lệ"`
	StartedAt   string  `json:"started_at"`
	FinishedAt  string  `json:"finished_at"`
}

type MaintenanceBranchCost struct {
	BranchID   uint    `json:"branch_id"`
	BranchName string  `json:"branch_name"`
	TotalCost  float64 `json:"total_cost"`
	TotalLogs  int64   `json:"total_logs"`
}

type MaintenanceMonthCost struct {
	Month     string  `json:"month"`
	TotalCost float64 `json:"total_cost"`
	TotalLogs int64   `json:"total_logs"`
}

// ---- Thiết bị (/equipments) -----------------------------------------------------

type ListEquipmentsReq struct {
	g.Meta `path:"/equipments" method:"get" tags:"Thiết bị" summary:"Danh sách thiết bị phòng tập" perm:"equipment.view,equipment.manage"`
	dto.PageReq
	BranchID uint   `json:"branch_id" in:"query"`
	RoomID   uint   `json:"room_id"   in:"query"`
	Category string `json:"category"  in:"query" v:"in:CARDIO,STRENGTH,FREE_WEIGHT,FUNCTIONAL,ACCESSORY,OTHER#Phân loại không hợp lệ"`
	Status   string `json:"status"    in:"query" v:"in:ACTIVE,BROKEN,MAINTENANCE,RETIRED#Trạng thái không hợp lệ"`
	Search   string `json:"search"    in:"query"`
}
type ListEquipmentsRes = dto.PageRes[*Equipment]

type GetEquipmentReq struct {
	g.Meta `path:"/equipments/{id}" method:"get" tags:"Thiết bị" summary:"Chi tiết thiết bị & lịch sử bảo trì" perm:"equipment.view,equipment.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetEquipmentRes = Equipment

type CreateEquipmentReq struct {
	g.Meta `path:"/equipments" method:"post" tags:"Thiết bị" summary:"Thêm thiết bị mới" perm:"equipment.manage"`
	EquipmentInput
}
type CreateEquipmentRes = Equipment

type UpdateEquipmentReq struct {
	g.Meta `path:"/equipments/{id}" method:"put" tags:"Thiết bị" summary:"Cập nhật thiết bị" perm:"equipment.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	EquipmentInput
}
type UpdateEquipmentRes = Equipment

type DeleteEquipmentReq struct {
	g.Meta `path:"/equipments/{id}" method:"delete" tags:"Thiết bị" summary:"Xóa thiết bị" perm:"equipment.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteEquipmentRes struct{}

type ReportIssueReq struct {
	g.Meta      `path:"/equipments/{id}/report-issue" method:"post" tags:"Thiết bị" summary:"Báo hỏng thiết bị" perm:"equipment.view,equipment.manage"`
	ID          uint   `json:"id"          in:"path" v:"required|min:1"`
	Description string `json:"description" v:"required#Vui lòng nhập mô tả sự cố"`
}
type ReportIssueRes = MaintenanceLog

// ---- Bảo trì (/maintenance-logs) ------------------------------------------------

type ListMaintenanceLogsReq struct {
	g.Meta `path:"/maintenance-logs" method:"get" tags:"Thiết bị" summary:"Danh sách phiếu bảo trì / sửa chữa" perm:"equipment.view,equipment.manage"`
	dto.PageReq
	EquipmentID uint   `json:"equipment_id" in:"query"`
	BranchID    uint   `json:"branch_id"    in:"query"`
	Type        string `json:"type"         in:"query" v:"in:PERIODIC,REPAIR"`
	Status      string `json:"status"       in:"query" v:"in:OPEN,IN_PROGRESS,DONE"`
}
type ListMaintenanceLogsRes = dto.PageRes[*MaintenanceLog]

type CreateMaintenanceLogReq struct {
	g.Meta `path:"/maintenance-logs" method:"post" tags:"Thiết bị" summary:"Tạo phiếu bảo trì / sửa chữa" perm:"equipment.manage"`
	MaintenanceLogInput
}
type CreateMaintenanceLogRes = MaintenanceLog

type UpdateMaintenanceLogReq struct {
	g.Meta `path:"/maintenance-logs/{id}" method:"put" tags:"Thiết bị" summary:"Cập nhật phiếu bảo trì" perm:"equipment.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
	MaintenanceLogInput
}
type UpdateMaintenanceLogRes = MaintenanceLog

type DeleteMaintenanceLogReq struct {
	g.Meta `path:"/maintenance-logs/{id}" method:"delete" tags:"Thiết bị" summary:"Xóa phiếu bảo trì" perm:"equipment.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteMaintenanceLogRes struct{}

type MaintenanceSummaryReq struct {
	g.Meta   `path:"/maintenance-logs/summary" method:"get" tags:"Thiết bị" summary:"Thống kê chi phí bảo trì theo chi nhánh/tháng" perm:"equipment.view,equipment.manage"`
	BranchID uint   `json:"branch_id" in:"query"`
	From     string `json:"from"      in:"query"`
	To       string `json:"to"        in:"query"`
}
type MaintenanceSummaryRes struct {
	TotalCost float64                  `json:"total_cost"`
	TotalLogs int64                    `json:"total_logs"`
	ByBranch  []*MaintenanceBranchCost `json:"by_branch"`
	ByMonth   []*MaintenanceMonthCost  `json:"by_month"`
}
