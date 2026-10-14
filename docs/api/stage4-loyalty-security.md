# Tài liệu API Giai đoạn 4 – Hợp đồng điện tử, Lịch nâng cao, KPI & Bảo mật Nghị định 13/2023 (Sprint 24–28)

## 1. Hợp đồng điện tử, Vé ngày & Gói Off-Peak (`Sprint 24` – `000023_contracts_passes`)
- Admin:
  - CRUD `/v1/admin/contract-templates`: Quản lý mẫu hợp đồng điện tử hỗ trợ biến `{{member_name}}`, `{{member_code}}`, `{{member_phone}}`, `{{plan_name}}`.
  - `GET/POST /v1/admin/member-contracts`: Danh sách & khởi tạo hợp đồng điện tử cho hội viên.
  - `GET/POST /v1/admin/day-passes`: Bán vé tập theo ngày (`DP-...`, quét 1 lần tại quầy/kiosk).
- Member:
  - `GET /v1/member/contracts`, `POST /v1/member/contracts/{id}/sign`: Xem và ký hợp đồng điện tử trực tuyến.
  - `GET/POST /v1/member/guest-passes`: Hội viên tự tạo mã QR vé khách mời (`DP-...`) theo hạn mức gói tập.
- Rule Engine Check-in mở rộng:
  - Hỗ trợ quét mã vé ngày / vé khách `DP-...` (`DAY_PASS_USED`, `DAY_PASS_EXPIRED`).
  - Kiểm tra khung giờ gói tập off-peak `membership_plans.access_hours` (`OUTSIDE_ACCESS_HOURS`).
  - Kiểm tra trả góp quá hạn (`INSTALLMENT_OVERDUE`) và hợp đồng chưa ký (`CONTRACT_UNSIGNED`).

## 2. Lịch rảnh HLV, Đơn nghỉ phép, HLV dạy thay & Lịch nghỉ lễ (`Sprint 25` – `000024_availability_holidays`)
- `GET/PUT /v1/admin/trainers/{id}/availability`: Thiết lập khung giờ làm việc trong tuần của HLV.
- `GET/POST /v1/admin/leave-requests`, `POST /v1/admin/leave-requests/{id}/approve`, `POST /v1/admin/leave-requests/{id}/reject`.
- `GET /v1/admin/leave-requests/{id}/impacts`: Liệt kê các buổi lớp nhóm & buổi PT bị ảnh hưởng trong thời gian xin nghỉ.
- `POST /v1/admin/class-schedules/{id}/substitute`: Gán HLV dạy thay cho buổi lớp nhóm.
- `POST /v1/admin/pt-sessions/{id}/reassign`: Chuyển buổi tập PT sang HLV dạy thay (`original_trainer_id` được lưu lại).
- `GET/POST/DELETE /v1/admin/branch-holidays` & `GET /v1/public/holidays`: Quản lý lịch nghỉ lễ, tùy chọn tự động cộng bù ngày cho các gói tập `TIME` đang `ACTIVE`.

## 3. Mật độ phòng tập Realtime, Chỉ tiêu KPI & Cohort Retention (`Sprint 26` – `000025_kpi`)
- `GET /v1/public/branches/occupancy` & `GET /v1/member/occupancy`: Mật độ người đang tập tại từng chi nhánh (`present_count / capacity`, mức `QUIET`, `NORMAL`, `BUSY`, `FULL`).
- `GET/POST/DELETE /v1/admin/sales-targets`: Thiết lập chỉ tiêu doanh số theo chi nhánh / nhân viên (`REVENUE`, `NEW_MEMBERS`, `RENEWALS`, `PT_SALES`).
- `GET /v1/admin/reports/kpi?period=YYYY-MM`: Báo cáo hoàn thành KPI thực tế so với mục tiêu.
- `GET /v1/admin/reports/cohort`: Ma trận tỷ lệ giữ chân hội viên (Cohort Retention) theo tháng gia nhập.

## 4. Xác thực 2 lớp (2FA TOTP), Quản lý phiên & Nghị định 13/2023 (`Sprint 27` – `000026_security_privacy`)
- Bảo mật 2FA & Phiên đăng nhập (`/v1/admin/security/*` và `/v1/member/security/*`):
  - `GET /security/mfa`, `POST /security/mfa/setup`, `POST /security/mfa/enable`, `POST /security/mfa/disable`, `POST /security/mfa/verify` (RFC 6238 TOTP + mã khôi phục).
  - `GET /security/sessions`, `DELETE /security/sessions/{id}`, `POST /security/sessions/revoke-others`.
- Quyền riêng tư & Dữ liệu cá nhân (Nghị định 13/2023):
  - `GET/PUT /v1/member/consents`: Quản lý đồng ý điều khoản (`TERMS`), tiếp thị (`MARKETING`), hình ảnh (`PHOTO`).
  - `GET/POST /v1/member/data-requests`: Hội viên yêu cầu xuất dữ liệu (`EXPORT`) hoặc yêu cầu xóa dữ liệu (`DELETE`).
  - `GET /v1/admin/data-requests`, `POST /v1/admin/data-requests/{id}/process`: Admin duyệt yêu cầu dữ liệu (ẩn danh hóa PII khi xóa để bảo toàn khóa ngoại hóa đơn).

## 5. Kiểm thử E2E, Kiểm thử tải & CI (`Sprint 28`)
- `e2e/`: Bộ kiểm thử giao diện End-to-End bằng Playwright (`playwright.config.ts`, `tests/gymmaster.spec.ts`).
- `perf/`: Kịch bản kiểm thử tải & đồng thời bằng k6 (`checkin-load.js`, `members-list.js`, `class-booking-race.js`).
- `postman/`: Bộ kịch bản kiểm thử API Newman (`gymmaster-api.postman_collection.json`).
