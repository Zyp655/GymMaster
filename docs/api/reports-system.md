# API – Báo cáo, Dashboard, Thông báo, Nhật ký thao tác, Job định kỳ (migration 000010)

Base URL (dev): `http://localhost:8000/v1`. Mọi endpoint cần `Authorization: Bearer <access_token>`.
Envelope, mã lỗi, phân trang: xem `sprint-01-auth-rbac-branch.md`. Tóm tắt:

- Thành công: `{ "code": 0, "message": "OK", "data": ... }`. Lỗi: `code` 5 chữ số, HTTP = code/100, `message` tiếng Việt.
  Lỗi validate tag: `code = 51` (HTTP 400). Nghiệp vụ: `40000`, `40300`, `40400`, `40900`.
- Ngày `"YYYY-MM-DD"` (lịch Việt Nam `Asia/Ho_Chi_Minh`). Thời điểm RFC3339. Tiền là `number` VND đã làm tròn đến đồng.
- Tỷ lệ/phần trăm (`pct`, `fill_rate`, `change_pct`, …) là số **0–100**, làm tròn 1 chữ số.

## Danh sách endpoint

| Method | Path | Quyền | Mô tả |
|---|---|---|---|
| GET | `/admin/reports/overview` | `report.view` | KPI dashboard |
| GET | `/admin/reports/revenue` | `report.view` | Doanh thu thực thu theo thời gian / loại / phương thức / chi nhánh |
| GET | `/admin/reports/members` | `report.view` | Hội viên mới, phân bố nguồn/giới tính/độ tuổi/chi nhánh, trạng thái |
| GET | `/admin/reports/retention` | `report.view` | Tỷ lệ gia hạn & churn theo tháng |
| GET | `/admin/reports/checkin-heatmap` | `report.view` | Heatmap 7 thứ × 24 giờ, giờ cao điểm |
| GET | `/admin/reports/class-occupancy` | `report.view` | Lấp đầy lớp theo loại lớp / HLV / khung giờ / thứ |
| GET | `/admin/reports/trainer-performance` | `report.view` | Hiệu suất & hoa hồng HLV |
| GET | `/admin/reports/sales-products` | `report.view` | Top sản phẩm, doanh thu, lợi nhuận gộp |
| GET | `/admin/reports/subscriptions` | `report.view` | Doanh số theo gói, biến động đăng ký |
| GET | `/admin/reports/export` | `report.export` | Xuất Excel (.xlsx) |
| GET | `/admin/notifications` | đăng nhập (nhân viên) | Thông báo của tôi |
| GET | `/admin/notifications/unread-count` | đăng nhập | Số chưa đọc (badge chuông) |
| POST | `/admin/notifications/{id}/read` | đăng nhập | Đánh dấu đã đọc |
| POST | `/admin/notifications/read-all` | đăng nhập | Đánh dấu tất cả đã đọc |
| GET | `/member/notifications` | role `MEMBER` | Thông báo của hội viên (cùng format) |
| GET | `/member/notifications/unread-count` | role `MEMBER` | |
| POST | `/member/notifications/{id}/read` | role `MEMBER` | |
| POST | `/member/notifications/read-all` | role `MEMBER` | |
| GET | `/admin/audit-logs` | `audit.view` | Nhật ký thao tác |
| GET | `/admin/audit-logs/options` | `audit.view` | Giá trị `entity`, `action` đang có (cho dropdown lọc) |
| GET | `/admin/audit-logs/{id}` | `audit.view` | Chi tiết + before/after + diff |
| GET | `/admin/jobs` | `job.view` | Danh sách job + lịch + lần chạy gần nhất |
| GET | `/admin/jobs/{name}/runs` | `job.view` | Lịch sử chạy |
| POST | `/admin/jobs/{name}/run` | `job.view` | Chạy ngay (đồng bộ), trả kết quả |
| POST | `/admin/jobs/test-email` | `job.view` | Gửi email thử (kiểm tra SMTP) |

Quyền mặc định (migration 000010): OWNER có tất cả; MANAGER có `report.view`, `report.export`. `audit.view`, `job.view` chỉ OWNER
(có thể gán thêm qua màn hình vai trò).

---

## 1. Báo cáo

### Tham số chung

| Query | Mặc định | Ghi chú |
|---|---|---|
| `branch_id` | (trống) | Trống = mọi chi nhánh người dùng được xem (OWNER: toàn hệ thống; nhân viên gắn chi nhánh: các chi nhánh của mình). Chi nhánh ngoài phạm vi → `40300`. |
| `from` | ngày 1 tháng hiện tại | `YYYY-MM-DD`, giờ VN. Chỉ có `to` → `from` = ngày 1 tháng của `to`. |
| `to` | hôm nay | `YYYY-MM-DD`, tính **cả ngày** `to`. |
| `group_by` | `day` | `day` \| `week` \| `month` (báo cáo có trục thời gian). Tuần bắt đầu **thứ 2** (mốc = ngày thứ 2); tháng = ngày 1. |

- `from > to` → `40000` "Từ ngày phải trước hoặc bằng đến ngày". Khoảng tối đa 3 năm. `group_by=day` tối đa 400 ngày
  (quá → `40000` "…hãy chọn theo tuần hoặc tháng").
- Mọi response có `period = {from, to, days}` (khoảng thực tế sau khi áp mặc định).
- Trục thời gian (`series`, `new_series`, `daily`, `months`) **luôn đủ** mọi ngày/tuần/tháng trong khoảng (giá trị 0 nếu không phát sinh).
  Mỗi điểm có `period` (ngày đầu nhóm, dùng làm key) và `label` gợi ý cho trục X: day `"14/10"`, week `"Tuần 12/10"`, month `"10/2026"`.
  Nhóm tuần/tháng đầu tiên có thể bắt đầu trước `from` (tuần chứa `from`), nhưng số liệu chỉ tính trong `[from, to]`.

### Cách tính chính
- **Thực thu** = Σ `payments.amount` (theo `paid_at`) − Σ `refunds.amount` (theo `refunds.created_at`) trong kỳ, không phụ thuộc trạng thái hóa đơn.
- **Tách theo loại** (`MEMBERSHIP`, `PT`, `PRODUCT`, `FEE`, `OTHER` = `invoice_items.item_type`): mỗi lần thu/hoàn được phân bổ cho các dòng hóa đơn
  theo tỷ trọng `amount` của dòng. Ví dụ hóa đơn gói 1.000.000 + sản phẩm 200.000, thu 1.200.000 → gói 1.000.000, sản phẩm 200.000;
  hoàn 120.000 → gói −100.000, sản phẩm −20.000. Chi nhánh = `invoices.branch_id`.
- **Hội viên đang hoạt động** = số hội viên (chưa xóa) có ít nhất một gói `ACTIVE`.
- Mọi quy đổi giờ dùng `AT TIME ZONE 'Asia/Ho_Chi_Minh'`; mỗi báo cáo là vài câu SQL gộp (GROUP BY / FILTER / generate_series), không N+1.

### GET `/admin/reports/overview` – KPI dashboard
Query: `branch_id`. (Không có `from/to` – luôn tính theo hôm nay / tháng này.)
```json
{
  "date": "2026-10-14",
  "revenue": { "today": 9415000, "month": 45200000, "prev_month_to_date": 40100000, "prev_month": 98000000, "change_pct": 12.7 },
  "members": { "active": 412, "new_this_month": 37, "new_prev_month": 51, "new_change_pct": -27.5, "expiring_7d": 18 },
  "checkins": { "today": 126, "present": 23 },
  "classes":  { "today": 9, "capacity": 180, "booked": 131, "fill_rate": 72.8 },
  "pt":       { "today": 14, "completed": 6 },
  "pending_freezes": 3,
  "low_stock_count": 2,
  "low_stock_items": [
    { "product_id": 8, "sku": "PRE-C4", "name": "Pre-workout C4 30 lần dùng", "branch_id": 3,
      "branch_name": "GymMaster Quận 1", "quantity": 1, "min_stock": 2 }
  ]
}
```
- `revenue.change_pct` = tháng này (đến hôm nay) so với **cùng số ngày** tháng trước (`prev_month_to_date`); `null` khi kỳ trước = 0.
  `prev_month` = cả tháng trước (hiển thị phụ). `members.new_change_pct` = tháng này so với cả tháng trước.
- `members.expiring_7d`: hội viên có gói `ACTIVE` hết hạn trong `[hôm nay, hôm nay+7]` và chưa có gói nối tiếp.
- `checkins.today`: lượt `ALLOWED` hôm nay; `present`: đang trong phòng tập (vào hôm nay, chưa checkout).
- `classes`: buổi lớp không bị hủy bắt đầu hôm nay; `fill_rate = booked / capacity`.
- `pt`: buổi PT hôm nay (`SCHEDULED` + `COMPLETED`), `completed` = đã hoàn thành.
- `pending_freezes`: yêu cầu bảo lưu chờ duyệt. `low_stock_*`: sản phẩm `ACTIVE` có `min_stock > 0` và tồn ≤ `min_stock` (tối đa 5 dòng, thiếu nhất trước).

### GET `/admin/reports/revenue`
Query: `branch_id`, `from`, `to`, `group_by`.
```json
{
  "period": { "from": "2026-10-01", "to": "2026-10-14", "days": 14 },
  "group_by": "day",
  "totals": { "net": 1080000, "payments": 1200000, "refunds": 120000 },
  "previous": { "from": "2026-09-17", "to": "2026-09-30", "days": 14, "net": 720000, "change_pct": 50 },
  "series": [
    { "period": "2026-10-01", "label": "01/10", "total": 0, "membership": 0, "pt": 0, "product": 0, "fee": 0, "other": 0, "payments": 0, "refunds": 0 },
    { "period": "2026-10-02", "label": "02/10", "total": 1080000, "membership": 900000, "pt": 0, "product": 180000,
      "fee": 0, "other": 0, "payments": 1200000, "refunds": 120000 }
  ],
  "by_type": [
    { "key": "MEMBERSHIP", "label": "Gói tập", "value": 900000, "pct": 83.3 },
    { "key": "PT", "label": "Huấn luyện cá nhân", "value": 0, "pct": 0 },
    { "key": "PRODUCT", "label": "Sản phẩm", "value": 180000, "pct": 16.7 },
    { "key": "FEE", "label": "Phí", "value": 0, "pct": 0 },
    { "key": "OTHER", "label": "Khác", "value": 0, "pct": 0 }
  ],
  "by_method": [
    { "key": "CASH", "label": "Tiền mặt", "value": 580000, "pct": 53.7 },
    { "key": "TRANSFER", "label": "Chuyển khoản", "value": 500000, "pct": 46.3 },
    { "key": "CARD", "label": "Thẻ", "value": 0, "pct": 0 }
  ],
  "by_branch": [ { "branch_id": 1, "branch_name": "GymMaster Cầu Giấy", "amount": 1080000, "pct": 100 } ]
}
```
- `series[].total` = thực thu của nhóm (= membership + pt + product + fee + other = payments − refunds). Biểu đồ gợi ý: cột chồng theo loại + đường `total`.
- `previous`: kỳ liền trước **cùng số ngày**; `change_pct` = % thay đổi `totals.net` (null khi kỳ trước = 0).
- `by_type`, `by_method` luôn đủ các khóa theo thứ tự cố định (pie/donut); `by_method` theo phương thức của lần thu/hoàn. `by_branch` sắp giảm dần.

### GET `/admin/reports/members`
Query: `branch_id` (chi nhánh chính của hội viên), `from`, `to`, `group_by`,
`population` = `new` (mặc định – phân bố trên hội viên **mới trong kỳ**) | `all` (toàn bộ hội viên).
```json
{
  "period": { "from": "2026-10-01", "to": "2026-10-14", "days": 14 },
  "group_by": "week",
  "population": "new",
  "new_total": 37,
  "new_series": [
    { "period": "2026-09-28", "label": "Tuần 28/09", "count": 6 },
    { "period": "2026-10-05", "label": "Tuần 05/10", "count": 17 },
    { "period": "2026-10-12", "label": "Tuần 12/10", "count": 14 }
  ],
  "by_source": [
    { "key": "WALK_IN", "label": "Khách vãng lai", "count": 12, "pct": 32.4 },
    { "key": "REFERRAL", "label": "Giới thiệu", "count": 9, "pct": 24.3 },
    { "key": "FACEBOOK", "label": "Facebook", "count": 10, "pct": 27 },
    { "key": "WEBSITE", "label": "Website", "count": 4, "pct": 10.8 },
    { "key": "OTHER", "label": "Khác", "count": 2, "pct": 5.4 }
  ],
  "by_gender": [
    { "key": "MALE", "label": "Nam", "count": 20, "pct": 54.1 },
    { "key": "FEMALE", "label": "Nữ", "count": 16, "pct": 43.2 },
    { "key": "OTHER", "label": "Khác", "count": 1, "pct": 2.7 }
  ],
  "by_age": [
    { "key": "UNDER_18", "label": "Dưới 18", "count": 1, "pct": 2.7 }, { "key": "18_24", "label": "18–24", "count": 11, "pct": 29.7 },
    { "key": "25_34", "label": "25–34", "count": 15, "pct": 40.5 }, { "key": "35_44", "label": "35–44", "count": 6, "pct": 16.2 },
    { "key": "45_54", "label": "45–54", "count": 2, "pct": 5.4 }, { "key": "55_PLUS", "label": "Từ 55", "count": 0, "pct": 0 },
    { "key": "UNKNOWN", "label": "Chưa rõ", "count": 2, "pct": 5.4 }
  ],
  "by_branch": [
    { "key": "1", "label": "GymMaster Cầu Giấy", "count": 25, "pct": 67.6 },
    { "key": "3", "label": "GymMaster Quận 1", "count": 12, "pct": 32.4 }
  ],
  "status": [
    { "key": "ACTIVE", "label": "Đang hoạt động", "count": 412, "pct": 68.7 },
    { "key": "FROZEN", "label": "Bảo lưu", "count": 9, "pct": 1.5 },
    { "key": "PENDING", "label": "Chờ thanh toán", "count": 4, "pct": 0.7 },
    { "key": "EXPIRED", "label": "Hết hạn", "count": 150, "pct": 25 },
    { "key": "NONE", "label": "Chưa có gói", "count": 25, "pct": 4.2 }
  ],
  "total_members": 600
}
```
- `status` là **hiện tại** (không phụ thuộc from/to) trên toàn bộ hội viên của chi nhánh, ưu tiên: ACTIVE (có gói ACTIVE) > FROZEN > PENDING > EXPIRED (từng có gói, nay hết) > NONE.
- `by_branch[].key` là `branch_id` dạng chuỗi. Độ tuổi tính theo `dob` tại hôm nay.

### GET `/admin/reports/retention`
Query: `branch_id`, `from`, `to`, `window_days` (1–180, mặc định 30).
```json
{
  "period": { "from": "2026-07-01", "to": "2026-10-14", "days": 106 },
  "window_days": 30,
  "summary": { "month": "", "label": "Tổng", "expired": 120, "renewed": 78, "pending": 9, "churned": 33, "renewal_rate": 65, "churn_rate": 27.5 },
  "months": [
    { "month": "2026-07", "label": "07/2026", "expired": 35, "renewed": 24, "pending": 0, "churned": 11, "renewal_rate": 68.6, "churn_rate": 31.4 },
    { "month": "2026-10", "label": "10/2026", "expired": 18, "renewed": 7, "pending": 9, "churned": 2, "renewal_rate": 38.9, "churn_rate": 11.1 }
  ]
}
```
- Tập xét: gói `ACTIVE/FROZEN/EXPIRED` có `end_date` trong `[from, to]` và **trước hôm nay** (gói hủy/chuyển nhượng/nâng cấp không tính).
  Gom theo tháng của `end_date`, đếm **số hội viên** (distinct).
- `renewed`: hội viên có gói khác (đã kích hoạt) kéo dài quá `end_date` và được tạo **trước `end_date + window_days`** (gia hạn sớm cũng tính).
- `pending`: chưa gia hạn nhưng còn trong cửa sổ; `churned = expired − renewed − pending`.
- `renewal_rate = renewed/expired`, `churn_rate = churned/expired`. `months` đủ mọi tháng trong kỳ.

### GET `/admin/reports/checkin-heatmap`
Query: `branch_id`, `from`, `to`. Chỉ tính lượt `ALLOWED`.
```json
{
  "period": { "from": "2026-10-01", "to": "2026-10-14", "days": 14 },
  "matrix": [
    { "dow": 1, "label": "T2", "hours": [0,0,0,0,0,3,18,25,12,6,4,3,5,4,3,5,9,22,31,27,14,5,0,0], "total": 196, "avg_per_day": 98 },
    { "dow": 7, "label": "CN", "hours": [0,0,0,0,0,0,4,9,14,12,8,5,2,1,1,3,6,8,7,4,1,0,0,0], "total": 85, "avg_per_day": 42.5 }
  ],
  "by_hour": [0,0,0,0,0,9,61,90,60,40,30,20,22,18,15,25,48,95,120,101,50,15,0,0],
  "max": 31,
  "total": 1029,
  "denied": 14,
  "unique_members": 233,
  "avg_per_day": 73.5,
  "peak": { "dow": 1, "hour": 18, "label": "T2 18:00", "count": 31 },
  "peak_hours": [18, 19, 17],
  "daily": [ { "period": "2026-10-01", "label": "01/10", "count": 70 } ]
}
```
- `matrix` luôn 7 dòng (`dow` 1 = Thứ 2 … 7 = Chủ nhật), mỗi dòng `hours` 24 phần tử (index = giờ 0–23, giờ VN). Dùng `max` để chia thang màu.
- `avg_per_day` từng thứ = total / số lần thứ đó xuất hiện trong kỳ; `avg_per_day` gốc = total / số ngày.
- `peak_hours`: tối đa 3 giờ đông nhất (theo `by_hour`). `daily`: lượt/ngày (rỗng nếu kỳ > 400 ngày). `denied`, `unique_members` để hiển thị KPI phụ.

### GET `/admin/reports/class-occupancy`
Query: `branch_id`, `from`, `to`. Buổi lớp không bị hủy, bắt đầu trong kỳ.
```json
{
  "period": { "from": "2026-10-01", "to": "2026-10-14", "days": 14 },
  "summary": { "key": "ALL", "label": "Tổng", "sessions": 42, "capacity": 840, "booked": 590, "attended": 498,
               "no_show": 61, "cancelled": 37, "waitlist": 12, "fill_rate": 70.2, "attendance_rate": 89.1 },
  "by_class_type": [
    { "key": "3", "label": "Yoga", "id": 3, "color": "#22C55E", "sessions": 14, "capacity": 280, "booked": 250, "attended": 220,
      "no_show": 18, "cancelled": 9, "waitlist": 10, "fill_rate": 89.3, "attendance_rate": 92.4 }
  ],
  "by_trainer": [
    { "key": "12", "label": "Trần Minh", "id": 12, "sessions": 10, "capacity": 200, "booked": 150, "attended": 130,
      "no_show": 12, "cancelled": 6, "waitlist": 0, "fill_rate": 75, "attendance_rate": 91.5 }
  ],
  "by_time_slot": [
    { "key": "06:00", "label": "06:00", "sessions": 8, "capacity": 160, "booked": 90, "attended": 80,
      "no_show": 6, "cancelled": 3, "waitlist": 0, "fill_rate": 56.3, "attendance_rate": 93 }
  ],
  "by_weekday": [
    { "key": "1", "label": "T2", "sessions": 6, "capacity": 120, "booked": 100, "attended": 90,
      "no_show": 5, "cancelled": 2, "waitlist": 3, "fill_rate": 83.3, "attendance_rate": 94.7 }
  ]
}
```
- `booked` = lượt `BOOKED + ATTENDED + NO_SHOW` (chỗ thực sự giữ); `fill_rate = booked/capacity`; `attendance_rate = attended/(attended+no_show)`.
- `by_class_type`, `by_trainer` sắp theo số buổi giảm dần (`id` = class_type_id / trainer user id, `color` chỉ có ở loại lớp);
  `by_time_slot` theo giờ bắt đầu tăng dần; `by_weekday` theo thứ.

### GET `/admin/reports/trainer-performance`
Query: `branch_id`, `from`, `to`.
```json
{
  "period": { "from": "2026-10-01", "to": "2026-10-14", "days": 14 },
  "items": [
    { "trainer_id": 12, "trainer_name": "Trần Minh", "level": "SENIOR",
      "pt_completed": 28, "pt_scheduled": 6, "pt_no_show": 1, "pt_cancelled": 2,
      "classes_taught": 10, "class_attendees": 130, "active_pt_members": 9,
      "pt_contracts_sold": 3, "pt_revenue": 21000000, "commission": 4200000, "commission_paid": 0 }
  ],
  "totals": { "trainer_id": 0, "trainer_name": "Tổng", "level": "", "pt_completed": 28, "pt_scheduled": 6, "pt_no_show": 1,
              "pt_cancelled": 2, "classes_taught": 10, "class_attendees": 130, "active_pt_members": 9,
              "pt_contracts_sold": 3, "pt_revenue": 21000000, "commission": 4200000, "commission_paid": 0 }
}
```
- `pt_*`: buổi PT có `start_at` trong kỳ theo trạng thái. `classes_taught`: buổi lớp (không hủy) đã bắt đầu trong kỳ; `class_attendees`: lượt ATTENDED.
- `active_pt_members`: số hội viên có hợp đồng PT `ACTIVE` với HLV (hiện tại). `pt_contracts_sold` / `pt_revenue`: hợp đồng tạo trong kỳ,
  trạng thái khác `PENDING/CANCELLED` (giá trị hợp đồng).
- `commission` / `commission_paid`: `trainer_commissions` tạo trong kỳ (tổng / đã chi).
- Danh sách gồm mọi HLV có hồ sơ HLV (lọc chi nhánh theo `user_roles`) + HLV có phát sinh số liệu ở chi nhánh; sắp theo `pt_completed`, `classes_taught` giảm dần.

### GET `/admin/reports/sales-products`
Query: `branch_id`, `from`, `to`, `limit` (1–100, mặc định 10).
```json
{
  "period": { "from": "2026-10-01", "to": "2026-10-14", "days": 14 },
  "totals": { "product_id": 0, "sku": "", "name": "Tổng", "category_name": "", "quantity": 154, "revenue": 18350000,
              "cost": 11200000, "gross_profit": 7150000, "margin": 39 },
  "top": [
    { "product_id": 3, "sku": "WHEY-ON-2LB", "name": "Whey Gold Standard 2lb", "category_name": "Thực phẩm bổ sung",
      "quantity": 12, "revenue": 13200000, "cost": 9600000, "gross_profit": 3600000, "margin": 27.3 }
  ],
  "by_category": [ { "category_name": "Thực phẩm bổ sung", "quantity": 40, "revenue": 15000000, "gross_profit": 5000000, "pct": 81.7 } ],
  "product_count": 17
}
```
- Dòng `PRODUCT` của hóa đơn `PAID/PARTIAL` lập (`issued_at`) trong kỳ. `revenue` đã trừ giảm giá cấp hóa đơn (phân bổ theo `total/subtotal`).
- `cost`: lấy từ thẻ kho `stock_transactions` loại `SALE` của chính hóa đơn (`reference_id` = invoice id, `unit_cost`); không có thì `products.cost_price × số lượng`.
- `gross_profit = revenue − cost`, `margin = gross_profit / revenue`. Sản phẩm không có danh mục → "Chưa phân loại". `totals` tính trên mọi sản phẩm (không chỉ top).

### GET `/admin/reports/subscriptions`
Query: `branch_id` (chi nhánh của gói), `from`, `to`, `group_by`.
```json
{
  "period": { "from": "2026-10-01", "to": "2026-10-14", "days": 14 },
  "group_by": "month",
  "counts": { "new": 25, "renewed": 18, "upgraded": 3, "transferred": 1, "cancelled": 2, "expired": 20, "frozen": 4 },
  "series": [
    { "period": "2026-10-01", "label": "10/2026", "new": 25, "renewed": 18, "upgraded": 3, "transferred": 1, "cancelled": 2, "expired": 20, "frozen": 4 }
  ],
  "by_plan": [
    { "plan_id": 2, "plan_code": "M3", "plan_name": "Gói 3 tháng", "plan_type": "TIME", "sold": 30, "new_sales": 18,
      "continued": 12, "sold_amount": 52650000, "collected": 49800000, "pct": 61.2 }
  ]
}
```
- `counts` / `series` đếm sự kiện `subscription_events` trong kỳ: `new` = gói tạo mới không có gói trước (`previous_id` null);
  `renewed` / `upgraded` / `transferred` / `cancelled` / `expired` / `frozen` = sự kiện tương ứng.
- `by_plan`: `sold` = gói tạo trong kỳ (trừ `PENDING/CANCELLED`), `new_sales` / `continued` (có gói trước: gia hạn, nâng cấp, chuyển nhượng),
  `sold_amount` = Σ giá gói; `collected` = thực thu phân bổ cho dòng `MEMBERSHIP` của gói đó trong kỳ; `pct` = tỷ trọng `collected`.

### GET `/admin/reports/export` – Xuất Excel (`report.export`)
Query: `type` (bắt buộc) = `revenue` | `members` | `checkins` | `subscriptions` | `invoices`, cùng `branch_id`, `from`, `to`.

Response là **file** (không bọc envelope):
```
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename="bao-cao-doanh-thu_20261001_20261014.xlsx"; filename*=UTF-8''bao-cao-doanh-thu_20261001_20261014.xlsx
Access-Control-Expose-Headers: Content-Disposition
```
Tên file: `bao-cao-{doanh-thu|hoi-vien|check-in|goi-tap|hoa-don}_{yyyyMMdd from}_{yyyyMMdd to}.xlsx`.
Lỗi (validate, quyền…) vẫn trả JSON envelope → FE kiểm tra `Content-Type` trước khi lưu file.

| `type` | Sheet "Tổng hợp" | Sheet chi tiết |
|---|---|---|
| `revenue` | Tổng thu, hoàn, thực thu, kỳ trước, % thay đổi; bảng theo loại / phương thức / chi nhánh | "Theo thời gian": theo ngày (kỳ ≤ 62 ngày) hoặc theo tháng; cột từng loại + tổng thu/hoàn/thực thu, dòng Tổng |
| `members` | Phân bố hội viên mới theo nguồn, giới tính, độ tuổi, chi nhánh; trạng thái hiện tại | "Danh sách": hội viên **đăng ký trong kỳ** (mã, họ tên, ngày sinh, SĐT, email, chi nhánh, nguồn, gói hiện tại, hết hạn) |
| `checkins` | Lượt hợp lệ/từ chối, số hội viên, TB/ngày, giờ cao điểm, heatmap thứ × giờ | "Danh sách": nhật ký ra vào (giờ vào/ra, hội viên, phương thức, kết quả, lý do từ chối, nhân viên) |
| `subscriptions` | Bán mới / gia hạn / nâng cấp / chuyển nhượng / hủy / hết hạn / bảo lưu; doanh số theo gói | "Danh sách": đăng ký tạo trong kỳ |
| `invoices` | Số HĐ, tổng tiền, đã thu, đã hoàn theo trạng thái | "Danh sách": hóa đơn lập trong kỳ |

Định dạng: tiêu đề bảng in đậm nền cam, cố định dòng tiêu đề + AutoFilter, độ rộng cột đặt sẵn, tiền `#,##0 ₫`, ngày `dd/mm/yyyy`,
thời điểm `dd/mm/yyyy hh:mm` (giờ VN), dòng Tổng in đậm. Tối đa 50.000 dòng chi tiết (vượt → ghi chú ở ô A3 sheet Tổng hợp).

FE tải file:
```ts
const res = await fetch(`${API}/v1/admin/reports/export?type=revenue&from=2026-10-01&to=2026-10-14`, {
  headers: { Authorization: `Bearer ${token}` },
});
if (res.headers.get("content-type")?.includes("application/json")) throw await res.json();
const name = /filename="([^"]+)"/.exec(res.headers.get("content-disposition") ?? "")?.[1] ?? "bao-cao.xlsx";
const url = URL.createObjectURL(await res.blob()); // <a download={name} href={url}>
```

---

## 2. Thông báo in-app

### Đối tượng `Notification`
```json
{
  "id": 20, "type": "SUBSCRIPTION_EXPIRED", "title": "Gói tập đã hết hạn",
  "body": "Gói 3 tháng của bạn đã hết hạn ngày 13/10/2026. Liên hệ lễ tân GymMaster Cầu Giấy để gia hạn.",
  "link": "/member/subscriptions", "is_read": false, "read_at": null, "created_at": "2026-10-14T00:05:01+07:00"
}
```
`type`: `SUBSCRIPTION_EXPIRING` · `SUBSCRIPTION_EXPIRED` · `FREEZE_REQUEST` · `FREEZE_RESULT` · `BOOKING_PROMOTED` · `CLASS_CANCELLED` ·
`PT_SCHEDULED` · `PT_CANCELLED` · `LOW_STOCK` · `SYSTEM`. `link` là đường dẫn FE (có thể rỗng); `body` có thể nhiều dòng (`\n`).

### Endpoint (nhân viên `/admin/…` và hội viên `/member/…` giống hệt nhau)
Luôn thao tác trên thông báo của **người đang đăng nhập**.
- `GET /notifications` – query: `page`, `limit`, `unread=true` (chỉ chưa đọc), `type`, `order_by` (`created_at` | `id`), `order_dir`.
  Response: `{ items: Notification[], total, total_page, current_page, limit }` (mới nhất trước).
- `GET /notifications/unread-count` → `{ "count": 3 }` (gợi ý FE poll 30–60 giây cho badge chuông).
- `POST /notifications/{id}/read` → `Notification` sau khi đánh dấu (gọi lại không lỗi). Thông báo của người khác / không tồn tại → `40400`.
- `POST /notifications/read-all` → `{ "updated": 5 }`.

`/admin/notifications` mở cho mọi tài khoản đã đăng nhập (không cần quyền riêng); `/member/notifications` cần role `MEMBER`
(nhân viên gọi → `40300`).

### Thông báo do job hệ thống tạo

| `type` | Khi nào | Người nhận | `link` |
|---|---|---|---|
| `SUBSCRIPTION_EXPIRED` | Job `expire_subscriptions` chuyển gói sang EXPIRED (và hội viên chưa có gói khác còn hiệu lực) | Hội viên có tài khoản cổng | `/member/subscriptions` |
| `SUBSCRIPTION_EXPIRING` | Job `remind_expiring` (còn N ngày theo `subscription.remind_before_days`) | Hội viên có tài khoản cổng (+ email nếu có) | `/member/subscriptions` |
| `SUBSCRIPTION_EXPIRING` (tổng hợp) | Job `remind_expiring`, mỗi chi nhánh một thông báo, liệt kê tối đa 10 hội viên (tên, SĐT, gói, ngày hết hạn) | Nhân viên có quyền `subscription.sell` tại chi nhánh (+ OWNER) | `/admin/subscriptions?status=expiring&branch_id={id}` |

---

## 3. Nhật ký thao tác

### GET `/admin/audit-logs`
Query: `page`, `limit`, `keyword` (tên/email người thao tác, entity, action), `actor_id`, `entity`, `entity_id`, `action`,
`from`, `to` (ngày giờ VN, tính cả ngày `to`), `order_by` (`a.created_at` | `a.id`), `order_dir`.
```json
{
  "items": [
    { "id": 439, "actor_id": 1, "actor_name": "Quản trị viên", "actor_email": "admin@gymmaster.vn",
      "action": "UPGRADE", "entity": "subscription", "entity_id": 329, "ip": "::1", "created_at": "2026-10-14T09:55:17.784669+07:00" }
  ],
  "total": 144, "total_page": 72, "current_page": 1, "limit": 2
}
```
`actor_id = null` → `actor_name = "Hệ thống"` (job định kỳ). Danh sách không trả before/after (cho nhẹ); xem chi tiết để có diff.

### GET `/admin/audit-logs/options`
```json
{ "entities": ["invoice", "member", "subscription", "subscription_freeze"], "actions": ["CREATE", "DELETE", "SELL", "UPDATE", "UPGRADE"] }
```

### GET `/admin/audit-logs/{id}`
```json
{
  "id": 439, "actor_id": 1, "actor_name": "Quản trị viên", "actor_email": "admin@gymmaster.vn",
  "action": "UPGRADE", "entity": "subscription", "entity_id": 329, "ip": "::1", "created_at": "2026-10-14T09:55:17.784669+07:00",
  "before": { "status": "ACTIVE", "plan_id": 2, "subscription_id": 329 },
  "after": { "total": 3450000, "credit": 1350000, "plan_id": 3, "invoice_id": 62, "new_subscription_id": 330 },
  "changes": [
    { "field": "credit", "before": null, "after": 1350000 },
    { "field": "invoice_id", "before": null, "after": 62 },
    { "field": "new_subscription_id", "before": null, "after": 330 },
    { "field": "plan_id", "before": 2, "after": 3 },
    { "field": "status", "before": "ACTIVE", "after": null },
    { "field": "subscription_id", "before": 329, "after": null },
    { "field": "total", "before": null, "after": 3450000 }
  ]
}
```
- `before` / `after`: JSON gốc (object, mảng hoặc `null`). `changes`: các trường **cấp 1** khác nhau (so sánh theo giá trị JSON, sắp theo tên);
  trường thiếu ở một bên hiển thị `null`. Nếu before/after không phải object → một phần tử với `field: ""`. Không tìm thấy → `40400`.

---

## 4. Job định kỳ

### Danh sách job hệ thống (giờ `Asia/Ho_Chi_Minh`)

| `name` | `spec` | Việc làm | `affected` |
|---|---|---|---|
| `expire_subscriptions` | `5 0 * * *` (00:05) | `Subscriptions.ExpireDue(today)`: ACTIVE/FROZEN quá `end_date` → EXPIRED; PENDING chưa thu đồng nào quá 3 ngày → void → CANCELLED. Sau đó báo `SUBSCRIPTION_EXPIRED` cho hội viên có tài khoản cổng. | số gói đổi trạng thái |
| `apply_freezes` | `10 0 * * *` (00:10) | `Subscriptions.ApplyDueFreezes(today)`: kết thúc bảo lưu đến hạn, áp bảo lưu đã duyệt đến ngày. | số bảo lưu xử lý |
| `remind_expiring` | `0 8 * * *` (08:00) | `ListExpiring(settings subscription.remind_before_days, mặc định [7,1])` → in-app + email cho hội viên, thông báo tổng hợp cho lễ tân từng chi nhánh. | số gói được nhắc |
| `mark_class_no_shows` | `*/15 * * * *` | `BookingService.MarkNoShows(now)`: BOOKED của lớp đã kết thúc > 30 phút → NO_SHOW, khóa đặt lớp khi vắng quá giới hạn, lớp đã qua → COMPLETED. | số lượt NO_SHOW |
| `cleanup_tokens` | `0 3 * * *` (03:00) | Xóa `refresh_tokens` hết hạn hoặc bị thu hồi quá 30 ngày. | số token xóa |

Đặc điểm runner (`internal/cron`):
- Scheduler (robfig/cron v3, múi giờ VN) chỉ chạy khi `CRON_ENABLED=true`; khởi động trong `cmd` **sau** `router.Register`.
- Mỗi lần chạy (theo lịch hoặc thủ công) ghi `job_runs`: `RUNNING` → `SUCCESS` / `FAILED` + `affected` + `message` (lỗi).
  Job lỗi một phần vẫn ghi `affected` = số đã xử lý và `FAILED`. Panic trong job được bắt → `FAILED`.
- Chống chạy chồng: mutex riêng mỗi job trong tiến trình + `pg_try_advisory_lock(hashtext('gymmaster.job.<name>'))` giữa nhiều instance
  (instance không lấy được khóa bỏ qua lượt đó, không ghi `job_runs`).
- Job luôn chạy với tư cách **hệ thống** (không mang thông tin đăng nhập của người bấm "Chạy ngay"; `actor_id = NULL` trong audit/sự kiện),
  timeout mặc định 10 phút, không bị hủy khi client ngắt kết nối.
- Khi bật scheduler, các `job_runs` còn `RUNNING` quá 6 giờ (tiến trình chết giữa chừng) được đánh dấu `FAILED`.
- Chạy thủ công `remind_expiring` nhiều lần trong ngày sẽ gửi lặp thông báo/email (không chống trùng theo ngày).

### GET `/admin/jobs`
```json
[
  { "name": "expire_subscriptions", "spec": "5 0 * * *",
    "description": "Chuyển gói tập quá hạn sang EXPIRED, hủy gói chờ thanh toán quá 3 ngày, báo hội viên",
    "scheduled": true, "running": false, "next_run_at": "2026-10-15T00:05:00+07:00",
    "last_run": { "id": 12, "job": "expire_subscriptions", "started_at": "2026-10-14T00:05:00.077+07:00",
                  "finished_at": "2026-10-14T00:05:00.107+07:00", "duration_ms": 30, "status": "SUCCESS", "affected": 1, "message": "" } },
  { "name": "mark_class_no_shows", "spec": "*/15 * * * *",
    "description": "Đánh dấu NO_SHOW lượt đặt lớp đã kết thúc, khóa đặt lớp khi vắng quá giới hạn",
    "scheduled": true, "running": false, "next_run_at": "2026-10-14T10:15:00+07:00", "last_run": null }
]
```
`scheduled = false` khi `CRON_ENABLED=false` (khi đó `next_run_at` vẫn được tính theo `spec` để hiển thị). `running`: đang chạy trong instance này. Sắp theo tên.

### GET `/admin/jobs/{name}/runs`
Query: `page`, `limit`, `status` (`RUNNING` | `SUCCESS` | `FAILED`), `order_by` (`started_at` | `id`), `order_dir`.
Response: `{ items: JobRun[], total, total_page, current_page, limit }` – `JobRun` như `last_run` ở trên (`duration_ms` null khi đang chạy).
Job không tồn tại → `40400`.

### POST `/admin/jobs/{name}/run`
Chạy ngay, **đồng bộ** (chờ job xong), trả `JobRun`. Lỗi nghiệp vụ của job nằm trong `status: "FAILED"` + `message` (HTTP 200).
Job đang chạy (ở đây hoặc instance khác) → `40900` "Job đang chạy, vui lòng thử lại sau". Không tồn tại → `40400`.

### POST `/admin/jobs/test-email`
Body `{ "to": "ban@example.com" }` → gửi đồng bộ một email thử, trả `{ "sent": true, "to": "ban@example.com" }`; lỗi SMTP → `40000` kèm lý do.

### Thêm job mới từ module khác (PT, POS…)
Đăng ký trong hàm `Register<M>` của module (`internal/module/<m>.go`). Hàm này được gọi trong `router.Register`, tức **trước** khi `cmd` bật scheduler:
```go
import "gymmaster-api/internal/cron"

func RegisterPT(d *Deps, admin *ghttp.RouterGroup) {
	svc := /* ... */
	cron.Register(cron.Job{
		Name:        "expire_pt_contracts",          // snake_case, duy nhất, ≤ 50 ký tự (= job_runs.job)
		Spec:        "15 0 * * *",                   // cron 5 trường hoặc @daily/@hourly/@every 1h, giờ VN
		Description: "Chuyển hợp đồng PT quá hạn sang EXPIRED",
		Run: func(ctx context.Context) (int, error) { // trả số bản ghi bị ảnh hưởng
			return svc.ExpireContracts(ctx, time.Now())
		},
		// Timeout: 5 * time.Minute,                 // tùy chọn, mặc định 10 phút
	})
	admin.Bind(/* controller */)
}
```
- Tên trùng / spec sai / thiếu `Run` → panic khi khởi động (phát hiện ngay).
- Job tự xuất hiện ở `GET /admin/jobs`, chạy thủ công được qua API và được scheduler lên lịch – không cần sửa file nào khác.
- Gợi ý job sẽ nối: `expire_pt_contracts` (PT – `ExpireContracts`), `low_stock_alert` (POS – `ListLowStock` →
  `Notifier.NotifyPermission("inventory.view", &branchID, service.NotifyLowStock, …)`).
- Job nên idempotent (chạy lại không sai dữ liệu) và xử lý từng bản ghi trong transaction riêng để một lỗi không chặn cả lô.

---

## 5. Cấu hình email

| Biến môi trường | Mặc định | Ghi chú |
|---|---|---|
| `SMTP_HOST` | `localhost` | Docker compose: `mailpit` |
| `SMTP_PORT` | `1025` | Mailpit SMTP |
| `SMTP_USER` / `SMTP_PASS` | (trống) | Để trống khi dùng Mailpit |
| `SMTP_FROM` | `GymMaster <no-reply@gymmaster.vn>` | |
| `CRON_ENABLED` | `true` | `false` = tắt scheduler (job chỉ chạy thủ công qua API) |

- Dev: `docker compose up -d mailpit` → mọi email xem tại http://localhost:8025 (không gửi ra ngoài). Kiểm tra nhanh: `POST /admin/jobs/test-email`.
- Gói `internal/pkg/mailer`: `mailer.Default(cfg.Mail)` (hoặc `module.NewMailer(d)`), `mailer.Render(template, data)` với template HTML tiếng Việt
  `subscription_expiring` (`ExpiringData`), `subscription_expired` (`ExpiredData`), `generic` (`GenericData`);
  `SendAsync` gửi nền, thử lại 3 lần (chờ 2s, 4s), tối đa 4 kết nối SMTP đồng thời, lỗi cuối cùng chỉ ghi log (không làm hỏng nghiệp vụ).
  `Send` gửi đồng bộ một lần.
