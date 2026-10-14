package v1

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type AccessDeviceItem struct {
	ID         uint       `json:"id"`
	BranchID   uint       `json:"branch_id"`
	BranchName string     `json:"branch_name,omitempty"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ---- Admin (/v1/admin) ----------------------------------------------------------

type ListAccessDevicesReq struct {
	g.Meta   `path:"/access-devices" method:"get" tags:"Thiết bị & Kiosk" summary:"Danh sách thiết bị kiểm soát ra vào" perm:"device.view,device.manage"`
	BranchID uint   `json:"branch_id" in:"query"`
	Status   string `json:"status"    in:"query"`
}
type ListAccessDevicesRes []*AccessDeviceItem

type CreateAccessDeviceReq struct {
	g.Meta   `path:"/access-devices" method:"post" tags:"Thiết bị & Kiosk" summary:"Thêm thiết bị kiểm soát ra vào" perm:"device.manage"`
	BranchID uint   `json:"branch_id" v:"required|min:1"`
	Code     string `json:"code"      v:"required"`
	Name     string `json:"name"      v:"required"`
	Type     string `json:"type"      d:"KIOSK" v:"in:TURNSTILE,DOOR,KIOSK"`
}
type CreateAccessDeviceRes struct {
	Device    *AccessDeviceItem `json:"device"`
	RawAPIKey string            `json:"raw_api_key"`
}

type UpdateAccessDeviceReq struct {
	g.Meta `path:"/access-devices/{id}" method:"put" tags:"Thiết bị & Kiosk" summary:"Cập nhật thiết bị kiểm soát ra vào" perm:"device.manage"`
	ID     uint   `json:"id"     in:"path" v:"required|min:1"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status" v:"in:ACTIVE,OFFLINE,DISABLED"`
}
type UpdateAccessDeviceRes = AccessDeviceItem

type DeleteAccessDeviceReq struct {
	g.Meta `path:"/access-devices/{id}" method:"delete" tags:"Thiết bị & Kiosk" summary:"Xóa thiết bị kiểm soát ra vào" perm:"device.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteAccessDeviceRes struct{}

type RotateDeviceKeyReq struct {
	g.Meta `path:"/access-devices/{id}/rotate-key" method:"post" tags:"Thiết bị & Kiosk" summary:"Cấp lại API key cho thiết bị" perm:"device.manage"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type RotateDeviceKeyRes struct {
	Device    *AccessDeviceItem `json:"device"`
	RawAPIKey string            `json:"raw_api_key"`
}

// ---- Device Public (/v1/public & /v1/device) ------------------------------------

type DeviceCheckinReq struct {
	g.Meta    `path:"/device/checkin" method:"post" tags:"Thiết bị & Kiosk" summary:"Quét QR check-in qua thiết bị hoặc Kiosk"`
	DeviceKey string `json:"device_key"`
	QRToken   string `json:"qr_token"   v:"required"`
}
type DeviceCheckinRes struct {
	Allow      bool   `json:"allow"`
	Result     string `json:"result"`
	DenyReason string `json:"deny_reason,omitempty"`
	Message    string `json:"message"`
	MemberName string `json:"member_name,omitempty"`
	MemberCode string `json:"member_code,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	PlanName   string `json:"plan_name,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
	BranchName string `json:"branch_name,omitempty"`
}

type DeviceHeartbeatReq struct {
	g.Meta    `path:"/device/heartbeat" method:"post" tags:"Thiết bị & Kiosk" summary:"Báo trạng thái hoạt động (heartbeat) từ thiết bị"`
	DeviceKey string `json:"device_key"`
}
type DeviceHeartbeatRes struct {
	Device     *AccessDeviceItem `json:"device"`
	ServerTime time.Time         `json:"server_time"`
}
