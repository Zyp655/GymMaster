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
  "password": "Password@123",
  "branch_id": 1
}
```

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
  Sinh mã OTP 6 chữ số (lưu SHA-256 trong `member_register_otps`, TTL 10 phút), gửi email xác thực và trả:
  ```json
  {
    "requires_otp": true,
    "message": "Số điện thoại đã có hồ sơ tại phòng tập. Vui lòng nhập mã OTP vừa gửi tới email để liên kết tài khoản."
  }
  ```

### POST `/v1/auth/verify-member-otp` (public)
Xác thực mã OTP 6 số để tạo tài khoản `users` và liên kết với hồ sơ `members` có sẵn.

Request:
```json
{
  "phone": "0912345678",
  "email": "an.nguyen@example.com",
  "otp": "482910"
}
```

Response `data`: `TokenRes` (giống `/v1/auth/login`).
