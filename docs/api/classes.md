# API contract – Lớp nhóm: loại lớp, lịch lớp, đặt chỗ, waitlist, điểm danh (migration 000007)

Base URL (dev): `http://localhost:8000/v1`. Swagger: `/swagger`, tag "Lớp nhóm".
Envelope, mã lỗi, phân trang và header dùng chung: xem `sprint-01-auth-rbac-branch.md`.

Ký hiệu trong tài liệu này:
- **Ngày** là `"YYYY-MM-DD"`.
- **Thời điểm** là RFC3339, ví dụ `"2026-10-15T18:00:00+07:00"`. Khi gửi lên, server cũng nhận `"2026-10-15T18:00"` (hiểu theo giờ Việt Nam).
- Lỗi validate tag trả `code = 51` (HTTP 400), `message` là câu tiếng Việt.
- Mọi lỗi nghiệp vụ dùng mã chung: `40000` (vi phạm quy tắc), `40300`, `40400`, `40900` (trùng lịch / trùng đặt chỗ). Hiển thị `message` cho người dùng.

## Quyền

| Quyền | Vai trò mặc định | Dùng cho |
|---|---|---|
| `class.view` | OWNER, MANAGER, RECEPTIONIST, TRAINER | Xem loại lớp, lịch lớp, danh sách đặt chỗ |
| `class.manage` | OWNER, MANAGER | Thêm/sửa/xóa loại lớp; tạo/sửa/hủy lịch lớp |
| `booking.manage` | OWNER, MANAGER, RECEPTIONIST, TRAINER | Đặt chỗ hộ, hủy hộ, điểm danh |

## Phạm vi chi nhánh

Giống các module khác (xem `staff-rooms-shifts.md`):
- Truyền `branch_id` ngoài phạm vi vào **danh sách** → `40300`.
- Tạo lịch cho chi nhánh ngoài phạm vi → `40300`.
- Xem, sửa, hủy, đặt chỗ, điểm danh trên buổi lớp ngoài phạm vi → `40400`.
- Loại lớp là danh mục dùng chung toàn hệ thống (không theo chi nhánh).

## Dropdown dùng lại từ module khác

- Chi nhánh: `GET /admin/branches?limit=100`.
- Phòng: `GET /admin/rooms/options?branch_id=` → `[{id, branch_id, name, capacity}]`.
- HLV: `GET /admin/trainers/options?branch_id=`.
- Hội viên (đặt hộ): `GET /admin/members/options?keyword=` (xem `members.md`).

---

## 1. Loại lớp – `/admin/class-types`

### Đối tượng `ClassType`
```json
{
  "id": 5,
  "code": "YOGA-BASIC",
  "name": "Yoga cơ bản",
  "description": "",
  "color": "#22C55E",
  "default_duration_min": 60,
  "default_capacity": 20,
  "level": "BEGINNER",
  "status": "ACTIVE",
  "created_at": "2026-10-14T09:18:58+07:00",
  "updated_at": "2026-10-14T09:18:58+07:00"
}
```
- `code`: luôn lưu IN HOA, duy nhất (không phân biệt hoa thường). Chỉ gồm chữ, số, `-`, `_`; tối đa 30 ký tự.
- `color`: `#RRGGBB` (lưu IN HOA) – dùng làm màu sự kiện trên lịch tuần.
- `level`: `ALL` | `BEGINNER` | `INTERMEDIATE` | `ADVANCED`.
- `status`: `ACTIVE` | `INACTIVE`. Loại lớp `INACTIVE` không dùng để tạo lịch mới, nhưng các lịch cũ vẫn giữ.

### GET `/admin/class-types` (perm `class.view`)
Query: `page`, `limit`, `keyword` (mã, tên), `order_by` (`id`, `code`, `name`, `level`, `status`, `created_at`; mặc định `id`), `order_dir`, `status`, `level`.

Response `data`: `PageRes<ClassType>`.

### GET `/admin/class-types/options` (perm `class.view`)
Các loại lớp `ACTIVE`, sắp theo tên:
```json
[{ "id": 5, "code": "YOGA-BASIC", "name": "Yoga cơ bản", "color": "#22C55E", "level": "BEGINNER", "default_duration_min": 60, "default_capacity": 20 }]
```
Khi chọn loại lớp trong form tạo lịch, FE có thể điền sẵn `duration_min` và `capacity` từ đây.

### GET `/admin/class-types/{id}` (perm `class.view`)
Trả `ClassType`; `40400` nếu không có.

### POST `/admin/class-types` (perm `class.manage`)
```json
{
  "code": "yoga-basic",
  "name": "Yoga cơ bản",
  "description": "Lớp cho người mới",
  "color": "#22c55e",
  "default_duration_min": 60,
  "default_capacity": 20,
  "level": "BEGINNER",
  "status": "ACTIVE"
}
```
- Bắt buộc: `code`, `name`.
- Mặc định: `color` `#F97316`; `default_duration_min` 60 (15–240); `default_capacity` 20 (1–500); `level` `ALL`; `status` `ACTIVE`.
- Trùng mã → `40900` "Mã loại lớp đã tồn tại".

Response: `ClassType`.

### PUT `/admin/class-types/{id}` (perm `class.manage`)
Body như POST (gửi đủ các trường). Response: `ClassType`.

### DELETE `/admin/class-types/{id}` (perm `class.manage`)
Xóa mềm. Còn lịch `SCHEDULED` chưa diễn ra → `40900` "Không thể xóa: loại lớp còn lịch sắp diễn ra". Response `data: null`.

---

## 2. Lịch lớp – `/admin/class-schedules`

### Đối tượng `ClassSchedule`
```json
{
  "id": 4,
  "class_type_id": 5,
  "class_type_code": "YOGA-BASIC",
  "class_type_name": "Yoga cơ bản",
  "class_type_color": "#22C55E",
  "branch_id": 2,
  "branch_name": "GymMaster Hai Bà Trưng",
  "room_id": 2,
  "room_name": "Phòng Yoga 1",
  "trainer_id": 11,
  "trainer_name": "Nguyễn Văn HLV",
  "start_at": "2026-10-15T18:00:00+07:00",
  "end_at": "2026-10-15T19:00:00+07:00",
  "duration_min": 60,
  "capacity": 15,
  "booked_count": 12,
  "waitlist_count": 3,
  "attended_count": 0,
  "available_slots": 3,
  "status": "SCHEDULED",
  "series_id": "10653478-816a-420e-8981-9380748aff6d",
  "cancel_reason": "",
  "note": "Mang thảm",
  "created_at": "2026-10-14T09:19:08+07:00",
  "updated_at": "2026-10-14T09:19:08+07:00"
}
```
- `status`: `SCHEDULED` (sắp/đang diễn ra) | `CANCELLED` | `COMPLETED` (cron chuyển sau khi lớp kết thúc).
- `booked_count`: số chỗ đã chiếm = số lượt `BOOKED` + `ATTENDED` + `NO_SHOW` (điểm danh không làm giảm). `available_slots = capacity − booked_count` (≥ 0).
- `waitlist_count`: số lượt `WAITLIST`. `attended_count`: số lượt `ATTENDED`.
- `series_id`: `""` nếu là buổi lẻ; các buổi tạo bằng lịch lặp tuần có cùng `series_id`.

Gợi ý hiển thị trên lịch tuần (FullCalendar): `title = class_type_name`, `start/end = start_at/end_at`, `color = class_type_color`, phụ đề `room_name · trainer_name · booked_count/capacity (+waitlist_count chờ)`; buổi `CANCELLED` hiển thị mờ/gạch ngang.

### GET `/admin/class-schedules` – lịch trong khoảng ngày (perm `class.view`)
Query:
- `from`, `to` (**bắt buộc**, `YYYY-MM-DD`, bao gồm cả ngày `to`); tối đa 62 ngày.
- `branch_id`: lọc chi nhánh (không truyền = mọi chi nhánh trong phạm vi).
- `trainer_id`, `class_type_id`, `room_id`.
- `status`: `SCHEDULED` | `CANCELLED` | `COMPLETED`. Không truyền = tất cả (gồm cả buổi đã hủy).

Quy tắc: nếu người gọi **chỉ có vai trò TRAINER** (và không có `class.manage`) mà không truyền `trainer_id` thì chỉ trả lớp do chính họ dạy.

Lọc theo **giờ bắt đầu** trong khoảng `[from 00:00, to 24:00)`, sắp theo `start_at`.

Response `data`: mảng `ClassSchedule` (không phân trang).

Lỗi: `to < from` hoặc > 62 ngày → `40000`; `branch_id` ngoài phạm vi → `40300`.

### GET `/admin/class-schedules/{id}` (perm `class.view`)
Trả `ClassSchedule`.

### POST `/admin/class-schedules` – tạo một buổi (perm `class.manage`)
```json
{
  "class_type_id": 5,
  "branch_id": 2,
  "room_id": 2,
  "trainer_id": 11,
  "start_at": "2026-10-15T18:00:00+07:00",
  "duration_min": 60,
  "capacity": 15,
  "note": "Mang thảm"
}
```
- Bắt buộc: `class_type_id`, `branch_id`, `room_id`, `trainer_id`, `start_at`.
- `duration_min` (15–240): mặc định theo loại lớp.
- `capacity`: mặc định `min(default_capacity của loại lớp, sức chứa phòng)`.

Quy tắc kiểm tra (theo thứ tự):
1. Chi nhánh thuộc phạm vi (`40300`).
2. Loại lớp tồn tại và `ACTIVE`; chi nhánh tồn tại và `ACTIVE`.
3. Phòng tồn tại, `ACTIVE`, thuộc đúng chi nhánh.
4. `capacity ≤ rooms.capacity` → nếu vượt: `40000` "Sức chứa vượt quá sức chứa phòng (15)".
5. HLV là tài khoản `ACTIVE`, là HLV (hồ sơ `position = TRAINER` hoặc có vai trò TRAINER), có vai trò tại chi nhánh (hoặc vai trò toàn hệ thống).
6. `start_at` ở tương lai.
7. Buổi lớp nằm trọn trong giờ mở cửa chi nhánh, trong cùng một ngày → `40000` "Lịch lớp phải nằm trong giờ mở cửa của chi nhánh (05:30 - 22:00)".
8. Trùng lịch → `40900`:
   - "Phòng đã có lớp khác trùng giờ" (EXCLUDE `ex_class_room_overlap`);
   - "HLV đã có lớp khác trùng giờ" (EXCLUDE `ex_class_trainer_overlap`);
   - "HLV đã có buổi PT trùng giờ" (bảng `pt_sessions` trạng thái SCHEDULED/COMPLETED).
   Buổi đã hủy không tính.

Response: `ClassSchedule`. Ghi audit `CREATE class_schedule`.

### POST `/admin/class-schedules/recurring` – tạo lịch lặp tuần (perm `class.manage`)
```json
{
  "class_type_id": 5,
  "branch_id": 2,
  "room_id": 2,
  "trainer_id": 11,
  "start_date": "2026-10-12",
  "end_date": "2026-12-31",
  "weekdays": [1, 3, 5],
  "start_time": "18:00",
  "duration_min": 60,
  "capacity": 15,
  "note": ""
}
```
- `weekdays`: 1 = Thứ Hai … 7 = Chủ nhật.
- `end_date − start_date ≤ 83 ngày` (tối đa 12 tuần) → nếu vượt: `40000` "Lịch lặp tối đa 12 tuần".
- Các kiểm tra 1–5, 7 như tạo một buổi; vi phạm thì **cả request** lỗi.
- Mỗi ngày khớp `weekdays` được xét riêng; buổi nào không tạo được thì **bỏ qua** và ghi vào `skipped`:
  - `"Thời điểm đã qua"`;
  - `"Phòng đã có lớp khác trùng giờ"`;
  - `"HLV đã có lớp khác trùng giờ"`;
  - `"HLV đã có buổi PT trùng giờ"`;
  - `"Trùng lịch phòng hoặc HLV"` (bị EXCLUDE chặn do tạo đồng thời).
- Các buổi tạo được dùng chung `series_id`. Toàn bộ chạy trong 1 transaction.

Response:
```json
{
  "series_id": "10653478-816a-420e-8981-9380748aff6d",
  "created": 2,
  "schedule_ids": [5, 6],
  "skipped": [
    { "date": "2026-10-12", "reason": "Thời điểm đã qua" },
    { "date": "2026-10-15", "reason": "Phòng đã có lớp khác trùng giờ" }
  ]
}
```
Nếu `created = 0` thì `series_id = ""`. FE nên hiện thông báo "Đã tạo N buổi, bỏ qua M buổi" kèm bảng `skipped`.

### PUT `/admin/class-schedules/{id}` – sửa buổi (perm `class.manage`)
```json
{
  "room_id": 2,
  "trainer_id": 11,
  "start_at": "2026-10-15T18:30:00+07:00",
  "duration_min": 60,
  "capacity": 20,
  "note": "Dời 30 phút"
}
```
- Tất cả bắt buộc trừ `note` (gửi giá trị hiện tại cho trường không đổi). Không đổi được loại lớp và chi nhánh.
- Chỉ sửa buổi `SCHEDULED` chưa bắt đầu; giờ mới phải ở tương lai.
- `capacity ≥ booked_count` → nếu không: `40000` "Sức chứa không được nhỏ hơn số chỗ đã đặt (12)".
- Các kiểm tra phòng, HLV, giờ mở cửa, trùng lịch như khi tạo (bỏ qua chính buổi này).
- **Tăng sức chứa** khi đang có waitlist → tự đẩy người đầu danh sách chờ lên `BOOKED` và gửi thông báo `BOOKING_PROMOTED`.

Response: `ClassSchedule`. Ghi audit `UPDATE class_schedule`.

### POST `/admin/class-schedules/{id}/cancel` – hủy buổi (perm `class.manage`)
```json
{ "reason": "HLV nghỉ phép", "scope": "following" }
```
- `reason`: bắt buộc, tối đa 255 ký tự.
- `scope`: `one` (mặc định) – chỉ buổi này; `following` – buổi này và mọi buổi `SCHEDULED` **sau nó** trong cùng `series_id` (buổi lẻ thì như `one`).
- Không hủy được buổi đã bắt đầu/diễn ra hoặc đã hủy/hoàn thành → `40000`.
- Mỗi buổi bị hủy: `status = CANCELLED`, `cancel_reason`, `booked_count = 0`; mọi lượt `BOOKED`/`WAITLIST` → `CANCELLED`. Hội viên có tài khoản cổng nhận thông báo in-app `CLASS_CANCELLED`.
- Ghi audit `CANCEL class_schedule` cho từng buổi.

Response:
```json
{ "cancelled_schedules": 2, "cancelled_bookings": 1, "schedule_ids": [5, 6] }
```

---

## 3. Đặt chỗ, waitlist, điểm danh

### Đối tượng `ClassBooking`
```json
{
  "id": 102,
  "schedule_id": 4,
  "member_id": 135,
  "member_code": "GM000135",
  "member_name": "Trần Thị B",
  "member_phone": "0900000932",
  "subscription_id": 131,
  "status": "WAITLIST",
  "waitlist_position": 1,
  "source": "COUNTER",
  "booked_at": "2026-10-14T09:19:18+07:00",
  "cancelled_at": null,
  "attended_at": null,
  "class_type_id": 5,
  "class_type_name": "Yoga cơ bản",
  "class_type_color": "#22C55E",
  "branch_id": 2,
  "branch_name": "GymMaster Hai Bà Trưng",
  "room_name": "Phòng Yoga 1",
  "trainer_id": 11,
  "trainer_name": "Nguyễn Văn HLV",
  "start_at": "2026-10-15T18:00:00+07:00",
  "end_at": "2026-10-15T19:00:00+07:00",
  "schedule_status": "SCHEDULED"
}
```
- `status`: `BOOKED` | `WAITLIST` | `CANCELLED` | `ATTENDED` | `NO_SHOW`.
- `waitlist_position`: chỉ có khi `WAITLIST` (thứ tự tăng dần, có thể không liên tục sau khi có người hủy/được đẩy lên – FE hiển thị theo thứ tự sắp xếp); còn lại `null`.
- `source`: `COUNTER` (nhân viên đặt hộ) | `PORTAL` (hội viên tự đặt).
- `subscription_id`: gói tập dùng để đặt (đặt không trừ buổi).

### GET `/admin/class-schedules/{id}/bookings` (perm `class.view` hoặc `booking.manage`)
Query: `status` (tùy chọn) để lọc.

Response: mảng `ClassBooking`, đã sắp: nhóm `BOOKED`/`ATTENDED`/`NO_SHOW` (theo giờ đặt) → `WAITLIST` (theo vị trí) → `CANCELLED`. Dùng cho màn quản lý đặt chỗ và điểm danh.

### POST `/admin/class-schedules/{id}/bookings` – đặt hộ tại quầy (perm `booking.manage`)
```json
{ "member_id": 135 }
```
Quy tắc (kiểm tra trong transaction, sau khi `SELECT … FOR UPDATE` dòng lịch):
1. Buổi lớp thuộc phạm vi chi nhánh của nhân viên (`40400`).
2. Buổi `SCHEDULED` và chưa bắt đầu.
3. Ngày lớp ≤ hôm nay + `booking.advance_days` (mặc định 7) → "Chỉ được đặt lớp trước tối đa 7 ngày".
4. Hội viên `ACTIVE`; không bị khóa (`booking_blocked_until` > hiện tại → "Hội viên bị khóa đặt lớp đến 15:04 21/10/2026 do vắng mặt nhiều lần").
5. Có subscription `ACTIVE` thỏa **tất cả**: gói `includes_classes = true`; `start_date ≤ ngày lớp ≤ end_date`; phạm vi chi nhánh (`SINGLE_BRANCH` → subscription cùng chi nhánh lớp, `ALL_BRANCHES` → mọi chi nhánh); gói `SESSION` còn `remaining_sessions > 0`. Không thỏa thì thông báo theo điều kiện gần đạt nhất:
   - "Hội viên không có gói tập đang hoạt động";
   - "Gói tập của hội viên không bao gồm lớp nhóm";
   - "Gói tập của hội viên không còn hiệu lực vào ngày diễn ra lớp";
   - "Gói tập của hội viên không áp dụng tại chi nhánh này";
   - "Gói tập theo buổi của hội viên đã hết buổi".
6. Chưa có lượt `BOOKED`/`WAITLIST`/`ATTENDED` trong buổi này → `40900` "Hội viên đã đặt buổi lớp này".
7. Không trùng giờ với lớp khác hội viên đã đặt (`BOOKED`/`WAITLIST`) → `40900` "Hội viên đã đặt một lớp khác trùng giờ".
8. `booked_count < capacity` → `BOOKED` (`booked_count + 1`); hết chỗ → `WAITLIST` với `waitlist_position = max + 1`.

Lỗi 1–5 là `40000`/`40400`. Response: `ClassBooking` (FE báo "Đã đặt chỗ" hoặc "Đã vào danh sách chờ (#n)" theo `status`).

### POST `/admin/class-bookings/{id}/cancel` – hủy hộ (perm `booking.manage`)
Không có body.
- Chỉ hủy lượt `BOOKED`/`WAITLIST` → ngược lại `40000` "Lượt đặt chỗ không còn hiệu lực để hủy".
- Nhân viên hủy hộ **không giới hạn thời gian** (hội viên tự hủy qua cổng phải trước `booking.cancel_before_minutes`, mặc định 120 phút).
- Hủy lượt `BOOKED` của buổi chưa bắt đầu: `booked_count − 1` rồi **tự đẩy người đầu waitlist lên `BOOKED`** trong cùng transaction; người được đẩy lên nhận thông báo `BOOKING_PROMOTED`.

Response: `ClassBooking` sau khi hủy.

### PUT `/admin/class-schedules/{id}/attendance` – điểm danh (perm `booking.manage`)
```json
{
  "items": [
    { "booking_id": 101, "status": "ATTENDED" },
    { "booking_id": 102, "status": "NO_SHOW" }
  ]
}
```
- Chỉ cho phép từ **30 phút trước giờ bắt đầu** đến **24 giờ sau khi kết thúc**:
  - "Chưa đến giờ điểm danh (mở từ 30 phút trước giờ học)";
  - "Đã quá thời hạn điểm danh (24 giờ sau khi lớp kết thúc)".
- Buổi đã hủy → `40000`.
- Mỗi `booking_id` phải thuộc buổi này và đang `BOOKED`, `ATTENDED` hoặc `NO_SHOW` (được sửa lại). Lượt `WAITLIST`/`CANCELLED` → `40000`.
- `ATTENDED` ghi `attended_at = now`; `NO_SHOW` xóa `attended_at`. Cả request là 1 transaction (một dòng lỗi → không lưu dòng nào).

Response: `{ "updated": 2 }`.

### GET `/admin/class-bookings?member_id=&from=&to=` – lịch sử đặt lớp của hội viên (perm `class.view` hoặc `booking.manage`)
- `member_id` bắt buộc; `from`/`to` (`YYYY-MM-DD`, theo giờ bắt đầu lớp) mặc định từ 30 ngày trước đến 30 ngày tới; tối đa 1 năm.
- Chỉ trả lượt đặt ở các chi nhánh trong phạm vi người gọi.

Response: mảng `ClassBooking`, sắp theo `start_at`. Dùng cho tab "Lớp đã đặt" trên hồ sơ hội viên.

---

## 4. Tác vụ nền (cron) – không có endpoint

`BookingService.MarkNoShows(ctx, now)`:
- Lượt `BOOKED` của buổi (không bị hủy) đã kết thúc hơn 30 phút → `NO_SHOW`.
- Buổi `SCHEDULED` đã kết thúc → `COMPLETED`.
- Hội viên có ≥ `booking.no_show_limit` (mặc định 3) lượt `NO_SHOW` trong 30 ngày → `booking_blocked_until = now + booking.block_days` (mặc định 7) ngày.

## 5. Cấu hình liên quan (bảng `settings`, sửa qua module cấu hình)

| Khóa | Mặc định | Ý nghĩa |
|---|---|---|
| `booking.advance_days` | 7 | Đặt trước tối đa (ngày) |
| `booking.cancel_before_minutes` | 120 | Hội viên phải tự hủy trước giờ học (phút) |
| `booking.no_show_limit` | 3 | Số NO_SHOW trong 30 ngày thì bị khóa |
| `booking.block_days` | 7 | Số ngày bị khóa đặt lớp |

## 6. Thông báo in-app

| `type` | Khi nào | Người nhận | `link` |
|---|---|---|---|
| `BOOKING_PROMOTED` | Được đẩy từ waitlist lên BOOKED (do người khác hủy hoặc tăng sức chứa) | Hội viên có tài khoản cổng | `/member/bookings` |
| `CLASS_CANCELLED` | Buổi lớp bị hủy | Hội viên có lượt BOOKED/WAITLIST và có tài khoản cổng | `/member/bookings` |

## 7. Chống vượt sức chứa

Mọi thao tác thay đổi chỗ (đặt, hủy, đẩy waitlist, sửa sức chứa) chạy trong transaction và khóa dòng `class_schedules` bằng `SELECT … FOR UPDATE`; CHECK `booked_count <= capacity` và unique index `uq_class_bookings_active` là chốt chặn cuối. Integration test `internal/service/booking_concurrency_itest_test.go` chạy 50 goroutine đặt đồng thời vào lớp capacity 10 → đúng 10 BOOKED + 40 WAITLIST (`make itest` với `GYMMASTER_TEST_DSN`).

## 8. Dành cho module cổng hội viên (backend)

Dùng lại `module.NewClassBookingService(d)` → `service.BookingService`:
```go
Book(ctx, scheduleID, memberID uint, source string) (*entity.ClassBooking, error) // source = entity.BookingSourcePortal
Cancel(ctx, bookingID uint, byMember bool) error   // byMember=true: áp hạn hủy; người gọi tự kiểm tra lượt đặt thuộc hội viên
ListMemberBookings(ctx, memberID uint, from, to time.Time) ([]entity.ClassBookingDetail, error)
GetBooking(ctx, id uint) (*entity.ClassBookingDetail, error)
MarkAttendance(ctx, scheduleID uint, items []service.BookingAttendance) (int, error)
MarkNoShows(ctx, now time.Time) (int, error)
```
`service.ClassBookingToRes(*entity.ClassBookingDetail)` chuyển sang DTO `ClassBooking` ở trên.
