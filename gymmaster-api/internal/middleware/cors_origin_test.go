package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Hồi quy QA vòng 2: whitelist so khớp chính xác origin, không chấp nhận cổng/scheme/subdomain khác.
func TestOriginAllowed_ExactMatch(t *testing.T) {
	wl := ParseCORSOrigins("http://localhost:3000, http://localhost:3001/,https://admin.gymmaster.vn, app.gymmaster.vn")
	cases := map[string]bool{
		"http://localhost:3000":           true,
		"http://localhost:3001":           true,
		"http://LOCALHOST:3001":           true,
		"http://localhost:3002":           false, // khác cổng
		"https://localhost:3000":          false, // khác scheme
		"http://localhost":                false,
		"https://admin.gymmaster.vn":      true,
		"http://admin.gymmaster.vn":       false,
		"https://evil.admin.gymmaster.vn": false, // subdomain
		"https://app.gymmaster.vn":        true,  // mục không scheme → http/https
		"http://app.gymmaster.vn":         true,
		"https://app.gymmaster.vn:8443":   false,
		"null":                            false,
		"":                                false,
		"file://localhost:3000":           false,
		"http://localhost:3000/path":      false,
	}
	for origin, want := range cases {
		assert.Equal(t, want, OriginAllowed(origin, wl), origin)
	}
	// Cấu hình cũ chỉ có host (config.corsHosts bỏ scheme) vẫn hoạt động, nhưng đúng cổng.
	legacy := []string{"localhost:3000"}
	assert.True(t, OriginAllowed("http://localhost:3000", legacy))
	assert.False(t, OriginAllowed("http://localhost:4000", legacy))
}

func TestBearerToken_CaseInsensitiveScheme(t *testing.T) {
	for h, want := range map[string]string{
		"Bearer abc": "abc", "bearer abc": "abc", "BEARER   abc ": "abc", " Bearer abc": "abc",
	} {
		got, ok := BearerToken(h)
		assert.True(t, ok, h)
		assert.Equal(t, want, got, h)
	}
	for _, h := range []string{"", "Bearer", "Bearer ", "Basic abc", "Bearerabc", "Token abc"} {
		_, ok := BearerToken(h)
		assert.False(t, ok, h)
	}
}
