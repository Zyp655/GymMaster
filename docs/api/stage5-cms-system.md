# Tài liệu API Bổ sung: Giai đoạn 5 (Sprint 29–30 & Tiện ích Hệ thống)

Tài liệu này mô tả các API thuộc **Sprint 29–30 và Mục 15.1–15.2** trong `docs/PLAN.md`:
- Quản trị nội dung Landing / Marketing (`cms` module: Hồ sơ HLV công khai, Chi tiết bộ môn, Cảm nhận hội viên, Blog/Tin tức, Liên hệ).
- Nhắn tin trực tiếp HLV ↔ Hội viên theo hợp đồng PT, Tùy chọn nhận thông báo, Đồng bộ lịch `.ics`.
- Công cụ hệ thống (`systemext` module: Tìm kiếm nhanh `Ctrl+K`, Thao tác hàng loạt, Chuyển chi nhánh chính, Hoàn tiền VNPay Online, Chuyển đổi dữ liệu cũ `Dry-Run`/`Commit`, Wizard thiết lập ban đầu).

---

## 1. Nội dung Công khai & Marketing (`/v1/public` & `/v1/admin`)

| Phương thức | Đường dẫn | Quyền | Mô tả |
|---|---|---|---|
| `GET` | `/v1/public/trainers` | Công khai | Danh sách hồ sơ HLV công khai kèm điểm đánh giá trung bình (`rating_avg`, `rating_count`). |
| `GET` | `/v1/public/trainers/{slug}` | Công khai | Chi tiết hồ sơ HLV công khai theo `slug`. |
| `GET` | `/v1/public/class-types` | Công khai | Danh sách các bộ môn lớp học nhóm đang hoạt động. |
| `GET` | `/v1/public/class-types/{id}` | Công khai | Chi tiết bộ môn kèm danh sách lịch học sắp diễn ra. |
| `GET` | `/v1/public/testimonials` | Công khai | Danh sách cảm nhận hội viên đã được duyệt (`APPROVED`). |
| `POST` | `/v1/public/testimonials` | Công khai | Hội viên gửi cảm nhận mới (trạng thái `PENDING` chờ duyệt). |
| `GET` | `/v1/public/posts` | Công khai | Danh sách bài viết Blog / Cẩm nang tập luyện đã xuất bản (`PUBLISHED`). |
| `GET` | `/v1/public/posts/{slug}` | Công khai | Chi tiết bài viết theo `slug`. |
| `POST` | `/v1/public/contact-messages` | Công khai | Gửi lời nhắn liên hệ từ trang chủ (tự động thông báo cho lễ tân). |
| `GET/POST/PUT/DELETE` | `/v1/admin/trainer-profiles` | `cms.manage` | Quản lý hồ sơ HLV công khai. |
| `GET/POST/DELETE` | `/v1/admin/testimonials` | `cms.manage` | Duyệt (`/approve`), từ chối (`/reject`) hoặc xóa cảm nhận hội viên. |
| `GET/POST/PUT/DELETE` | `/v1/admin/posts` | `cms.manage` | Quản lý bài viết Blog / Tin tức. |
| `GET/PUT` | `/v1/admin/contact-messages` | `cms.manage,lead.view` | Xem và cập nhật trạng thái xử lý lời nhắn liên hệ. |

---

## 2. Nhắn tin HLV ↔ Hội viên, Tùy chọn Thông báo & Lịch `.ics`

Các endpoint dưới đây được gắn trên cả `/v1/member/*` (dành cho Hội viên) và `/v1/admin/*` (dành cho HLV / Nhân viên):

| Phương thức | Đường dẫn | Mô tả |
|---|---|---|
| `GET` | `/conversations` | Danh sách cuộc trò chuyện gắn với hợp đồng PT (`pt_contract_id`). |
| `GET` | `/conversations/{id}/messages` | Danh sách tin nhắn trong cuộc trò chuyện (tự động đánh dấu `read_at`). |
| `POST` | `/conversations/{id}/messages` | Gửi tin nhắn mới (chỉ cho phép khi hợp đồng PT còn `ACTIVE`, đẩy realtime qua SSE). |
| `GET` | `/notification-preferences` | Xem tùy chọn bật/tắt thông báo theo từng loại & kênh (`IN_APP`, `EMAIL`). |
| `PUT` | `/notification-preferences` | Lưu tùy chọn thông báo (các loại bắt buộc như Hóa đơn/Bảo mật luôn bật). |
| `GET` | `/calendar-token` | Lấy đường dẫn đồng bộ lịch `.ics` cá nhân. |
| `POST` | `/calendar-token/rotate` | Cấp lại mã token lịch `.ics` (thu hồi link cũ). |
| `GET` | `/v1/public/calendar.ics?token=...` | Xuất file lịch chuẩn iCalendar (RFC 5545) gồm lịch lớp đã đặt và lịch PT. |

---

## 3. Tiện ích Quản trị, Chuyển Chi nhánh, Hoàn tiền Online & Chuyển đổi Dữ liệu Cũ (`/v1/admin`)

| Phương thức | Đường dẫn | Quyền | Mô tả |
|---|---|---|---|
| `GET` | `/v1/admin/search?q=...` | Nhân viên đăng nhập | Tìm kiếm nhanh toàn hệ thống (`Ctrl+K`): hội viên, hóa đơn, sản phẩm POS, gói tập. |
| `POST` | `/v1/admin/members/bulk-action` | `member.update` | Thao tác hàng loạt trên danh sách hội viên (gửi thông báo, chuyển chi nhánh, kích hoạt/ngừng). |
| `GET` | `/v1/admin/branch-transfers` | `member.view,subscription.transfer` | Danh sách lịch sử chuyển chi nhánh chính của hội viên. |
| `POST` | `/v1/admin/branch-transfers` | `member.update,subscription.transfer` | Chuyển chi nhánh chính cho hội viên (tự động chuyển theo các gói `SINGLE_BRANCH` đang chạy). |
| `POST` | `/v1/admin/online-payments/{id}/refund` | `invoice.refund` | Hoàn tiền giao dịch VNPay trực tuyến qua cổng sandbox, ghi nhận `refunds.online_txn_id`. |
| `GET` | `/v1/admin/legacy-imports` | `import.legacy,member.import` | Lịch sử các phiên chuyển đổi dữ liệu cũ (`legacy_import_jobs`). |
| `POST` | `/v1/admin/legacy-imports` | `import.legacy,member.import` | Chạy thử (`DRY_RUN`) hoặc ghi chính thức (`COMMIT`) dữ liệu hội viên, gói đang chạy, hợp đồng PT và công nợ (`source = 'IMPORT'`). |
| `GET` | `/v1/admin/setup-wizard/status` | `setup.wizard,setting.manage` | Kiểm tra tiến độ 6 bước thiết lập ban đầu của hệ thống. |
| `POST` | `/v1/admin/setup-wizard/complete` | `setup.wizard,setting.manage` | Lưu cấu hình nhanh hoặc đánh dấu hoàn tất Wizard thiết lập ban đầu. |
