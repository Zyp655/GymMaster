package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "gymmaster-api/api/cms/v1"
	"gymmaster-api/internal/pkg/rescode"
)

// FD-11: tin nhắn chỉ gồm khoảng trắng/ký tự vô hình bị từ chối; emoji ghép ZWJ vẫn hợp lệ.
func TestHasVisibleText(t *testing.T) {
	for _, bad := range []string{"", "   ", "\u200B", "\u200B\u200C\u200D", "\uFEFF \u2060", "\u00AD\u3164", "\t\n\u2800"} {
		assert.False(t, hasVisibleText(bad), "%q", bad)
	}
	for _, ok := range []string{"Chào HLV", "👨\u200D👩\u200D👧", "\u200Bok", "👍"} {
		assert.True(t, hasVisibleText(ok), "%q", ok)
	}
}

// NT-07: loại thông báo ngoài danh mục → 400; NT-01: enabled=false được giữ (loại bắt buộc luôn bật).
func TestNotificationPrefs_ValidateAndKeepFalse(t *testing.T) {
	err := validateNotificationPrefs([]v1.SaveNotificationPreferenceInput{{Type: "INVOICE", Channel: "IN_APP"}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	err = validateNotificationPrefs([]v1.SaveNotificationPreferenceInput{{Type: "BIRTHDAY", Channel: "SMS"}})
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))
	require.NoError(t, validateNotificationPrefs([]v1.SaveNotificationPreferenceInput{{Type: "BIRTHDAY", Channel: "EMAIL"}}))

	now := time.Now()
	p := buildNotificationPref(7, v1.SaveNotificationPreferenceInput{Type: "BIRTHDAY", Channel: "EMAIL", Enabled: false}, now)
	assert.False(t, p.Enabled)
	p = buildNotificationPref(7, v1.SaveNotificationPreferenceInput{Type: "SYSTEM", Channel: "EMAIL", Enabled: false}, now)
	assert.True(t, p.Enabled)
}

// CMS-01: nội dung bài viết được lọc HTML; SEC-04: tiêu đề rỗng → 400.
func TestPostContentSanitizeAndTitle(t *testing.T) {
	out, err := sanitizePostContent(`<p onclick="x()">Hi</p><script>alert(1)</script><a href="javascript:alert(1)">l</a><img src=x onerror=alert(1)>`)
	require.NoError(t, err)
	for _, bad := range []string{"<script", "onclick", "javascript:", "onerror"} {
		assert.NotContains(t, strings.ToLower(out), bad)
	}
	assert.Contains(t, out, "<p>Hi</p>")
	_, err = sanitizePostContent("<script>alert(1)</script>")
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(err))

	assert.Equal(t, rescode.BadRequest.Code(), codeOf(validatePostTitle("")))
	assert.Equal(t, rescode.BadRequest.Code(), codeOf(validatePostTitle(strings.Repeat("a", 256))))
	assert.NoError(t, validatePostTitle("Bài viết"))
}

// CAL-05: điều kiện lọc lớp của HLV dùng HLV thực dạy (dạy thay ưu tiên).
func TestTrainerTeachingClassCond(t *testing.T) {
	assert.Equal(t, "COALESCE(cs.substitute_trainer_id, cs.trainer_id) = ?", trainerTeachingClassCond)
}
