package service

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gymmaster-api/internal/entity"
	"gymmaster-api/internal/pkg/htmlsafe"
	"gymmaster-api/internal/pkg/rescode"
)

// Quy tắc dữ liệu bài viết Blog (dùng chung cho tạo/cập nhật và trang công khai).

const (
	postSlugMinLen = 3
	postSlugMaxLen = 180
)

var postSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// postCategories khớp nhãn POST_CATEGORY_LABELS ở admin/web.
var postCategories = map[string]bool{"TAP_LUYEN": true, "DINH_DUONG": true, "SU_KIEN": true}

var postStatuses = map[string]bool{
	entity.PostStatusDraft: true, entity.PostStatusPublished: true, entity.PostStatusArchived: true,
}

// validatePostSlug: chỉ chữ thường a-z, số, dấu '-' (không ở đầu/cuối, không lặp), dài 3–180.
func validatePostSlug(slug string) error {
	n := utf8.RuneCountInString(slug)
	if n < postSlugMinLen || n > postSlugMaxLen {
		return rescode.New(rescode.BadRequest, "Đường dẫn bài viết phải từ 3 đến 180 ký tự")
	}
	if !postSlugPattern.MatchString(slug) {
		return rescode.New(rescode.BadRequest, "Đường dẫn bài viết chỉ gồm chữ thường không dấu (a-z), số và dấu gạch ngang '-'")
	}
	return nil
}

// validatePostTitle: tiêu đề (đã trim) bắt buộc, tối đa 255 ký tự.
func validatePostTitle(title string) error {
	if title == "" {
		return rescode.New(rescode.BadRequest, "Vui lòng nhập tiêu đề bài viết")
	}
	if utf8.RuneCountInString(title) > 255 {
		return rescode.New(rescode.BadRequest, "Tiêu đề tối đa 255 ký tự")
	}
	return nil
}

// sanitizePostContent lọc HTML nội dung bài viết theo allowlist (bỏ script, on*, javascript:...)
// trước khi lưu; nội dung chỉ còn rỗng sau khi lọc → 400.
func sanitizePostContent(raw string) (string, error) {
	content := strings.TrimSpace(htmlsafe.Sanitize(raw))
	if content == "" {
		return "", rescode.New(rescode.BadRequest, "Vui lòng nhập nội dung bài viết")
	}
	return content, nil
}

func validatePostCategory(category string) error {
	if !postCategories[category] {
		return rescode.New(rescode.BadRequest, "Chuyên mục bài viết không hợp lệ")
	}
	return nil
}

func validatePostStatus(status string) error {
	if !postStatuses[status] {
		return rescode.New(rescode.BadRequest, "Trạng thái bài viết không hợp lệ")
	}
	return nil
}

// postIsPublic: chỉ bài PUBLISHED và đã tới thời điểm xuất bản mới hiển thị công khai.
func postIsPublic(p *entity.Post, now time.Time) bool {
	if p == nil || p.Status != entity.PostStatusPublished {
		return false
	}
	return p.PublishedAt == nil || !p.PublishedAt.After(now)
}

func postSlugConflictError() error {
	return rescode.New(rescode.Conflict, "Đường dẫn bài viết đã được sử dụng cho bài viết khác")
}
