package repository

import (
	"testing"
	"time"
)

// D7 (FIN-PAY-06): P&L kỳ lẻ ngày chỉ tính phần quỹ lương tương ứng số ngày, không cộng nguyên tháng.
func TestPayrollPeriodShare(t *testing.T) {
	loc := time.FixedZone("ICT", 7*3600)
	d := func(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, loc) }
	cases := []struct {
		name     string
		period   string
		from, to time.Time
		want     float64
	}{
		{"cả tháng", "2026-10", d(2026, 10, 1), d(2026, 10, 31), 1},
		{"một ngày", "2026-10", d(2026, 10, 22), d(2026, 10, 22), 1.0 / 31},
		{"nửa tháng 9", "2026-09", d(2026, 9, 16), d(2026, 10, 5), 15.0 / 30},
		{"phần tháng 10 của khoảng vắt tháng", "2026-10", d(2026, 9, 16), d(2026, 10, 5), 5.0 / 31},
		{"ngoài khoảng", "2026-08", d(2026, 9, 1), d(2026, 9, 30), 0},
		{"phủ nhiều tháng", "2026-02", d(2026, 1, 1), d(2026, 12, 31), 1},
		{"kỳ sai định dạng", "2026/02", d(2026, 1, 1), d(2026, 12, 31), 0},
	}
	for _, c := range cases {
		got := PayrollPeriodShare(c.period, c.from, c.to)
		if diff := got - c.want; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
