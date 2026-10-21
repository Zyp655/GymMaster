# API contract – Sprint 12 & 17 (Quên mật khẩu, Rate Limit & Tự đăng ký hội viên)

Base URL (dev): `http://localhost:8000/v1`.

## 1. Mã lỗi bổ sung

| code | HTTP | Ý nghĩa |
|---|---|---|
| 40104 | 401 | `RESET_TOKEN_INVALID` – Token đặt lại mật khẩu hoặc OTP không hợp lệ / đã hết hạn / đã sử dụng |
| 40105 | 401 | `TOO_MANY_ATTEMPTS` – Đăng nhập sai quá 5 lần trong 15 phút (theo email + IP) |
| 42900 | 429 | `TOO_MANY_REQUESTS` – Vượt giới hạn tần suất gửi yêu cầu công khai |

---

## 2. Quên mật khẩu & Đặt lại mật khẩu

### POST `/v1/auth/forgot-password` (public)
Luôn trả HTTP 200 (không tiết lộ email có tồn tại trong hệ thống hay không). Nếu email khớp với tài khoản `ACTIVE`, hệ thống sinh token ngẫu nhiên 32 byte (lưu SHA-256 trong bảng `password_resets`, TTL 30 phút) và gửi email chứa liên kết `<WEB_URL hoặc ADMIN_URL>/reset-password?token=...`.

Request:
```json
{
  "email": "hoivien@gymmaster.vn",
  "portal": "WEB"
}
```
- `portal`: `"WEB"` (mặc định, dẫn về cổng hội viên) hoặc `"ADMIN"` (dẫn về trang quản trị).

Response `data`:
```json
{
  "message": "Nếu email tồn tại trong hệ thống, hướng dẫn đặt lại mật khẩu đã được gửi"
}
```

### POST `/v1/auth/reset-password` (public)
Đặt lại mật khẩu mới bằng token một lần. Trong cùng một transaction:
1. Khóa dòng `password_resets` (`FOR UPDATE`), kiểm tra chưa dùng (`used_at IS NULL`) và còn hạn (`expires_at > now()`).
2. Cập nhật `users.password_hash` (bcrypt).
3. Đánh dấu `password_resets.used_at = now()`.
4. Thu hồi toàn bộ `refresh_tokens` đang hoạt động của người dùng.

Request:
```json
{
  "token": "f29a8c...",
  "new_password": "NewPassword@123"
}
```

Response `data`:
```json
{
  "message": "Đặt lại mật khẩu thành công, vui lòng đăng nhập lại"
}
```

---

## 3. Chống Brute-Force khi đăng nhập (`POST /v1/auth/login`)

- Mỗi lần gọi `/v1/auth/login`, hệ thống kiểm tra bảng `login_attempts` qua interface `LoginLimiter`.
- Nếu trong **15 phút gần nhất** (tính từ sau lần đăng nhập thành công cuối cùng) cặp `(lower(email), ip)` có **≥ 5 lần sai**, trả ngay lỗi `40105 TOO_MANY_ATTEMPTS`.
- Job định kỳ `cleanup_tokens` (03:00 hằng ngày) tự động dọn các bản ghi `password_resets` hết hạn và `login_attempts` cũ.

---

## 4. Hội viên tự tạo tài khoản & Liên kết OTP (Sprint 17)

### POST `/v1/auth/register-member` (public)
Request:
```json
{
  "full_name": "Nguyễn Văn An",
  "phone": "0912345678",
  "email": "an.nguyen@example.com",
  "password": "Password123",
  "branch_id": 1
}
```

- `email`, `password`: **bắt buộc khi tạo hội viên mới** (trường hợp 1); **bị bỏ qua** khi SĐT trùng hồ sơ có sẵn (trường hợp 2).
- `branch_id`: không gửi = chi nhánh mặc định; nếu gửi phải `> 0` (`0`/âm → `40000`), chi nhánh phải tồn tại & đang hoạt động.
- Chính sách mật khẩu mới (mọi nơi đặt mật khẩu mới: đăng ký, verify OTP, đổi/đặt lại mật khẩu, tạo/đặt lại mật khẩu nhân viên):
  ≥ 8 ký tự, có ít nhất 1 chữ cái và 1 chữ số, tối đa 72 byte → sai trả `40000` "Mật khẩu phải có ít nhất 8 ký tự, gồm cả chữ và số".
  Mật khẩu cũ đang dùng không bị kiểm tra lại.

- **Trường hợp 1 – Số điện thoại chưa có trong `members`**:
  Tạo mới `users` (role `MEMBER`) + hồ sơ `members` và trả luôn cặp token đăng nhập:
  ```json
  {
    "requires_otp": false,
    "message": "Đăng ký tài khoản hội viên thành công",
    "token": {
      "access_token": "eyJ...",
      "refresh_token": "...",
      "expires_in": 900,
      "user": { "id": 250, "email": "an.nguyen@example.com", "roles": ["MEMBER"], "permissions": [], "branch_ids": [1] }
    }
  }
  ```
- **Trường hợp 2 – Số điện thoại đã có hồ sơ `members` (lập tại quầy, `user_id IS NULL`)**:
  Sinh mã OTP 6 chữ số (lưu SHA-256 trong `member_register_otps`, TTL 10 phút) và gửi tới **email lưu trong hồ sơ**
  (không phải email trong request). Tài khoản tạo ra dùng email hồ sơ. **Mật khẩu không được lưu ở bước này.**
  ```json
  {
    "requires_otp": true,
    "masked_email": "an*******@example.com",
    "message": "Số điện thoại đã có hồ sơ tại phòng tập. Mã OTP đã được gửi tới email đăng ký trong hồ sơ (an*******@example.com); ..."
  }
  ```
  FE hiển thị `masked_email` (không hiển thị email người dùng gõ).
  Lỗi: hồ sơ đã có tài khoản / email hồ sơ đã dùng → `40900`; hồ sơ chưa có email → `40000`;
  gửi lại trong 60 giây hoặc quá 5 OTP/giờ → `42900` (message có thời gian chờ).

### POST `/v1/auth/verify-member-otp` (public)
Xác thực mã OTP 6 số, **đặt mật khẩu** và tạo tài khoản `users` liên kết với hồ sơ `members` có sẵn.
Mật khẩu gửi ở bước này (không lấy từ lần gọi register-member gần nhất) để người thứ ba biết SĐT không thể
yêu cầu lại OTP rồi đặt mật khẩu thay chủ hồ sơ.

Request:
```json
{
  "phone": "0912345678",
  "otp": "482910",
  "password": "MatKhau123"
}
```
- `email`: không còn dùng (vẫn chấp nhận để tương thích, bị bỏ qua).
- `password`: **bắt buộc** (chính sách mật khẩu ở trên; thiếu/yếu → `40000`, không tính vào số lần nhập sai OTP).

Response `data`: `TokenRes` (giống `/v1/auth/login`) – FE dùng ngay để đăng nhập (hoặc đăng nhập bằng `user.email` + mật khẩu vừa đặt).

Lỗi: OTP sai/hết hạn → HTTP 401 `40104` "Mã OTP không đúng hoặc đã hết hạn"; sai 5 lần → `40105` (OTP bị vô hiệu,
cần gửi lại mã); hồ sơ đã có tài khoản → `40900`.


---

## QA vòng 2 – thay đổi hợp đồng (2026-10)

### Access token có claim `sid`
`sid` = id phiên (refresh token) cấp cùng access token. Token cũ (không có `sid`) vẫn hợp lệ.

### Phiên đăng nhập (`/v1/admin|member/security/sessions`)
- `GET .../security/sessions`: mỗi phần tử có thêm `is_current: boolean` (phiên của access token đang gọi).
- `POST .../security/sessions/revoke-others` body `{}`: thu hồi mọi phiên **khác** phiên hiện tại (lấy từ `sid`).
  Token cũ không có `sid` → gửi `current_session_id` hoặc `refresh_token` của phiên hiện tại; không xác định được → `40000`
  (không còn thu hồi luôn phiên đang dùng). Response `{ "revoked_count": n }`.

### 2FA
- Mã TOTP chỉ dùng được **một lần**: mã thuộc bước thời gian (30 giây) đã dùng hoặc cũ hơn bị từ chối (`40108`), kể cả mã vừa dùng để bật 2FA.
- Mã khôi phục bị tiêu hủy nguyên tử (đăng nhập song song chỉ một lần thành công). Chấp nhận chữ thường và thiếu dấu gạch (`abcd1234` = `ABCD-1234`).

### Yêu cầu dữ liệu cá nhân (Nghị định 13/2023)
- `POST /v1/member/data-requests`: đã có yêu cầu cùng loại đang `PENDING` → `40900`.
- Yêu cầu `EXPORT` (tự hoàn tất) lưu bản trích xuất; `file_url` = `/v1/member/data-requests/{id}/download`.
- `GET /v1/member/data-requests/{id}/download` (role MEMBER, cần access token): trả file JSON
  (`Content-Disposition: attachment; filename="du-lieu-ca-nhan-<mã HV>-<id>.json"`). Không phải yêu cầu EXPORT đã xong của chính mình → `40400`.
- Admin: `GET /v1/admin/data-requests` cần `privacy.manage` **hoặc** `data_request.process`; `POST /v1/admin/data-requests/{id}/process` cần `data_request.process` (OWNER/MANAGER được cấp ở migration 000036).

### Audit đăng nhập
`LOGIN`, `LOGIN_FAILED` (khi email tồn tại), `RESET_PASSWORD_REQUEST`, `RESET_PASSWORD` ghi `actor_id` = tài khoản liên quan (trước đây NULL).

### Khác
- Header `Authorization`: scheme không phân biệt hoa thường (`bearer xxx` hợp lệ).
- CORS: `CORS_ORIGINS` so khớp **chính xác** origin (scheme + host + cổng), không còn chấp nhận subdomain/cổng khác.
- `APP_ENV=production|prod`: tắt `/swagger` và `/api.json`.
- SSE `/v1/*/notifications/stream`: kiểm tra lại trạng thái tài khoản mỗi 45 giây (bị khóa/xóa → gửi `event: auth_revoked` rồi đóng)
  và đóng khi access token hết hạn (`event: auth_expired`) – client làm mới token rồi mở lại luồng.
- Quản lý nhân viên (`/v1/admin/staff/{id}`: sửa, gán vai trò, khóa, mở khóa, đặt lại mật khẩu, xóa): người không phải OWNER bị `40300` khi
  nhân viên đích là OWNER, đồng cấp (giữ một vai trò mà người thao tác cũng giữ), có quyền ngoài tập quyền của người thao tác,
  hoặc (trừ gán vai trò) có vai trò ngoài chi nhánh người thao tác quản lý. Tự thao tác trên chính mình (đặt lại mật khẩu, khóa, mở khóa, xóa,
  gán vai trò; sửa thông tin với người không phải OWNER) → `40000`, dùng trang hồ sơ/đổi mật khẩu.
- Job: `POST /v1/admin/jobs/{name}/run` và `POST /v1/admin/jobs/test-email` cần quyền mới `job.manage` (`job.view` chỉ còn xem).
