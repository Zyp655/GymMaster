package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// B9: đường dẫn /uploads/ phải được giải mã %XX lặp tới khi ổn định và chuẩn hóa trước khi kiểm tra.
var qa5BadUploadPaths = []string{
	"/uploads/%2e%2e/x.png", "/uploads/%2E%2E/x.png", "/uploads/..%2fx", "/uploads/./../x", "/uploads/../x.png",
	"/uploads/%252e%252e/x.png", "/uploads/%25252e%25252e/x.png", "/uploads/%2e%2e%2fetc%2fpasswd",
	"/uploads/a/%2e%2e/%2e%2e/x.png", "/uploads/./a.png", "/uploads//a.png", "/uploads/a/", "/uploads/",
	"/uploads/%2fetc/passwd", "/uploads/a%5c..%5cx.png", "/uploads/a%22onerror%3d.png", "/uploads/a%3cb%3e.png",
	"/uploads/a.png?x=1", "/uploads/a.png#f", "/uploads/a%3fx.png", "/uploads/a b.png", "/uploads/a%20b.png",
	"/uploads/%00.png", "/uploads/%zz.png", "/uploadsx/a.png", "/static/a.png",
	"/uploads/%2525252e%2525252e/x.png", // mã hóa lồng quá nhiều lớp
}

func TestSafeUploadPath(t *testing.T) {
	for _, ok := range []string{"/uploads/a.png", "/uploads/2026/10/a-b_c.webp", "/uploads/sig/abc123.png"} {
		got, valid := safeUploadPath(ok)
		assert.True(t, valid, ok)
		assert.Equal(t, ok, got)
	}
	for _, bad := range qa5BadUploadPaths {
		_, valid := safeUploadPath(bad)
		assert.False(t, valid, bad)
	}
}

func TestValidateSignature_UploadTraversal(t *testing.T) {
	assert.NoError(t, validateSignature("/uploads/2026/10/sig.png"))
	for _, bad := range qa5BadUploadPaths {
		err := validateSignature(bad)
		if assert.Error(t, err, bad) {
			assert.Equal(t, 40000, codeOf(err), bad)
		}
	}
}

func TestUploadPathCheckers_SharedRule(t *testing.T) {
	orig := memberUploadPublicURL
	memberUploadPublicURL = func() string { return "http://localhost:8000/uploads" }
	defer func() { memberUploadPublicURL = orig }()

	for _, bad := range qa5BadUploadPaths {
		assert.False(t, validMemberAvatarURL(bad), "avatar %s", bad)
		assert.False(t, productValidImageURL(bad), "product %s", bad)
		assert.False(t, validImageURLStrict(bad), "image %s", bad)
		assert.False(t, validAttachmentURL(bad), "attachment %s", bad)
	}
	assert.False(t, validMemberAvatarURL("http://localhost:8000/uploads/%252e%252e/x.png"))
	assert.False(t, validAttachmentURL("http://localhost:8000/uploads/..%2fx"))
	assert.False(t, validAttachmentURL("javascript:alert(1)"))
	assert.False(t, validAttachmentURL("data:text/html;base64,AAA"))

	assert.True(t, validMemberAvatarURL("http://localhost:8000/uploads/2026/a.png"))
	assert.True(t, productValidImageURL("/uploads/p/a.jpg"))
	assert.True(t, validAttachmentURL(""))
	assert.True(t, validAttachmentURL("/uploads/doc/a.pdf"))
	assert.True(t, validAttachmentURL("http://localhost:8000/uploads/doc/a.pdf"))
	assert.True(t, validAttachmentURL("https://drive.example.com/file/1"))
}
