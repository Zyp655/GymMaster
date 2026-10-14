# API contract – Nhân sự, HLV, Phòng tập, Ca làm (migration 000002)

Base URL (dev): `http://localhost:8000/v1`. Swagger: `/swagger` (tags "Nhân sự", "Phòng tập", "Ca làm").
Envelope, mã lỗi, phân trang và header dùng chung: xem `sprint-01-auth-rbac-branch.md`.

Ký hiệu trong tài liệu này:
- **Ngày** là `"YYYY-MM-DD"`.
- **Thời điểm** là RFC3339, ví dụ `"2026-10-12T06:00:00+07:00"`.
- Lỗi validate tag trả `code = 51` (HTTP 400), `message` là câu tiếng Việt.

## Phạm vi chi nhánh (áp dụng cho mọi endpoint bên dưới)

Người dùng có `branch_ids` trong token là người dùng gắn chi nhánh: có vai trò gắn chi nhánh và không có vai trò toàn hệ thống. Họ chỉ thấy và thao tác dữ liệu thuộc các chi nhánh đó.

- **OWNER**, hoặc người có vai trò không gắn chi nhánh, được thao tác trên toàn hệ thống.
- Truyền `branch_id` không thuộc phạm vi của mình vào **danh sách** hoặc **options** → `40300`.
- Gọi chi tiết, sửa hoặc xóa một bản ghi ngoài phạm vi → `40400` (không lộ sự tồn tại của bản ghi).
- Một nhân viên "thuộc" chi nhánh X khi có ít nhất một dòng `user_roles` với `branch_id = X`.

---

## 1. Nhân viên – `/admin/staff`

### Đối tượng `Staff`
```json
{
  "id": 3,
  "email": "hlv1@gymmaster.vn",
  "phone": "0912345678",
  "full_name": "Nguyễn Văn HLV",
  "avatar_url": "",
  "status": "ACTIVE",
  "last_login_at": "2026-10-14T08:58:14+07:00",
  "created_at": "2026-10-14T08:58:24+07:00",
  "updated_at": "2026-10-14T08:58:24+07:00",
  "staff_profile": {
    "staff_code": "NV0001",
    "position": "TRAINER",
    "gender": "MALE",
    "dob": "1995-05-20",
    "id_card": "",
    "address": "",
    "hire_date": "2026-10-14",
    "base_salary": 8000000,
    "commission_rate": 15,
    "note": ""
  },
  "roles": [
    { "role_id": 4, "role_code": "TRAINER", "role_name": "Huấn luyện viên", "branch_id": 1, "branch_name": "GymMaster Cầu Giấy" },
    { "role_id": 3, "role_code": "RECEPTIONIST", "role_name": "Lễ tân", "branch_id": null, "branch_name": "" }
  ],
  "trainer_profile": {
    "level": "SENIOR",
    "specialties": ["Yoga", "Giảm cân"],
    "bio": "HLV 5 năm",
    "certifications": "",
    "years_experience": 5
  }
}
```

Giải thích các trường:
- **`status`**: `ACTIVE` hoặc `LOCKED`.
- **`last_login_at`**: có thể là `null`.
- **`staff_profile`**: là `null` với tài khoản quản trị cũ chưa có hồ sơ (ví dụ OWNER được seed sẵn).
- **`staff_profile.staff_code`**: hệ thống tự sinh (`NV0001`, `NV0002`…), không sửa được.
- **`staff_profile.position`**: `MANAGER`, `RECEPTIONIST`, `TRAINER` hoặc `OTHER`.
- **`staff_profile.gender`**: `MALE`, `FEMALE` hoặc `OTHER`.
- **`staff_profile.dob`**: có thể là `null`.
- **`roles[].branch_id`**: `null` nghĩa là vai trò toàn hệ thống.
- **`trainer_profile`**: chỉ có khi `position = TRAINER`; trường hợp khác là `null`.
- **`trainer_profile.level`**: `JUNIOR`, `SENIOR` hoặc `MASTER`.

### GET `/admin/staff` – danh sách (perm `staff.view`)
Query:
- `page`, `limit`, `keyword`: tìm theo họ tên, email, SĐT hoặc mã NV.
- `order_by`: `id`, `full_name`, `email`, `status`, `created_at`, `staff_code`, `position` hoặc `hire_date`. Mặc định `id`.
- `order_dir`.
- `branch_id`: chỉ lấy nhân viên có vai trò tại chi nhánh này.
- `position`: `MANAGER`, `RECEPTIONIST`, `TRAINER` hoặc `OTHER`.
- `status`: `ACTIVE` hoặc `LOCKED`.

Danh sách gồm mọi người dùng có hồ sơ nhân viên **hoặc** có ít nhất một vai trò khác `MEMBER`. Tài khoản hội viên thuần túy không xuất hiện.

Response `data`: `PageRes<Staff>`.

### GET `/admin/staff/{id}` (perm `staff.view`)
Trả `Staff`. Trả `40400` khi không tồn tại, đã xóa, hoặc nằm ngoài phạm vi chi nhánh.

### POST `/admin/staff` – thêm nhân viên (perm `staff.create`)
Toàn bộ thao tác chạy trong **một transaction**:
1. Tạo `users`, mật khẩu được băm bcrypt.
2. Tạo `staff_profiles`.
3. Tạo `trainer_profiles` nếu `position = TRAINER`.
4. Tạo các dòng `user_roles`.

Request:
```json
{
  "full_name": "Nguyễn Văn HLV",
  "email": "hlv1@gymmaster.vn",
  "phone": "0912345678",
  "password": "Trainer@123",
  "position": "TRAINER",
  "gender": "MALE",
  "dob": "1995-05-20",
  "id_card": "001095000001",
  "address": "Hà Nội",
  "hire_date": "2026-10-14",
  "base_salary": 8000000,
  "commission_rate": 15,
  "note": "",
  "trainer_profile": {
    "level": "SENIOR",
    "specialties": ["Yoga", "Giảm cân"],
    "bio": "…",
    "certifications": "…",
    "years_experience": 5
  },
  "roles": [
    { "role_code": "TRAINER", "branch_id": 1 },
    { "role_code": "RECEPTIONIST", "branch_id": 1 }
  ]
}
```

| Trường | Bắt buộc | Ràng buộc |
|---|---|---|
| `full_name` | ✓ | ≤ 150 ký tự |
| `email` | ✓ | Email hợp lệ, ≤ 150. Duy nhất, **không phân biệt hoa thường**. Được lưu ở dạng chữ thường. |
| `phone` | | `^[0-9+ ]{8,20}$` |
| `password` | ✓ | 8–72 ký tự (mật khẩu ban đầu do người tạo nhập) |
| `position` | ✓ | `MANAGER`, `RECEPTIONIST`, `TRAINER` hoặc `OTHER` |
| `gender` | | Mặc định `OTHER` |
| `dob` | | Ngày, phải trước hôm nay |
| `hire_date` | | Ngày, mặc định hôm nay |
| `id_card` | | ≤ 20 ký tự |
| `address` | | ≤ 255 ký tự |
| `base_salary` | | ≥ 0 |
| `commission_rate` | | Từ 0 đến 100 (%) |
| `note` | | ≤ 2000 ký tự |
| `trainer_profile` | | Chỉ dùng khi `position = TRAINER`. Bỏ trống thì `level = JUNIOR`. `specialties` được bỏ phần tử rỗng và trùng (không phân biệt hoa thường). `years_experience` từ 0 đến 60. |
| `roles` | ✓ | Ít nhất 1 dòng. Xem quy tắc bên dưới. |

**Quy tắc gán vai trò** (áp dụng cho cả `POST /staff` và `PUT /staff/{id}/roles`):

| Trường hợp | Kết quả |
|---|---|
| `role_code` là `MEMBER` | `40000` "Không thể gán vai trò hội viên cho nhân viên" |
| `role_code` không tồn tại | `40000` |
| `role_code` là `OWNER` và người gọi không phải OWNER | `40300` |
| `role_code` là `OWNER` và người gọi là OWNER | Luôn lưu với `branch_id = null` |
| Người gọi không phải OWNER, `branch_id` là `null` | `40000` "Vui lòng chọn chi nhánh cho vai trò …" |
| Người gọi không phải OWNER, `branch_id` ngoài phạm vi của mình | `40300` |
| `branch_id` không tồn tại | `40000` |
| Dòng trùng (cùng role và branch) | Tự gộp |

`role_code` không phân biệt hoa thường. `branch_id = 0` được coi như `null`.

Response `data`: `Staff`.

Lỗi:
- `40900` "Email đã được sử dụng".
- Các lỗi `40000` / `40300` theo bảng quy tắc vai trò ở trên.

### PUT `/admin/staff/{id}` – cập nhật thông tin + hồ sơ HLV (perm `staff.update`)
Body: giống POST nhưng **không có** `password` và `roles`.

Xử lý hồ sơ HLV:
- `position = TRAINER`: tạo hoặc cập nhật `trainer_profile`.
- Đổi sang chức vụ khác: hồ sơ HLV bị xóa.

Với tài khoản chưa có hồ sơ nhân viên, hệ thống tạo hồ sơ và cấp mã NV mới.

Response `data`: `Staff`.

Lỗi:
- `40900`: email trùng.
- `40300`: người gọi không phải OWNER mà sửa tài khoản OWNER.
- `40400`: không tìm thấy hoặc ngoài phạm vi chi nhánh.

### PUT `/admin/staff/{id}/roles` – gán lại vai trò (perm `staff.update`)
Request:
```json
{ "roles": [ { "role_code": "MANAGER", "branch_id": 1 } ] }
```

Hành vi:
- **OWNER:** thay toàn bộ vai trò bằng danh sách mới.
- **Người không phải OWNER:** chỉ thay các vai trò ở chi nhánh trong phạm vi của mình. Hai loại vai trò sau được **giữ nguyên**:
  - vai trò toàn hệ thống của nhân viên;
  - vai trò ở chi nhánh khác.
- **Không được tự đổi vai trò của chính mình** → `40000`.

Quyền mới có hiệu lực khi nhân viên refresh token (tối đa 15 phút).

Response `data`: `Staff`.

### POST `/admin/staff/{id}/lock` và `/admin/staff/{id}/unlock` (perm `staff.update`)
Body rỗng.

- **Lock:** đặt `status = LOCKED` và **thu hồi mọi refresh token** của nhân viên. Từ lần refresh tiếp theo, nhân viên bị đăng xuất; đăng nhập lại nhận `40301`.
- **Unlock:** đặt `status = ACTIVE`.

Lỗi:
- `40000` "Không thể tự khóa tài khoản của chính mình".
- `40300`: người gọi không phải OWNER mà khóa tài khoản OWNER.

Response `data`: `Staff`.

### POST `/admin/staff/{id}/reset-password` (perm `staff.update`)
Request:
```json
{ "new_password": "NewPass@123" }
```

`new_password` dài 8–72 ký tự. Sau khi đặt lại, mọi refresh token của nhân viên bị thu hồi.

Response `data`: `null`.

### DELETE `/admin/staff/{id}` – xóa mềm (perm `staff.delete`)
Thứ tự kiểm tra:
1. Tự xóa chính mình → `40000`.
2. Người gọi không phải OWNER mà xóa tài khoản OWNER → `40300`.
3. Nhân viên còn `class_schedules` hoặc `pt_sessions` ở trạng thái `SCHEDULED` có `start_at > now()` mà mình phụ trách → `40900` "Không thể xóa: nhân viên còn X lịch lớp và Y buổi PT sắp diễn ra…".

Khi xóa thành công, trong cùng một transaction:
- Xóa các ca làm (`shifts`) chưa bắt đầu.
- Thu hồi refresh token.
- Xóa mềm `users`.

Email của nhân viên đã xóa có thể dùng lại cho tài khoản khác.

Response `data`: `null`.

### GET `/admin/staff/options?branch_id=` (perm `staff.view` | `shift.view` | `shift.manage`)
Trả mảng (không phân trang, tối đa 500) các nhân viên thỏa mãn cả ba điều kiện:
- đang `ACTIVE`;
- có hồ sơ nhân viên;
- có vai trò tại `branch_id` (khi truyền `branch_id`).

```json
[ { "id": 4, "full_name": "Lê Quản Lý", "staff_code": "NV0002", "position": "MANAGER" } ]
```

Dùng cho dropdown chọn nhân viên khi xếp ca.

### GET `/admin/trainers/options?branch_id=` (perm `staff.view` | `class.view` | `pt.view`)
Trả mảng các HLV `ACTIVE` có `position = TRAINER`. Khi truyền `branch_id`, chỉ lấy HLV có vai trò tại chi nhánh đó.

```json
[ { "id": 3, "full_name": "Nguyễn Văn HLV", "level": "SENIOR", "specialties": ["Yoga"] } ]
```

Dùng cho dropdown ở module lớp học và PT.

**Audit log** (`audit_logs.entity = "staff"`):

| `action` | Thao tác |
|---|---|
| `CREATE` | Thêm nhân viên |
| `UPDATE` | Cập nhật thông tin |
| `ASSIGN_ROLES` | Gán lại vai trò |
| `LOCK` | Khóa tài khoản |
| `UNLOCK` | Mở khóa tài khoản |
| `RESET_PASSWORD` | Đặt lại mật khẩu |
| `DELETE` | Xóa nhân viên |

---

## 2. Phòng tập – `/admin/rooms`

### Đối tượng `Room`
```json
{
  "id": 1,
  "branch_id": 1,
  "branch_name": "GymMaster Cầu Giấy",
  "name": "Phòng Yoga 1",
  "capacity": 20,
  "description": "",
  "status": "ACTIVE",
  "created_at": "…",
  "updated_at": "…"
}
```

### GET `/admin/rooms` (perm `room.view`)
Query:
- `page`, `limit`, `keyword`: tìm theo tên hoặc mô tả.
- `order_by`: `id`, `name`, `capacity`, `status`, `branch_id` hoặc `created_at`.
- `order_dir`.
- `branch_id`.
- `status`: `ACTIVE` hoặc `INACTIVE`.

Response `data`: `PageRes<Room>`.

### GET `/admin/rooms/{id}` (perm `room.view`)
Trả `Room`. Trả `40400` khi không tồn tại hoặc nằm ngoài phạm vi chi nhánh.

### POST `/admin/rooms` và PUT `/admin/rooms/{id}` (perm `room.manage`)
Request:
```json
{ "branch_id": 1, "name": "Phòng Yoga 1", "capacity": 20, "description": "Tầng 2", "status": "ACTIVE" }
```

| Trường | Bắt buộc | Ràng buộc |
|---|---|---|
| `branch_id` | ✓ | Chi nhánh tồn tại và nằm trong phạm vi của người gọi (ngoài phạm vi → `40300`) |
| `name` | ✓ | ≤ 100 ký tự. **Duy nhất trong chi nhánh**, không phân biệt hoa thường. Trùng → `40900` "Tên phòng đã tồn tại trong chi nhánh". |
| `capacity` | ✓ | 1–1000 |
| `description` | | ≤ 255 ký tự |
| `status` | | `ACTIVE` (mặc định) hoặc `INACTIVE` |

Response `data`: `Room`.

### DELETE `/admin/rooms/{id}` – xóa mềm (perm `room.manage`)
Phòng còn `class_schedules` ở trạng thái `SCHEDULED` có `start_at > now()` → `40900` "Không thể xóa: phòng còn lịch lớp sắp diễn ra".

### GET `/admin/rooms/options?branch_id=` (perm `room.view` | `class.view`)
Trả mảng các phòng `ACTIVE`, sắp theo chi nhánh rồi tên.

```json
[ { "id": 1, "branch_id": 1, "name": "Phòng Yoga 1", "capacity": 20 } ]
```

---

## 3. Ca làm – `/admin/shifts`

### Đối tượng `Shift`
```json
{
  "id": 1,
  "branch_id": 1,
  "branch_name": "GymMaster Cầu Giấy",
  "staff_id": 3,
  "staff_name": "Nguyễn Văn HLV",
  "staff_code": "NV0001",
  "position": "TRAINER",
  "start_at": "2026-10-12T06:00:00+07:00",
  "end_at": "2026-10-12T14:00:00+07:00",
  "note": "Ca sáng"
}
```

### GET `/admin/shifts?branch_id=&staff_id=&from=&to=` (perm `shift.view`)
- `from` và `to` là **ngày** (bắt buộc), tính theo giờ Việt Nam và bao gồm cả hai đầu.
- Trả các ca **giao** với khoảng `[from 00:00, to + 1 ngày 00:00)`.
- `to` phải lớn hơn hoặc bằng `from`, khoảng tối đa 62 ngày.
- `branch_id` và `staff_id` là tùy chọn.
- Response `data` là **mảng** `Shift[]` (không phân trang), sắp theo `start_at`.

Ví dụ lấy lưới tuần: `from` = thứ Hai, `to` = Chủ nhật.

### POST `/admin/shifts` và PUT `/admin/shifts/{id}` (perm `shift.manage`)
Request:
```json
{ "branch_id": 1, "staff_id": 3, "start_at": "2026-10-12T06:00:00+07:00", "end_at": "2026-10-12T14:00:00+07:00", "note": "Ca sáng" }
```

`start_at` và `end_at` nhận RFC3339. Ngoài ra chấp nhận `"YYYY-MM-DDTHH:mm"` không kèm múi giờ, khi đó hiểu là giờ Việt Nam.

| Lỗi | Khi nào |
|---|---|
| `40000` "Giờ kết thúc phải sau giờ bắt đầu" | `end_at <= start_at` |
| `40000` "Một ca làm không được dài quá 16 giờ" | Ca dài quá 16 giờ |
| `40000` "Nhân viên không thuộc chi nhánh của ca làm hoặc đã bị khóa" | Nhân viên không có vai trò tại `branch_id`, hoặc không `ACTIVE`, hoặc đã xóa |
| `40300` | `branch_id` ngoài phạm vi của người gọi |
| `40900` "Nhân viên đã có ca trùng giờ" | Vi phạm EXCLUDE `ex_shifts_staff_overlap` (SQLSTATE 23P01). Kiểm tra trên mọi chi nhánh. Hai ca nối tiếp nhau (10:00–12:00 và 12:00–14:00) **không** bị coi là trùng. |

Response `data`: `Shift`.

### DELETE `/admin/shifts/{id}` (perm `shift.manage`)
Xóa cứng ca làm. Response `data`: `null`.

### POST `/admin/shifts/copy-week` (perm `shift.manage`)
Request:
```json
{ "branch_id": 1, "from_week_start": "2026-10-12", "to_week_start": "2026-10-19" }
```

Điều kiện:
- Cả `from_week_start` và `to_week_start` phải là **thứ Hai**, nếu không trả `40000`.
- Hai tuần phải khác nhau.
- Tuần đích có thể ở trước hoặc sau tuần nguồn.

Cách hoạt động:
- Sao chép mọi ca của chi nhánh **bắt đầu** trong tuần nguồn (giờ VN) sang tuần đích, giữ nguyên giờ và thứ trong tuần.
- Ca bị **bỏ qua** khi:
  - trùng giờ với ca đã có của nhân viên; hoặc
  - nhân viên không còn thuộc chi nhánh hoặc đã bị khóa.
- Gọi lại lần hai sẽ không nhân đôi ca, vì các ca đã chép đều bị bỏ qua do trùng giờ.

Response `data`:
```json
{ "created": 2, "skipped": 0 }
```

---

## Tóm tắt endpoint

| Method | Path | Perm |
|---|---|---|
| GET | `/admin/staff` | staff.view |
| GET | `/admin/staff/{id}` | staff.view |
| POST | `/admin/staff` | staff.create |
| PUT | `/admin/staff/{id}` | staff.update |
| PUT | `/admin/staff/{id}/roles` | staff.update |
| POST | `/admin/staff/{id}/lock` | staff.update |
| POST | `/admin/staff/{id}/unlock` | staff.update |
| POST | `/admin/staff/{id}/reset-password` | staff.update |
| DELETE | `/admin/staff/{id}` | staff.delete |
| GET | `/admin/staff/options` | staff.view, shift.view, shift.manage |
| GET | `/admin/trainers/options` | staff.view, class.view, pt.view |
| GET | `/admin/rooms` | room.view |
| GET | `/admin/rooms/{id}` | room.view |
| POST | `/admin/rooms` | room.manage |
| PUT | `/admin/rooms/{id}` | room.manage |
| DELETE | `/admin/rooms/{id}` | room.manage |
| GET | `/admin/rooms/options` | room.view, class.view |
| GET | `/admin/shifts` | shift.view |
| POST | `/admin/shifts` | shift.manage |
| PUT | `/admin/shifts/{id}` | shift.manage |
| DELETE | `/admin/shifts/{id}` | shift.manage |
| POST | `/admin/shifts/copy-week` | shift.manage |

Khi một endpoint ghi nhiều quyền, người dùng chỉ cần có **một** trong số đó.

Danh sách vai trò để hiển thị trong dropdown: `GET /admin/roles` (perm `role.view`). Danh sách chi nhánh: `GET /admin/branches?limit=100` (perm `branch.view`). Hai endpoint này thuộc sprint 01.
