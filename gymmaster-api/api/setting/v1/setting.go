package v1

import (
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type Setting struct {
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"       dc:"Giá trị JSON: số nguyên hoặc mảng số nguyên"`
	Description string          `json:"description"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ListReq struct {
	g.Meta `path:"/settings" method:"get" tags:"Cấu hình" summary:"Danh sách cấu hình nghiệp vụ" perm:"setting.manage"`
}
type ListRes []*Setting

type UpdateReq struct {
	g.Meta `path:"/settings/{key}" method:"put" tags:"Cấu hình" summary:"Cập nhật một mục cấu hình" perm:"setting.manage"`
	Key    string `json:"key"   in:"path" v:"required"`
	Value  any    `json:"value" v:"required#Vui lòng nhập giá trị"`
}
type UpdateRes = Setting
