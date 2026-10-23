package repository

import (
	"fmt"
	"time"
)

// Định nghĩa CHUNG "đang trong phòng tập" (dùng cho Mật độ /occupancy, dashboard /reports/overview, danh sách
// /checkins/present và thống kê /checkins/stats): lượt check-in ALLOWED chưa checkout, vào trong PresentWindow
// gần nhất (lượt quên checkout quá cửa sổ coi như đã ra về); đếm theo người (hội viên / vé ngày / vé tập thử).
const PresentWindow = 3 * time.Hour

// PresentSince: mốc bắt đầu cửa sổ "đang trong phòng tập" tại thời điểm now.
func PresentSince(now time.Time) time.Time { return now.Add(-PresentWindow) }

// presentPersonKey: khóa người của lượt check-in (bảng alias a) – hội viên, khách vé ngày hoặc vé tập thử.
func presentPersonKey(a string) string {
	return fmt.Sprintf("COALESCE('m' || %[1]s.member_id::text, 'd' || %[1]s.day_pass_id::text, 't' || %[1]s.trial_pass_id::text)", a)
}

// presentCond: điều kiện SQL "đang trong phòng tập" (tham số ? = PresentSince(now)).
func presentCond(a string) string {
	return fmt.Sprintf("%[1]s.result = 'ALLOWED' AND %[1]s.checked_out_at IS NULL AND %[1]s.checked_in_at >= ?", a)
}
