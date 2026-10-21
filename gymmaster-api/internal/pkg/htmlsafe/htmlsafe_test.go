package htmlsafe

import (
	"strings"
	"testing"
)

func TestSanitizeStripsDangerous(t *testing.T) {
	cases := map[string]string{
		`X<img src=x onerror="window.__pwn=1">`:             `X<img>`,
		`<script>alert(1)</script>ok`:                       `ok`,
		`<svg onload=alert(1)><circle/></svg>after`:         `after`,
		`<a href="javascript:alert(1)">x</a>`:               `<a target="_blank" rel="noopener noreferrer nofollow">x</a>`,
		`<a href="java&#x09;script:alert(1)">x</a>`:         `<a target="_blank" rel="noopener noreferrer nofollow">x</a>`,
		`<style>body{display:none}</style><p>hi</p>`:        `<p>hi</p>`,
		`<p style="background:url(javascript:x)">t</p>`:     `<p>t</p>`,
		`<iframe src="https://evil"></iframe>z`:             `z`,
		`<div onclick="x()" title="a&quot;b">c</div>`:       `<div title="a&#34;b">c</div>`,
		`<!-- c --><p>a<b>b</p>`:                            `<p>a<b>b</b></p>`,
		`</p></div>t`:                                       `t`,
		`<img src="data:image/svg+xml;base64,PHN2Zz4=">`:    `<img>`,
		`<math><mi xlink:href="javascript:x">m</mi></math>`: ``,
	}
	for in, want := range cases {
		got := Sanitize(in)
		if !strings.Contains(got, want) && !(want == "" && got == "") {
			t.Errorf("Sanitize(%q) = %q, want contains %q", in, got, want)
		}
		low := strings.ToLower(got)
		for _, bad := range []string{"<script", "onerror", "onload", "javascript:", "<style", "<svg", "<iframe", "style="} {
			if strings.Contains(low, bad) {
				t.Errorf("Sanitize(%q) = %q still contains %q", in, got, bad)
			}
		}
	}
}

func TestSanitizeKeepsFormatting(t *testing.T) {
	in := `<h3>HỢP ĐỒNG</h3><p>Họ tên: <strong>{{member_name}}</strong></p><table><tr><td colspan="2">x</td></tr></table><a href="https://gymmaster.vn">w</a><img src="/uploads/a.png" alt="a">`
	got := Sanitize(in)
	for _, want := range []string{"<h3>HỢP ĐỒNG</h3>", "<strong>{{member_name}}</strong>", `<td colspan="2">x</td>`, `href="https://gymmaster.vn"`, `src="/uploads/a.png"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if Sanitize("   ") != "" {
		t.Error("blank must stay blank")
	}
	// idempotent
	if Sanitize(got) != got {
		t.Errorf("not idempotent: %q vs %q", Sanitize(got), got)
	}
}
