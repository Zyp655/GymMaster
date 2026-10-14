# Tài liệu API Giai đoạn 3 – Tự động hóa vận hành, Nhân sự & Tài chính nâng cao (Sprint 18–23)

## 1. Thiết bị kiểm soát ra vào & Kiosk (`Sprint 18` – `000017_access_devices`)
- `GET /v1/admin/access-devices` (`perm: device.view,device.manage`): Danh sách thiết bị Cổng xoay / Cửa từ / Kiosk.
- `POST /v1/admin/access-devices` (`perm: device.manage`): Tạo thiết bị mới, trả `raw_api_key` một lần duy nhất (`GMD-...`).
- `PUT /v1/admin/access-devices/{id}`, `DELETE /v1/admin/access-devices/{id}` (`perm: device.manage`).
- `POST /v1/admin/access-devices/{id}/rotate-key` (`perm: device.manage`): Cấp lại `raw_api_key` mới.
- `POST /v1/public/device/checkin` (hoặc `/v1/device/checkin`, header `X-Device-Key`): Quét mã QR hội viên / vé tập thử / vé ngày tại Kiosk/Cổng xoay, tái sử dụng `CheckinService` rule engine.
- `POST /v1/public/device/heartbeat` (hoặc `/v1/device/heartbeat`, header `X-Device-Key`): Cập nhật `last_seen_at` và trạng thái `ACTIVE`.
- Cron `device_heartbeat_check` (`*/5 * * * *`): Chuyển thiết bị quá 10 phút không gửi heartbeat sang `OFFLINE` và gửi thông báo cho quản lý.

## 2. Tủ đồ & Dịch vụ theo lượt (`Sprint 19` – `000018_lockers`)
- `GET /v1/admin/lockers` (`perm: locker.view,locker.manage`): Danh sách tủ đồ.
- `GET /v1/admin/lockers/board?branch_id=` (`perm: locker.view,locker.manage`): Sơ đồ lưới tủ đồ theo khu vực kèm trạng thái (`AVAILABLE`, `OCCUPIED`, `RENTED`, `BROKEN`) và hội viên đang sử dụng.
- `POST /v1/admin/lockers/{id}/assign` (`perm: locker.manage`): Gán tủ theo lượt (`DAILY`) hoặc cho thuê tháng (`MONTHLY`).
- `POST /v1/admin/lockers/{id}/release` (`perm: locker.manage`): Giải phóng tủ đồ về `AVAILABLE`.
- `POST /v1/admin/locker-rentals`, `GET /v1/admin/locker-rentals`.
- `GET /v1/member/locker` (`role: MEMBER`): Xem tủ đồ đang sử dụng trên cổng hội viên.

## 3. Chấm công & Bảng lương (`Sprint 20` – `000019_payroll`)
- `POST /v1/admin/attendances/scan` (`perm: payroll.manage,shift.manage`): Chấm công vào/ra cho nhân viên, tự tính phút đi muộn/về sớm theo ca trực.
- `GET /v1/admin/attendances` (`perm: payroll.view,payroll.manage`): Lịch sử chấm công.
- `POST /v1/admin/payrolls/generate` (`perm: payroll.manage`): Tính bảng lương tháng (`YYYY-MM`) tổng hợp ngày công, hoa hồng PT (`trainer_commissions`), hoa hồng bán gói tập (`subscriptions.sold_by`).
- `GET /v1/admin/payrolls`, `GET /v1/admin/payrolls/{id}`, `PUT /v1/admin/payrolls/{id}` (`perm: payroll.view,payroll.manage`).
- `POST /v1/admin/payrolls/{id}/lock`, `POST /v1/admin/payrolls/{id}/pay`, `GET /v1/admin/payrolls/{id}/export`.
- `GET /v1/admin/my-payslips`: Nhân viên xem phiếu lương cá nhân.

## 4. Chi phí, Đối soát ca, Trả góp, P&L & Hóa đơn điện tử (`Sprint 21` – `000020_finance_ext`)
- `GET/POST /v1/admin/expense-categories`, CRUD `/v1/admin/expenses`, `POST /v1/admin/expenses/{id}/approve`.
- `POST /v1/admin/cash-sessions/open`, `POST /v1/admin/cash-sessions/{id}/close`, `GET /v1/admin/cash-sessions`: Mở/đóng ca thu ngân & đối soát chênh lệch tiền mặt.
- `GET /v1/admin/reports/profit-loss`: Báo cáo Lãi/Lỗ (P&L) theo chi nhánh & kỳ.
- `POST /v1/admin/invoices/{id}/installments`, `GET /v1/admin/installments`, `POST /v1/admin/installments/{id}/pay`, `GET /v1/member/installments`: Quản lý trả góp N kỳ.
- `POST /v1/admin/invoices/{id}/einvoice`: Phát hành hóa đơn điện tử (`EInvoiceProvider`).

## 5. Điểm thưởng Loyalty, Huy hiệu, Thử thách & Giới thiệu (`Sprint 22` – `000021_loyalty`)
- Admin: CRUD `/v1/admin/loyalty-rewards`, `/v1/admin/badges`, `/v1/admin/challenges`, `POST /v1/admin/members/{id}/points/adjust`.
- Member: `GET /v1/member/loyalty`, `POST /v1/member/loyalty/redeem` (sinh mã voucher 1 lần dùng trong `promotions`), `GET /v1/member/badges`, `GET /v1/member/challenges`, `POST /v1/member/challenges/{id}/join`, `GET /v1/member/leaderboard`, `GET /v1/member/referral`.

## 6. Dinh dưỡng, Bảng tin & Khảo sát NPS (`Sprint 23` – `000022_engagement`)
- Admin: CRUD `/v1/admin/foods`, `/v1/admin/meal-plans`, `/v1/admin/announcements`, `/v1/admin/surveys`, `GET /v1/admin/reports/nps`.
- Member: `GET /v1/member/meal-plans` (kèm gợi ý calo BMR Mifflin–St Jeor), `GET/POST/DELETE /v1/member/food-logs`, `GET /v1/member/announcements`, `GET /v1/member/surveys`, `POST /v1/member/surveys/{id}/responses`.
