package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// UploadReq: multipart/form-data, field `file`. Chỉ nhận ảnh JPG/PNG/WEBP, dung lượng ≤ UPLOAD_MAX_SIZE_MB.
// Không khai báo perm: mọi nhân viên đã đăng nhập đều được tải ảnh (tài khoản chỉ có role MEMBER bị chặn ở service).
type UploadReq struct {
	g.Meta `path:"/uploads" method:"post" mime:"multipart/form-data" tags:"Tải tệp" summary:"Tải ảnh lên (jpg/png/webp)"`
	File   *ghttp.UploadFile `json:"file" type:"file" v:"required#Vui lòng chọn tệp ảnh" dc:"Tệp ảnh"`
}

type UploadRes struct {
	URL string `json:"url" dc:"URL công khai của ảnh"`
}
