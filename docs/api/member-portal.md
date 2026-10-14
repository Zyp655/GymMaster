# API – Cổng hội viên (`/v1/member/*`)

Base URL (dev): `http://localhost:8000/v1`. Mọi endpoint cần `Authorization: Bearer <access_token>` của tài khoản có role
**`MEMBER`** (cấp qua `POST /admin/members/{id}/portal-account`). Đăng nhập/làm mới/đổi mật khẩu dùng chung `/auth/*`.
Envelope, mã lỗi, phân trang: xem `sprint-01-auth-rbac-branch.md`.

- Mỗi API khai báo `role:"MEMBER"` trong `g.Meta` → tài khoản nhân viên (kể cả OWNER) gọi vào nhận `40300`;
  tài khoản MEMBER gọi `/admin/*` cũng nhận `40300`.
- **Hội viên được xác định từ tài khoản đăng nhập** (`members.user_id = uid` trong token). API **không bao giờ** nhận
  `member_id` từ client. Truy cập bản ghi của hội viên khác (lượt đặt lớp, buổi PT, hợp đồng PT, gói tập, hóa đơn) → `40400`
  (không tiết lộ bản ghi có tồn tại).
- Tài khoản MEMBER chưa gắn hội viên → `40400 "Tài khoản chưa được liên kết với hội viên"`.
- Ngày `YYYY-MM-DD` (giờ VN), thời điểm RFC3339. Lỗi nghiệp vụ trả `message` tiếng Việt – FE hiển thị thẳng.

Mã nguồn: `api/portal/v1/portal.go`, `internal/service/portal_service.go` (+ `_test.go`), `internal/repository/portal_repository.go`,
`internal/controller/portal/`, `internal/module/portal.go` (`module.RegisterPortal`). Service dùng lại:
`CheckinService.IssueQR/ListMemberCheckins`, `SubscriptionService.ListMemberSubscriptions/ListMemberInvoices/RequestFreezeByMember`,
`InvoiceService.FindInvoice/Detail`, `BookingService.Book(…, PORTAL)/Cancel(id, true)/ListMemberBookings/GetBooking`,
`PTService.ListMemberContracts/ListMemberSessions/BookSessionByMember/CancelSessionByMember`, `MemberService.Get/ListBodyMetrics`.

## Danh sách endpoint

| Method | Path | Mô tả |
|---|---|---|
| GET | `/member/profile` | Hồ sơ + chi nhánh chính + gói hiện tại |
| PUT | `/member/profile` | Sửa email, địa chỉ, người liên hệ khẩn cấp, ảnh đại diện |
| POST | `/member/uploads` | Tải ảnh đại diện (multipart `file`, JPG/PNG/WEBP ≤ 2 MB) |
| GET | `/member/qr` | Mã QR vào cửa (động) |
| GET | `/member/subscriptions` | Gói tập (mới nhất trước) + hạn mức + yêu cầu bảo lưu |
| POST | `/member/subscriptions/{id}/freezes` | Yêu cầu bảo lưu |
| GET | `/member/checkins` | Lịch sử ra vào (phân trang) |
| GET | `/member/branches` | Chi nhánh được dùng (theo phạm vi gói) |
| GET | `/member/classes?branch_id&from&to` | Lịch lớp có thể đặt + trạng thái đặt của tôi |
| POST | `/member/classes/{scheduleId}/book` | Đặt chỗ (hết chỗ → danh sách chờ) |
| GET | `/member/bookings?from&to` | Lớp đã đặt |
| POST | `/member/bookings/{id}/cancel` | Hủy lượt đặt (trước hạn hủy) |
| GET | `/member/pt-contracts` | Hợp đồng PT |
| GET | `/member/pt-contracts/{id}/trainer-availability?date=` | Khung giờ trống của HLV trong ngày |
| GET | `/member/pt-sessions?from&to` | Buổi PT |
| POST | `/member/pt-sessions` | Đặt buổi PT |
| POST | `/member/pt-sessions/{id}/cancel` | Hủy buổi PT (trước hạn hủy) |
| GET | `/member/body-metrics` | Chỉ số cơ thể (tăng dần theo `measured_at`) |
| GET | `/member/invoices` | Hóa đơn (phân trang) |
| GET | `/member/invoices/{id}` | Chi tiết hóa đơn |
| GET/POST | `/member/notifications…` | Thông báo – module hệ thống (xem `reports-system.md`) |

---

## 1. Hồ sơ

### GET `/member/profile`
```json
{
  "member": { "...": "đối tượng Member của members.md §2 (note luôn rỗng)" },
  "branch": { "id": 1, "code": "HN01", "name": "GymMaster Cầu Giấy", "address": "...", "phone": "...",
              "open_time": "05:30", "close_time": "22:00", "is_home": true },
  "current_subscription": { "...": "Subscription (mục 3)" },
  "has_valid_subscription": true
}
```
- `current_subscription`: gói `ACTIVE/FROZEN` có `start_date ≤ hôm nay ≤ end_date` và `end_date` xa nhất; nếu không có thì gói
  `ACTIVE` sắp bắt đầu sớm nhất (FE hiển thị "Bắt đầu từ …"); `null` nếu không có gói nào.
- `has_valid_subscription`: có gói `ACTIVE` trong hạn hôm nay (gói theo buổi còn buổi) – tức là vào cửa được.

### PUT `/member/profile`
```json
{ "email": "an@example.com", "address": "Cầu Giấy", "emergency_contact_name": "Mẹ",
  "emergency_contact_phone": "0988111222", "avatar_url": "http://localhost:8000/uploads/2026/10/x.png" }
```
Gửi **đủ** 5 trường (trống = xóa). Không sửa được họ tên, SĐT, ngày sinh, CCCD, chi nhánh (liên hệ lễ tân).
- `email` hợp lệ (lưu chữ thường) – `51 "Email không hợp lệ"`.
- `emergency_contact_phone`: SĐT Việt Nam (chuẩn hóa như hồ sơ) – `40000 "Số điện thoại người liên hệ khẩn cấp không hợp lệ"`.
- `avatar_url`: rỗng hoặc URL do `/member/uploads` trả (cùng tiền tố `UPLOAD_PUBLIC_URL`) – `40000 "Ảnh đại diện không hợp lệ, vui lòng tải ảnh lên lại"`.
- Ghi audit `PORTAL_UPDATE_PROFILE` (entity `member`). Response: như GET.

### POST `/member/uploads`
`multipart/form-data`, field `file`. Định dạng kiểm tra theo nội dung thật. Response `{ "url": "http://…/uploads/2026/10/<uuid>.png" }`.
Lỗi `40000`: `"Chỉ chấp nhận ảnh JPG, PNG hoặc WEBP"`, `"Ảnh vượt quá dung lượng cho phép (2 MB)"`, `"Tệp rỗng"`; `51 "Vui lòng chọn tệp ảnh"`.

## 2. QR vào cửa

### GET `/member/qr`
```json
{ "token": "GM1.382.1791947408.B-K2HlsDRFyma-RIMvN1Wr", "expires_at": "2026-10-14T10:10:08+07:00",
  "ttl": 60, "server_time": "2026-10-14T10:09:08.81+07:00", "has_valid_subscription": true }
```
- Cùng định dạng/chữ ký với `GET /admin/members/{id}/qr` (`checkin.md` §1). `ttl` = `checkin.qr_ttl_seconds`.
- FE: thời gian còn lại = `expires_at − server_time` (bù lệch đồng hồ máy), đếm ngược và gọi lại khi còn ~5 giây.
- `has_valid_subscription = false` → vẫn trả mã nhưng FE nên cảnh báo (quét sẽ bị từ chối).

## 3. Gói tập & bảo lưu

### GET `/member/subscriptions` → `Subscription[]`
`Subscription` = đối tượng `Subscription` của `sales.md` §1 (`note` luôn rỗng) **+**
```json
{ "includes_classes": true, "max_freeze_days": 15, "max_freeze_times": 1,
  "freezes": [ { "id": 6, "subscription_id": 341, "start_date": "2026-10-20", "end_date": "2026-10-25", "days": 6,
                 "reason": "Đi công tác", "status": "PENDING", "review_note": "", "reviewed_at": null, "created_at": "..." } ] }
```
Hạn mức còn lại: `max_freeze_times − freeze_times_used` lần, `max_freeze_days − freeze_days_used` ngày (yêu cầu PENDING/APPROVED
chưa áp cũng bị tính khi gửi yêu cầu mới).

### POST `/member/subscriptions/{id}/freezes`
Body `{ "start_date": "2026-10-20", "end_date": "2026-10-25", "reason": "Đi công tác" }`. Cùng quy tắc `POST /admin/subscriptions/{id}/freezes`
(`sales.md` §2) → yêu cầu `PENDING`, nhân viên có quyền duyệt nhận thông báo; hội viên nhận `FREEZE_RESULT` khi được xử lý.
Response: `Freeze` (rút gọn như trên). Gói của người khác → `40400 "Không tìm thấy gói tập của hội viên"`.

## 4. Check-in

### GET `/member/checkins?page=&limit=`
`PageRes<CheckinItem>` (`checkin.md` §4), mới nhất trước – gồm cả lượt `DENIED` (`deny_message`).

## 5. Lớp nhóm

### GET `/member/branches` → `Branch[]`
Chi nhánh chính + chi nhánh của gói `ACTIVE/FROZEN` còn hạn; có gói `ALL_BRANCHES` → mọi chi nhánh đang hoạt động.
Chi nhánh chính đứng đầu (`is_home = true`).

### GET `/member/classes?branch_id=&from=&to=`
- `branch_id` mặc định = phần tử đầu của `/member/branches`; ngoài danh sách → `40300 "Gói tập của bạn không áp dụng tại chi nhánh này"`.
- `from` mặc định hôm nay, `to` mặc định `from + booking.advance_days`; tối đa 31 ngày.
- Chỉ buổi `SCHEDULED` **chưa bắt đầu**, sắp theo giờ.
```json
{
  "branch_id": 1, "from": "2026-10-14", "to": "2026-10-21", "advance_days": 7, "cancel_before_minutes": 120,
  "items": [ {
    "...": "ClassSchedule (classes.md §2: class_type_name, room_name, trainer_name, start_at, end_at, capacity, booked_count, waitlist_count, available_slots…)",
    "my_booking": { "...": "ClassBooking (classes.md §3) hoặc null" },
    "cancel_deadline": "2026-10-15T16:00:00+07:00",
    "bookable": true
  } ]
}
```
`my_booking`: lượt đặt chưa hủy của chính hội viên (`BOOKED`/`WAITLIST`/`ATTENDED`/`NO_SHOW`). `bookable = false` khi lớp quá xa
(> `advance_days`). Gợi ý nút: `my_booking` null → "Đặt chỗ" (`available_slots > 0`) / "Vào danh sách chờ"; có `my_booking` BOOKED/WAITLIST
→ "Hủy" (khóa khi đã quá `cancel_deadline`).

### POST `/member/classes/{scheduleId}/book`
Không body. `BookingService.Book(…, PORTAL)` – quy tắc như đặt hộ tại quầy (`classes.md` §3): đặt trước tối đa `advance_days`,
hội viên không bị khóa đặt lớp, có gói bao gồm lớp hợp lệ ngày đó và đúng chi nhánh, không trùng giờ lớp khác; hết chỗ → `WAITLIST`.
Response `Booking` = `ClassBooking` + `{ "cancel_deadline": "...", "can_cancel": true }`.
Lỗi thường gặp: `40000 "Gói tập của hội viên không bao gồm lớp nhóm"`, `"Chỉ được đặt lớp trước tối đa 7 ngày"`,
`"Buổi lớp đã bắt đầu, không thể đặt chỗ"`, `"Hội viên bị khóa đặt lớp đến … do vắng mặt nhiều lần"`; `40900 "Hội viên đã đặt buổi lớp này"`,
`"Hội viên đã đặt một lớp khác trùng giờ"`.

### GET `/member/bookings?from=&to=` → `Booking[]`
Mặc định 30 ngày trước → 30 ngày tới (theo giờ bắt đầu lớp), tối đa 366 ngày; sắp theo `start_at`.
`can_cancel` = `BOOKED/WAITLIST`, lớp chưa hủy, chưa quá `cancel_deadline` (= `start_at − booking.cancel_before_minutes`).

### POST `/member/bookings/{id}/cancel`
Kiểm tra lượt đặt thuộc hội viên (khác → `40400 "Không tìm thấy lượt đặt chỗ"`) rồi `Cancel(id, byMember=true)`.
Quá hạn → `40000 "Chỉ được hủy đặt lớp trước giờ học 120 phút"`. Hủy lượt `BOOKED` tự đẩy người đầu danh sách chờ lên (thông báo `BOOKING_PROMOTED`).
Response: `Booking` sau khi hủy.

## 6. PT

### GET `/member/pt-contracts` → `Contract[]` (`pt.md` §1, `note` rỗng), mới nhất trước.

### GET `/member/pt-sessions?from=&to=`
Mặc định 30 ngày trước → 30 ngày tới, tối đa 62 ngày.
```json
{ "cancel_before_minutes": 720,
  "items": [ { "...": "Session (pt.md §1: workout_note, trainer_note…)", "cancel_deadline": "2026-10-15T20:00:00+07:00", "can_cancel": true } ] }
```

### GET `/member/pt-contracts/{id}/trainer-availability?date=YYYY-MM-DD`
Khung giờ bước **30 phút** trong giờ mở cửa chi nhánh của hợp đồng; buổi dài `session_minutes` phải kết thúc trước giờ đóng cửa.
```json
{ "date": "2026-10-15", "contract_id": 44, "trainer_id": 62, "trainer_name": "Lê Văn HLV", "branch_id": 1, "branch_name": "GymMaster Cầu Giấy",
  "open_time": "05:30", "close_time": "22:00", "session_minutes": 60,
  "slots": [ { "start_at": "2026-10-15T05:30:00+07:00", "end_at": "2026-10-15T06:30:00+07:00", "available": true, "reason": "" },
             { "start_at": "2026-10-15T17:30:00+07:00", "end_at": "2026-10-15T18:30:00+07:00", "available": false, "reason": "TRAINER_BUSY" } ] }
```
`reason` (theo thứ tự ưu tiên): `OUT_OF_CONTRACT` (ngày ngoài thời hạn hợp đồng) · `PAST` (đã qua) · `TRAINER_BUSY`
(HLV có buổi PT `SCHEDULED/COMPLETED` hoặc lớp chưa hủy giao nhau) · `MEMBER_BUSY` (chính hội viên đã có buổi PT / lớp đã đặt giao nhau).
Hợp đồng không thuộc hội viên → `40400 "Không tìm thấy hợp đồng PT"`; hợp đồng khác `ACTIVE` → `40000`.
Khung "trống" chỉ là gợi ý – khi đặt, service PT kiểm tra lại toàn bộ trong transaction.

### POST `/member/pt-sessions`
Body `{ "contract_id": 44, "start_at": "2026-10-16T08:00" }` (RFC3339 hoặc `YYYY-MM-DDTHH:mm` giờ VN).
`PTService.BookSessionByMember` – cùng quy tắc `POST /admin/pt-sessions` (`pt.md` §3), `source = PORTAL`; hợp đồng của người khác → `40400`.
Response: `PTSession` (`Session` + `cancel_deadline`, `can_cancel`).

### POST `/member/pt-sessions/{id}/cancel`
Body `{ "reason": "Bận" }` (trống → "Hội viên hủy qua cổng hội viên"). Chỉ hủy **đúng hạn** (`start_at − pt.cancel_before_minutes`):
muộn → `40000 "Chỉ được hủy buổi PT trước giờ tập ít nhất 12 giờ, vui lòng liên hệ lễ tân"`. Buổi của người khác → `40400 "Không tìm thấy buổi PT"`.

## 7. Chỉ số cơ thể & hóa đơn

- `GET /member/body-metrics` → `BodyMetric[]` (`members.md` §3), tăng dần theo `measured_at` – dùng làm dữ liệu biểu đồ.
- `GET /member/invoices?page=&limit=` → `PageRes<Invoice>` (`sales.md` §3).
- `GET /member/invoices/{id}` → `InvoiceDetail`. Hóa đơn của người khác / khách vãng lai → `40400 "Không tìm thấy hóa đơn"`.

## 8. Thông báo
`/member/notifications`, `/member/notifications/unread-count`, `/member/notifications/{id}/read`, `/member/notifications/read-all`
(module hệ thống). `link` trong thông báo có dạng `/member/bookings`, `/member/pt-sessions`… – FE cổng hội viên đổi tiền tố
`/member/` → `/me/` (và `pt-sessions` → `pt`).

## 9. Kiểm thử
- Unit test (fake): `internal/service/portal_service_test.go` – mọi quy tắc "chỉ dữ liệu của mình" (hủy lượt đặt / hóa đơn /
  hợp đồng PT / bảo lưu của người khác → 404 và không gọi service ghi), lọc phòng thủ danh sách, phạm vi chi nhánh, khung giờ trống, hồ sơ, tải ảnh.
- Đã kiểm tra bằng curl trên server thật với 2 hội viên tạm (A không xem/hủy được lượt đặt, buổi PT, hợp đồng, gói, hóa đơn của B;
  nhân viên gọi `/member/*` và hội viên gọi `/admin/*` → `40300`).

## 10. Giới hạn
- Hội viên chưa tự mua/gia hạn gói qua cổng (thanh toán online chưa có) – hiển thị hóa đơn, mua tại quầy.
- Không giới hạn số lần tải ảnh; ảnh cũ không tự xóa khi đổi ảnh đại diện.
