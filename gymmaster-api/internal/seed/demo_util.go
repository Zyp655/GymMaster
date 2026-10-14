package seed

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------------
// Ngẫu nhiên có seed cố định (kết quả lặp lại được với cùng "hôm nay").
// ---------------------------------------------------------------------

type demoRand struct{ *rand.Rand }

func (r demoRand) between(lo, hi int) int { // [lo, hi]
	if hi <= lo {
		return lo
	}
	return lo + r.Intn(hi-lo+1)
}

func (r demoRand) chance(p float64) bool { return r.Float64() < p }

func (r demoRand) normal(mean, sd float64) float64 { return mean + sd*r.NormFloat64() }

func (r demoRand) pickS(list []string) string { return list[r.Intn(len(list))] }

// weighted chọn chỉ số theo trọng số.
func (r demoRand) weighted(weights []float64) int {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	x := r.Float64() * sum
	for i, w := range weights {
		if x < w {
			return i
		}
		x -= w
	}
	return len(weights) - 1
}

// ---------------------------------------------------------------------
// Thời gian (giờ Việt Nam, không có DST nên Add theo phút là an toàn).
// ---------------------------------------------------------------------

func demoLoc() *time.Location {
	if l, err := time.LoadLocation("Asia/Ho_Chi_Minh"); err == nil {
		return l
	}
	return time.FixedZone("ICT", 7*3600)
}

// dayOf trả 00:00 (giờ VN) của ngày chứa t.
func dayOf(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func addDays(d time.Time, n int) time.Time { return d.AddDate(0, 0, n) }

// atMin: ngày d (00:00 VN) + số phút trong ngày.
func atMin(d time.Time, minute int) time.Time { return d.Add(time.Duration(minute) * time.Minute) }

func daysBetween(a, b time.Time) int { // b - a (theo ngày lịch)
	return int(math.Round(b.Sub(a).Hours() / 24))
}

// pgDate: giá trị cho cột DATE (UTC midnight để driver không lệch ngày).
func pgDate(d time.Time) time.Time { return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC) }

func ymd(d time.Time) string   { return d.Format("2006-01-02") }
func dmy(d time.Time) string   { return d.Format("02/01/2006") }
func isoWeekday(d time.Time) int { // 1 = Thứ Hai … 7 = Chủ nhật
	w := int(d.Weekday())
	if w == 0 {
		return 7
	}
	return w
}

func hhmmToMin(s string) int {
	var h, m int
	_, _ = fmt.Sscanf(s, "%d:%d", &h, &m)
	return h*60 + m
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

type interval struct{ from, to time.Time }

func overlaps(list []interval, from, to time.Time) bool {
	for _, iv := range list {
		if from.Before(iv.to) && iv.from.Before(to) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------
// Tiền, chuỗi
// ---------------------------------------------------------------------

func roundVND(x float64) float64 { return math.Round(x) }

func roundTo(x float64, step float64) float64 { return math.Round(x/step) * step }

func moneyVN(x float64) string {
	n := int64(math.Round(x))
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return b.String() + " đ"
}

var vnFold = map[rune]string{}

func init() {
	groups := map[string]string{
		"a": "àáạảãâầấậẩẫăằắặẳẵ", "e": "èéẹẻẽêềếệểễ", "i": "ìíịỉĩ", "o": "òóọỏõôồốộổỗơờớợởỡ",
		"u": "ùúụủũưừứựửữ", "y": "ỳýỵỷỹ", "d": "đ",
	}
	for ascii, chars := range groups {
		for _, c := range chars {
			vnFold[c] = ascii
		}
	}
}

// asciiFold bỏ dấu tiếng Việt, chữ thường, bỏ khoảng trắng.
func asciiFold(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if c == ' ' {
			continue
		}
		if r, ok := vnFold[c]; ok {
			b.WriteString(r)
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

// ---------------------------------------------------------------------
// Ghi theo lô: INSERT nhiều dòng một câu lệnh ($1..$n), tối đa ~60k tham số.
// ---------------------------------------------------------------------

type bulkRows struct {
	table string
	cols  []string
	rows  [][]any
}

func newBulk(table string, cols ...string) *bulkRows { return &bulkRows{table: table, cols: cols} }

func (b *bulkRows) add(vals ...any) {
	if len(vals) != len(b.cols) {
		panic(fmt.Sprintf("bulk %s: %d giá trị cho %d cột", b.table, len(vals), len(b.cols)))
	}
	b.rows = append(b.rows, vals)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (b *bulkRows) flush(ctx context.Context, tx *gorm.DB) error {
	if len(b.rows) == 0 {
		return nil
	}
	ex, ok := tx.Statement.ConnPool.(execer)
	if !ok {
		return fmt.Errorf("bulk: kết nối không hỗ trợ ExecContext")
	}
	per := 60000 / len(b.cols)
	if per > 2000 {
		per = 2000
	}
	head := "INSERT INTO " + b.table + " (" + strings.Join(b.cols, ", ") + ") VALUES "
	for start := 0; start < len(b.rows); start += per {
		end := start + per
		if end > len(b.rows) {
			end = len(b.rows)
		}
		var sb strings.Builder
		sb.WriteString(head)
		args := make([]any, 0, (end-start)*len(b.cols))
		n := 1
		for i := start; i < end; i++ {
			if i > start {
				sb.WriteByte(',')
			}
			sb.WriteByte('(')
			for j := range b.cols {
				if j > 0 {
					sb.WriteByte(',')
				}
				fmt.Fprintf(&sb, "$%d", n)
				n++
			}
			sb.WriteByte(')')
			args = append(args, b.rows[i]...)
		}
		if _, err := ex.ExecContext(ctx, sb.String(), args...); err != nil {
			return fmt.Errorf("insert %s: %w", b.table, err)
		}
	}
	return nil
}

// nullable helpers
func nUint(p *uint) any {
	if p == nil {
		return nil
	}
	return int64(*p)
}

func nTime(p *time.Time) any {
	if p == nil {
		return nil
	}
	return *p
}

func nFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrUint(v uint) *uint           { return &v }
func ptrFloat(v float64) *float64    { return &v }
