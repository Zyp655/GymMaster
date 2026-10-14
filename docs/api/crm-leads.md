# API contract – Sprint 16 & 17 (CRM Khách tiềm năng, Chăm sóc tự động & Realtime SSE)

Base URL (dev): `http://localhost:8000/v1`.

## 1. Quyền (Permissions)

| Mã quyền | Mô tả | Vai trò mặc định |
|---|---|---|
| `lead.view` | Xem khách tiềm năng (CRM) | `OWNER`, `MANAGER`, `RECEPTIONIST` |
| `lead.manage` | Quản lý khách tiềm năng, cấp vé tập thử, chuyển đổi hội viên | `OWNER`, `MANAGER`, `RECEPTIONIST` |

---

## 2. API Công khai cho Landing Page (`/v1/public/*`)

### POST `/v1/public/leads`
Khách truy cập trang chủ gửi form "Đăng ký tập thử miễn phí". Có giới hạn tần suất theo IP (tối đa 5 lần / 15 phút). Tự động gửi thông báo `LEAD_NEW` cho lễ tân/quản lý chi nhánh.

Request:
```json
{
  "branch_id": 1,
  "full_name": "Trần Minh Khoa",
  "phone": "0987654321",
  "email": "khoa.tran@example.com",
  "interest_plan_id": 2,
  "note": "Muốn tập thử khung giờ 18h"
}
```

### GET `/v1/public/class-schedules`
Lịch lớp nhóm công khai trong tuần cho trang chủ.
- Query: `branch_id` (tùy chọn), `from` (`YYYY-MM-DD`), `to` (`YYYY-MM-DD`).

---

## 3. Quản lý CRM Khách tiềm năng (`/v1/admin/leads`)

- Nguồn (`source`): `WEBSITE` | `WALK_IN` | `REFERRAL` | `FACEBOOK` | `OTHER`.
- Trạng thái (`status`): `NEW` → `CONTACTED` → `TRIAL` → `CONVERTED` | `LOST`.
- Loại tương tác (`lead_activities.type`): `CALL` | `SMS` | `VISIT` | `NOTE`.

| Method | Path | Quyền | Mô tả |
|---|---|---|---|
| GET | `/v1/admin/leads` | `lead.view,lead.manage` | Danh sách lead (phân trang, lọc `branch_id`, `status`, `source`, `search`) |
| GET | `/v1/admin/leads/{id}` | `lead.view,lead.manage` | Chi tiết lead kèm `activities` và `trial_passes` |
| POST | `/v1/admin/leads` | `lead.manage` | Tạo lead mới |
| PUT | `/v1/admin/leads/{id}` | `lead.manage` | Cập nhật thông tin / trạng thái lead |
| DELETE | `/v1/admin/leads/{id}` | `lead.manage` | Xóa mềm lead |
| POST | `/v1/admin/leads/{id}/activities` | `lead.manage` | Ghi nhật ký chăm sóc (nếu đang `NEW` tự chuyển sang `CONTACTED`) |
| POST | `/v1/admin/leads/{id}/trial-pass` | `lead.manage` | Cấp vé tập thử QR (`qr_token = "TP-..."`), chuyển lead sang `TRIAL` |
| POST | `/v1/admin/leads/{id}/convert` | `lead.manage` | Chuyển đổi lead thành Hội viên (`members`), chuyển lead sang `CONVERTED` |
| GET | `/v1/admin/members/at-risk` | `member.view,lead.view` | Danh sách hội viên có gói `ACTIVE` nhưng ≥ 14 ngày không check-in |
| GET | `/v1/admin/reports/leads` | `report.view,lead.view` | Báo cáo phễu chuyển đổi theo trạng thái và nguồn |

> **Quét vé tập thử tại quầy Check-in (`POST /v1/admin/checkins/scan`)**:
> Màn hình quét QR tại quầy lễ tân chấp nhận cả mã QR động của hội viên lẫn mã vé tập thử `TP-...` trong ngày hiệu lực (`valid_date`), tự động đánh dấu `trial_passes.used_at`.

---

## 4. Chăm sóc hội viên tự động (Cron Jobs)

- **`birthday_greetings`** (`0 8 * * *` – 08:00 hằng ngày):
  Tìm hội viên `ACTIVE` có ngày/tháng sinh nhật trùng hôm nay, tự động sinh mã voucher ưu đãi sinh nhật và gửi thông báo in-app `BIRTHDAY` + email chúc mừng.
- **`churn_risk_scan`** (`0 6 * * *` – 06:00 hằng ngày):
  Quét hội viên có gói `ACTIVE` nhưng 14 ngày không check-in và gửi thông báo nhắc chăm sóc cho lễ tân/quản lý từng chi nhánh.

---

## 5. Thông báo thời gian thực qua Server-Sent Events (Sprint 17)

### GET `/v1/notifications/stream`
- Xác thực bằng header `Authorization: Bearer <access_token>` hoặc query `?token=<access_token>`.
- Giữ kết nối `text/event-stream`, đẩy sự kiện `event: notification` ngay khi `Notifier.Notify` hoặc `Notifier.NotifyPermission` được gọi.
