package v1

import (
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"gymmaster-api/internal/dto"
)

type AuditLog struct {
	ID         uint      `json:"id"`
	ActorID    *uint     `json:"actor_id"`   // null = hệ thống (cron)
	ActorName  string    `json:"actor_name"` // "Hệ thống" khi actor_id null
	ActorEmail string    `json:"actor_email"`
	Action     string    `json:"action"`
	Entity     string    `json:"entity"`
	EntityID   *uint     `json:"entity_id"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

type AuditLogDetail struct {
	AuditLog
	Before  json.RawMessage `json:"before"` // JSON gốc (null nếu không có)
	After   json.RawMessage `json:"after"`
	Changes []FieldChange   `json:"changes"` // các trường khác nhau giữa before/after (cấp 1) – FE hiển thị diff
}

type FieldChange struct {
	Field  string          `json:"field"`
	Before json.RawMessage `json:"before"`
	After  json.RawMessage `json:"after"`
}

type ListReq struct {
	g.Meta `path:"/audit-logs" method:"get" tags:"Nhật ký thao tác" summary:"Danh sách nhật ký thao tác" perm:"audit.view"`
	dto.PageReq
	ActorID  uint   `json:"actor_id"  in:"query" dc:"Người thao tác"`
	Entity   string `json:"entity"    in:"query" dc:"Đối tượng (branch, member, subscription, invoice...)"`
	EntityID uint   `json:"entity_id" in:"query" dc:"ID đối tượng"`
	Action   string `json:"action"    in:"query" dc:"Hành động (CREATE, UPDATE, DELETE, SELL...)"`
	From     string `json:"from"      in:"query" v:"date-format:Y-m-d#Từ ngày phải có dạng YYYY-MM-DD" dc:"Từ ngày (giờ VN)"`
	To       string `json:"to"        in:"query" v:"date-format:Y-m-d#Đến ngày phải có dạng YYYY-MM-DD" dc:"Đến ngày (giờ VN, tính cả ngày)"`
}
type ListRes = dto.PageRes[*AuditLog]

type GetReq struct {
	g.Meta `path:"/audit-logs/{id}" method:"get" tags:"Nhật ký thao tác" summary:"Chi tiết nhật ký (before/after)" perm:"audit.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = AuditLogDetail

type OptionsReq struct {
	g.Meta `path:"/audit-logs/options" method:"get" tags:"Nhật ký thao tác" summary:"Giá trị lọc (entity, action) đang có" perm:"audit.view"`
}
type OptionsRes struct {
	Entities []string `json:"entities"`
	Actions  []string `json:"actions"`
}
