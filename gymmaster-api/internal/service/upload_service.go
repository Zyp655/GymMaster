package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"gymmaster-api/internal/consts"
	"gymmaster-api/internal/pkg/reqctx"
	"gymmaster-api/internal/pkg/rescode"
)

// UploadService lưu ảnh tải lên vào thư mục Upload.Dir/<yyyy>/<mm>/<uuid>.<ext>
// (được serve tĩnh tại /uploads) và trả URL công khai.
type UploadService interface {
	SaveImage(ctx context.Context, size int64, file io.Reader) (url string, err error)
}

type uploadService struct {
	dir       string
	publicURL string
	maxBytes  int64
	now       func() time.Time
}

func NewUploadService(dir, publicURL string, maxSizeMB int) UploadService {
	if maxSizeMB <= 0 {
		maxSizeMB = 5
	}
	return &uploadService{
		dir:       dir,
		publicURL: strings.TrimSuffix(publicURL, "/"),
		maxBytes:  int64(maxSizeMB) << 20,
		now:       time.Now,
	}
}

// Chỉ nhận ảnh – phần mở rộng lấy theo nội dung thật, không tin tên file của client.
var uploadImageExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

func (s *uploadService) SaveImage(ctx context.Context, size int64, file io.Reader) (string, error) {
	if !uploadIsStaff(ctx) {
		return "", rescode.New(rescode.PermissionDenied, "Bạn không có quyền tải tệp lên")
	}
	if size > s.maxBytes {
		return "", rescode.New(rescode.BadRequest, fmt.Sprintf("Ảnh vượt quá dung lượng cho phép (%d MB)", s.maxBytes>>20))
	}
	// Đọc tối đa maxBytes+1 để phát hiện file lớn hơn khai báo.
	data, err := io.ReadAll(io.LimitReader(file, s.maxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > s.maxBytes {
		return "", rescode.New(rescode.BadRequest, fmt.Sprintf("Ảnh vượt quá dung lượng cho phép (%d MB)", s.maxBytes>>20))
	}
	if len(data) == 0 {
		return "", rescode.New(rescode.BadRequest, "Tệp rỗng")
	}
	ext, ok := uploadImageExt[http.DetectContentType(data)]
	if !ok {
		return "", rescode.New(rescode.BadRequest, "Chỉ chấp nhận ảnh JPG, PNG hoặc WEBP")
	}

	now := s.now()
	rel := fmt.Sprintf("%04d/%02d/%s.%s", now.Year(), int(now.Month()), uuid.NewString(), ext)
	full := filepath.Join(s.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return s.publicURL + "/" + rel, nil
}

// uploadIsStaff: người dùng có ít nhất một role khác MEMBER (tài khoản cổng hội viên không được tải lên).
func uploadIsStaff(ctx context.Context) bool {
	p := reqctx.Auth(ctx)
	if p == nil {
		return false
	}
	for _, r := range p.Roles {
		if r != consts.RoleMember {
			return true
		}
	}
	return false
}
