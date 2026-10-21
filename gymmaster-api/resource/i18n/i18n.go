// Package i18n nhúng (go:embed) các file ngôn ngữ trong resource/i18n vào binary và nạp vào
// i18n manager mặc định của GoFrame, để thông báo lỗi validate (gvalid) mặc định là tiếng Việt.
// Nhúng vào binary để không phụ thuộc thư mục làm việc khi chạy (Docker image chỉ chứa file binary).
package i18n

import (
	"archive/zip"
	"bytes"
	"embed"
	"io/fs"
	"path"
	"time"

	"github.com/gogf/gf/v2/encoding/gcompress"
	"github.com/gogf/gf/v2/i18n/gi18n"
	"github.com/gogf/gf/v2/os/gres"
)

//go:embed vi/*.toml
var files embed.FS

// resPath là thư mục ảo trong gres chứa file ngôn ngữ (mỗi thư mục con là một ngôn ngữ).
const resPath = "resource/i18n"

// DefaultLanguage là ngôn ngữ mặc định cho thông báo lỗi validate.
const DefaultLanguage = "vi"

// Init nạp file ngôn ngữ nhúng vào gres rồi cấu hình gi18n.Instance() (gvalid dùng instance này).
func Init() error {
	packed, err := pack()
	if err != nil {
		return err
	}
	if err := gres.Add(string(packed)); err != nil {
		return err
	}
	m := gi18n.Instance()
	if err := m.SetPath(resPath); err != nil {
		return err
	}
	m.SetLanguage(DefaultLanguage)
	return nil
}

// pack tạo nội dung theo định dạng gres (zip đã gzip) từ embed.FS.
func pack() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	now := time.Now()
	dirs := map[string]bool{}
	addDir := func(name string) error {
		if dirs[name] {
			return nil
		}
		dirs[name] = true
		h := &zip.FileHeader{Name: name, Modified: now}
		h.SetMode(fs.ModeDir | 0o755)
		_, err := zw.CreateHeader(h)
		return err
	}
	for _, d := range []string{"resource", resPath} {
		if err := addDir(d); err != nil {
			return nil, err
		}
	}
	err := fs.WalkDir(files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == "." {
			return err
		}
		name := path.Join(resPath, p)
		if d.IsDir() {
			return addDir(name)
		}
		content, err := files.ReadFile(p)
		if err != nil {
			return err
		}
		h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: now}
		h.SetMode(0o644)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = w.Write(content)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return gcompress.Gzip(buf.Bytes())
}
