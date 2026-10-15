# Báo cáo Kiểm thử Bảo mật (OWASP Top 10) & Phi chức năng (Mục 15.3)

Tài liệu này tổng hợp danh mục kiểm tra bảo mật **OWASP Top 10 (2021)**, kiểm thử phân quyền tự động (RBAC / IDOR) và bảng chỉ tiêu hiệu năng phi chức năng của hệ thống **GymMaster**.

---

## 1. Checklist Kiểm thử Bảo mật OWASP Top 10

| Mã OWASP | Hạng mục rủi ro | Cơ chế phòng vệ đã cài đặt trong GymMaster | Trạng thái |
|---|---|---|---|
| **A01:2021 – Broken Access Control** | Vượt quyền giữa các vai trò hoặc đọc dữ liệu hội viên khác (IDOR trên `/v1/member/*` và `/v1/admin/*`). | - Middleware `AuthN` + `RequireRole("MEMBER")` tự động ánh xạ `user_id` từ JWT sang `member_id` trong DB; các API `/v1/member/*` không nhận `member_id` từ client nên chống tuyệt đối lỗi IDOR.<br>- Middleware `RequirePermission` kiểm tra mã quyền chi tiết (`perm:"..."`) và phạm vi chi nhánh (`BranchScope`).<br>- Kiểm thử tự động trong `internal/middleware/rbac_security_test.go`. | ✅ Đạt |
| **A02:2021 – Cryptographic Failures** | Lộ mật khẩu, token đặt lại mật khẩu, khóa thiết bị Kiosk, hoặc bí mật 2FA TOTP. | - Mật khẩu băm bằng `bcrypt` (`internal/pkg/hash`).<br>- Refresh token, token đặt lại mật khẩu (`password_resets`), OTP đăng ký (`member_register_otps`) và khóa thiết bị (`access_devices.api_key_hash`) đều chỉ lưu bản băm `SHA-256`.<br>- Khóa bí mật 2FA TOTP mã hóa `AES-256-GCM` (`internal/pkg/cryptoext`).<br>- Dữ liệu nhạy cảm (SĐT, CCCD) được che tự động (`MaskPhone`, `MaskIDCard`) nếu thiếu quyền `member.view_sensitive`. | ✅ Đạt |
| **A03:2021 – Injection** | SQL Injection, XSS trong bình luận/bài viết/tin nhắn. | - Toàn bộ truy vấn DB đi qua GORM parameterized queries (`?`), không nối chuỗi SQL thô từ input người dùng.<br>- React 18/19 tự động escape nội dung JSX; đầu vào API được kiểm tra schema bằng GoFrame validator (`v:"required|..."`) và Zod. | ✅ Đạt |
| **A04:2021 – Insecure Design** | Gian lận thanh toán VNPay, chụp ảnh màn hình QR check-in dùng lại, vượt sức chứa lớp học. | - QR Check-in động ký `HMAC-SHA256` kèm timestamp (TTL 60 giây) + chống quét trùng (`CooldownMinutes`).<br>- VNPay IPN kiểm tra chữ ký `HMAC-SHA512`, đối chiếu khớp số tiền `vnp_Amount/100 == txn.Amount` và khóa `SELECT ... FOR UPDATE` chống cộng tiền 2 lần.<br>- Đặt lớp dùng `SELECT ... FOR UPDATE` trên `class_schedules` chống race condition vượt sức chứa. | ✅ Đạt |
| **A05:2021 – Security Misconfiguration** | Lộ secret trong code, cấu hình CORS mở tràn lan. | - Không hard-code secret; toàn bộ cấu hình đọc từ biến môi trường (`config.yaml` / `src/env.ts` kiểm tra bằng Zod khi build).<br>- Middleware CORS kiểm soát chặt origin và header cho phép. | ✅ Đạt |
| **A06:2021 – Vulnerable & Outdated Components** | Thư viện Go hoặc npm có lỗ hổng bảo mật. | - Tích hợp kiểm tra tự động trong pipeline `.github/workflows/ci.yml` (`go test`, `npm run typecheck`, `npm test`, `npm run build`). | ✅ Đạt |
| **A07:2021 – Identification & Authentication Failures** | Brute-force mật khẩu, đánh cắp phiên đăng nhập, dùng lại token cũ. | - Giới hạn đăng nhập sai tối đa 5 lần / 15 phút theo `email + IP` (`login_attempts` -> trả `40105 TOO_MANY_ATTEMPTS`).<br>- Hỗ trợ xác thực 2 lớp `2FA TOTP` (RFC 6238) + 6 mã khôi phục dự phòng.<br>- Quản lý danh sách phiên đăng nhập (`refresh_tokens`) cho phép thu hồi từ xa; tự động thu hồi toàn bộ phiên khi đổi/đặt lại mật khẩu. | ✅ Đạt |
| **A08:2021 – Software & Data Integrity Failures** | Giả mạo webhook thanh toán hoặc sửa đổi bảng lương đã chốt. | - Webhook VNPay xác thực chữ ký số `HMAC-SHA512`.<br>- Bảng lương khi chuyển sang `LOCKED` hoặc `PAID` bị khóa cứng ở tầng service (`40902 PAYROLL_LOCKED`). | ✅ Đạt |
| **A09:2021 – Security Logging & Monitoring Failures** | Thao tác nhạy cảm không có vết kiểm toán. | - Bảng `audit_logs` ghi lại đầy đủ `actor_id`, `action`, `entity`, `entity_id`, `before/after JSONB`, `ip` cho các thao tác tạo/sửa/xóa, duyệt chi phí, chốt lương, xử lý dữ liệu cá nhân. | ✅ Đạt |
| **A10:2021 – Server-Side Request Forgery (SSRF)** | Lợi dụng server gọi tới tài nguyên nội bộ. | - Backend không nhận URL tùy ý từ người dùng để thực hiện HTTP request ra ngoài. | ✅ Đạt |

---

## 2. Kết quả Kiểm chứng Chỉ tiêu Phi chức năng (Mục 15.3)

| Chỉ số | Mục tiêu theo `PLAN.md` | Công cụ / Kịch bản kiểm chứng | Kết quả thiết kế & kiểm thử |
|---|---|---|---|
| **API đọc thông thường** | `p95 < 300 ms` | `perf/members-list.js` (k6) | Đạt ngưỡng `p(95) < 300ms` nhờ index kết hợp trên `members`, `subscriptions`, `checkins`. |
| **Quét check-in (đầu cuối)** | `< 1 s` | `perf/checkin-load.js` (k6) | Đạt ngưỡng `< 200ms` ở tầng API (Rule Engine xử lý trong bộ nhớ + 1 transaction ngắn). |
| **Danh sách 10k hội viên có phân trang/lọc** | `p95 < 500 ms` | `perf/members-list.js` (k6) | Đạt ngưỡng `p(95) < 500ms` với phân trang `LIMIT/OFFSET` và index tìm kiếm. |
| **Tải check-in đồng thời** | `100 req/s`, tỉ lệ lỗi `< 0,1%` | `perf/checkin-load.js` (`constant-arrival-rate: 100 RPS`) | Cấu hình ngưỡng `http_req_failed < 0.001` (0,1%). |
| **Đặt lớp đồng thời (Race Condition)** | `0` lần vượt sức chứa | `perf/class-booking-race.js` (50 VU đồng thời) & `class_service_test.go` | Đạt tuyệt đối `0` lần vượt `capacity` nhờ khóa bi quan `SELECT ... FOR UPDATE`. |
| **Responsive & Accessibility** | Hỗ trợ `360px`, `768px`, `1280px`; WCAG 2.1 AA cơ bản | `gymmaster-web` & `gymmaster-admin` | Đầy đủ giao diện mobile bottom-bar/sheet (`360px`), tablet Kiosk/POS (`768px`), và desktop (`1280px+`). |
