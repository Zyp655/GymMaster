package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ImportTemplateReq trả thẳng file .xlsx (không bọc envelope JSON).
type ImportTemplateReq struct {
	g.Meta `path:"/members/import-template" method:"get" tags:"Hội viên" summary:"Tải file Excel mẫu để nhập hội viên" perm:"member.import"`
}
type ImportTemplateRes struct{}

type ImportReq struct {
	g.Meta   `path:"/members/import" method:"post" mime:"multipart/form-data" tags:"Hội viên" summary:"Nhập hội viên từ Excel" perm:"member.import"`
	File     *ghttp.UploadFile `json:"file"      type:"file" v:"required#Vui lòng chọn file Excel (.xlsx)"`
	BranchID uint              `json:"branch_id" v:"required|min:1#Vui lòng chọn chi nhánh|Vui lòng chọn chi nhánh"`
}

type ImportError struct {
	Row     int    `json:"row"     dc:"Số dòng trong file Excel (dòng tiêu đề = 1)"`
	Message string `json:"message"`
}

type ImportRes struct {
	Created int           `json:"created" dc:"Số hội viên đã tạo"`
	Skipped int           `json:"skipped" dc:"Số dòng bị bỏ qua"`
	Errors  []ImportError `json:"errors"  dc:"Lý do bỏ qua từng dòng"`
}
