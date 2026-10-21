package service

import "testing"

func TestValidEmailStrict(t *testing.T) {
	for _, ok := range []string{"", "a@b.vn", "nguyen.van-a+1@gym-master.com.vn"} {
		if !validEmailStrict(ok) {
			t.Errorf("%q phải hợp lệ", ok)
		}
	}
	for _, bad := range []string{"abc", "a@b", "a @b.vn", "A <a@b.vn>", "a@b.vn\"", "a@.vn", "a@b.vn.", "@b.vn", "a@b.vn<script>"} {
		if validEmailStrict(bad) {
			t.Errorf("%q phải bị từ chối", bad)
		}
	}
}

func TestValidImageURLStrict(t *testing.T) {
	for _, ok := range []string{"", "/uploads/2026/10/a.png", "https://cdn.gymmaster.vn/a.jpg?x=1"} {
		if !validImageURLStrict(ok) {
			t.Errorf("%q phải hợp lệ", ok)
		}
	}
	for _, bad := range []string{"javascript:alert(1)", `https://x.vn/a.png" onerror="x`, "https://x.vn/a b.png", "/uploads/../etc/passwd",
		"//evil.vn/a.png", "data:image/png;base64,AAA", "https://u:p@x.vn/a.png", "https://x.vn/<a>.png", "/uploads/", "ftp://x.vn/a.png", "https://x.vn/ả.png"} {
		if validImageURLStrict(bad) {
			t.Errorf("%q phải bị từ chối", bad)
		}
	}
}
