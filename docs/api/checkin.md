# API – Check-in / Ra vào (migration 000006)

Base URL (dev): `http://localhost:8000/v1`. Mọi endpoint cần `Authorization: Bearer <access_token>`.
Envelope, mã lỗi, phân trang: xem `sprint-01-auth-rbac-branch.md`. Tóm tắt:

- Thành công: `{ "code": 0, "message": "OK", "data": ... }`.
- Lỗi: `code` 5 chữ số, HTTP = code/100. `message` là câu tiếng Việt, hiển thị thẳng cho người dùng được.
- Lỗi validate tag của GoFrame: `code = 51`, HTTP 400.
- Ngày: `"YYYY-MM-DD"`. Thời điểm: RFC3339 (`2026-10-14T09:12:53+07:00`).

> **Quan trọng cho màn hình quầy.** Kết quả quét `DENIED` **không phải lỗi**: API trả HTTP 200, `code = 0`,
> và `data.result = "DENIED"` + `deny_reason` + `deny_message`. Chỉ lỗi hệ thống/validate/quyền mới trả mã lỗi.

| code | Khi nào (module này) |
|---|---|
| 51 | Thiếu `token`/`branch_id`/`member_id`, ngày sai định dạng, `result`/`method` không hợp lệ |
| 40000 | Chi nhánh đã ngừng hoạt động; checkout lượt bị từ chối; `to` < `from` |
| 40300 | Thiếu quyền, **hoặc** chi nhánh nằm ngoài phạm vi chi nhánh của nhân viên |
| 40400 | Không tìm thấy hội viên (thủ công, lấy QR) / chi nhánh / lượt check-in |
| 40900 | Checkout lượt đã checkout |

**Quyền**

| Quyền | Vai trò mặc định | Dùng cho |
|---|---|---|
| `checkin.scan` | OWNER, MANAGER, RECEPTIONIST | Lấy QR hội viên, quét, thủ công, checkout, xem đang tập & thống kê |
| `checkin.view` | OWNER, MANAGER, RECEPTIONIST, TRAINER | Lịch sử, đang tập, thống kê |

**Phạm vi chi nhánh.** OWNER thấy toàn hệ thống. Nhân viên gắn chi nhánh chỉ quét/checkout/xem tại chi nhánh của mình.
Hội viên **không** bị giới hạn theo chi nhánh chính: hội viên gói `ALL_BRANCHES` được vào ở mọi chi nhánh.

**Enum**

| Trường | Giá trị | Nhãn gợi ý |
|---|---|---|
| `result` | `ALLOWED` · `DENIED` | Cho vào · Từ chối |
| `method` | `QR` · `MANUAL` | Quét QR · Thủ công |
| `subscription.type` | `TIME` · `SESSION` | Theo thời gian · Theo buổi |
| `subscription.status` | `ACTIVE` · `FROZEN` | Đang hiệu lực · Bảo lưu |

### Bảng `deny_reason` → thông điệp

`deny_message` trong kết quả quét có thể **chi tiết hơn** cột "Thông điệp mặc định" (in đậm phần động). Lịch sử
(`GET /admin/checkins`) và thống kê luôn trả thông điệp mặc định. FE nên hiển thị `deny_message` từ API, không tự map.

| deny_reason | Thông điệp mặc định | Thông điệp trong kết quả quét | Ghi DB? |
|---|---|---|---|
| `QR_INVALID` | Mã QR không hợp lệ | (như mặc định) | Có nếu đọc được mã hội viên từ QR; **không** nếu QR hỏng hoàn toàn / hội viên không tồn tại |
| `QR_EXPIRED` | Mã QR đã hết hạn, vui lòng mở lại mã mới trên ứng dụng | (như mặc định) | Có |
| `BRANCH_CLOSED` | Ngoài giờ mở cửa của chi nhánh | Ngoài giờ mở cửa của chi nhánh (**05:00 - 22:00**) | Có |
| `MEMBER_INACTIVE` | Hội viên đang ngừng hoạt động hoặc bị cấm ra vào | Hội viên đã ngừng hoạt động / Hội viên đang bị cấm ra vào | Có |
| `DUPLICATE` | Hội viên vừa check-in, không cần quét lại | Hội viên đã check-in lúc **08:15** tại **GymMaster Cầu Giấy** (trong vòng **60** phút), không cần quét lại | Có |
| `SUBSCRIPTION_FROZEN` | Gói tập đang bảo lưu | Gói tập "**Gói 3 tháng**" đang bảo lưu, chưa thể vào tập | Có |
| `NO_SESSIONS_LEFT` | Gói tập theo buổi đã dùng hết số buổi | Gói "**Gói 12 buổi**" đã dùng hết số buổi, vui lòng gia hạn | Có |
| `WRONG_BRANCH` | Gói tập không áp dụng tại chi nhánh này | Gói tập chỉ dùng tại chi nhánh **GymMaster Hà Đông** | Có |
| `NO_ACTIVE_SUBSCRIPTION` | Hội viên không có gói tập còn hiệu lực | Hội viên không có gói tập còn hiệu lực, vui lòng mua hoặc gia hạn gói · *hoặc* Gói tập "**X**" bắt đầu từ ngày **15/10/2026** (gói nối tiếp chưa tới ngày) | Có |

### Thứ tự kiểm tra (một transaction)

1. Chi nhánh trong phạm vi nhân viên (403), tồn tại (404), đang hoạt động (400) – **lỗi**, không ghi log.
2. QR: đọc mã hội viên → khóa dòng hội viên (`FOR NO KEY UPDATE`, tuần tự hóa các lần quét đồng thời cùng hội viên) →
   kiểm tra chữ ký/hạn → `QR_INVALID` / `QR_EXPIRED`.
3. Giờ hiện tại (VN) ngoài `[open_time, close_time)` của chi nhánh → `BRANCH_CLOSED`.
4. Hội viên khác `ACTIVE` → `MEMBER_INACTIVE`.
5. Khóa các gói `ACTIVE`/`FROZEN` của hội viên (`FOR UPDATE`) và chọn gói hợp lệ:
   `ACTIVE`, `start_date ≤ hôm nay ≤ end_date` (giờ VN), đúng phạm vi (`SINGLE_BRANCH`: gói mua tại chi nhánh này; `ALL_BRANCHES`: mọi nơi),
   gói `SESSION` còn buổi. **Ưu tiên** gói `TIME` → gói `SESSION` → hết hạn sớm nhất → id nhỏ nhất.
6. Chống quẹt trùng: đã có lượt `ALLOWED` (ở bất kỳ chi nhánh nào, kể cả đã checkout) trong `checkin.duplicate_window_minutes` phút → `DUPLICATE`.
7. Không có gói hợp lệ: có gói `FROZEN` → `SUBSCRIPTION_FROZEN`; có gói đúng hạn, đúng chi nhánh nhưng hết buổi → `NO_SESSIONS_LEFT`;
   có gói đúng hạn nhưng khác chi nhánh → `WRONG_BRANCH`; còn lại → `NO_ACTIVE_SUBSCRIPTION`.
8. Ghi `checkins` (ALLOWED hoặc DENIED + `deny_reason`). ALLOWED với gói `SESSION`: `remaining_sessions - 1`
   và ghi `subscription_events` `SESSION_USED` (`data = {checkin_id, remaining, branch_id}`).

Lượt `DENIED` không gắn `subscription_id` và không trừ buổi.

---

## 1. Mã QR động

Định dạng `GM1.<memberID>.<expUnix>.<sig>` (ký HMAC bằng `members.qr_secret`), sống `checkin.qr_ttl_seconds` giây (mặc định 60).
FE hiển thị bằng `qrcode.react` và **tự gọi lại API trước khi `expires_at`** (ví dụ khi còn 5 giây) để làm mới mã.
Đặt lại bí mật (vô hiệu mã cũ): `POST /admin/members/{id}/reset-qr` (module hội viên).

### GET `/admin/members/{id}/qr` — `checkin.scan`
Cho lễ tân hiển thị/in mã khi hội viên quên điện thoại. (Cổng hội viên sẽ có `GET /member/qr` dùng cùng service.)

```json
{
  "member_id": 30,
  "token": "GM1.30.1791944033.3g_X0FAsybnwM8nDVC5UiK",
  "expires_at": "2026-10-14T09:13:53+07:00",
  "ttl_seconds": 60
}
```

---

## 2. Quét QR / check-in thủ công

### POST `/admin/checkins/scan` — `checkin.scan`
```json
{ "token": "GM1.30.1791944033.3g_X0FAsybnwM8nDVC5UiK", "branch_id": 1 }
```

### POST `/admin/checkins/manual` — `checkin.scan`
Dùng khi hội viên không có QR (chọn hội viên bằng `GET /admin/members/options?keyword=` – đã cho phép quyền `checkin.scan`).
Đi qua đúng các bước kiểm tra như quét QR (trừ bước QR); `method = MANUAL`.
```json
{ "member_id": 31, "branch_id": 1, "note": "Quên điện thoại" }
```

### Response chung `ScanResult`

**ALLOWED** (gói theo buổi, có cảnh báo):
```json
{
  "result": "ALLOWED",
  "deny_reason": "",
  "deny_message": "",
  "member": {
    "id": 30, "code": "GM000030", "full_name": "Nguyễn Văn An",
    "avatar_url": "", "phone": "0911000001", "status": "ACTIVE"
  },
  "subscription": {
    "id": 26, "plan_name": "Gói 12 buổi", "type": "SESSION", "status": "ACTIVE",
    "end_date": "2026-10-16", "days_left": 2, "remaining_sessions": 1
  },
  "checkin": { "id": 51, "checked_in_at": "2026-10-14T09:12:53.530064+07:00", "branch_id": 1, "branch_name": "GymMaster Cầu Giấy" },
  "previous_checkin": null,
  "warnings": ["Gói sắp hết hạn trong 2 ngày", "Gói chỉ còn 1 buổi"]
}
```
- `subscription.remaining_sessions`: số buổi còn lại **sau** lần vào này; `null` với gói `TIME`.
- `subscription.days_left`: số ngày tới `end_date` (0 = hết hạn hôm nay).
- `warnings` (chỉ khi ALLOWED, có thể rỗng): `"Gói hết hạn hôm nay"`, `"Gói sắp hết hạn trong N ngày"` (N ≤ 7),
  `"Đây là buổi cuối cùng của gói"`, `"Gói chỉ còn N buổi"` (N ≤ 3). Gợi ý FE: hiển thị dạng tag vàng dưới kết quả xanh.

**DENIED – DUPLICATE** (vẫn trả gói để lễ tân xem):
```json
{
  "result": "DENIED",
  "deny_reason": "DUPLICATE",
  "deny_message": "Hội viên đã check-in lúc 09:12 tại GymMaster Cầu Giấy (trong vòng 60 phút), không cần quét lại",
  "member": { "id": 30, "code": "GM000030", "full_name": "Nguyễn Văn An", "avatar_url": "", "phone": "0911000001", "status": "ACTIVE" },
  "subscription": { "id": 26, "plan_name": "Gói 12 buổi", "type": "SESSION", "status": "ACTIVE", "end_date": "2026-10-16", "days_left": 2, "remaining_sessions": 1 },
  "checkin": { "id": 52, "checked_in_at": "2026-10-14T09:12:53.549548+07:00", "branch_id": 1, "branch_name": "GymMaster Cầu Giấy" },
  "previous_checkin": { "id": 51, "checked_in_at": "2026-10-14T09:12:53.530064+07:00", "branch_id": 1, "branch_name": "GymMaster Cầu Giấy" },
  "warnings": []
}
```

**DENIED – WRONG_BRANCH**: `subscription` là gói liên quan (gói ở chi nhánh khác).
```json
{
  "result": "DENIED", "deny_reason": "WRONG_BRANCH",
  "deny_message": "Gói tập chỉ dùng tại chi nhánh GymMaster Hà Đông",
  "member": { "id": 31, "code": "GM000031", "full_name": "Trần Thị B", "avatar_url": "", "phone": "0911000002", "status": "ACTIVE" },
  "subscription": { "id": 27, "plan_name": "Gói 1 tháng", "type": "TIME", "status": "ACTIVE", "end_date": "2026-11-03", "days_left": 20, "remaining_sessions": null },
  "checkin": { "id": 53, "checked_in_at": "2026-10-14T09:12:53+07:00", "branch_id": 1, "branch_name": "GymMaster Cầu Giấy" },
  "previous_checkin": null, "warnings": []
}
```

**DENIED – QR hỏng hoàn toàn** (không xác định được hội viên, không ghi DB):
```json
{ "result": "DENIED", "deny_reason": "QR_INVALID", "deny_message": "Mã QR không hợp lệ",
  "member": null, "subscription": null, "checkin": null, "previous_checkin": null, "warnings": [] }
```

Quy tắc trường `null`:
- `member = null` chỉ khi QR hỏng hoàn toàn. `checkin = null` cùng trường hợp đó.
- `subscription`: gói được dùng (ALLOWED / DUPLICATE) hoặc gói liên quan (`SUBSCRIPTION_FROZEN`, `NO_SESSIONS_LEFT`, `WRONG_BRANCH`,
  gói nối tiếp chưa tới ngày); `null` với `QR_*`, `BRANCH_CLOSED`, `MEMBER_INACTIVE` và khi không có gói nào.
- `previous_checkin`: chỉ có khi `DUPLICATE`.

**Gợi ý màn hình quầy:** chọn chi nhánh (lưu localStorage) → camera `html5-qrcode` quét liên tục, gửi `token` khi đọc được
(bỏ qua cùng token trong ~3 giây để tránh gửi lặp) → khung kết quả lớn **xanh** (ALLOWED: ảnh, tên, mã, gói, số buổi/ngày còn lại, warnings)
hoặc **đỏ** (DENIED: `deny_message`), tự ẩn sau ~5 giây. Có nút "Check-in thủ công" (ô tìm hội viên + ghi chú)
và panel "Đang tập" + thống kê nhỏ bên cạnh (làm mới sau mỗi lần quét).

---

## 3. Checkout & đang tập

### POST `/admin/checkins/{id}/checkout` — `checkin.scan`
Không body. Chỉ lượt `ALLOWED`, chưa checkout. Trả `CheckinItem` (mục 4) với `checked_out_at`.

| Lỗi | message |
|---|---|
| 40000 | Chỉ checkout được lượt check-in thành công |
| 40300 | Lượt check-in thuộc chi nhánh bạn không quản lý |
| 40400 | Không tìm thấy lượt check-in |
| 40900 | Hội viên đã checkout lúc 09:13 |

### GET `/admin/checkins/present?branch_id=` — `checkin.view` hoặc `checkin.scan`
Hội viên đang trong phòng tập: lượt `ALLOWED`, chưa checkout, vào **trong ngày hôm nay** (giờ VN). Mảng thường, mới nhất trước.
`branch_id` bỏ trống = mọi chi nhánh trong phạm vi.
```json
[
  {
    "checkin_id": 54, "member_id": 31, "member_code": "GM000031", "member_name": "Trần Thị B",
    "member_phone": "0911000002", "member_avatar_url": "", "branch_id": 1, "branch_name": "GymMaster Cầu Giấy",
    "plan_name": "Gói 1 tháng", "checked_in_at": "2026-10-14T09:12:53+07:00", "minutes": 42
  }
]
```

---

## 4. Lịch sử ra vào

### GET `/admin/checkins` — `checkin.view`
Query: `page`, `limit`, `keyword` (tên / mã / SĐT hội viên), `branch_id`, `member_id`, `result` (`ALLOWED|DENIED`),
`method` (`QR|MANUAL`), `from`, `to` (`YYYY-MM-DD`, giờ VN, tính trọn ngày). Sắp xếp `checked_in_at` (mặc định DESC; `order_dir=ASC` được).

```json
{
  "items": [
    {
      "id": 51,
      "member_id": 30, "member_code": "GM000030", "member_name": "Nguyễn Văn An",
      "member_phone": "0911000001", "member_avatar_url": "",
      "branch_id": 1, "branch_name": "GymMaster Cầu Giấy",
      "subscription_id": 26, "plan_name": "Gói 12 buổi", "plan_type": "SESSION",
      "checked_in_at": "2026-10-14T09:12:53.530064+07:00",
      "checked_out_at": "2026-10-14T10:30:02+07:00",
      "method": "QR", "result": "ALLOWED", "deny_reason": "", "deny_message": "",
      "staff_id": 1, "staff_name": "Quản trị viên", "note": ""
    }
  ],
  "total": 1, "total_page": 1, "current_page": 1, "limit": 10
}
```
`subscription_id`/`plan_name`/`plan_type` = `null` với lượt DENIED. `staff_id`/`staff_name` = người quét.

### GET `/admin/checkins/stats?branch_id=&date=` — `checkin.view` hoặc `checkin.scan`
`date` mặc định hôm nay. `present` = số lượt ALLOWED trong ngày chưa checkout.
```json
{
  "date": "2026-10-14",
  "total": 3, "allowed": 2, "denied": 1,
  "unique_members": 2,
  "present": 1,
  "by_hour": [ { "hour": 0, "allowed": 0, "denied": 0 }, "... đủ 24 phần tử ...", { "hour": 9, "allowed": 2, "denied": 1 } ],
  "deny_reasons": [ { "reason": "DUPLICATE", "message": "Hội viên vừa check-in, không cần quét lại", "count": 1 } ]
}
```
Gợi ý: biểu đồ cột `recharts` theo `by_hour` (chỉ hiển thị khoảng giờ mở cửa), 4 thẻ số: lượt vào / từ chối / đang tập / hội viên.

---

## 5. Service dùng chung (cho cổng hội viên – backend)

```go
// internal/service/checkin_service.go
IssueQR(ctx context.Context, memberID uint) (token string, expiresAt time.Time, err error)
ListMemberCheckins(ctx context.Context, memberID uint, page dto.PageReq) (*dto.PageRes[*v1.CheckinItem], error)
```
Khởi tạo: `service.NewCheckinService(repository.NewCheckinRepository(d.DB), d.Tx, d.Settings)`.

## 6. Cấu hình (bảng `settings`, sửa qua `PUT /admin/settings/{key}`)

| key | Mặc định | Ý nghĩa |
|---|---|---|
| `checkin.qr_ttl_seconds` | 60 | Thời gian sống của mã QR (giây) |
| `checkin.duplicate_window_minutes` | 60 | Không cho check-in lại trong khoảng (phút); 0 = tắt |
