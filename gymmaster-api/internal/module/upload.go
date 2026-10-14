package module

import (
	"github.com/gogf/gf/v2/net/ghttp"

	uploadctrl "gymmaster-api/internal/controller/upload"
	"gymmaster-api/internal/service"
)

// RegisterUpload: POST /v1/admin/uploads – ảnh dùng chung cho mọi module (hội viên, sản phẩm...).
func RegisterUpload(d *Deps, admin *ghttp.RouterGroup) {
	svc := service.NewUploadService(d.Config.Upload.Dir, d.Config.Upload.PublicURL, d.Config.Upload.MaxSizeMB)
	admin.Bind(uploadctrl.NewV1(svc))
}
