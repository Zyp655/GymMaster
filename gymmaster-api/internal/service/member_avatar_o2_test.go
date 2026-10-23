package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// UIA-079: avatar_url hội viên chỉ nhận ảnh tải lên của hệ thống (/uploads/... hoặc UPLOAD_PUBLIC_URL/...) hoặc rỗng.
func TestValidMemberAvatarURL(t *testing.T) {
	orig := memberUploadPublicURL
	memberUploadPublicURL = func() string { return "http://localhost:8000/uploads" }
	defer func() { memberUploadPublicURL = orig }()

	for _, ok := range []string{"", "/uploads/abc.png", "http://localhost:8000/uploads/2026/10/a-b.webp"} {
		assert.True(t, validMemberAvatarURL(ok), ok)
	}
	for _, bad := range []string{
		"http://evil.com/a.png", "https://cdn.example.com/uploads/a.png", "/uploads/../etc/passwd",
		"/uploads/%2e%2e/x.png", "/uploads/a.png?x=1", "javascript:alert(1)", "//evil.com/uploads/a.png",
		"http://localhost:8000/uploads/../a.png", "/static/a.png",
	} {
		assert.False(t, validMemberAvatarURL(bad), bad)
	}
}
