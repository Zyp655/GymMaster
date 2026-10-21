# Kế hoạch sửa & kiểm thử – QA vòng 3 (GymMaster)

> Tài liệu bàn giao cho agent/dev tiếp theo. Lập ngày 21/10/2026 sau 3 vòng QA (vòng 1 → sửa → vòng 2 → sửa → vòng 3).
> Toàn bộ thay đổi của các vòng trước **chưa commit** (xem `git status` / `git diff HEAD`). Không được `git checkout/stash/reset`.

---

## 0. Bối cảnh & cách chạy

| Thành phần | Lệnh | Cổng |
|---|---|---|
| Postgres + Mailpit | `docker compose up -d postgres mailpit` | 5432 / 8025 |
| API | `cd gymmaster-api && go run . migrate up && go run . http` | 8000 (`/health`, `/swagger`) |
| Admin | `cd gymmaster-admin && npm run dev` | 3001 |
| Web | `cd gymmaster-web && npm run dev` | 3000 |

- Migration hiện có tới **000041** (không có 000039, 000040 – golang-migrate chấp nhận khoảng trống). Migration mới dùng số **≥ 000042**.
- Tài khoản demo (mật khẩu `Admin@123`): OWNER `admin@gymmaster.vn`, MANAGER `quanly.cg@gymmaster.vn` (CN1), RECEPTIONIST `letan.cg@gymmaster.vn`, TRAINER `hlv.cg@gymmaster.vn`, MEMBER `hoivien@gymmaster.vn` (web).
- DB hiện chứa nhiều dữ liệu test (tiền tố `QA-`, `FIX-`, `FIX2-`, `QA2-`, `QA3-`, `E2E-`). **Nên reset trước khi test lại** (lệnh xóa volume đã bị chặn tự động, cần người chạy tay):
  ```bash
  docker compose down -v && docker compose up -d postgres mailpit
  cd gymmaster-api && go run . migrate up && go run . seed
  ```
- Quy ước đã áp dụng ở vòng trước (giữ nguyên):
  - HTTP status chuẩn hóa theo nhóm mã (`internal/pkg/rescode` – `HTTPStatus()`), trường `code` trong body không đổi; validate → `40000`.
  - Lỗi DB được dịch sang 4xx ở `internal/middleware/errmap.go` + `internal/pkg/dberr`.
  - Nhãn trường tiếng Việt: `internal/pkg/fieldlabel`.
  - Mật khẩu mới: ≥ 8 ký tự, có chữ và số (`ValidateNewPassword`).
  - Lọc HTML: API `internal/pkg/htmlsafe`, admin `routes/contracts-passes/sanitizeHtml.ts`, web `src/lib/sanitize-html.ts`.

### Quy tắc khi sửa
1. Tái hiện lỗi trước (curl/script), sửa **nguyên nhân gốc**, thêm/mở rộng unit test.
2. Kiểm chứng runtime trên một API riêng (`SERVER_PORT=81xx CRON_ENABLED=false`), không đụng tiến trình :8000 đang dùng chung.
3. Test lại các luồng xung quanh (các vòng trước đã từng sinh regression khi sửa).
4. Với component/hàm dùng chung: grep mọi chỗ dùng và kiểm từng chỗ.
5. Không commit (theo yêu cầu chủ dự án).

---

## 1. Danh sách lỗi cần sửa (gom theo gói việc)

Mức độ: **H** = Cao, **M** = Trung bình, **L** = Thấp. Đường dẫn tính từ `gymmaster-api/` nếu không ghi rõ.

### Gói A – Phân quyền, phiên đăng nhập, bảo mật

| # | Mức | Lỗi | Vị trí / gợi ý sửa |
|---|---|---|---|
| A1 | **H** | Người có `role.update` (không phải OWNER) sửa được quyền vai trò hệ thống **MEMBER** (vd thêm `member.view`) → mọi hội viên gọi được `/v1/admin/*`. `branch_ids` rỗng còn được hiểu là “không giới hạn chi nhánh”. | `internal/service/role_service.go` `Update` (~105–118): cấm sửa/gán quyền cho các vai trò hệ thống không dành cho nhân viên (MEMBER; cân nhắc khóa cả OWNER). `internal/middleware/auth.go` `Authorization`: chặn token chỉ có role MEMBER ở nhóm `/admin` (hiện MEMBER còn gọi được `/admin/my-payslips`, `/admin/notifications`, `/admin/security/*`, `/admin/calendar-token`). `internal/pkg/reqctx/branch.go:14`: không coi `BranchIDs` rỗng = toàn quyền với non-OWNER. Role-admin gắn CN1 không nên sửa vai trò toàn cục đang gán cho NV chi nhánh khác. |
| A2 | M | “Đăng xuất thiết bị khác”/xóa 1 phiên → khi thiết bị bị thu hồi tự refresh, hệ thống tưởng là dùng lại token sau xoay vòng và `RevokeAllByUser` → phiên hiện tại cũng bị đăng xuất. | `internal/service/auth_service.go:274-277`. Thêm cột `revoked_reason` (ROTATED / LOGOUT / REVOKED_BY_USER / ADMIN) vào `refresh_tokens` (migration), chỉ coi là reuse khi reason = ROTATED. Cân nhắc chặn luôn access token theo claim `sid` khi phiên bị thu hồi. |
| A3 | M | SSE: client admin/web không xử lý `auth_expired`/`auth_revoked`, `onerror` gọi `es.close()` → mất thông báo realtime sau ~15 phút. Web toast đọc `data.content` trong khi API gửi `body`. | `gymmaster-admin/src/layouts/NotificationBell.tsx:72-75`, `gymmaster-web/src/features/portal/components/PortalShell.tsx:101,115-118`, `internal/pkg/realtime/hub.go`. Khi nhận 2 event đó → refresh token rồi mở lại EventSource (backoff). |
| A4 | M | Khóa đăng nhập theo (email, IP) khi login qua web: API thấy IP của server Next → kẻ xấu khóa tài khoản nạn nhân cho mọi người dùng web. Nếu bật `TRUSTED_PROXIES` thì `clientIpFrom` lấy XFF đầu tiên do client tự gửi. | `gymmaster-web/src/lib/client-info.ts:29-32`, `internal/pkg/clientip/clientip.go`. Lấy IP từ phải sang trái qua chuỗi proxy tin cậy; cấu hình `TRUSTED_PROXIES` trong `docker-compose.yml` cho IP container web; cân nhắc CAPTCHA/backoff thay vì khóa cứng theo email. |
| A5 | L/M | Legacy import: `pt_package_code`/`pt_remaining_sessions` nhận nhưng không ghi (vẫn báo COMPLETED); `plan_code` sai không báo lỗi khi thiếu ngày bắt đầu. | `internal/service/systemext_service.go` `importLegacyRow`, `planLegacyRows`. Hoặc import PT thật (docs `stage5-cms-system.md:57`), hoặc báo lỗi dòng. |
| A6 | L | Audit `LOGIN_FAILED` lý do `TOO_MANY_ATTEMPTS` không có `actor_id` dù email tồn tại. | `auth_service.go:178` (`auditLoginFailed(ctx, 0, …)`) → tra user theo email. |
| A7 | L | Audit còn mã thô: action `REASSIGN_TRAINER`, `SUBSTITUTE_TRAINER`; entity `locker_rental`, `online_payment_txn`. | `gymmaster-admin/src/api/audit.ts`. Nên viết test liệt kê mọi `audit.Log(` trong `internal/service` và so với từ điển nhãn. |
| A8 | L | Admin “Đổi mật khẩu”: lỗi chính sách mật khẩu hiện dưới ô “Mật khẩu hiện tại”; client chỉ kiểm min 8. | `gymmaster-admin/src/components/ChangePasswordModal/index.tsx:46` (regex chọn ô); thêm rule chữ + số phía client. |
| A9 | L | Nhỏ: `APP_ENV=Production` (hoa) không tắt swagger (`config.go:161` so sánh phân biệt hoa thường); thông báo bulk “Đã thực hiện ACTIVATE…” còn mã tiếng Anh; menu Nhân viên vẫn hiện Khóa/Đặt lại mật khẩu cho người không quản lý được (server đã chặn); forgot-password lộ email qua thời gian phản hồi (25 ms vs 2 ms). | Tương ứng từng file. |

### Gói B – Tiền: hóa đơn, VNPay, hoàn tiền, P&L

| # | Mức | Lỗi | Vị trí / gợi ý sửa |
|---|---|---|---|
| B1 | **H** | **Thất thoát tiền:** hoàn được hóa đơn gốc sau khi gói đã **nâng cấp** (credit đã dùng cho gói mới) hoặc **chuyển nhượng** (`carried_paid_base` vẫn còn). | `internal/service/invoice_service.go:643-714` (`invoiceApplyRefund`/`refund`). Thêm “refund guard” (tương tự `RegisterPayGuard`): chặn (409) khi gói của hóa đơn đã TRANSFERRED hoặc CANCELLED kèm sự kiện UPGRADED; hoặc chỉ cho hoàn phần chưa bị khấu trừ. |
| B2 | **H** | **P&L tính phần nợ đã xóa (`write_off_amount`) là doanh thu** (vd HĐ thu 300k, xóa nợ 1,05tr → P&L vẫn +1,35tr; tháng 10 bị thổi 4,65tr). Lệch với báo cáo Doanh thu/KPI/Excel (đã đúng). | `internal/repository/finance_repository.go:327-354` (`CalcProfitLoss` cộng `SUM(ii.amount)`): trừ `i.write_off_amount` (phân bổ theo dòng) hoặc dựa trên `total`. Cân nhắc cả phần chưa thu của HĐ PARTIAL theo định nghĩa đã ghi ở tooltip. |
| B3 | M | Nhân viên ghi được phiếu thu **VNPAY giả** qua `POST /admin/subscriptions` (và renew/upgrade/transfer) với `payment.method = VNPAY`. | `internal/service/subscription_service.go:319` (`subPayment` dùng `invoiceValidMethod` cho phép VNPAY – `invoice_service.go:415-420`); `api/subscription/v1/subscription.go:120` thêm `in:CASH,TRANSFER,CARD`. Rà mọi chỗ nhận `method` ở API quầy. |
| B4 | M | Sổ thu tiền bỏ sót VNPay trong `summary`/tổng (UI thẻ “VNPay Online” = 0); lọc `method=VNPAY` → 400. | `invoice_service.go:845-859` (`invoiceSummarize` chỉ lặp CASH/TRANSFER/CARD); `api/invoice/v1/invoice.go:175`. |
| B5 | M | Hoàn hết số đã thu trên HĐ **PARTIAL** → HĐ REFUNDED nhưng vẫn hiện “Còn nợ”, gói PENDING kẹt, không thu tiếp/không VNPay được. | `invoice_service.go:657-659`, `toInvoiceRes` `:968-976`. Chọn 1: chặn hoàn toàn bộ khi còn nợ, hoặc tự xóa nợ/hủy gói; Due = 0 khi REFUNDED. |
| B6 | L/M | Hóa đơn bán **vé ngày** ghi `source = POS` → tổng kết POS bị thổi (2 đơn, 240k, SL 0, giá vốn 0); mở trả hàng báo lỗi. | `internal/service/contractpass_service.go:462`; `internal/repository/pos_repository.go:104-120`. Dùng source COUNTER (hoặc nguồn riêng) và lọc POS theo dòng PRODUCT. |
| B7 | L | `/payment/result` báo “Thanh toán thành công” khi txn NEEDS_REFUND (HĐ đã VOID/đủ). | `internal/service/online_payment_service.go:509-526` – xét `txn.status`/`recorded_amount`. |
| B8 | L | Txn EXPIRED đã nhận IPN thất bại vẫn chuyển SUCCESS khi có IPN 00 sau đó. | `online_payment_service.go:367-390`. |
| B9 | L | Nhỏ: `gateway_ref` hoàn VNPay trùng khi tạo cùng giây (`systemext_service.go:530` – thêm id/ngẫu nhiên); vé mời không giới hạn ngày tương lai & 1 SĐT nhiều vé cùng ngày (`contractpass_service.go:93-106`); chữ ký chấp nhận `/uploads/%2e%2e/x.png` (decode trước khi kiểm); `access_hours.dow` không khử trùng, khung giờ chồng nhau vẫn nhận. | – |

### Gói C – Vận hành: lịch lớp, dạy thay, ngày lễ, check-in

| # | Mức | Lỗi | Vị trí / gợi ý sửa |
|---|---|---|---|
| C1 | M | Tạo/sửa lớp không xét lớp HLV đang **dạy thay** → HLV trùng lịch. | `internal/repository/class_schedule_repository.go:176-177` (`TrainerOverlap`) → `COALESCE(substitute_trainer_id, trainer_id)`; `class_schedule_service.go:571-585`. Ràng buộc EXCLUDE trong DB cũng chỉ theo `trainer_id`. |
| C2 | M | Chọn lại **HLV chính** cho lớp (`substitute_trainer_id = trainer_id`) bỏ qua mọi kiểm tra → trùng PT / giao lớp cho HLV đang nghỉ phép. | `internal/service/scheduleext_service.go` `SubstituteClassTrainer` nhánh `else if` (~413–419): gọi `checkNewTrainer` cả khi trả về HLV chính. |
| C3 | M | Người dạy thay “vô hình”: lịch dạy HLV (`/admin/trainer-schedule`), `/admin/class-schedules` (+ `/{id}`), lọc `trainer_id`, `/member/classes` đều không phản ánh người dạy thay; admin Lịch lớp không hiển thị. | `internal/repository/pt_repository.go:478` (`TrainerClasses`), DTO class schedule thêm `substitute_trainer_id/name`, bộ lọc dùng COALESCE; UI lịch lớp & portal hiển thị HLV thực dạy. |
| C4 | L/M | Ngày lễ: tạo được nhiều bản ghi cùng CN/cùng ngày; bản “đóng cửa” thêm sau bị bỏ qua (lấy `ORDER BY id LIMIT 1`). | `scheduleext_service.go:613` (Create/Update) – kiểm trùng → 409; migration UNIQUE `(branch_id, date)` (xử lý NULL branch cho lễ toàn hệ thống); `HolidayOn` (`:840`), `checkin_repository.go:459`. |
| C5 | L/M | Sửa điểm danh NO_SHOW → ATTENDED không gỡ khóa đặt lớp. | `internal/service/booking_service.go` ~363–377 (`applyNoShowBlocks` chỉ tăng) – tính lại số NO_SHOW trong cửa sổ và gỡ/giảm khóa. |
| C6 | L | `/member/classes` trả `bookable: true` cho lớp ngoài giờ lễ hoặc HLV nghỉ phép (bấm đặt → 403). | `internal/service/portal_service.go:556` – gọi `bookingCheckScheduleGuard` khi tính `bookable`. |
| C7 | L | Excel check-in: cột “Lý do từ chối” hiện mã thô (`DAY_PASS_USED`, `TRIAL_PASS_USED`, `OUTSIDE_ACCESS_HOURS`…), không dùng `deny_message`; “Số hội viên khác nhau” không đếm vé (lệch `/checkins/stats`). | `internal/service/report_export.go:414-418, 487-490` (`reportDenyLabel`). |
| C8 | L | Nhãn trường `size` = “kích thước trang” → lỗi tủ đồ “Kích thước trang không hợp lệ”. | `internal/pkg/fieldlabel/fieldlabel.go:148` – nhãn theo ngữ cảnh hoặc custom message trong DTO tủ. |
| C9 | L | `GET /admin/trainers/{id}/availability` không kiểm phạm vi (HLV/lễ tân đọc lịch HLV chi nhánh khác). | `scheduleext_service.go:110-116`. |
| C10 | L (bảo mật) | Kiosk tự dùng device key demo cứng `GMD-demo-kiosk-cg01` kể cả production, còn ghi key demo vào localStorage. | `gymmaster-web/src/features/kiosk/kiosk-utils.ts:4,22-29`, `KioskView.tsx:85-86` – chỉ fallback khi `NODE_ENV=development`. |
| C11 | L (UX) | HLV mới thấy tin của HLV cũ là chưa đọc; bảng ngày lễ hiện “08:00–17:00” khi closed=false mà không có giờ (API cho tạo vậy – nên bắt buộc giờ); lịch rảnh nhận ca chồng nhau; Alert info light mode nền xám đậm; bảng Lịch sử ra vào 1440 cắt cột Nhân viên. | – |

### Gói D – Tài chính, kho, KPI

| # | Mức | Lỗi | Vị trí / gợi ý sửa |
|---|---|---|---|
| D1 | M | Khoản chi sửa/duyệt/từ chối/xóa **không ghi audit**; khoản đã duyệt vẫn sửa → duyệt lại → **xóa hẳn**; đổi thẳng APPROVED → REJECTED. | `internal/service/finance_service.go` UpdateExpense (~296-356), DeleteExpense (357-369), ApproveExpense (371-398); `repository/finance_repository.go:224` (xóa cứng → xóa mềm hoặc chặn xóa khi APPROVED). |
| D2 | M | KPI “Gia hạn gói tập” đếm cả chuyển nhượng & nâng cấp (33 vs 24 sự kiện RENEWED). | `internal/repository/kpi_repository.go:211-215` – đếm theo `subscription_events.type = 'RENEWED'`. |
| D3 | L | Số lượng lẻ bị cắt phần thập phân không báo lỗi (nhập 1.5 → 1, kiểm kê 2.7 → 2, bán 1.9 → 1). | `api/inventory/v1/inventory.go:14,29,44`, `api/pos/v1` (OrderItem ~45, ReturnItem ~106) thêm rule `integer`. |
| D4 | L | Mở ca thu ngân ở CN ngừng hoạt động; ghi chú > 500 ký tự → lỗi chung “Dữ liệu quá dài”. | `finance_service.go:401-415`; `api/finance/v1/finance.go:159,167` thêm `max-length`. |
| D5 | L | Khoản chi nhận ngày quá cũ (1999) và CN ngừng hoạt động; sales-target nhận CN ngừng hoạt động (`kpi_repository.go:158` – `BranchExists` không xét status). | – |
| D6 | L | `profit-loss`, `revenue`, `overview` với `branch_id` không tồn tại → 200 toàn số 0 (kpi/cohort/sales-targets đã → 404). Thống nhất 404. | report/finance services. |
| D7 | L | Excel doanh thu sheet “Theo thời gian”: cột theo loại là thực thu (đã trừ hoàn) cạnh “Tổng thu” gộp → dễ nhầm. P&L kỳ lẻ ngày cộng lương cả tháng (`finance_repository.go:405-416`, chưa tái hiện). | – |
| D8 | L (UI) | 1440: bảng Kho cuộn ngang, cột “Giá trị tồn” bị cột thao tác che; bảng Nhà cung cấp cắt cột Ghi chú; Gói tập: cột “Vé mời/tháng”, “Khung giờ” bị đẩy ra ngoài; tab VNPay: nút “Hoàn tiền” ngoài màn hình. 390: `/app/finance` tràn 7px; drawer lead cắt nhãn bước, warning antd `Descriptions span={2}` khi `column=1` (`routes/leads/index.tsx:497`). “Thu theo phương thức” (POS orders) là số gộp – thêm chú thích; “Đang trong phòng tập” Dashboard (từ 00:00) khác trang Mật độ (3 giờ) – thống nhất hoặc ghi rõ. | – |

---

## 2. Thứ tự đề xuất

1. **Đợt 1 (bắt buộc):** A1, B1, B2 → B3, B5, A2.
2. **Đợt 2:** C1–C3 (dạy thay – cùng một chủ đề), B4, D1, D2, A3, A4.
3. **Đợt 3:** các mục L còn lại theo gói.

Có thể giao song song theo gói (A / B / C / D) nếu mỗi agent chỉ sửa file trong gói của mình; các file dùng chung (`invoice_service.go`, `scheduleext_service.go`, `report_export.go`, `fieldlabel.go`) cần đọc lại trước mỗi lần sửa.

---

## 3. Kế hoạch kiểm thử

### 3.1 Với mỗi lỗi
- Viết **unit test** tái hiện (fail trước, pass sau). Với lỗi đồng thời: test chạy song song + `go test -race`.
- Kiểm **runtime** bằng curl/script trên API riêng, ghi lại request/response.
- Với lỗi UI: Playwright (`e2e/node_modules/playwright`) chụp 1440 & 390, đọc console/network.

### 3.2 Kịch bản kiểm chứng then chốt
| Lỗi | Kịch bản chấp nhận |
|---|---|
| A1 | Role-admin (non-OWNER) `PUT /admin/roles/{MEMBER}` → 403; token MEMBER gọi mọi `/v1/admin/*` → 403; ma trận RBAC 5 vai trò × toàn bộ endpoint trong `/api.json` khớp `role_permissions`. |
| A2 | Đăng nhập A, B → A “đăng xuất thiết bị khác” → B refresh (401) → **A vẫn refresh được**; dùng lại refresh token đã xoay vòng vẫn thu hồi toàn chuỗi. |
| B1 | Bán M1 thu đủ → nâng cấp M3 → hoàn HĐ M1 → 409. Chuyển nhượng → hoàn HĐ gốc → 409 (hoặc chỉ phần chưa khấu trừ). |
| B2 | HĐ 1,35tr thu 300k → hủy gói (xóa nợ 1,05tr) → P&L kỳ đó chỉ tăng 300k; P&L = Doanh thu thuần (cùng định nghĩa) đối chiếu SQL. |
| B3 | `POST /admin/subscriptions` (và renew/upgrade/transfer) với `method: VNPAY` → 400. |
| B5 | HĐ PARTIAL hoàn toàn bộ → theo phương án đã chọn, không còn “Còn nợ” trên HĐ REFUNDED, gói không kẹt PENDING. |
| C1–C3 | Gán dạy thay t2 cho lớp X → tạo lớp khác cho t2 trùng giờ → 409; chọn lại HLV chính đang nghỉ phép/bận → 409/43006; lịch dạy t2, `/admin/class-schedules?trainer_id=t2`, `/member/classes` đều hiện t2 là HLV thực dạy. |
| D1 | Mọi thao tác khoản chi có audit; khoản đã duyệt không xóa cứng được. |

### 3.3 Hồi quy (chạy sau mỗi đợt)
```bash
cd gymmaster-api && gofmt -l . && go vet ./... && go test -race -count=1 ./... && go build ./...
cd gymmaster-admin && npm run typecheck && npm test && npm run lint && npm run build
cd gymmaster-web && npm run typecheck && npm test && npm run lint && npm run i18n:check
cd e2e && npx playwright test        # 6 test, chạy 2 lần để chắc lặp lại được
npx -y newman run postman/gymmaster-api.postman_collection.json
```
Thêm:
- Migration trên DB trống: `migrate up` → `seed` (2 lần) → `migrate down` hết → `up` lại; so `pg_dump -s` với DB đang chạy.
- `docker compose build api admin web`.

### 3.4 Phần vòng 3 **chưa test xong** (bị dừng để tiết kiệm token) – cần làm sau khi sửa
- **Web/Cổng hội viên vòng 3:** chưa có báo cáo. Cần kiểm lại: `/me/workout` (mức tạ), lịch lớp trang chủ (`available_slots`), bài nháp không lộ + `/blog/[slug]` (XSS trong nội dung bài), open redirect (`/login?returnUrl=` các biến thể), trang Tài khoản (phiên, tải dữ liệu, 2FA), link thông báo, `?ref=` không lỗi hydration, thử thách đã kết thúc, dinh dưỡng (gam/khẩu phần), chữ mờ `text-muted`, email hồ sơ trùng, sổ điểm admin, giao diện mobile, **CSP không vi phạm trên mọi trang** (VNPay redirect, kiosk camera, SSE), luồng đăng ký liên kết hồ sơ, banner khóa đặt lớp; IDOR mọi `/member/*` với hội viên thứ hai.
- **Kiểm tra xuyên suốt vòng 3:** chưa có báo cáo. Cần: `go test -race`, build web/admin, migration up/down trên DB trống, `docker compose build`, crawl 4 vai trò × mọi route admin (+ dark, 390), crawl web 390/1024/1280/1440, **fuzz toàn API** (không được có 5xx; mã HTTP khớp `code`; thông báo tiếng Việt không lộ tên field Go), header bảo mật API/admin/web, swagger ở `APP_ENV=production`.
- **Tài chính vòng 3 – phần UI chưa xong:** `BranchSelect` mới kiểm trên màn POS – còn 24 màn (inventory adjust/transfer/import, members import, manual checkin, shifts, checkin, trainer-schedules ngày lễ, KPI target, lockers, rooms, MemberForm, lịch lớp, chấm công, khoản chi, bán gói/PT, leads, equipments, access-devices); chữ nút `ConfirmButton` trên mọi chỗ dùng; dark mode reports/kpi/finance/dashboard; padding `page-card`; tooltip giải thích doanh thu; tràn ngang 390 ở finance/reports/kpi/dashboard/payrolls.

### 3.5 Tiêu chí hoàn thành
- Không còn lỗi mức H/M ở mục 1; mọi lỗi đã sửa có unit test.
- Toàn bộ lệnh ở 3.3 xanh; fuzz API 0 lỗi 5xx.
- Một vòng QA độc lập (theo 3.4) không tìm thêm lỗi mức H/M.

---

## 4. Tài liệu tham khảo
- Kế hoạch & nghiệp vụ: `docs/PLAN.md`, `docs/api/*.md`, `docs/user-guide/README.md`.
- Báo cáo & ảnh chụp các vòng QA (thư mục tạm, có thể bị xóa khi khởi động lại máy): `/private/tmp/claude-501/-Users-hieu-Work/1e838077-5da3-43a6-8a7d-59287bf8ebd2/scratchpad/qa3-*/`, `qa2-*/`, `fix2-*/`.
