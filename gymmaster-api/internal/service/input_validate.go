package service

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"path"
	"strings"
	"time"

	"gymmaster-api/internal/pkg/rescode"
)

// Bộ kiểm tra dữ liệu nhập dùng chung (email, URL ảnh) – dùng cho lead, hội viên...

// validEmailStrict: email dạng a@b.c (không tên hiển thị, không khoảng trắng), tối đa 150 ký tự. Rỗng → true.
func validEmailStrict(raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > 150 || strings.ContainsAny(raw, " \t\r\n<>\"'(),;:\\") {
		return false
	}
	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address != raw {
		return false
	}
	at := strings.LastIndex(raw, "@")
	domain := raw[at+1:]
	return at > 0 && strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// validImageURLStrict: URL ảnh an toàn để gắn vào thuộc tính src – http(s) có host hoặc /uploads/... (đường dẫn tải lên
// của hệ thống); không chứa khoảng trắng, dấu nháy, < >, `, \, không có ".." hoặc thông tin đăng nhập. Rỗng → true.
func validImageURLStrict(raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > 500 || strings.ContainsAny(raw, " \t\r\n\"'<>`\\{}|^") || strings.Contains(raw, "..") {
		return false
	}
	for _, r := range raw {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return false
	}
	if strings.HasPrefix(raw, "/uploads/") {
		_, ok := safeUploadPath(raw)
		return ok
	}
	scheme := strings.ToLower(u.Scheme)
	return (scheme == "http" || scheme == "https") && u.Hostname() != ""
}

// uploadPathMaxDecode: số lần giải mã %XX tối đa – chuỗi còn thay đổi sau ngần ấy lần (mã hóa lồng nhiều lớp) bị từ chối.
const uploadPathMaxDecode = 4

// safeUploadPath (B9) kiểm tra đường dẫn ảnh/tệp đã tải lên của hệ thống "/uploads/...": chỉ ký tự ASCII hiển thị
// (không khoảng trắng, nháy, < >, \ ...), không scheme/host/query/fragment; giải mã %XX LẶP tới khi ổn định
// (chặn %2e%2e, %252e%252e, ..%2f) rồi chuẩn hóa – đường dẫn sau giải mã phải bằng chính nó sau path.Clean
// (không "..", ".", "//", "/" cuối) và vẫn nằm dưới /uploads/. Trả về đường dẫn đã giải mã khi hợp lệ.
func safeUploadPath(raw string) (string, bool) {
	const prefix = "/uploads/"
	if !strings.HasPrefix(raw, prefix) || len(raw) > 500 || strings.ContainsAny(raw, "\"'<>`\\{}|^") {
		return "", false
	}
	for _, r := range raw {
		if r < 0x21 || r > 0x7e {
			return "", false
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", false
	}
	decoded := raw
	stable := false
	for range uploadPathMaxDecode {
		next, err := url.PathUnescape(decoded)
		if err != nil {
			return "", false
		}
		if next == decoded {
			stable = true
			break
		}
		decoded = next
	}
	if !stable {
		return "", false
	}
	for _, r := range decoded {
		if r < 0x21 || r == 0x7f {
			return "", false
		}
	}
	if strings.ContainsAny(decoded, "\"'<>`\\?#") || strings.Contains(decoded, "..") {
		return "", false
	}
	if path.Clean(decoded) != decoded || !strings.HasPrefix(decoded, prefix) || len(decoded) <= len(prefix) {
		return "", false
	}
	return decoded, true
}

// validAttachmentURL: đường dẫn tệp đính kèm (tin nhắn, chứng từ chi) – rỗng, tệp đã tải lên của hệ thống
// (/uploads/... hoặc UPLOAD_PUBLIC_URL/..., kiểm bằng safeUploadPath) hoặc URL http(s) an toàn (validImageURLStrict).
func validAttachmentURL(raw string) bool {
	if raw == "" {
		return true
	}
	if pub := memberUploadPublicURL(); pub != "" && strings.HasPrefix(raw, pub+"/") {
		_, ok := safeUploadPath("/uploads/" + strings.TrimPrefix(raw, pub+"/"))
		return ok
	}
	if strings.HasPrefix(raw, "/") {
		_, ok := safeUploadPath(raw)
		return ok
	}
	return validImageURLStrict(raw)
}

// checkPassDayOpen: vé ngày / vé khách / vé tập thử chỉ dùng được trong ngày hiệu lực → không phát hành cho ngày chi nhánh
// nghỉ lễ đóng cửa cả ngày (lịch ngày lễ theo ScheduleGuard; chưa đăng ký guard – unit test – thì bỏ qua).
func checkPassDayOpen(ctx context.Context, branchID uint, day time.Time, what string) error {
	h, err := currentScheduleGuard().ClosedHoliday(ctx, branchID, day)
	if err != nil || h == nil {
		return err
	}
	return rescode.New(rescode.BadRequest, fmt.Sprintf("Chi nhánh nghỉ lễ \"%s\" và đóng cửa cả ngày %s – không thể %s cho ngày này, vui lòng chọn ngày khác",
		h.Name, day.In(subVNLoc).Format("02/01/2006"), what))
}
