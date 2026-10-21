// Package htmlsafe: bộ lọc HTML theo danh sách cho phép (allowlist) cho nội dung do người dùng/nhân viên
// soạn (mẫu hợp đồng điện tử...) trước khi lưu hoặc trả về client để render bằng innerHTML.
//
// Nguyên tắc: chỉ giữ các thẻ định dạng văn bản/bảng; mọi thuộc tính ngoài danh sách (on*, style, class...)
// bị bỏ; href/src chỉ nhận URL an toàn; thẻ nguy hiểm (script, style, svg, iframe...) bị bỏ cả nội dung.
// Kết quả được tuần tự hóa lại từ token nên mọi chữ/thuộc tính đều đã được escape.
package htmlsafe

import (
	"html"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
)

var allowedTags = map[string]bool{
	"a": true, "b": true, "blockquote": true, "br": true, "caption": true, "code": true, "col": true,
	"colgroup": true, "dd": true, "div": true, "dl": true, "dt": true, "em": true, "h1": true, "h2": true,
	"h3": true, "h4": true, "h5": true, "h6": true, "hr": true, "i": true, "img": true, "li": true, "ol": true,
	"p": true, "pre": true, "s": true, "small": true, "span": true, "strong": true, "sub": true, "sup": true,
	"table": true, "tbody": true, "td": true, "tfoot": true, "th": true, "thead": true, "tr": true, "u": true,
	"ul": true,
}

// dropWithContent: thẻ bị bỏ cả phần nội dung bên trong.
var dropWithContent = map[string]bool{
	"script": true, "style": true, "iframe": true, "object": true, "embed": true, "noscript": true,
	"template": true, "svg": true, "math": true, "form": true, "input": true, "button": true, "select": true,
	"textarea": true, "link": true, "meta": true, "base": true, "title": true, "head": true, "frame": true,
	"frameset": true, "applet": true, "xmp": true, "noembed": true, "noframes": true, "plaintext": true,
}

var voidTags = map[string]bool{"br": true, "hr": true, "img": true, "col": true}

var globalAttrs = map[string]bool{"title": true, "colspan": true, "rowspan": true, "align": true}

var tagAttrs = map[string]map[string]bool{
	"a":   {"href": true},
	"img": {"src": true, "alt": true, "width": true, "height": true},
	"th":  {"scope": true},
}

var (
	safeURL = regexp.MustCompile(`(?i)^(https?:|mailto:|tel:|#|/[^/\\])`)
	safeImg = regexp.MustCompile(`(?i)^(https?:|/[^/\\]|data:image/(png|jpe?g|gif|webp);base64,[a-z0-9+/=]+$)`)
	// ký tự điều khiển/khoảng trắng mà trình duyệt bỏ qua trong scheme (vd "java\tscript:").
	urlNoise = regexp.MustCompile(`[\x00-\x20\x7f]+`)
)

func cleanURL(v string, img bool) (string, bool) {
	v = strings.TrimSpace(v)
	probe := urlNoise.ReplaceAllString(v, "")
	if img {
		return v, safeImg.MatchString(probe)
	}
	return v, safeURL.MatchString(probe)
}

// Sanitize lọc HTML theo allowlist. Chuỗi rỗng → rỗng.
func Sanitize(in string) string {
	if strings.TrimSpace(in) == "" {
		return ""
	}
	z := xhtml.NewTokenizer(strings.NewReader(in))
	var b strings.Builder
	drop := 0 // độ sâu đang nằm trong thẻ dropWithContent
	var dropTag []string
	open := []string{} // ngăn xếp thẻ đã mở (được giữ) để tự đóng ở cuối
	for {
		tt := z.Next()
		switch tt {
		case xhtml.ErrorToken:
			for i := len(open) - 1; i >= 0; i-- {
				b.WriteString("</" + open[i] + ">")
			}
			return b.String()
		case xhtml.TextToken:
			if drop > 0 {
				continue
			}
			b.WriteString(html.EscapeString(string(z.Text())))
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			tok := z.Token()
			name := strings.ToLower(tok.Data)
			if drop > 0 {
				if tt == xhtml.StartTagToken && name == dropTag[len(dropTag)-1] {
					drop++
					dropTag = append(dropTag, name)
				}
				continue
			}
			if dropWithContent[name] {
				if tt == xhtml.StartTagToken && !voidTags[name] && name != "input" && name != "link" &&
					name != "meta" && name != "base" && name != "embed" {
					drop++
					dropTag = append(dropTag, name)
				}
				continue
			}
			if !allowedTags[name] {
				continue
			}
			b.WriteString("<" + name)
			for _, a := range tok.Attr {
				key := strings.ToLower(a.Key)
				if a.Namespace != "" || !(globalAttrs[key] || tagAttrs[name][key]) {
					continue
				}
				val := a.Val
				if key == "href" || key == "src" {
					v, ok := cleanURL(val, key == "src")
					if !ok {
						continue
					}
					val = v
				}
				b.WriteString(" " + key + `="` + html.EscapeString(val) + `"`)
			}
			if name == "a" {
				b.WriteString(` target="_blank" rel="noopener noreferrer nofollow"`)
			}
			b.WriteString(">")
			if !voidTags[name] && tt == xhtml.StartTagToken {
				open = append(open, name)
			}
		case xhtml.EndTagToken:
			tok := z.Token()
			name := strings.ToLower(tok.Data)
			if drop > 0 {
				if name == dropTag[len(dropTag)-1] {
					drop--
					dropTag = dropTag[:len(dropTag)-1]
				}
				continue
			}
			if !allowedTags[name] || voidTags[name] {
				continue
			}
			// Chỉ đóng thẻ đã mở (đóng luôn các thẻ con chưa đóng) – không để thẻ đóng lạc.
			idx := -1
			for i := len(open) - 1; i >= 0; i-- {
				if open[i] == name {
					idx = i
					break
				}
			}
			if idx < 0 {
				continue
			}
			for i := len(open) - 1; i >= idx; i-- {
				b.WriteString("</" + open[i] + ">")
			}
			open = open[:idx]
		default: // comment, doctype
			continue
		}
	}
}
