# API contract – Sprint 01 (Auth, RBAC, Chi nhánh)

Base URL (dev): `http://localhost:8000/v1`. Swagger: `http://localhost:8000/swagger`.

## Quy ước chung

**Envelope** – mọi response (kể cả lỗi):
```json
{ "code": 0, "message": "OK", "data": { } }
```
- `code = 0` → thành công.
- Lỗi nghiệp vụ là số 5 chữ số; **HTTP status là status chuẩn suy theo bảng dưới** (`rescode.HTTPStatus`).
  FE nên dựa vào `code` trong body; HTTP status chỉ dùng để phân loại 4xx/5xx.
- Lỗi validate / body sai kiểu dữ liệu → `code = 40000`, HTTP 400; `message` là câu tiếng Việt dùng nhãn
  tiếng Việt của trường (vd "Chi nhánh là bắt buộc"), không lộ tên field Go. (Trước đây validate trả `code = 51`.)
- Lỗi DB do dữ liệu đầu vào không còn trả 500: trùng khóa → 40900 "Dữ liệu đã tồn tại (trùng …)";
  khóa ngoại không tồn tại → 40000 "Dữ liệu tham chiếu không tồn tại (…)"; xóa bản ghi đang được tham chiếu →
  40900 "Không thể xóa vì dữ liệu đang được sử dụng"; chuỗi quá dài → 40000 "Dữ liệu quá dài"; số/ID vượt giới hạn
  kiểu cột → 40000 "Giá trị số vượt giới hạn"; ký tự NUL → 40000 "Dữ liệu chứa ký tự không hợp lệ"
  (bảng đầy đủ: `gymmaster-api/internal/pkg/dberr`).

| Dải code | HTTP |
|---|---|
| 400xx | 400 |
| 401xx | 401 |
| 403xx | 403 |
| 404xx | 404 |
| 409xx | 409 |
| 429xx | 429 |
| 41xxx (gói tập) | 409 |
| 42xxx (check-in bị từ chối: chưa ký HĐ, ngoài giờ, nợ trả góp, nghỉ lễ) | 403 |
| 43xxx (lịch PT: ngoài lịch rảnh, HLV nghỉ) | 409 |
| 44xxx (thanh toán: sai chữ ký, lệch tiền, hết hạn) | 400 |
| 45xxx, 46xxx (POS/tồn kho, thiết bị/tủ đồ) | 409 |
| 47001 (kỳ lương không hợp lệ) | 400 |
| 47002, 47003 (ca thu ngân) | 409 |
| 5xxxx | 500 |

| code | HTTP | Ý nghĩa |
|---|---|---|
| 40000 | 400 | BAD_REQUEST |
| 40100 | 401 | UNAUTHORIZED (thiếu/sai token) |
| 40101 | 401 | TOKEN_EXPIRED → FE gọi refresh |
| 40102 | 401 | INVALID_CREDENTIALS (sai email/mật khẩu) |
| 40103 | 401 | REFRESH_TOKEN_INVALID → FE logout |
| 40300 | 403 | PERMISSION_DENIED |
| 40301 | 403 | ACCOUNT_LOCKED |
| 40400 | 404 | NOT_FOUND |
| 40900 | 409 | CONFLICT (trùng mã, đang được sử dụng…) |
| 50000 | 500 | INTERNAL |

**Header**
- Endpoint cần đăng nhập: `Authorization: Bearer <access_token>`.

**Danh sách có phân trang**
- Query: `page` (mặc định 1), `limit` (mặc định 10, tối đa 100), `keyword`, `order_by`, `order_dir` (`ASC`|`DESC`).
- `data` có dạng:
```json
{ "items": [], "total": 0, "total_page": 0, "current_page": 1, "limit": 10 }
```

**Định dạng dữ liệu**
- Thời gian: ISO-8601 (`2026-10-14T08:00:00+07:00`).
- Giờ mở cửa: `"HH:mm"`.
- Tiền: number (VND).

---

## Auth

### POST `/auth/login` (public)
Request:
```json
{ "email": "admin@gymmaster.vn", "password": "Admin@123" }
```

Response `data`:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "c0f1...(opaque, 64 hex)",
  "expires_in": 900,
  "user": {
    "id": 1, "email": "admin@gymmaster.vn", "full_name": "Quản trị viên",
    "phone": "0900000000", "avatar_url": "",
    "roles": ["OWNER"],
    "permissions": ["branch.view", "branch.create", "..."],
    "branch_ids": []
  }
}
```
- `branch_ids` rỗng nghĩa là được làm việc trên toàn hệ thống.

### POST `/auth/refresh` (public)
- Request: `{ "refresh_token": "..." }`
- Response: giống `/auth/login`.
- Refresh token được **xoay vòng**: token cũ bị thu hồi ngay sau khi dùng.

### POST `/auth/logout` (auth)
- Request: `{ "refresh_token": "..." }`
- Response: `data: null`.

### GET `/auth/me` (auth)
- Response: `data` = object `user` như trên.

### PUT `/auth/change-password` (auth)
- Request: `{ "old_password": "...", "new_password": "..." }`
- `new_password`: tối thiểu 8 ký tự.
- Sau khi đổi, mọi refresh token của user bị thu hồi.

---

## Chi nhánh – permission `branch.*`

Object `Branch`:
```json
{
  "id": 1, "code": "HN01", "name": "GymMaster Cầu Giấy",
  "address": "123 Xuân Thủy, Cầu Giấy, Hà Nội", "phone": "0241234567", "email": "caugiay@gymmaster.vn",
  "open_time": "05:30", "close_time": "22:00", "status": "ACTIVE",
  "created_at": "...", "updated_at": "..."
}
```

| Method | Path | Permission | Ghi chú |
|---|---|---|---|
| GET | `/admin/branches` | `branch.view` | Phân trang, thêm filter `status` |
| GET | `/admin/branches/{id}` | `branch.view` | |
| POST | `/admin/branches` | `branch.create` | Body: Branch không có `id`/timestamps. Bắt buộc `code`, `name` |
| PUT | `/admin/branches/{id}` | `branch.update` | Body: như POST |
| DELETE | `/admin/branches/{id}` | `branch.delete` | Xóa mềm. Trả 40900 nếu còn nhân viên gắn với chi nhánh |

Validate:
- `code`: 2–20 ký tự `[A-Z0-9_-]`, duy nhất.
- `status`: `ACTIVE` | `INACTIVE`.
- `close_time` phải sau `open_time`.

---

## Vai trò & quyền – permission `role.*`

Object `Role`:
```json
{ "id": 1, "code": "OWNER", "name": "Chủ trung tâm", "description": "", "is_system": true, "permissions": ["branch.view"] }
```

Object `Permission`:
```json
{ "id": 1, "code": "branch.view", "module": "branch", "name": "Xem chi nhánh" }
```

| Method | Path | Permission | Ghi chú |
|---|---|---|---|
| GET | `/admin/permissions` | `role.view` | Trả mảng (không phân trang) |
| GET | `/admin/roles` | `role.view` | Trả mảng (không phân trang) |
| GET | `/admin/roles/{id}` | `role.view` | |
| POST | `/admin/roles` | `role.create` | Body: `{code,name,description,permissions:[code]}` |
| PUT | `/admin/roles/{id}` | `role.update` | Body: `{name,description,permissions:[code]}`. Không sửa được `code` |
| DELETE | `/admin/roles/{id}` | `role.delete` | Không xóa được role `is_system`. Trả 40900 nếu còn user giữ role này |

Role hệ thống được seed sẵn:
- `OWNER`: có tất cả quyền.
- `MANAGER`
- `RECEPTIONIST`
- `TRAINER`
- `MEMBER`

Danh sách permission code hiện có:
- `branch.view`, `branch.create`, `branch.update`, `branch.delete`
- `role.view`, `role.create`, `role.update`, `role.delete`

Các module sau sẽ bổ sung permission mới qua migration.
