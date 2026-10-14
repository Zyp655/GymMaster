package service

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

var invoiceDigitWords = [10]string{"không", "một", "hai", "ba", "bốn", "năm", "sáu", "bảy", "tám", "chín"}

// invoiceReadTriple đọc một nhóm 3 chữ số (0..999).
// full = true khi đã có nhóm cao hơn khác 0 đứng trước → đọc đủ "không trăm", "linh".
func invoiceReadTriple(n int, full bool) []string {
	h, t, u := n/100, (n/10)%10, n%10
	var w []string
	if h > 0 || full {
		w = append(w, invoiceDigitWords[h], "trăm")
	}
	switch t {
	case 0:
		if u > 0 {
			if h > 0 || full {
				w = append(w, "linh")
			}
			w = append(w, invoiceDigitWords[u])
		}
	case 1:
		w = append(w, "mười")
		switch u {
		case 0:
		case 5:
			w = append(w, "lăm")
		default:
			w = append(w, invoiceDigitWords[u])
		}
	default:
		w = append(w, invoiceDigitWords[t], "mươi")
		switch u {
		case 0:
		case 1:
			w = append(w, "mốt")
		case 4:
			w = append(w, "tư")
		case 5:
			w = append(w, "lăm")
		default:
			w = append(w, invoiceDigitWords[u])
		}
	}
	return w
}

// invoiceReadBelowBillion đọc phần < 1 tỷ (tối đa 3 nhóm: triệu, nghìn, đơn vị).
// full = true khi phía trước đã có phần cao hơn (tỷ) → nhóm đầu cũng đọc đủ "không trăm".
func invoiceReadBelowBillion(n int64, full bool) []string {
	groups := [3]int{int(n / 1_000_000), int(n / 1000 % 1000), int(n % 1000)}
	scales := [3]string{"triệu", "nghìn", ""}
	var w []string
	started := full
	for i, g := range groups {
		if g == 0 {
			continue
		}
		w = append(w, invoiceReadTriple(g, started)...)
		if scales[i] != "" {
			w = append(w, scales[i])
		}
		started = true
	}
	return w
}

func invoiceReadNumber(n int64, full bool) []string {
	if n >= 1_000_000_000 {
		w := append(invoiceReadNumber(n/1_000_000_000, full), "tỷ")
		if lo := n % 1_000_000_000; lo > 0 {
			w = append(w, invoiceReadBelowBillion(lo, true)...)
		}
		return w
	}
	return invoiceReadBelowBillion(n, full)
}

// NumberToVietnameseWords đọc số nguyên không âm thành chữ tiếng Việt (chữ thường), ví dụ
// 1215000 → "một triệu hai trăm mười lăm nghìn"; 1500000000000 → "một nghìn năm trăm tỷ"; 0 → "không".
func NumberToVietnameseWords(n int64) string {
	if n < 0 {
		return "âm " + NumberToVietnameseWords(-n)
	}
	if n == 0 {
		return "không"
	}
	return strings.Join(invoiceReadNumber(n, false), " ")
}

// VNDInWords đọc số tiền VND thành chữ để in hóa đơn, viết hoa chữ cái đầu và thêm "đồng".
// Ví dụ 1215000 → "Một triệu hai trăm mười lăm nghìn đồng".
func VNDInWords(amount float64) string {
	s := NumberToVietnameseWords(int64(math.Round(amount))) + " đồng"
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}
