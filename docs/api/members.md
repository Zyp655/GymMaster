# API – Hội viên, chỉ số cơ thể, tải ảnh (migration 000003)

Base URL (dev): `http://localhost:8000/v1`. Mọi endpoint cần `Authorization: Bearer <access_token>`.
Envelope, mã lỗi, phân trang: xem `sprint-01-auth-rbac-branch.md`. Tóm tắt:

- Thành công: `{ "code": 0, "message": "OK", "data": ... }`.
- Lỗi: `code` 5 chữ số, HTTP status chuẩn theo bảng trong `sprint-01-auth-rbac-branch.md`. `message` là câu tiếng Việt, hiển thị thẳng cho người dùng được.
- Lỗi validate tag của GoFrame: `code = 51`, HTTP 400.
- Ngày: `"YYYY-MM-DD"`. Thời điểm: RFC3339 (`2026-10-14T08:00:00+07:00`).

| code | Khi nào (module này) |
|---|---|
| 51 / 40000 | Dữ liệu không hợp lệ (SĐT, ngày sinh, file không phải ảnh/Excel, chi nhánh không tồn tại…) |
| 40300 | Thiếu quyền, **hoặc** hội viên/chi nhánh nằm ngoài phạm vi chi nhánh của nhân viên |
| 40400 | Không tìm thấy hội viên / lần đo / tài khoản cổng |
| 40900 | SĐT trùng, email trùng, đã có tài khoản cổng, xóa hội viên còn gói ACTIVE/FROZEN |

**Phạm vi chi nhánh.** OWNER (hoặc user có role không gắn chi nhánh) thấy toàn hệ thống.
Nhân viên gắn chi nhánh chỉ thấy và thao tác hội viên có `home_branch_id` thuộc chi nhánh của mình.
Nếu truy cập hội viên ngoài phạm vi, API trả `40300`.

**Enum**

| Trường | Giá trị | Nhãn gợi ý |
|---|---|---|
| `gender` | `MALE` · `FEMALE` · `OTHER` | Nam · Nữ · Khác |
| `source` | `WALK_IN` · `REFERRAL` · `FACEBOOK` · `WEBSITE` · `OTHER` | Vãng lai · Giới thiệu · Facebook · Website · Khác |
| `status` (hội viên) | `ACTIVE` · `INACTIVE` · `BANNED` | Đang hoạt động · Ngừng · Cấm |
| `active_subscription.status` | `ACTIVE` · `FROZEN` | Đang hiệu lực · Bảo lưu |
| `portal_account.status` | `ACTIVE` · `LOCKED` | Hoạt động · Đã khóa |

---

## 1. Tải ảnh

### POST `/admin/uploads`
Dành cho mọi nhân viên đã đăng nhập. Không cần quyền riêng. Tài khoản chỉ có role MEMBER bị từ chối (40300).

- Request: `multipart/form-data`, field `file`.
- Chỉ nhận JPG, PNG hoặc WEBP. Định dạng được kiểm tra theo **nội dung thật** của file (`http.DetectContentType`), không theo đuôi tên file.
- Dung lượng tối đa: `UPLOAD_MAX_SIZE_MB` (mặc định 5 MB).
- File được lưu tại `UPLOAD_DIR/<yyyy>/<mm>/<uuid>.<ext>` và serve tĩnh tại `/uploads/...`.

```bash
curl -X POST $API/admin/uploads -H "Authorization: Bearer $T" -F file=@avatar.png
```
Response `data`:
```json
{ "url": "http://localhost:8000/uploads/2026/10/d8b3a667-6853-412c-b859-1d338987b355.png" }
```
| Lỗi | message |
|---|---|
| 51 | Vui lòng chọn tệp ảnh |
| 40000 | Chỉ chấp nhận ảnh JPG, PNG hoặc WEBP |
| 40000 | Ảnh vượt quá dung lượng cho phép (5 MB) |
| 40300 | Bạn không có quyền tải tệp lên |

Muốn lưu ảnh vào bản ghi thì FE gửi `url` này trong trường tương ứng, ví dụ `avatar_url` của hội viên.

---

## 2. Hội viên

### Đối tượng `MemberItem` (dòng của danh sách)
```json
{
  "id": 1,
  "member_code": "GM000001",
  "full_name": "Nguyễn Văn An",
  "gender": "MALE",
  "dob": "1995-08-15",
  "phone": "0912345678",
  "email": "an@example.com",
  "avatar_url": "",
  "home_branch_id": 1,
  "home_branch_name": "GymMaster Cầu Giấy",
  "source": "WALK_IN",
  "status": "ACTIVE",
  "has_portal_account": false,
  "active_subscription": {
    "id": 1,
    "plan_id": 6,
    "plan_name": "Gói 1 tháng",
    "status": "ACTIVE",
    "start_date": "2026-09-19",
    "end_date": "2026-10-19",
    "remaining_sessions": null,
    "days_left": 5
  },
  "created_at": "2026-10-14T08:58:04.827946+07:00"
}
```
- `dob` có thể `null`.
- `active_subscription`:
  - Là gói `ACTIVE`/`FROZEN` có `end_date` xa nhất của hội viên.
  - `null` nếu hội viên không có gói đang hiệu lực.
  - `days_left` = `end_date` − hôm nay (giờ VN). FE hiện badge cảnh báo khi `days_left <= 7`.
  - `remaining_sessions` chỉ có giá trị với gói theo buổi, còn lại là `null`.

### Đối tượng `Member` (hồ sơ chi tiết)
```json
{
  "id": 1,
  "member_code": "GM000001",
  "full_name": "Nguyễn Văn An",
  "gender": "MALE",
  "dob": "1995-08-15",
  "phone": "0912345678",
  "email": "an@example.com",
  "id_card": "012345678901",
  "address": "",
  "avatar_url": "",
  "emergency_contact_name": "",
  "emergency_contact_phone": "0988111222",
  "home_branch_id": 1,
  "home_branch": { "id": 1, "code": "HN01", "name": "GymMaster Cầu Giấy" },
  "source": "WALK_IN",
  "status": "ACTIVE",
  "note": "VIP",
  "booking_blocked_until": null,
  "portal_account": { "user_id": 5, "email": "an.member@example.com", "status": "ACTIVE", "last_login_at": null },
  "latest_metric": { "...": "đối tượng BodyMetric (mục 3)" },
  "active_subscription": { "...": "như MemberItem" },
  "stats": { "total_checkins": 12, "last_checkin_at": "2026-10-14T08:58:24+07:00" },
  "created_at": "...",
  "updated_at": "..."
}
```
- `portal_account`, `latest_metric`, `active_subscription` và `booking_blocked_until` có thể là `null`.
- `stats.total_checkins` chỉ đếm các lượt check-in `ALLOWED`.
- `booking_blocked_until`: hội viên bị khóa đặt lớp đến thời điểm này (phạt no-show).
- Không bao giờ trả `qr_secret`.

### GET `/admin/members` – danh sách (perm `member.view`)
| Query | Mô tả |
|---|---|
| `page`, `limit`, `order_by`, `order_dir` | `order_by` ∈ `id` (mặc định), `member_code`, `full_name`, `status`, `created_at` |
| `keyword` | Tìm theo họ tên, SĐT, mã `GM…` hoặc email. SĐT có thể gõ dạng `+84 912…` |
| `branch_id` | Chi nhánh chính. Ngoài phạm vi → 40300. Bỏ trống → mọi chi nhánh trong phạm vi |
| `status` | `ACTIVE` \| `INACTIVE` \| `BANNED` |
| `gender` | `MALE` \| `FEMALE` \| `OTHER` |
| `source` | `WALK_IN` \| `REFERRAL` \| `FACEBOOK` \| `WEBSITE` \| `OTHER` |
| `subscription_status` | `active` (có gói ACTIVE/FROZEN) · `expiring` (có gói và `end_date` ≤ hôm nay + 7) · `none` (không có gói) |

Response `data`: `{ items: MemberItem[], total, total_page, current_page, limit }`.

### GET `/admin/members/options` – tìm nhanh cho ô chọn hội viên
Quyền: một trong `member.view`, `subscription.sell`, `checkin.scan`, `booking.manage`, `pt.sell`, `pos.sell`.

| Query | Mô tả |
|---|---|
| `keyword` | Tên, SĐT hoặc mã. Kết quả khớp đúng SĐT/mã được xếp lên đầu |
| `branch_id` | Tùy chọn, áp phạm vi chi nhánh như danh sách |

Response `data` gồm tối đa 20 phần tử (không phân trang):
```json
[{ "id": 1, "member_code": "GM000001", "full_name": "Nguyễn Văn An", "phone": "0912345678", "avatar_url": "", "status": "ACTIVE" }]
```
FE nên debounce khoảng 300 ms. `keyword` rỗng → trả 20 hội viên đầu tiên theo tên.

### GET `/admin/members/{id}` – hồ sơ (perm `member.view`)
Response `data`: `Member`.

### POST `/admin/members` – thêm (perm `member.create`)
### PUT `/admin/members/{id}` – sửa (perm `member.update`)
Body (cùng cho tạo và sửa; PUT gửi **đầy đủ** trường):
```json
{
  "full_name": "Nguyễn Văn An",
  "phone": "0912 345 678",
  "gender": "MALE",
  "dob": "1995-08-15",
  "email": "an@example.com",
  "id_card": "012345678901",
  "address": "Cầu Giấy, Hà Nội",
  "avatar_url": "http://localhost:8000/uploads/2026/10/....png",
  "emergency_contact_name": "Nguyễn Thị B",
  "emergency_contact_phone": "0988111222",
  "home_branch_id": 1,
  "source": "WALK_IN",
  "status": "ACTIVE",
  "note": ""
}
```
| Trường | Bắt buộc | Quy tắc |
|---|---|---|
| `full_name` | ✓ | ≤ 150 ký tự |
| `phone` | ✓ | SĐT Việt Nam: di động `0[35789]xxxxxxxx` hoặc máy bàn `02xxxxxxxxx`. Chấp nhận khoảng trắng, `.`, `-`, tiền tố `+84`/`84`. Server lưu dạng chuẩn `0xxxxxxxxx`. **Duy nhất** trong các hội viên chưa xóa (409) |
| `gender` | | mặc định `OTHER` |
| `dob` | | `YYYY-MM-DD`, không ở tương lai (theo giờ VN), năm ≥ 1900 |
| `email` | | email hợp lệ, server chuyển về chữ thường |
| `id_card` | | 9–12 chữ số |
| `avatar_url` | | URL lấy từ `/admin/uploads` |
| `emergency_contact_phone` | | nếu có thì phải là SĐT VN hợp lệ |
| `home_branch_id` | ✓ | chi nhánh phải tồn tại (400) và nằm trong phạm vi người dùng (403). Khi sửa, nếu đổi chi nhánh thì chi nhánh mới cũng phải trong phạm vi |
| `source` | | mặc định `WALK_IN` |
| `status` | | mặc định `ACTIVE` |
| `note` | | ≤ 2000 ký tự |

- `member_code` (`GM000001`…) do DB tự sinh, không gửi lên.
- Response `data`: `Member`.
- Có ghi audit `CREATE` / `UPDATE`.

| Lỗi | message |
|---|---|
| 40000 | Số điện thoại không hợp lệ (SĐT Việt Nam 10 số) |
| 40000 | Ngày sinh không được ở tương lai |
| 40000 | Chi nhánh không tồn tại |
| 40300 | Bạn không được thao tác trên chi nhánh này |
| 40900 | Số điện thoại đã được dùng cho hội viên khác |

### DELETE `/admin/members/{id}` – xóa mềm (perm `member.delete`)
- Hội viên còn gói `ACTIVE`/`FROZEN` → 409 "Không thể xóa hội viên đang có gói tập còn hiệu lực hoặc đang bảo lưu".
- Nếu hội viên có tài khoản cổng, tài khoản bị **khóa** và mọi phiên đăng nhập bị thu hồi.
- Sau khi xóa, SĐT được phép dùng lại cho hội viên mới.
- Có ghi audit `DELETE`.
- Response `data`: `null`.

### POST `/admin/members/{id}/portal-account` – cấp tài khoản cổng hội viên (perm `member.update`)
Body:
```json
{ "email": "an.member@example.com", "password": "Member@123" }
```
- `email` bắt buộc, hợp lệ, server chuyển về chữ thường. `password` dài 8–72 ký tự.
- Server tạo user (`full_name`, `phone`, `avatar_url` lấy từ hội viên), gán role `MEMBER` với `branch_id = NULL`, rồi gán `members.user_id`.
- Hội viên đã có tài khoản **đang hoạt động** → 409 "Hội viên đã có tài khoản cổng hội viên".
- Hội viên có tài khoản **đã bị khóa** → mở lại tài khoản đó với email và mật khẩu mới, dùng thay cho chức năng "mở khóa".
- Email đã thuộc tài khoản khác → 409 "Email đã được sử dụng cho tài khoản khác".
- Có ghi audit `GRANT_PORTAL`.

Response `data`:
```json
{ "user_id": 5, "email": "an.member@example.com", "status": "ACTIVE", "last_login_at": null }
```

### DELETE `/admin/members/{id}/portal-account` – khóa tài khoản cổng (perm `member.update`)
- Đặt `users.status = LOCKED` và thu hồi refresh token. Hội viên đăng nhập sẽ nhận `40301`.
- Hội viên chưa có tài khoản → 404.
- Có ghi audit `LOCK_PORTAL`.
- Response `data`: `null`.

### POST `/admin/members/{id}/reset-qr` – đặt lại QR (perm `member.update`)
- Sinh lại `qr_secret` (64 hex). Mọi mã QR cũ của hội viên lập tức hết hiệu lực (xem `pkg/qrtoken`).
- Có ghi audit `RESET_QR`.
- Response `data`: `null`.

---

## 3. Chỉ số cơ thể

### Đối tượng `BodyMetric`
```json
{
  "id": 1, "member_id": 1,
  "measured_at": "2026-09-01T08:00:00+07:00",
  "weight_kg": 72.5, "height_cm": 172,
  "body_fat_pct": 18.2, "muscle_mass_kg": null,
  "waist_cm": null, "chest_cm": null, "hip_cm": null,
  "bmi": 24.51,
  "note": "",
  "measured_by": 1, "measured_by_name": "Quản trị viên",
  "created_at": "2026-10-14T08:58:12+07:00"
}
```
`bmi` do DB tự tính (cột GENERATED, làm tròn 2 chữ số). Các chỉ số tùy chọn có thể là `null`.

### GET `/admin/members/{id}/body-metrics` (perm `member.view`)
Response `data`: `BodyMetric[]`, sắp **tăng dần** theo `measured_at`, dùng trực tiếp làm dữ liệu biểu đồ (trục X là `measured_at`). Không phân trang.

### POST `/admin/members/{id}/body-metrics` (perm `member.metric`)
```json
{
  "measured_at": "2026-09-01T08:00:00+07:00",
  "weight_kg": 72.5,
  "height_cm": 172,
  "body_fat_pct": 18.2,
  "muscle_mass_kg": 32,
  "waist_cm": 80, "chest_cm": 95, "hip_cm": 92,
  "note": "Sau 1 tháng"
}
```
| Trường | Quy tắc |
|---|---|
| `weight_kg` ✓ | 1–500 |
| `height_cm` ✓ | 50–260 |
| `body_fat_pct` | 0–100 |
| `muscle_mass_kg` | 0–300 |
| `waist_cm`, `chest_cm`, `hip_cm` | 0–999 |
| `measured_at` | RFC3339, mặc định là thời điểm hiện tại, không được ở tương lai |
| `note` | ≤ 255 ký tự |

- `measured_by` là người đang đăng nhập.
- Response `data`: `BodyMetric`, đã có `bmi`.

### DELETE `/admin/members/{id}/body-metrics/{metricId}` (perm `member.metric`)
- Lần đo không thuộc hội viên `{id}` → 404.
- Có ghi audit `DELETE` với entity `body_metric`.
- Response `data`: `null`.

---

## 4. Nhập Excel

### GET `/admin/members/import-template` (perm `member.import`)
- Trả **file nhị phân**, không bọc envelope JSON.
  - Content-Type: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - Tên file: `mau-nhap-hoi-vien.xlsx`
- FE tải bằng `fetch` có header Bearer, nhận `blob`, rồi tạo link download. Nếu lỗi quyền thì body là JSON envelope (`40300`).

Sheet "Hội viên" có các cột sau, theo đúng thứ tự (dòng 1 là tiêu đề):

| Cột | Tiêu đề | Ghi chú |
|---|---|---|
| A | Họ tên* | bắt buộc |
| B | SĐT* | bắt buộc. Ô định dạng Text; ô số bị mất số 0 đầu (912345678) vẫn được nhận |
| C | Email | |
| D | Giới tính (Nam/Nữ/Khác) | dropdown; để trống = Khác |
| E | Ngày sinh (dd/mm/yyyy) | nhận `dd/mm/yyyy`, `yyyy-mm-dd` hoặc ô kiểu ngày của Excel |
| F | CCCD | 9–12 chữ số |
| G | Địa chỉ | |
| H | Nguồn | dropdown: Vãng lai / Giới thiệu / Facebook / Website / Khác; để trống = Vãng lai |

File còn có sheet "Hướng dẫn".

### POST `/admin/members/import` (perm `member.import`)
`multipart/form-data`:

| Field | Mô tả |
|---|---|
| `file` | file `.xlsx`. Server đọc sheet **đầu tiên** và bỏ qua dòng 1 |
| `branch_id` | chi nhánh chính gán cho mọi hội viên được nhập. Phải tồn tại và trong phạm vi |

Xử lý:
1. Bỏ qua im lặng các dòng trống hoàn toàn.
2. Validate từng dòng theo cùng quy tắc với API tạo hội viên.
3. Bỏ qua (kèm lý do) các dòng có SĐT trùng một dòng phía trên trong file, hoặc trùng SĐT đã có trong hệ thống.
4. Insert mọi dòng hợp lệ trong **một transaction**. Hội viên mới có `status = ACTIVE`.
5. Tối đa 2000 dòng dữ liệu mỗi lần.
6. Ghi một bản ghi audit `IMPORT`.

Response `data` (`row` là số dòng trong Excel, tiêu đề = 1; `errors` sắp tăng dần theo `row`):
```json
{
  "created": 2,
  "skipped": 8,
  "errors": [
    { "row": 4,  "message": "Thiếu họ tên" },
    { "row": 5,  "message": "SĐT 0912345678 đã tồn tại trong hệ thống" },
    { "row": 6,  "message": "SĐT 0901234567 trùng với dòng 2" },
    { "row": 7,  "message": "Ngày sinh không hợp lệ: 31/02/2000 (dd/mm/yyyy)" },
    { "row": 8,  "message": "Ngày sinh không được ở tương lai" },
    { "row": 9,  "message": "Giới tính không hợp lệ: XYZ (Nam/Nữ/Khác)" },
    { "row": 11, "message": "Email không hợp lệ: abc@" },
    { "row": 12, "message": "Nguồn không hợp lệ: TikTok" }
  ]
}
```
| Lỗi toàn file | message |
|---|---|
| 51 | Vui lòng chọn file Excel (.xlsx) / Vui lòng chọn chi nhánh |
| 40000 | File không đúng định dạng Excel (.xlsx) |
| 40000 | File Excel không có dòng dữ liệu nào |
| 40000 | Mỗi lần chỉ nhập tối đa 2000 dòng |
| 40000 / 40300 | Chi nhánh không tồn tại / ngoài phạm vi |

---

## 5. Bảng quyền

| Endpoint | Quyền |
|---|---|
| POST `/admin/uploads` | đã đăng nhập (nhân viên) |
| GET `/admin/members`, `/{id}`, `/{id}/body-metrics` | `member.view` |
| GET `/admin/members/options` | `member.view` \| `subscription.sell` \| `checkin.scan` \| `booking.manage` \| `pt.sell` \| `pos.sell` |
| POST `/admin/members` | `member.create` |
| PUT `/admin/members/{id}`, POST/DELETE `/{id}/portal-account`, POST `/{id}/reset-qr` | `member.update` |
| DELETE `/admin/members/{id}` | `member.delete` |
| POST/DELETE `/admin/members/{id}/body-metrics[/{metricId}]` | `member.metric` |
| GET `/admin/members/import-template`, POST `/admin/members/import` | `member.import` |

Quyền mặc định theo vai trò (migration 000003):

| Vai trò | Quyền hội viên |
|---|---|
| OWNER, MANAGER | tất cả `member.*` |
| RECEPTIONIST | `member.view`, `member.create`, `member.update`, `member.metric` |
| TRAINER | `member.view`, `member.metric` |

## 6. Ghi chú cho module khác
- **Bán gói / check-in:** `active_subscription` trong danh sách và hồ sơ đọc trực tiếp từ bảng `subscriptions` + `membership_plans`, không cần sửa module hội viên.
- **Check-in:** `stats` trong hồ sơ đọc trực tiếp từ bảng `checkins`, cũng không cần sửa module hội viên.
- **Cổng hội viên:** user MEMBER được nối với hội viên qua `members.user_id`.
- **QR:** dùng `members.qr_secret` với `pkg/qrtoken`. Sau `reset-qr`, mã cũ báo `ErrSignature`.
- **Hàm dùng chung (Go):** `service.NormalizeVNPhone(raw) (string, bool)` chuẩn hóa và kiểm tra SĐT VN.
