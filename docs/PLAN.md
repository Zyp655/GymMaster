# GymMaster – Kế hoạch phát triển

> Đồ án tốt nghiệp: **Phát triển hệ thống số hóa quy trình quản lý trung tâm Fitness trên nền tảng Web**
> Stack: **Go (GoFrame v2) + PostgreSQL** cho backend, **React (Next.js App Router) + TypeScript** cho frontend.
> **Đã chốt:** mở rộng = **POS bán hàng + tồn kho** (tuần 14); **thanh toán online VNPay để sau** (giai đoạn sau đồ án, cổng hội viên chỉ xem hóa đơn).
> **Cập nhật kiến trúc FE:** tách thành 2 ứng dụng như hệ StrongBody:
> - `gymmaster-admin` – **trang quản trị** cho nhân viên/chủ trung tâm, React 18 + Vite + TypeScript + **Ant Design v5**, kiến trúc theo `strongbody_admin` (Sider/Header, menu lọc theo quyền, ActionBar/FilterBar/Table/Modal form).
> - `gymmaster-web` – **trang công khai + cổng hội viên**, Next.js (theo `strongbody-web`).
> Tổ chức **1 repo** (`GymMaster/` gồm `gymmaster-api/`, `gymmaster-web/`, `docs/`, `docker-compose.yml`). Giao diện **chỉ tiếng Việt**.
> Kiến trúc lấy theo `strongbody-api` và `strongbody-web`. Các điểm yếu mà chính 2 repo đó đã ghi nhận sẽ được sửa lại.

---

## 1. Mục tiêu & phạm vi

### 1.1 Bài toán
Nhiều phòng gym vừa và nhỏ vẫn quản lý bằng sổ sách hoặc Excel, nên thường gặp các vấn đề sau:
- Không biết chính xác hội viên nào sắp hết hạn, ai đang bảo lưu.
- Check-in thủ công, hay bị gian lận (mượn thẻ, tập quá hạn).
- Lịch lớp và lịch PT bị trùng, không kiểm soát được sức chứa.
- Không có báo cáo doanh thu, tỷ lệ gia hạn, giờ cao điểm.
- Hoa hồng PT phải tính tay.

### 1.2 Mục tiêu hệ thống
Số hóa toàn bộ vòng đời hội viên: **Tiếp nhận → Đăng ký gói → Thanh toán → Check-in → Tập lớp/PT → Theo dõi tiến độ → Gia hạn/Bảo lưu → Báo cáo**.

### 1.3 Tác nhân (Actors)
| Actor | Mô tả |
|---|---|
| **Owner / Admin** | Chủ trung tâm. Toàn quyền: chi nhánh, nhân sự, gói tập, báo cáo, cấu hình |
| **Manager** | Quản lý một chi nhánh: nhân sự ca, lịch lớp, duyệt bảo lưu/hoàn tiền, báo cáo chi nhánh |
| **Receptionist (Lễ tân)** | Đăng ký hội viên, bán gói, thu tiền, check-in, bán hàng tại quầy |
| **Trainer (PT/HLV)** | Xem lịch dạy, điểm danh buổi PT/lớp, ghi chỉ số cơ thể và giáo án cho học viên, xem hoa hồng |
| **Member (Hội viên)** | Cổng hội viên: xem gói, QR check-in, đặt lớp, đặt lịch PT, xem lịch sử và biểu đồ tiến độ, thanh toán online |
| **System (Cron)** | Nhắc hết hạn, tự chuyển trạng thái gói, tự hủy booking no-show, tổng hợp báo cáo |

### 1.4 Phạm vi chức năng

**MVP (bắt buộc):**
- Auth và RBAC
- Chi nhánh
- Hội viên
- Gói tập
- Đăng ký/gia hạn/bảo lưu
- Thanh toán và hóa đơn
- Check-in QR
- Lớp nhóm và đặt chỗ
- PT
- Nhân sự
- Dashboard báo cáo
- Cổng hội viên

**Mở rộng (nếu còn thời gian, giúp điểm cộng):**
- POS bán hàng và tồn kho
- Quản lý thiết bị và bảo trì
- Thanh toán VNPay sandbox
- Thông báo email/realtime
- Khuyến mãi/voucher
- Audit log

**Giai đoạn sau (hướng phát triển, ngoài phạm vi 16 tuần):** xem §12–§15 (Sprint 12–30) và cách phân phạm vi ở §16.1.

---

## 2. Kiến trúc tổng thể

```
┌──────────────────────┐   HTTPS/JSON   ┌───────────────────────────┐
│  gymmaster-web       │ ─────────────▶ │  gymmaster-api (Go)       │
│  Next.js 15/16 App   │                │  GoFrame v2               │
│  Router + next-auth  │ ◀───────────── │  /v1/{admin,staff,member, │
│  (Admin + Staff +    │  {code,msg,    │       public}/...         │
│   Member portal)     │   data}        │                           │
└──────────────────────┘                │  controller → service →   │
                                        │  repository → GORM        │
                                        └──────┬───────────┬────────┘
                                               │           │
                                     ┌─────────▼──┐   ┌────▼─────┐
                                     │ PostgreSQL │   │  Redis   │ (cache, rate-limit,
                                     │   16       │   │  7       │  refresh-token blacklist)
                                     └────────────┘   └──────────┘
                                     MinIO/S3-compatible (ảnh hội viên, ảnh thiết bị)
```

Kiến trúc là **Monolith phân lớp (Layered / Clean-ish)**. Với quy mô đồ án, monolith dễ triển khai, dễ test và dễ trình bày. Nếu hội đồng hỏi về mở rộng sau này, có thể trả lời rằng các module đã tách rõ nên có thể chuyển sang microservice.

---

## 3. Backend – `gymmaster-api`

### 3.1 Giữ nguyên từ strongbody-api
- **GoFrame v2**: định nghĩa `api/<module>/v1` Req/Res có `g.Meta` (path, method, tags, summary). Từ đó GoFrame tự sinh route, Swagger và validation.
- **`gf gen ctrl`** sinh interface `I<Module>V1` và stub controller.
- Mỗi tầng theo pattern **interface + struct private + `NewX(deps)`**:
  - `controller/<module>/<module>_v1_<action>.go`
  - `service/<module>_service.go`
  - `repository/<module>_repository.go`
- **ResponseMiddleware** trả envelope `{code, message, data}`. Business code dùng `gcode` (ví dụ 40100 thì HTTP 401).
- **BaseEntity**: `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt` (soft delete), `CreatedBy`, `UpdatedBy`.
- **PaginationDto / BuildWhere / Paginate** với cùng shape list: `{data,total,total_page,current_page,limit}`.
- **AuthenticationMiddleware** (Bearer JWT) và **AuthorizationMiddleware**.
- Config đọc bằng `godotenv`, singleton `GetConfig()`.
- `.golangci.yml`, Makefile, Dockerfile multi-stage, testify.

### 3.2 Cải tiến so với strongbody-api
Các điểm này được rút từ chính `docs/HOC-GO-VA-KIEN-TRUC.md` §14 của repo gốc, và cũng là luận điểm để viết vào báo cáo.

| Vấn đề ở repo gốc | GymMaster làm |
|---|---|
| `AutoMigrate` mỗi lần boot | **golang-migrate** với file versioned `migrations/000001_init.up.sql` |
| `route.go` "God file" 1500 dòng | Mỗi module có `module.go` tự wire `repo→service→controller` và `Register(group)` |
| `dgrijalva/jwt-go` (deprecated) | `golang-jwt/jwt/v5` với access token 15 phút và refresh token 7 ngày (lưu hash trong DB, rotate) |
| RBAC là list role hard-code | **Permission-based RBAC**: `roles`, `permissions`, `role_permissions`, middleware `RequirePermission("member.create")` |
| Service gọi thẳng DB hoặc `ghttp` | Service chỉ phụ thuộc interface repository; user hiện tại lấy qua helper `auth.FromCtx(ctx)` |
| Entity trả thẳng ra response | Luôn map sang Res DTO |
| Không có transaction xuyên repo | `TxManager.WithTx(ctx, fn)` cho các nghiệp vụ như bán gói (subscription + invoice + payment) |
| Gần như không có test | Unit test service với mock (mục tiêu ≥ 60% coverage service), integration test repo bằng Postgres trong docker |
| Không có docker-compose, CI không test | `docker-compose.yml` (api, web, postgres, redis, minio) và GitHub Actions chạy lint + test + build |
| Cron chạy chung API | Vẫn chạy chung (đồ án) nhưng tách `internal/cron` và có cờ `CRON_ENABLED` |

### 3.3 Cấu trúc thư mục
```
gymmaster-api/
├── main.go
├── api/                         # Req/Res + g.Meta (gf gen ctrl đọc ở đây)
│   ├── auth/v1/  branch/v1/  member/v1/  package/v1/  subscription/v1/
│   ├── checkin/v1/  class/v1/  booking/v1/  trainer/v1/  pt_session/v1/
│   ├── invoice/v1/  payment/v1/  staff/v1/  report/v1/  ...
├── internal/
│   ├── cmd/cmd.go               # g.Server, global middleware, swagger, /health
│   ├── config/                  # Config struct, GetConfig()
│   ├── consts/                  # ctx keys, permission codes, enums
│   ├── controller/<module>/
│   ├── service/
│   ├── repository/              # base.go: BaseRepository[T] generic
│   ├── entity/
│   ├── dto/                     # pagination, mapper
│   ├── middleware/              # cors, response, authn, authz(permission), branch_scope, rate_limit, request_log
│   ├── module/<module>/module.go# wiring + Register routes  (thay route.go khổng lồ)
│   ├── router/router.go         # gọi module.Register theo nhóm /v1/admin, /v1/staff, /v1/member, /v1/public
│   ├── storage/{postgres,redis,objectstore}/
│   ├── cron/                    # expire_subscriptions, remind_expiring, no_show_bookings, daily_report
│   ├── pkg/                     # token(jwt), hash(bcrypt), qrcode, mailer, vnpay, txmanager
│   └── mocks/
├── migrations/                  # golang-migrate SQL
├── seeds/                       # dữ liệu mẫu demo (roles, permissions, gói tập, 200 hội viên giả)
├── resource/                    # swagger template, email template
├── hack/config.yaml             # gf cli
├── Makefile  Dockerfile  .golangci.yml  .env.example
└── docs/
```

### 3.4 Thư viện
| Mục đích | Thư viện |
|---|---|
| HTTP / router / swagger / validation / log | `gogf/gf/v2` |
| ORM | `gorm.io/gorm`, `gorm.io/driver/postgres` |
| Migration | `golang-migrate/migrate/v4` |
| JWT | `golang-jwt/jwt/v5` |
| Hash | `golang.org/x/crypto/bcrypt` |
| Cache | `redis/go-redis/v9` |
| Cron | `robfig/cron/v3` |
| QR | `skip2/go-qrcode` |
| Excel export | `xuri/excelize/v2` |
| Mail | `gopkg.in/gomail.v2` (SMTP, dev dùng Mailpit) |
| Object storage | `minio/minio-go/v7` |
| Test | `stretchr/testify`, `testcontainers-go` (tuỳ chọn) |

### 3.5 Quy ước API
- Prefix theo đối tượng:
  - `/v1/public/*` (không cần auth)
  - `/v1/auth/*`
  - `/v1/admin/*` (Owner/Manager/Staff, kiểm tra theo permission)
  - `/v1/member/*` (cổng hội viên, chỉ thao tác trên dữ liệu của chính mình)
- Header: `Authorization: Bearer <access>`. Header `X-Branch-Id` chọn chi nhánh làm việc; middleware `BranchScope` kiểm tra nhân viên có thuộc chi nhánh đó không.
- Response mẫu: `{ "code": 0, "message": "OK", "data": {...} }`, lỗi thì `{ "code": 40901, "message": "SUBSCRIPTION_OVERLAP", "data": null }`.
- Bảng mã lỗi theo module:

| Dải mã | Nhóm |
|---|---|
| 400xx | validate |
| 401xx | auth |
| 403xx | permission |
| 404xx | not found |
| 409xx | xung đột nghiệp vụ |
| 41xxx | subscription |
| 42xxx | checkin |
| 43xxx | booking |
| 44xxx | payment |
| 45xxx | POS / tồn kho |

Mã mới dự kiến cho giai đoạn 2–4 được tổng hợp ở §16.3.

---

## 4. Thiết kế cơ sở dữ liệu (PostgreSQL)

Quy ước:
- ID là `BIGSERIAL`, giống repo gốc.
- Mã hiển thị riêng: `member_code` (`GM000123`) và `invoice_no`.
- Thời gian là `timestamptz`; tiền là `NUMERIC(14,2)`.
- Có soft delete; trạng thái lưu bằng `varchar` + `CHECK`.

### 4.1 Danh sách bảng (≈ 30 bảng)

**Định danh & phân quyền**
- `users` (id, email, phone, password_hash, full_name, avatar_url, status, last_login_at)
- `roles` (id, code, name), `permissions` (id, code, module, description), `role_permissions`, `user_roles` (user_id, role_id, branch_id NULL = toàn hệ thống)
- `refresh_tokens` (id, user_id, token_hash, expires_at, revoked_at, user_agent, ip)
- `password_resets`

**Tổ chức**
- `branches` (id, code, name, address, phone, open_time, close_time, status)
- `staff_profiles` (user_id, branch_id, position, hire_date, base_salary, commission_rate)
- `trainer_profiles` (user_id, specialties[], bio, certifications, rating)
- `shifts` (id, branch_id, staff_id, start_at, end_at, note)

**Hội viên**
- `members` (id, user_id NULL, member_code, full_name, gender, dob, phone, email, id_card, address, avatar_url, emergency_contact, home_branch_id, source, status, qr_token, note)
- `body_metrics` (id, member_id, measured_at, weight, height, body_fat, muscle_mass, bmi, waist, chest, hip, measured_by)
- `member_notes` / `workout_plans` (giáo án PT cho hội viên)

**Gói tập & đăng ký**
- `membership_plans` (id, code, name, type[`TIME`|`SESSION`], duration_days, session_count, price, access_scope[`SINGLE_BRANCH`|`ALL`], max_freeze_days, max_freeze_times, includes_classes, status)
- `pt_packages` (id, name, session_count, price, validity_days, trainer_level)
- `subscriptions` (id, member_id, plan_id, branch_id, start_date, end_date, remaining_sessions, status[`PENDING`|`ACTIVE`|`FROZEN`|`EXPIRED`|`CANCELLED`|`TRANSFERRED`], price_paid, sold_by, invoice_id)
- `subscription_freezes` (id, subscription_id, from_date, to_date, reason, approved_by, status)
- `subscription_events` (lịch sử: created, renewed, frozen, unfrozen, transferred, cancelled, upgraded)
- `pt_contracts` (id, member_id, trainer_id, pt_package_id, total_sessions, used_sessions, start_date, end_date, status, invoice_id)

**Check-in**
- `checkins` (id, member_id, branch_id, subscription_id, checked_in_at, checked_out_at, method[`QR`|`MANUAL`|`CARD`], staff_id, result[`ALLOWED`|`DENIED`], deny_reason)

**Lớp nhóm & PT**
- `class_types` (id, name, description, color, default_duration)
- `rooms` (id, branch_id, name, capacity)
- `class_schedules` (id, class_type_id, branch_id, room_id, trainer_id, start_at, end_at, capacity, recurrence_rule, status)
- `class_bookings` (id, schedule_id, member_id, status[`BOOKED`|`WAITLIST`|`CANCELLED`|`ATTENDED`|`NO_SHOW`], booked_at, cancelled_at)
- `pt_sessions` (id, pt_contract_id, trainer_id, member_id, branch_id, start_at, end_at, status[`SCHEDULED`|`COMPLETED`|`CANCELLED`|`NO_SHOW`], trainer_note)

**Tài chính**
- `invoices` (id, invoice_no, member_id, branch_id, subtotal, discount, total, paid_amount, status[`UNPAID`|`PARTIAL`|`PAID`|`REFUNDED`|`VOID`], issued_by, issued_at)
- `invoice_items` (id, invoice_id, item_type[`SUBSCRIPTION`|`PT`|`PRODUCT`], ref_id, description, qty, unit_price, amount)
- `payments` (id, invoice_id, amount, method[`CASH`|`TRANSFER`|`CARD`|`VNPAY`], txn_ref, status, paid_at, received_by)
- `refunds`
- `promotions` (id, code, type[`PERCENT`|`FIXED`], value, start_at, end_at, usage_limit, used_count, applicable_plan_ids)
- `trainer_commissions` (id, trainer_id, source_type, source_id, amount, period, status)

**Mở rộng**
- `products`, `inventory_transactions` (POS)
- `equipments`, `maintenance_logs`
- `notifications` (id, user_id, type, title, body, read_at)
- `audit_logs` (id, actor_id, action, entity, entity_id, before jsonb, after jsonb, ip, created_at)

**Bổ sung giai đoạn 2–4** (chi tiết cột trong từng sprint ở §12–§15)
| Migration | Bảng mới / thay đổi |
|---|---|
| 000011 | `password_resets`, `login_attempts` |
| 000012 | `online_payment_txns`; `payments.method` thêm `VNPAY` |
| 000013 | `exercises`, `workout_plans`, `workout_plan_days`, `workout_plan_items`, `workout_logs` |
| 000014 | `reviews`; `trainer_profiles.rating_avg`, `rating_count` |
| 000015 | `equipments`, `maintenance_logs` (thay cho mục "Mở rộng" ở trên) |
| 000016 | `leads`, `lead_activities`, `trial_passes` |
| 000017 | `access_devices`; `checkins.device_id`, method `DEVICE` |
| 000018 | `lockers`, `locker_rentals` |
| 000019 | `staff_attendances`, `payrolls`, `payroll_items` |
| 000020 | `expense_categories`, `expenses`, `cash_sessions`, `installment_plans`, `installment_schedules` |
| 000021 | `loyalty_ledgers`, `loyalty_rewards`, `badges`, `member_badges`, `challenges`, `challenge_participants`; `members.loyalty_points`, `referral_code`, `referred_by` |
| 000022 | `foods`, `meal_plans`, `meal_plan_items`, `food_logs`, `announcements`, `surveys`, `survey_responses` |
| 000023 | `contract_templates`, `member_contracts`, `day_passes`; `membership_plans.access_hours`, `guest_passes_per_month` |
| 000024 | `trainer_availabilities`, `staff_leave_requests`, `branch_holidays`; HLV thay thế trên `class_schedules`/`pt_sessions` |
| 000025 | `sales_targets`; `branches.capacity` |
| 000026 | `user_mfa`, `consents`, `data_requests`; `refresh_tokens.device_name`, `last_used_at` |
| 000027 | `trainer_public_profiles`, `testimonials`, `posts`, `contact_messages`, `conversations`, `messages`, `notification_preferences` |
| 000028 | `refunds` thêm `online_txn_id`; `member_branch_transfers`; `legacy_import_jobs` |

### 4.2 Ràng buộc & index quan trọng (nên đưa vào báo cáo)
- **EXCLUDE constraint** (`btree_gist`) chống trùng lịch:
  - phòng tập: `EXCLUDE USING gist (room_id WITH =, tstzrange(start_at,end_at) WITH &&)`;
  - HLV: tương tự cho `trainer_id` trên `class_schedules` và `pt_sessions`.
- **Unique partial index**: mỗi hội viên chỉ có 1 booking active trong 1 lịch: `UNIQUE(schedule_id, member_id) WHERE status IN ('BOOKED','WAITLIST')`.
- **Index** `checkins(branch_id, checked_in_at)` cho heatmap giờ cao điểm; `subscriptions(member_id, status, end_date)`.
- **Khóa đặt chỗ**: `SELECT ... FOR UPDATE` trên `class_schedules` khi đặt chỗ để tránh vượt sức chứa (race condition), kèm test đồng thời 50 goroutine.

---

## 5. Nghiệp vụ chính (quy tắc cần cài đặt)

1. **Bán gói tập.** Trong một transaction:
   - tạo `invoice` + `invoice_items`;
   - áp voucher;
   - tạo `subscription` (`PENDING` nếu chưa thanh toán đủ, `ACTIVE` khi `paid_amount >= total`);
   - ghi `subscription_events`.

   Nếu hội viên còn gói đang chạy thì gói mới **nối tiếp**: `start_date` = `end_date` cũ + 1.
2. **Gia hạn / nâng cấp.** Nâng cấp tính tiền chênh lệch theo số ngày còn lại (pro-rata).
3. **Bảo lưu (freeze).**
   - Giới hạn theo `max_freeze_days` và `max_freeze_times` của plan; Manager duyệt.
   - Khi bảo lưu: status chuyển `FROZEN`; khi kết thúc bảo lưu: `end_date += số ngày bảo lưu`.
   - Cron tự mở bảo lưu khi đến hạn.
4. **Chuyển nhượng gói.** Chuyển sang hội viên khác, có tính phí chuyển nhượng.
5. **Check-in.** Hội viên đưa QR (token động có TTL 60s, ký HMAC, chống chụp màn hình dùng lại). Lễ tân quét. Hệ thống kiểm tra lần lượt:
   1. hội viên còn `ACTIVE`;
   2. có subscription `ACTIVE` hợp lệ tại chi nhánh (`access_scope`);
   3. gói theo buổi thì còn `remaining_sessions > 0` (sau đó trừ 1);
   4. không check-in trùng trong 1 giờ.

   Mọi lần quét đều ghi log, kể cả bị từ chối (`DENIED` + lý do).
6. **Đặt lớp.**
   - Đặt trước tối đa 7 ngày và hủy trước 2 giờ.
   - Hết chỗ thì vào `WAITLIST`; khi có người hủy, người đầu waitlist tự được lên và nhận thông báo.
   - Cron chuyển `NO_SHOW` sau khi lớp kết thúc; no-show 3 lần/tháng thì khóa đặt lớp 7 ngày.
7. **PT.** Hợp đồng PT gồm N buổi. Đặt lịch phải không trùng lịch HLV (EXCLUDE). Khi hoàn thành buổi:
   - `used_sessions++`;
   - sinh `trainer_commissions` = giá/buổi × `commission_rate`.
8. **Hết hạn & nhắc nhở.** Cron 00:05 hằng ngày:
   - chuyển `EXPIRED`;
   - email nhắc trước 7 ngày và 1 ngày;
   - tạo danh sách hội viên cần chăm sóc gia hạn cho lễ tân.
9. **Báo cáo.**
   - Doanh thu theo ngày/tháng/chi nhánh/loại gói.
   - Hội viên mới, active, hết hạn.
   - Tỷ lệ gia hạn (retention).
   - Heatmap check-in theo giờ × thứ.
   - Tỷ lệ lấp đầy lớp.
   - Hiệu suất và hoa hồng PT.
   - Xuất Excel.

---

## 6. Danh sách API (tóm tắt, ≈ 110 endpoint)

| Module | Endpoint chính |
|---|---|
| Auth | `POST /auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/forgot-password`, `/auth/reset-password`, `GET /auth/me`, `PUT /auth/change-password` |
| RBAC | CRUD `/admin/roles`, `GET /admin/permissions`, `PUT /admin/roles/{id}/permissions` |
| Branch / Room | CRUD `/admin/branches`, `/admin/rooms` |
| Staff / Trainer | CRUD `/admin/staff`, `/admin/trainers`, `/admin/shifts` |
| Member | CRUD `/admin/members`, `GET /admin/members/{id}/timeline`, `POST /admin/members/{id}/body-metrics`, `POST /admin/members/import` (Excel) |
| Plan | CRUD `/admin/membership-plans`, `/admin/pt-packages` |
| Subscription | `POST /admin/subscriptions` (bán gói), `/renew`, `/upgrade`, `/freeze`, `/unfreeze`, `/transfer`, `/cancel`, `GET /admin/subscriptions?status=expiring` |
| Check-in | `POST /admin/checkins/scan`, `POST /admin/checkins/manual`, `PUT /admin/checkins/{id}/checkout`, `GET /admin/checkins` |
| Class | CRUD `/admin/class-types`, `/admin/class-schedules` (hỗ trợ lặp tuần), `GET /admin/class-schedules/{id}/bookings`, `PUT .../attendance` |
| PT | CRUD `/admin/pt-contracts`, `/admin/pt-sessions`, `PUT /admin/pt-sessions/{id}/complete` |
| Finance | `/admin/invoices`, `/admin/payments`, `/admin/refunds`, `/admin/promotions`, `POST /public/payments/vnpay/ipn` |
| Report | `/admin/reports/overview`, `/revenue`, `/members`, `/checkin-heatmap`, `/class-occupancy`, `/trainer-performance`, `/export` |
| Member portal | `/member/profile`, `/member/qr`, `/member/subscriptions`, `/member/checkins`, `/member/classes` + `/book` + `/cancel`, `/member/pt-sessions`, `/member/body-metrics`, `/member/invoices`, `/member/payments/vnpay` |
| Public | `/public/branches`, `/public/plans`, `/public/class-schedules` (cho landing page) |
| Notification | `/notifications`, `PUT /notifications/{id}/read` |

Swagger tự sinh tại `/swagger`. Có thể đưa ảnh chụp Swagger vào phụ lục báo cáo.

> Bảng trên là API của giai đoạn 1. Endpoint của giai đoạn 2–4 liệt kê trong từng sprint ở §12–§15. Module mới: password reset, VNPay, workout, review, equipment, lead, access device, locker, payroll, expense/cash session/installment, loyalty, nutrition/announcement/survey, contract/day pass, availability/leave/holiday, KPI/cohort, MFA/session/privacy, CMS landing, messaging.

---

## 7. Frontend – `gymmaster-web`

### 7.1 Giữ nguyên từ strongbody-web
- **Next.js App Router** (React 19), TypeScript strict, alias `@/*`.
- **Tailwind CSS v4** với token đặt trong `@theme`; **shadcn/ui** + Radix + `cva` + `cn()`.
- **next-auth v4** (Credentials, JWT strategy):
  - lưu `accessToken`/`refreshToken` trong cookie mã hóa, không dùng localStorage;
  - refresh trong `jwt` callback;
  - `RefreshTokenGuard` tự logout khi refresh lỗi.
- `proxy.ts` (middleware) chặn route theo session; server guard dùng `getCachedSession()`.
- **react-hook-form + zod**, **zustand** (UI state), **sonner** (toast), **dayjs**, **Heroicons**.
- **next-intl**: chỉ dùng locale `vi`. Chuỗi giao diện vẫn tách theo domain `messages/vi/<domain>.json`, không hard-code trong component (giữ khả năng mở rộng ngôn ngữ sau này), không có prefix locale trên URL.
- DESIGN_SYSTEM (rút gọn):
  - 1 font (Be Vietnam Pro để hỗ trợ tiếng Việt), type scale semantic;
  - màu token `brand-*`, `success`/`warning`/`danger`/`info`;
  - nền xám + card trắng; cấm arbitrary value;
  - mọi view có đủ trạng thái loading skeleton, empty, error;
  - mọi thao tác xóa phải có confirm dialog.
- Prettier config giống hệt; ESLint flat config.

### 7.2 Cải tiến
| Vấn đề ở repo gốc | GymMaster làm |
|---|---|
| Không có HTTP client tập trung, token truyền tay | `lib/http.ts`: `apiClient` tự gắn Bearer từ session, parse envelope `{code,message,data}`, throw `ApiError`, xử lý 401 → refresh/logout |
| Trộn SWR, React Query và state thô | Chỉ dùng **TanStack Query v5** (query key factory theo module) |
| Không có thư viện bảng/biểu đồ | **TanStack Table** cho DataTable dùng chung (sort, filter, paginate server-side); **Recharts** cho dashboard |
| Dockerfile Node lệch phiên bản | Node 22, `output: 'standalone'` |
| Secret hard-code trong config | Chỉ đọc env, validate bằng zod khi build (`src/env.ts`) |

Thêm cho nghiệp vụ gym:
- `html5-qrcode`: quét QR bằng webcam ở quầy lễ tân.
- `qrcode.react`: hiển thị QR động cho hội viên.
- **FullCalendar** (hoặc `react-big-calendar`): lịch lớp và lịch PT dạng tuần.

### 7.3 Cấu trúc thư mục
```
gymmaster-web/
├── messages/vi/{common,auth,member,subscription,class,report,...}.json
├── src/
│   ├── proxy.ts                # auth gating
│   ├── env.ts                  # zod-validated env
│   ├── app/
│   │   ├── (auth)/login, forgot-password, reset-password
│   │   ├── (public)/           # landing: gói tập, lịch lớp, chi nhánh
│   │   ├── (admin)/admin/      # layout: Sidebar + Topbar + BranchSwitcher
│   │   │   ├── dashboard/
│   │   │   ├── members/  [id]/ (tabs: hồ sơ, gói, check-in, chỉ số, hóa đơn, timeline)
│   │   │   ├── subscriptions/  sell/ (wizard 3 bước)
│   │   │   ├── checkin/        # màn hình quầy: camera quét QR + kết quả lớn xanh/đỏ
│   │   │   ├── classes/  schedule/ (calendar)
│   │   │   ├── pt/  contracts/  sessions/
│   │   │   ├── finance/ invoices/ payments/ promotions/
│   │   │   ├── staff/ trainers/ shifts/
│   │   │   ├── plans/
│   │   │   ├── reports/
│   │   │   └── settings/ branches/ rooms/ roles/
│   │   ├── (trainer)/trainer/  # lịch dạy, học viên, hoa hồng
│   │   └── (member)/me/        # QR, gói của tôi, đặt lớp, lịch PT, tiến độ, hóa đơn
│   ├── app/api/auth/[...nextauth]/
│   ├── features/<module>/      # api.ts (gọi apiClient) · queries.ts (useXQuery/useXMutation) · schema.ts (zod) · types.ts · components/
│   ├── components/ui/          # shadcn primitives
│   ├── components/common/      # DataTable, PageHeader, ConfirmDialog, EmptyState, LoadError, StatCard, PermissionGate
│   ├── components/providers/
│   ├── guards/                 # authOptions.ts, getCachedSession.ts, RequirePermission
│   ├── lib/                    # http.ts, utils.ts(cn), format.ts (tiền VND, ngày)
│   ├── hooks/  store/  i18n/  styles/  types/
```
Điểm khác so với repo gốc: tổ chức theo **feature folder** (`features/<module>/`) thay vì tách rời `src/api` và `src/components`, để code của một module nằm cùng chỗ.

### 7.4 Phân quyền trên UI
- Session có `permissions: string[]`.
- `<PermissionGate perm="member.delete">` ẩn/hiện nút; sidebar lọc menu theo permission.
- `proxy.ts` chặn `/admin/*` nếu không phải staff và `/me/*` nếu không phải member.
- Backend vẫn là nơi kiểm tra cuối cùng; UI chỉ cải thiện trải nghiệm.

---

## 8. Hạ tầng & DevOps
- `docker-compose.yml` (thư mục gốc): `postgres:16`, `redis:7`, `minio`, `mailpit` (bắt email dev), `api`, `web`.
- `make dev` chạy hot reload backend bằng `air`; `npm run dev` cho frontend.
- `make migrate-up` / `migrate-down` / `migrate-create name=...`; `make seed` tạo dữ liệu demo.
- GitHub Actions:
  - backend: `golangci-lint` → `go test ./...` (service Postgres) → build image;
  - frontend: `tsc --noEmit` → `eslint` → `build`.
- Deploy demo: 1 VPS chạy docker compose + Caddy (HTTPS tự động), hoặc Render/Fly.io + Vercel.

---

## 9. Kiểm thử
| Loại | Công cụ | Phạm vi |
|---|---|---|
| Unit (BE) | testify + mock repo | Toàn bộ service nghiệp vụ: bán gói, nối gói, freeze, check-in rules, booking/waitlist, commission |
| Integration (BE) | Postgres thật (docker) | Repository, transaction, EXCLUDE constraint, test đồng thời đặt lớp |
| API | Postman/Newman collection | Kịch bản end-to-end chạy trong CI |
| Unit (FE) | Vitest + Testing Library | zod schema, format helper, component quan trọng |
| E2E | Playwright | 5 luồng chính: login → bán gói → check-in → đặt lớp → xem báo cáo |
| Hiệu năng | k6 | Check-in 100 req/s, list hội viên 10k bản ghi |

Bảng test case (ID, mô tả, input, expected, kết quả) sẽ được đưa vào chương Kiểm thử của báo cáo.

---

## 10. Lộ trình (16 tuần)

| Tuần | Giai đoạn | Đầu ra |
|---|---|---|
| 1 | Khảo sát & phân tích | Đặc tả yêu cầu, use case tổng quát và chi tiết, quy tắc nghiệp vụ |
| 2 | Thiết kế | ERD, sequence diagram (bán gói, check-in, đặt lớp), wireframe Figma, chốt API contract |
| 3 | Khởi tạo dự án | Scaffold BE + FE, docker-compose, CI, migrations init, seed RBAC; BE: config/logger/response/authn/authz/pagination |
| 4 | Auth & RBAC | Login/refresh/logout/quên mật khẩu, quản lý role-permission; FE: login, layout admin, guard, apiClient |
| 5 | Chi nhánh, nhân sự, HLV | CRUD branch/room/staff/trainer/shift; DataTable dùng chung |
| 6 | Hội viên | CRUD, upload ảnh, import Excel, hồ sơ chi tiết, chỉ số cơ thể |
| 7–8 | Gói tập & Đăng ký | Plans, bán gói (wizard), invoice/payment, nối gói, gia hạn, nâng cấp, bảo lưu, chuyển nhượng, hủy, voucher |
| 9 | Check-in | QR động, màn hình quét quầy, rule engine, log, checkout |
| 10 | Lớp nhóm | Class type, lịch lặp tuần, calendar, đặt chỗ, waitlist, điểm danh, cron no-show |
| 11 | PT | Hợp đồng PT, đặt lịch, chống trùng, hoàn thành buổi, hoa hồng |
| 12 | Cổng hội viên | QR, gói, đặt lớp/PT, lịch sử, biểu đồ tiến độ, xem hóa đơn |
| 13 | Báo cáo & thông báo | Dashboard, các báo cáo, xuất Excel, cron nhắc hạn, email, thông báo in-app |
| 14 | Mở rộng | POS bán hàng + tồn kho theo chi nhánh (sản phẩm, nhập/xuất/điều chuyển/kiểm kê, cảnh báo tồn thấp), audit log |
| 15 | Kiểm thử & tối ưu | Test, sửa lỗi, k6, index, deploy demo |
| 16 | Báo cáo & bảo vệ | Hoàn thiện quyển, slide, video demo, dữ liệu demo đẹp |

> Lộ trình 16 tuần trên là phạm vi đồ án (giai đoạn 1, đã hoàn thành Sprint 01–11). Sprint 12–30 là các giai đoạn tiếp theo (§12–§15), phạm vi xem §16.1.

---

## 11. Gợi ý cấu trúc quyển đồ án
1. **Mở đầu**: lý do chọn đề tài, mục tiêu, phạm vi, phương pháp.
2. **Cơ sở lý thuyết & công nghệ**: Go/GoFrame, PostgreSQL (transaction, EXCLUDE constraint, isolation), JWT/RBAC, React/Next.js (SSR, App Router), REST, Docker.
3. **Khảo sát & phân tích yêu cầu**: khảo sát quy trình gym thực tế, tác nhân, use case, đặc tả use case, yêu cầu phi chức năng.
4. **Thiết kế hệ thống**: kiến trúc tổng thể, kiến trúc phân lớp BE, FE, ERD, sequence/activity diagram, thiết kế API, thiết kế giao diện, bảo mật.
5. **Cài đặt**: môi trường, cấu trúc mã nguồn, các module tiêu biểu (kèm code trích dẫn), giải pháp cho các bài toán khó (trùng lịch, race condition đặt chỗ, QR chống gian lận, pro-rata).
6. **Kiểm thử & đánh giá**: test case, kết quả, hiệu năng.
7. **Kết luận & hướng phát triển**: mobile app, thiết bị cửa từ/vân tay, AI gợi ý giáo án, multi-tenant SaaS.

---

## 12. Giai đoạn 2 – Lộ trình phát triển tiếp (Sprint 12–17)

> Mỗi sprint ≈ 1 tuần. Tuân thủ pattern hiện có: `api/<module>/v1` → `controller` → `service` → `repository`, wiring trong `internal/module/<m>.go`. Migration đánh số tiếp từ `000011`, permission mới khai báo trong `consts/permission.go` + seed. Mỗi sprint xong: tick README, viết `docs/api/<module>.md`.

### Sprint 12 – Quên mật khẩu & chống brute-force
**DB – `000011_password_reset`**
- `password_resets` (id, user_id, token_hash, expires_at, used_at, ip, created_at) – token lưu SHA-256, TTL 30 phút, dùng 1 lần.
- `login_attempts` (email, ip, attempted_at, success) – đếm đăng nhập sai.

**API**
- `POST /v1/auth/forgot-password`: luôn trả 200 (không lộ email tồn tại), gửi mail link `web/reset-password?token=...`.
- `POST /v1/auth/reset-password`: đổi mật khẩu + thu hồi toàn bộ `refresh_tokens` của user.
- Rate limit: sai 5 lần/15 phút theo email+IP → khóa tạm, mã `40105`. Cài bằng DB qua interface `Limiter`; có thể thay bằng Redis (`go-redis`) sau.
- Dọn `password_resets` hết hạn gộp vào job `cleanup_tokens`.

**FE**: web `/forgot-password`, `/reset-password`, link "Quên mật khẩu?" ở `/login`; admin tương tự cho nhân viên.

**Test**: token hết hạn / đã dùng, vượt giới hạn đăng nhập sai, refresh token bị thu hồi sau reset.

### Sprint 13 – Thanh toán online VNPay sandbox (ưu tiên cao)
**DB – `000012_online_payment`**
- Bổ sung `'VNPAY'` vào CHECK `payments.method` (và `refunds.method` nếu cần).
- `online_payment_txns` (id, invoice_id, member_id, txn_ref UNIQUE, amount, status[`PENDING`|`SUCCESS`|`FAILED`|`EXPIRED`], provider_txn_no, bank_code, raw_ipn jsonb, created_at, paid_at).

**Package** `internal/pkg/vnpay`: build URL (ký HMAC-SHA512), verify chữ ký. Config `VNPAY_TMN_CODE`, `VNPAY_HASH_SECRET`, `VNPAY_RETURN_URL` thêm vào `.env.example`.

**API**
- `POST /v1/member/invoices/{id}/vnpay` → tạo txn, trả `payment_url`.
- `POST /v1/member/subscriptions/checkout`: hội viên tự chọn gói → tái dùng `SubscriptionService.Quote/Create` (invoice `UNPAID`, sub `PENDING`) → trả `payment_url`.
- `GET /v1/public/payments/vnpay/ipn`: verify chữ ký + số tiền, idempotent theo `txn_ref`; trong 1 transaction: ghi `payments` → kích hoạt subscription → thông báo.
- `GET /v1/public/payments/vnpay/return`: chỉ hiển thị kết quả, **không** cập nhật trạng thái.
- Cron `expire_online_txns` (mỗi 15 phút): txn `PENDING` quá 15 phút → `EXPIRED`.

**FE web**: nút "Thanh toán VNPay" ở `/me/invoices/[id]`; trang `/me/subscriptions/buy` (chọn gói, áp voucher qua `/promotions/check`, thanh toán); trang `/payment/result`.
**FE admin**: hiển thị phương thức VNPAY, lọc giao dịch online.

**Test**: IPN sai chữ ký, sai số tiền, IPN gọi trùng (idempotency), thanh toán sau khi txn hết hạn.

### Sprint 14 – Giáo án & nhật ký tập
**DB – `000013_workout`**
- `exercises` (id, name, muscle_group, equipment, video_url, description) – thư viện bài tập, seed ~40 bài.
- `workout_plans` (id, member_id, trainer_id, pt_contract_id NULL, name, goal, start_date, end_date, status).
- `workout_plan_days` (id, plan_id, day_index, title).
- `workout_plan_items` (id, day_id, exercise_id, sets, reps, weight, rest_sec, note, sort).
- `workout_logs` (id, member_id, plan_item_id NULL, exercise_id, performed_at, sets, reps, weight) – hội viên tự ghi.

**API**
- Admin/HLV: CRUD `/admin/exercises`, `/admin/workout-plans`, `POST /admin/workout-plans/{id}/clone` (giáo án mẫu). Permission `workout.manage`; HLV chỉ thao tác trên học viên của mình.
- Member: `GET /member/workout-plans`, `POST /member/workout-logs`, `GET /member/workout-logs?exercise_id=` (biểu đồ tiến bộ mức tạ).

**FE**: admin – màn soạn giáo án theo ngày, chọn bài từ thư viện; web – `/me/workout` (giáo án hôm nay, ghi set) + biểu đồ mức tạ trong `/me/progress`.

### Sprint 15 – Đánh giá HLV/lớp & Quản lý thiết bị
**Đánh giá – `000014_reviews`**
- `reviews` (id, member_id, target_type[`PT_SESSION`|`CLASS_SCHEDULE`], target_id, trainer_id, rating 1–5, comment, created_at), UNIQUE(member_id, target_type, target_id).
- Chỉ đánh giá buổi đã `COMPLETED`/`ATTENDED`, trong vòng 7 ngày.
- Thêm `trainer_profiles.rating_avg`, `rating_count`, cập nhật trong cùng transaction khi tạo review.
- API: `POST /member/reviews`, `GET /admin/reviews`; báo cáo `trainer-performance` thêm cột rating.
- Gửi thông báo "Đánh giá buổi tập" sau khi buổi hoàn thành.

**Thiết bị – `000015_equipment`**
- `equipments` (id, branch_id, room_id, code, name, category, purchase_date, price, warranty_until, status[`ACTIVE`|`BROKEN`|`MAINTENANCE`|`RETIRED`], maintenance_interval_days, next_maintenance_date, image_url).
- `maintenance_logs` (id, equipment_id, type[`PERIODIC`|`REPAIR`], reported_by, description, cost, status[`OPEN`|`IN_PROGRESS`|`DONE`], started_at, finished_at).
- API: CRUD `/admin/equipments`, `POST /admin/equipments/{id}/report-issue`, CRUD `/admin/maintenance-logs`.
- Cron `remind_maintenance` (07:00): thiết bị đến hạn → `NotifyPermission("equipment.manage", branch)`.
- Báo cáo chi phí bảo trì theo chi nhánh/tháng, thêm vào `/reports/export`.

### Sprint 16 – CRM khách tiềm năng & chăm sóc hội viên
**DB – `000016_crm`**
- `leads` (id, branch_id, full_name, phone, email, source[`WEBSITE`|`WALK_IN`|`REFERRAL`|`FACEBOOK`], interest_plan_id, status[`NEW`|`CONTACTED`|`TRIAL`|`CONVERTED`|`LOST`], assigned_to, trial_date, member_id NULL, note).
- `lead_activities` (id, lead_id, type[`CALL`|`SMS`|`VISIT`|`NOTE`], content, created_by, created_at).
- `trial_passes` (id, lead_id, branch_id, valid_date, qr_token, used_at) – màn quét check-in chấp nhận thêm vé tập thử.

**API**
- `POST /v1/public/leads`: form "Đăng ký tập thử" trên landing (rate limit theo IP).
- Admin: CRUD `/admin/leads`, `POST /admin/leads/{id}/activities`, `/trial-pass`, `/convert` (tạo member → mở màn bán gói).
- Báo cáo `/reports/leads`: phễu chuyển đổi, tỷ lệ theo nguồn.
- `GET /v1/public/class-schedules`: lịch lớp công khai cho landing.

**Chăm sóc tự động (cron)**
- `birthday_greetings` (08:00): email/thông báo chúc mừng, kèm voucher sinh nhật sinh tự động qua promotion service.
- `churn_risk_scan` (06:00): hội viên có gói ACTIVE nhưng 14 ngày không check-in → danh sách `GET /admin/members/at-risk`.

**FE**: landing – section + form tập thử, lịch lớp công khai; admin – Kanban lead theo trạng thái, tab "Hội viên cần chăm sóc".

### Sprint 17 – Realtime, PWA & hoàn thiện
- **SSE** `GET /v1/notifications/stream` (auth bằng access token): hub in-process `internal/pkg/realtime`, push sau `Notifier.Notify`. Admin nhận check-in mới, yêu cầu bảo lưu; hội viên nhận "lên từ waitlist", "thanh toán thành công". FE thay polling `unread-count` bằng `EventSource`, fallback polling khi lỗi.
- **PWA** cho `gymmaster-web`: manifest, service worker, icon, mở QR offline từ cache; Web Push (tùy chọn).
- **Hội viên tự tạo tài khoản**: `POST /v1/auth/register-member` (SĐT/email); trùng SĐT với `members` có sẵn → xác thực OTP email để liên kết.
- **Hoàn thiện**: cập nhật Swagger, `docs/api/*`, ERD; seed demo cho module mới; CI chạy integration test; kịch bản demo end-to-end: *khách đăng ký tập thử → convert → mua gói VNPay → check-in → tập theo giáo án → đánh giá HLV*.

### Tổng hợp
| Sprint | Nội dung | Migration | Độ khó | Giá trị demo |
|---|---|---|---|---|
| 12 | Quên mật khẩu, rate limit | 000011 | Thấp | Trung bình |
| 13 | VNPay sandbox, tự mua gói | 000012 | Cao | **Rất cao** |
| 14 | Giáo án, nhật ký tập | 000013 | Trung bình | Cao |
| 15 | Đánh giá HLV, thiết bị & bảo trì | 000014–15 | Trung bình | Cao |
| 16 | CRM lead, tập thử, chăm sóc tự động | 000016 | Trung bình | Cao |
| 17 | SSE realtime, PWA, tự đăng ký, hoàn thiện | — | Trung bình | Trung bình |

Nếu thời gian gấp, bản rút gọn: Sprint 12 → 13 → 14 → phần đánh giá của 15 (đủ vòng hội viên tự phục vụ).

---

## 13. Giai đoạn 3 – Mở rộng vận hành, tài chính & tương tác (Sprint 18–23)

> Tiếp nối Giai đoạn 2, cùng quy ước: migration đánh số tiếp từ `000017`, permission mới khai báo trong `consts/permission.go` + seed, cron đăng ký qua `cron.Register` trong `internal/module/<m>.go`.

### A. Tự động hóa vận hành tại quầy

#### Sprint 18 – Kiểm soát ra vào bằng thiết bị & Kiosk
**DB – `000017_access_devices`**
- `access_devices` (id, branch_id, code, name, type[`TURNSTILE`|`DOOR`|`KIOSK`], api_key_hash, last_seen_at, status[`ACTIVE`|`OFFLINE`|`DISABLED`]).
- Thêm `checkins.device_id` NULL và giá trị `DEVICE` vào CHECK `checkins.method`.

**API**
- Admin: CRUD `/admin/access-devices`, `POST /admin/access-devices/{id}/rotate-key` (chỉ hiện key 1 lần). Permission `device.manage`.
- Thiết bị: nhóm `/v1/device/*`, xác thực bằng header `X-Device-Key` (middleware riêng, so sánh hash):
  - `POST /v1/device/checkin` – nhận QR token → tái dùng rule engine `CheckinService` → trả `{allow, member_name, message}` để mở cửa.
  - `POST /v1/device/heartbeat` – cập nhật `last_seen_at`.
- Cron `device_heartbeat_check` (mỗi 5 phút): thiết bị quá 10 phút không heartbeat → `OFFLINE` + `NotifyPermission("device.manage", branch)`.

**FE**
- Admin: màn quản lý thiết bị (trạng thái online/offline, lần cuối hoạt động).
- Web: trang `/kiosk` toàn màn hình cho tablet đặt tại cửa – quét QR bằng camera (`html5-qrcode` đã có), hiển thị kết quả lớn (xanh/đỏ + tên hội viên), tự reset sau 3 giây. Kiosk đăng nhập bằng device key, không dùng tài khoản người.

**Test**: key sai/bị vô hiệu, rule check-in qua thiết bị giống check-in tại quầy, heartbeat timeout.

#### Sprint 19 – Tủ đồ & dịch vụ theo lượt
**DB – `000018_lockers`**
- `lockers` (id, branch_id, code, zone, size, status[`AVAILABLE`|`OCCUPIED`|`RENTED`|`BROKEN`]).
- `locker_rentals` (id, locker_id, member_id, type[`DAILY`|`MONTHLY`], start_at, end_at, invoice_id NULL, status[`ACTIVE`|`ENDED`|`EXPIRED`]).

**Nghiệp vụ**
- Tủ theo lượt: gán khi check-in (tự động hoặc lễ tân chọn), tự giải phóng khi checkout hoặc cuối ngày.
- Tủ thuê tháng: bán như một dòng `invoice_items` (`item_type = 'LOCKER'`), cron `expire_locker_rentals` (00:20) hết hạn → giải phóng + thông báo hội viên trước 3 ngày.
- Khăn/nước theo lượt bán qua POS hiện có (sản phẩm loại dịch vụ, không trừ tồn kho).

**API**: CRUD `/admin/lockers`, `GET /admin/lockers/board?branch_id=` (sơ đồ tủ), `POST /admin/lockers/{id}/assign`, `/release`, `POST /admin/locker-rentals`. Portal: `GET /member/locker`.

**FE**: admin – sơ đồ tủ dạng lưới theo khu, màu theo trạng thái; tích hợp chọn tủ vào màn check-in. Web – hiển thị số tủ đang dùng ở `/me`.

### B. Tài chính & nhân sự nâng cao

#### Sprint 20 – Chấm công & bảng lương
**DB – `000019_payroll`**
- `staff_attendances` (id, staff_id, branch_id, shift_id NULL, check_in_at, check_out_at, late_minutes, early_minutes, method[`QR`|`MANUAL`], note).
- `payrolls` (id, branch_id, period `YYYY-MM`, status[`DRAFT`|`LOCKED`|`PAID`], locked_by, locked_at). UNIQUE(branch_id, period).
- `payroll_items` (id, payroll_id, staff_id, base_salary, work_days, ot_hours, pt_commission, sales_commission, bonus, penalty, total, note).

**Nghiệp vụ**
- Nhân viên chấm công bằng QR cá nhân tại quầy/kiosk; đối chiếu `shifts` để tính đi muộn/về sớm (ngưỡng cấu hình trong `settings`).
- Tạo bảng lương tháng: lương cơ bản theo ngày công + hoa hồng PT (`trainer_commissions` của kỳ) + hoa hồng bán gói (theo `subscriptions.sold_by` × tỷ lệ cấu hình) + thưởng/phạt nhập tay.
- `LOCKED` thì không sửa được; `PAID` đánh dấu luôn các `trainer_commissions` liên quan đã trả.

**API**: `POST /admin/attendances/scan`, `GET /admin/attendances`, `POST /admin/payrolls/generate`, `GET/PUT /admin/payrolls/{id}`, `POST /admin/payrolls/{id}/lock`, `/pay`, `GET /admin/payrolls/{id}/export` (Excel). Nhân viên xem phiếu lương của mình: `GET /v1/staff/me/payslips`.

**Test**: tính công với ca qua đêm, bảng lương đã khóa không sửa được, không trả trùng hoa hồng.

#### Sprint 21 – Chi phí, P&L, đối soát ca & trả góp
**DB – `000020_finance_ext`**
- `expense_categories` (id, name, type[`FIXED`|`VARIABLE`]); `expenses` (id, branch_id, category_id, amount, spent_at, description, attachment_url, created_by, approved_by, status).
- `cash_sessions` (id, branch_id, staff_id, opened_at, closed_at, opening_cash, expected_cash, counted_cash, difference, note, status[`OPEN`|`CLOSED`]).
- `installment_plans` (id, invoice_id, total, installments) + `installment_schedules` (id, plan_id, seq, due_date, amount, paid_amount, status[`PENDING`|`PAID`|`OVERDUE`]).

**Nghiệp vụ**
- **Chi phí**: nhập chi phí theo chi nhánh, manager duyệt; chi phí bảo trì thiết bị (Sprint 15) tự đẩy sang `expenses`.
- **Báo cáo lãi/lỗ (P&L)**: doanh thu (gói, PT, POS, tủ đồ) – giá vốn POS – chi phí – lương, theo chi nhánh/tháng; xuất Excel.
- **Đối soát ca**: lễ tân mở ca (tiền đầu ca) → đóng ca nhập tiền đếm thực tế → hệ thống so với tổng `payments` CASH trong ca, ghi chênh lệch.
- **Trả góp/công nợ**: gói dài hạn cho chia N kỳ; subscription `ACTIVE` khi đã trả kỳ 1; cron `remind_installments` (08:00) nhắc trước 3 ngày, quá hạn 7 ngày → `OVERDUE` và chặn check-in (thêm rule vào rule engine, `deny_reason = INSTALLMENT_OVERDUE`).
- **Hóa đơn điện tử**: định nghĩa interface `EInvoiceProvider` + bản mock; tích hợp nhà cung cấp thật (VNPT/Viettel sandbox) để sau.

**API**: CRUD `/admin/expenses`, `/admin/expense-categories`; `POST /admin/cash-sessions/open`, `/{id}/close`, `GET /admin/cash-sessions`; `GET /admin/reports/profit-loss`; `POST /admin/invoices/{id}/installments`, `GET /admin/installments?status=OVERDUE`. Portal: hiển thị lịch trả góp trong `/me/invoices/[id]`.

**Test**: chênh lệch đối soát ca, P&L khớp với tổng báo cáo doanh thu hiện có, chặn check-in khi quá hạn trả góp.

### C. Tăng tương tác hội viên

#### Sprint 22 – Điểm thưởng, huy hiệu, thử thách, giới thiệu bạn bè
**DB – `000021_loyalty`**
- `loyalty_ledgers` (id, member_id, points (+/–), reason[`CHECKIN`|`RENEW`|`REFERRAL`|`REVIEW`|`REDEEM`|`ADJUST`], ref_type, ref_id, created_at); số dư lưu ở `members.loyalty_points` (cập nhật trong cùng transaction).
- `loyalty_rewards` (id, name, cost_points, reward_type[`VOUCHER`|`PRODUCT`], ref_id, stock, status).
- `badges` (id, code, name, icon, rule jsonb) + `member_badges` (member_id, badge_id, earned_at).
- `challenges` (id, branch_id NULL, name, metric[`CHECKINS`|`CLASSES`|`PT_SESSIONS`], target, start_date, end_date, reward_points) + `challenge_participants` (challenge_id, member_id, progress, completed_at).
- Thêm `members.referral_code` UNIQUE, `members.referred_by` NULL.

**Nghiệp vụ**
- Cộng điểm qua hook sau check-in hợp lệ (tối đa 1 lần/ngày), gia hạn gói, đánh giá HLV, giới thiệu thành công (khi người được giới thiệu thanh toán gói đầu tiên).
- Đổi điểm lấy voucher (sinh `promotions` mã riêng 1 lần dùng) hoặc sản phẩm (nhận tại quầy qua POS).
- Cron `evaluate_badges_challenges` (01:00): xét huy hiệu (chuỗi 30 ngày, 100 buổi…) và cập nhật tiến độ thử thách; bảng xếp hạng theo chi nhánh.
- Cấu hình tỷ lệ điểm trong `settings`.

**API**: admin – CRUD `/admin/loyalty-rewards`, `/admin/badges`, `/admin/challenges`, `POST /admin/members/{id}/points/adjust`. Portal – `GET /member/loyalty` (số dư, lịch sử), `POST /member/loyalty/redeem`, `GET /member/badges`, `GET/POST /member/challenges/{id}/join`, `GET /member/leaderboard`, `GET /member/referral`.

**FE**: web – trang `/me/rewards` (điểm, đổi quà, mã giới thiệu có nút chia sẻ), `/me/challenges` (tiến độ, bảng xếp hạng), huy hiệu trên `/me`. Landing nhận `?ref=CODE` → lưu vào form tập thử (Sprint 16).

**Test**: không cộng điểm check-in 2 lần/ngày, đổi quà khi thiếu điểm/hết quà, điểm giới thiệu chỉ cộng 1 lần.

#### Sprint 23 – Dinh dưỡng, bảng tin & khảo sát NPS
**DB – `000022_engagement`**
- `foods` (id, name, unit, calories, protein, carbs, fat) – seed món ăn Việt phổ biến.
- `meal_plans` (id, member_id, trainer_id, name, target_calories, start_date, end_date) + `meal_plan_items` (id, plan_id, day_index, meal[`BREAKFAST`|`LUNCH`|`DINNER`|`SNACK`], food_id, qty).
- `food_logs` (id, member_id, food_id, qty, meal, eaten_at).
- `announcements` (id, branch_id NULL, title, body, image_url, publish_at, expire_at, pinned, created_by).
- `surveys` (id, title, type[`NPS`|`CUSTOM`], start_at, end_at, status) + `survey_responses` (id, survey_id, member_id, score, answers jsonb, created_at).

**Nghiệp vụ**
- Gợi ý calo mục tiêu từ `body_metrics` (BMR Mifflin–St Jeor × hệ số vận động, điều chỉnh theo mục tiêu giảm/tăng cân); HLV giao thực đơn, hội viên ghi bữa ăn và xem tổng calo/macro theo ngày.
- Bảng tin theo chi nhánh (lịch nghỉ lễ, sự kiện), ghim tin quan trọng, gửi thông báo khi đăng.
- Khảo sát NPS định kỳ: cron `send_nps_survey` gửi cho hội viên đủ 30 ngày tập; báo cáo NPS theo chi nhánh, điểm < 7 tạo việc chăm sóc cho lễ tân (danh sách at-risk Sprint 16).

**API**: admin – CRUD `/admin/foods`, `/admin/meal-plans`, `/admin/announcements`, `/admin/surveys`, `GET /admin/reports/nps`. Portal – `GET /member/meal-plans`, `POST/GET /member/food-logs`, `GET /member/announcements`, `POST /member/surveys/{id}/responses`.

**FE**: web – `/me/nutrition` (thực đơn, ghi bữa ăn, vòng calo/macro), bảng tin trên `/me`, popup khảo sát NPS; admin – soạn thực đơn, quản lý bảng tin, dashboard NPS.

### Tổng hợp Giai đoạn 3
| Nhóm | Sprint | Nội dung | Migration | Độ khó |
|---|---|---|---|---|
| A | 18 | Thiết bị ra vào, kiosk tự check-in | 000017 | Trung bình |
| A | 19 | Tủ đồ, dịch vụ theo lượt | 000018 | Thấp–Trung bình |
| B | 20 | Chấm công, bảng lương | 000019 | Trung bình |
| B | 21 | Chi phí, P&L, đối soát ca, trả góp | 000020 | Cao |
| C | 22 | Điểm thưởng, huy hiệu, thử thách, giới thiệu | 000021 | Trung bình |
| C | 23 | Dinh dưỡng, bảng tin, NPS | 000022 | Trung bình |

Thứ tự ưu tiên gợi ý: **18 (kiosk) → 20 (lương) → 22 (điểm thưởng) → 21 → 19 → 23**.

---

## 14. Giai đoạn 4 – Hoàn thiện nghiệp vụ, bảo mật & chất lượng (Sprint 24–28) + Backlog

> Cùng quy ước các giai đoạn trước; migration đánh số tiếp từ `000023`. Phạm vi **không** gồm mobile app, tích hợp bên ngoài, AI, SaaS multi-tenant.
> Ký hiệu ưu tiên: ⭐⭐⭐ nên làm · ⭐⭐ hay · ⭐ thêm nếu còn thời gian. Các sprint 24–28 lấy từ các mục ⭐⭐⭐.

### Sprint 24 – Hợp đồng điện tử, vé ngày & gói off-peak
**DB – `000023_contracts_passes`**
- `contract_templates` (id, code, name, body_html, version, status) – mẫu hợp đồng/cam kết miễn trừ, có biến `{{member_name}}`, `{{plan_name}}`…
- `member_contracts` (id, member_id, template_id, template_version, subscription_id NULL, signed_at, signature_url, pdf_url, ip, user_agent).
- `day_passes` (id, branch_id, buyer_name, phone, member_id NULL, sponsor_member_id NULL, type[`DAY`|`GUEST`], valid_date, qr_token, invoice_id NULL, used_at).
- Thêm `membership_plans.access_hours` jsonb (vd `[{"dow":[1,2,3,4,5],"from":"09:00","to":"16:00"}]`) và `membership_plans.guest_passes_per_month`.

**Nghiệp vụ**
- Hội viên ký hợp đồng trên portal (canvas chữ ký) → render PDF lưu vào upload; cấu hình `settings` bắt buộc ký trước lần check-in đầu (rule engine: `deny_reason = CONTRACT_UNSIGNED`).
- Vé ngày bán tại quầy/POS, QR dùng 1 lần trong ngày; hội viên gói có `guest_passes_per_month` tự tạo vé khách trên portal (giới hạn theo tháng).
- Gói off-peak: rule engine check-in kiểm tra `access_hours` (`deny_reason = OUTSIDE_ACCESS_HOURS`).

**API**: CRUD `/admin/contract-templates`, `GET /admin/members/{id}/contracts`; `POST /admin/day-passes`, `GET /admin/day-passes`; portal – `GET /member/contracts/pending`, `POST /member/contracts/{id}/sign`, `POST /member/guest-passes`, `GET /member/guest-passes`. Màn quét check-in nhận thêm QR vé ngày.

**FE**: admin – soạn mẫu hợp đồng, bán vé ngày, cấu hình khung giờ trong form gói tập; web – trang ký hợp đồng, `/me/guest-passes`.

**Test**: check-in ngoài khung giờ, vé ngày dùng lại/sai ngày, vượt số vé khách tháng, chưa ký hợp đồng.

### Sprint 25 – Lịch rảnh HLV, đơn nghỉ & HLV thay thế, lịch ngày lễ
**DB – `000024_availability_holidays`**
- `trainer_availabilities` (id, trainer_id, branch_id, dow, start_time, end_time, effective_from, effective_to).
- `staff_leave_requests` (id, staff_id, from_at, to_at, reason, status[`PENDING`|`APPROVED`|`REJECTED`], approved_by).
- `class_schedules.substitute_trainer_id` NULL; `pt_sessions` thêm `original_trainer_id` NULL.
- `branch_holidays` (id, branch_id NULL, date, name, closed bool, open_time NULL, close_time NULL).

**Nghiệp vụ**
- `trainer-availability` (portal + admin) chỉ trả khung giờ nằm trong `trainer_availabilities`, trừ buổi đã đặt và thời gian nghỉ phép đã duyệt.
- Duyệt đơn nghỉ → liệt kê lớp/buổi PT bị ảnh hưởng → quản lý gán HLV thay (kiểm tra trùng lịch bằng EXCLUDE sẵn có) hoặc hủy; tự thông báo hội viên liên quan.
- Ngày lễ: chặn tạo/đặt lớp, buổi PT và check-in ngoài giờ mở đặc biệt; khi thêm ngày nghỉ, cảnh báo các lịch đã tồn tại. Tùy chọn cộng bù ngày cho gói `TIME` khi đóng cửa cả ngày.

**API**: `GET/PUT /admin/trainers/{id}/availability`, CRUD `/admin/leave-requests` + `/approve`, `/reject`, `GET /admin/leave-requests/{id}/impacts`, `POST /admin/class-schedules/{id}/substitute`, `POST /admin/pt-sessions/{id}/reassign`; CRUD `/admin/branch-holidays`; public – `GET /v1/public/branches/{id}/holidays`.

**FE**: admin – lịch rảnh dạng lưới tuần, màn duyệt nghỉ kèm danh sách ảnh hưởng, lịch ngày lễ; web – hiển thị ngày nghỉ trên lịch lớp và landing.

**Test**: đặt PT ngoài lịch rảnh, HLV thay bị trùng lịch, check-in ngày nghỉ, cộng bù ngày gói.

### Sprint 26 – Mật độ phòng realtime, KPI doanh số, cohort retention
**DB – `000025_kpi`**
- `sales_targets` (id, branch_id, staff_id NULL, period `YYYY-MM`, metric[`REVENUE`|`NEW_MEMBERS`|`RENEWALS`|`PT_SALES`], target_value).
- Thêm `branches.capacity` (sức chứa an toàn).

**Nghiệp vụ & API**
- Mật độ: `GET /v1/public/branches/occupancy` và `/member/occupancy` → số người đang tập (từ `checkins/present`) / `capacity` → mức `QUIET|NORMAL|BUSY|FULL`; kèm biểu đồ "giờ thường đông" lấy từ heatmap. Cache 60 giây.
- KPI: CRUD `/admin/sales-targets`, `GET /admin/reports/kpi?period=` → thực tế vs mục tiêu theo nhân viên/chi nhánh (tính từ invoices, subscriptions, pt_contracts).
- Cohort: `GET /admin/reports/cohort?from=&to=&branch_id=` → ma trận % hội viên còn gói ACTIVE theo tháng gia nhập × tháng thứ N; xuất Excel.

**FE**: web – chỉ báo mật độ trên `/me` và landing; admin – dashboard KPI (thanh tiến độ), heatmap cohort.

**Test**: cohort với hội viên bảo lưu/chuyển nhượng, KPI cá nhân khớp tổng chi nhánh.

### Sprint 27 – 2FA, quản lý phiên, tuân thủ Nghị định 13/2023
**DB – `000026_security_privacy`**
- `user_mfa` (user_id PK, totp_secret_enc, enabled_at, recovery_codes_hash text[]).
- `refresh_tokens` thêm `device_name`, `last_used_at`.
- `consents` (id, member_id, type[`TERMS`|`MARKETING`|`PHOTO`], granted bool, granted_at, ip).
- `data_requests` (id, member_id, type[`EXPORT`|`DELETE`], status[`PENDING`|`PROCESSING`|`DONE`|`REJECTED`], file_url, handled_by, created_at, done_at).

**Nghiệp vụ & API**
- 2FA TOTP: `POST /auth/mfa/setup` (trả QR otpauth), `/auth/mfa/enable`, `/auth/mfa/disable`; login trả `mfa_required` + token tạm → `POST /auth/mfa/verify`. Bắt buộc với OWNER/MANAGER (cấu hình trong `settings`); có mã khôi phục dùng 1 lần.
- Phiên đăng nhập: `GET /auth/sessions`, `DELETE /auth/sessions/{id}`, `POST /auth/sessions/revoke-others`.
- Quyền riêng tư: hội viên `GET/PUT /member/consents`, `POST /member/data-requests` (xuất JSON/ZIP dữ liệu của mình; yêu cầu xóa → admin duyệt → ẩn danh hóa thay vì xóa cứng để giữ toàn vẹn hóa đơn). Admin: `GET /admin/data-requests`, `/process`.
- Mã hóa `members.id_card` (AES-GCM, khóa từ env) + che SĐT/CCCD trong response nếu thiếu quyền `member.view_sensitive`; mọi lần xem dữ liệu nhạy cảm ghi audit log.

**FE**: admin + web – trang bảo mật tài khoản (bật 2FA, danh sách phiên); web – quản lý đồng ý, yêu cầu xuất/xóa dữ liệu; admin – hàng đợi yêu cầu dữ liệu.

**Test**: login thiếu/sai mã TOTP, mã khôi phục dùng lại, thu hồi phiên, ẩn danh hóa không phá hóa đơn, che dữ liệu theo quyền.

### Sprint 28 – Kiểm thử E2E, tải & API trong CI
- **Playwright** (`e2e/` ở thư mục gốc): 5 luồng chính – đăng nhập → bán gói → check-in → đặt lớp → xem báo cáo; thêm luồng hội viên tự mua gói (Sprint 13). Chạy trên docker compose + seed.
- **k6** (`perf/`): check-in 100 req/s, danh sách 10k hội viên, đặt lớp đồng thời; ghi p95/p99 vào chương Kiểm thử.
- **Postman/Newman**: collection kịch bản API end-to-end, chạy trong GitHub Actions.
- **Vitest** cho `gymmaster-web` và `gymmaster-admin`: schema form, helper định dạng, component quan trọng.
- CI: job e2e chạy khi merge vào nhánh chính, upload report/screenshot làm artifact.

### Tổng hợp Giai đoạn 4
| Sprint | Nội dung | Migration | Độ khó |
|---|---|---|---|
| 24 | Hợp đồng điện tử, vé ngày/khách, gói off-peak | 000023 | Trung bình |
| 25 | Lịch rảnh HLV, đơn nghỉ & HLV thay, ngày lễ | 000024 | Trung bình–Cao |
| 26 | Mật độ realtime, KPI doanh số, cohort retention | 000025 | Trung bình |
| 27 | 2FA, quản lý phiên, Nghị định 13/2023 | 000026 | Cao |
| 28 | Playwright, k6, Newman, Vitest trong CI | — | Trung bình |

### Backlog (bốc dần vào sprint khi còn thời gian)

**Hội viên & CRM**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| Hồ sơ sức khỏe (PAR-Q) | Câu hỏi tiền sử bệnh/chấn thương; cảnh báo cho HLV khi nhận học viên | ⭐⭐ |
| Ảnh tiến độ before/after | Upload ảnh theo mốc, so sánh cạnh nhau trong `/me/progress` | ⭐⭐ |
| Tag & phân khúc hội viên | Gắn tag, bộ lọc động lưu lại | ⭐⭐ |
| Chiến dịch email theo phân khúc | Soạn template, gửi theo phân khúc, thống kê mở/click | ⭐⭐ |
| Gói gia đình / nhóm | Một người trả, nhiều thành viên dùng chung, giới hạn số người | ⭐⭐ |
| Hội viên doanh nghiệp (B2B) | Hợp đồng với công ty, danh sách nhân viên, hóa đơn gộp theo tháng | ⭐ |
| Gộp hội viên trùng | Phát hiện trùng theo SĐT/CCCD, gộp lịch sử | ⭐ |

**Gói tập & giá bán**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| Ví trả trước | Nạp tiền, trừ khi mua POS/vé lẻ/gia hạn, lịch sử giao dịch ví | ⭐⭐ |
| Thẻ quà tặng | Mua tặng, mã kích hoạt, có hạn dùng | ⭐⭐ |
| Gói combo | Gói tập + N buổi PT + tủ đồ, bán 1 lần, tách dòng hóa đơn | ⭐⭐ |
| Tự động gia hạn | Bật auto-renew, cron tạo hóa đơn trước hạn 3 ngày | ⭐ |
| Giá theo thời điểm | Giá mùa/flash sale gắn trực tiếp vào gói | ⭐ |

**Lớp nhóm & PT**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| PT nhóm nhỏ (2–4 người) | Hợp đồng chia sẻ, buổi trừ cho nhiều hội viên, chia hoa hồng | ⭐⭐ |
| Chọn vị trí trong lớp | Sơ đồ phòng (xe spinning, thảm yoga), đặt đúng chỗ | ⭐⭐ |
| Lớp tính phí lẻ (drop-in) | Không có gói lớp vẫn đặt, trả theo buổi | ⭐⭐ |
| Đánh giá thể lực định kỳ | Bộ bài test, chấm điểm, so sánh theo kỳ | ⭐⭐ |
| Danh sách chờ PT | Khung giờ kín thì xếp hàng, có người hủy tự báo | ⭐ |

**Vận hành cơ sở**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| Checklist công việc ca | Checklist mở/đóng cửa, vệ sinh theo khu; quản lý xem tiến độ | ⭐⭐ |
| Báo cáo sự cố / chấn thương | Ghi sự cố, người xử lý, ảnh đính kèm | ⭐⭐ |
| Đặt chỗ tiện ích | Sauna, bể bơi, sân… theo khung giờ, giới hạn sức chứa | ⭐⭐ |
| Đồ thất lạc | Ghi nhận, hội viên tra cứu và nhận lại | ⭐ |

**POS & kho**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| Đơn đặt hàng NCC (PO) | Tạo PO từ cảnh báo tồn thấp → nhận hàng → tự tạo phiếu nhập | ⭐⭐ |
| Lô & hạn sử dụng | Nhập theo lô, xuất FEFO, cron cảnh báo sắp hết hạn | ⭐⭐ |
| Kiểm kê định kỳ | Phiên kiểm kê, nhập số đếm, tự tạo phiếu điều chỉnh | ⭐⭐ |
| Quầy bar có công thức | Trừ kho nguyên liệu theo công thức (BOM) | ⭐ |
| Đặt hàng trước trên portal | Đặt sản phẩm, nhận tại quầy khi check-in | ⭐ |

**Nhân sự**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| Quỹ phép năm | Theo dõi số ngày phép, trừ vào bảng lương | ⭐⭐ |
| Chứng chỉ HLV | Lưu chứng chỉ có hạn, cron nhắc sắp hết hạn | ⭐ |

**Báo cáo & phân tích**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| LTV & hiệu quả nguồn khách | Giá trị vòng đời, doanh thu theo nguồn lead | ⭐⭐ |
| So sánh chi nhánh | Xếp hạng theo doanh thu, retention, lấp đầy lớp | ⭐⭐ |
| Báo cáo gửi email theo lịch | Báo cáo tuần/tháng (PDF/Excel) gửi chủ phòng tập | ⭐⭐ |
| Lưu bộ lọc / báo cáo yêu thích | Lưu cấu hình bộ lọc, ghim lên dashboard | ⭐ |

**Hệ thống**
| Tính năng | Mô tả | Ưu tiên |
|---|---|---|
| Tối ưu dữ liệu lớn | Phân vùng `checkins`/`audit_logs` theo tháng, materialized view cho báo cáo | ⭐⭐ |
| Đa ngôn ngữ | Thêm `messages/en` cho web (đã có `next-intl`), i18n cho admin | ⭐ |

---

## 15. Bổ sung còn thiếu (Sprint 29–30 + Backlog)

> Migration tiếp `000027`, `000028`. Phạm vi vẫn không gồm mobile app, tích hợp bên ngoài, AI, SaaS.

### 15.1 Tính năng còn thiếu
| Mảng | Nội dung | Ưu tiên |
|---|---|---|
| Landing / marketing | Trang hồ sơ HLV công khai (`trainer_public_profiles`: ảnh, chuyên môn, chứng chỉ, rating), trang chi tiết lớp, cảm nhận hội viên (`testimonials`, admin duyệt), tin tức/blog (`posts` – CMS đơn giản: tiêu đề, slug, ảnh bìa, nội dung, trạng thái), form liên hệ (`contact_messages` → thông báo lễ tân), bản đồ chi nhánh (nhúng iframe) | ⭐⭐⭐ |
| SEO | `generateMetadata` từng trang, `sitemap.ts`, `robots.ts`, structured data `schema.org/ExerciseGym` cho chi nhánh, ảnh Open Graph | ⭐⭐ |
| Nhắn tin HLV ↔ hội viên | `conversations` (gắn `pt_contract_id`), `messages` (text/ảnh, read_at); chỉ nhắn trong hợp đồng PT còn hiệu lực; dùng SSE của Sprint 17 để đẩy tin mới | ⭐⭐ |
| Tùy chọn thông báo | `notification_preferences` (user_id, type, channel[`IN_APP`|`EMAIL`], enabled); `Notifier` kiểm tra trước khi gửi; thông báo bắt buộc (bảo mật, hóa đơn) không tắt được | ⭐⭐ |
| Hoàn tiền online | Hoàn tiền giao dịch VNPay qua API refund sandbox; `refunds.online_txn_id`; trạng thái chờ xử lý → thành công/thất bại | ⭐⭐ |
| Chuyển chi nhánh chính | `member_branch_transfers` (from, to, reason, approved_by); gói `SINGLE_BRANCH` đang chạy được chuyển theo hoặc tính phí theo cấu hình | ⭐ |
| Xuất lịch .ics | `GET /member/calendar.ics?token=` (token riêng, thu hồi được) và tương tự cho HLV – lịch lớp đã đặt + buổi PT | ⭐ |
| Trải nghiệm admin | Tìm kiếm nhanh toàn hệ thống (Ctrl+K: hội viên, hóa đơn, sản phẩm – `GET /admin/search?q=`), thao tác hàng loạt (gửi thông báo, gắn tag, xuất Excel), tùy biến mẫu in hóa đơn (logo, chân trang, khổ A5/80mm), dark mode | ⭐⭐ |

### 15.2 Triển khai thực tế
| Mảng | Nội dung | Ưu tiên |
|---|---|---|
| Chuyển đổi dữ liệu cũ | Import Excel nhiều sheet: hội viên → gói đang chạy (ngày bắt đầu/kết thúc, số buổi còn lại) → hợp đồng PT (buổi còn lại) → công nợ. `legacy_import_jobs` (file, trạng thái, số dòng OK/lỗi, file lỗi tải về); chạy thử (dry-run) trước khi ghi; dữ liệu nhập ghi `source = 'IMPORT'` để truy vết | ⭐⭐⭐ |
| Wizard thiết lập ban đầu | Lần đầu OWNER đăng nhập: thông tin trung tâm → chi nhánh → phòng → gói tập mẫu → nhân viên → cài đặt (giờ mở cửa, quy định hủy lớp); có thể bỏ qua và quay lại | ⭐⭐ |
| Tài liệu người dùng | Hướng dẫn theo vai trò (lễ tân, HLV, quản lý, hội viên) ở `docs/user-guide/`; nút "?" trên từng màn hình admin mở hướng dẫn tương ứng; tour giới thiệu lần đầu | ⭐⭐ |

**Gợi ý chia sprint**
| Sprint | Nội dung | Migration |
|---|---|---|
| 29 | Landing (hồ sơ HLV, chi tiết lớp, cảm nhận, blog, liên hệ) + SEO | 000027 (một phần) |
| 30 | Chuyển đổi dữ liệu cũ + wizard thiết lập + tài liệu người dùng | 000028 (một phần) |

Các mục còn lại của 15.1 (nhắn tin, tùy chọn thông báo, hoàn tiền online, chuyển chi nhánh, .ics, trải nghiệm admin) đưa vào Backlog §14.

### 15.3 Yêu cầu phi chức năng & kiểm thử bảo mật
**Mục tiêu đo được** (kiểm chứng ở Sprint 28 bằng k6, ghi vào chương Kiểm thử)
| Chỉ số | Mục tiêu |
|---|---|
| API đọc thông thường | p95 < 300 ms |
| Quét check-in (đầu cuối) | < 1 s |
| Danh sách 10k hội viên có phân trang/lọc | p95 < 500 ms |
| Tải check-in | 100 req/s, tỉ lệ lỗi < 0,1% |
| Đặt lớp đồng thời | 0 lần vượt sức chứa |
| Trang landing (Lighthouse mobile) | Performance ≥ 85, SEO ≥ 90 |

**Kiểm thử bảo mật**
- Checklist OWASP Top 10 cho từng module (ghi kết quả vào phụ lục): kiểm soát truy cập (IDOR trên `/member/*`), injection, xác thực, cấu hình CORS/header bảo mật, log sự kiện bảo mật.
- CI: `govulncheck ./...`, `npm audit --audit-level=high` cho web và admin; `gosec` trong `golangci-lint`.
- Test phân quyền tự động: mỗi endpoint admin gọi bằng token thiếu quyền → phải trả `40300`; hội viên A không đọc được dữ liệu hội viên B.

**Accessibility & responsive**
- Cổng hội viên và landing đạt WCAG 2.1 AA cơ bản: tương phản màu, nhãn form, điều hướng bàn phím, `alt` ảnh; kiểm bằng axe/Lighthouse.
- Kiểm tra responsive ở 360px, 768px, 1280px; admin ưu tiên ≥ 1280px nhưng các màn quầy (check-in, POS) dùng được trên tablet.

---

## 16. Quản lý lộ trình mở rộng

### 16.1 Phân phạm vi
| Phạm vi | Nội dung | Dùng trong quyển đồ án |
|---|---|---|
| **Đồ án (đã xong)** | Giai đoạn 1 – Sprint 01–11 | Chương Phân tích, Thiết kế, Cài đặt, Kiểm thử |
| **Đồ án (nếu kịp trước bảo vệ)** | Sprint 12 (quên mật khẩu), 13 (VNPay), 28 (E2E/k6) | Bổ sung chương Cài đặt & Kiểm thử |
| **Hướng phát triển** | Sprint 14–27, 29–30, Backlog | Chương Kết luận & hướng phát triển (tóm tắt theo nhóm) |

### 16.2 Phụ thuộc giữa các sprint
| Sprint | Cần có trước | Lý do |
|---|---|---|
| 13 VNPay | 12 | Luồng email/token dùng chung; hội viên tự thanh toán cần khôi phục mật khẩu |
| 16 CRM | 13 (khuyến nghị) | Convert lead → mua gói online |
| 17 SSE | — | Hạ tầng cho 15.1 nhắn tin, thông báo realtime |
| 18 Kiosk | 17 (khuyến nghị) | Admin nhận check-in mới realtime |
| 19 Tủ đồ | 18 | Gán tủ trong luồng check-in/kiosk |
| 20 Lương | 18 | Chấm công nhân viên qua kiosk |
| 21 Chi phí/P&L | 15, 20 | Chi phí bảo trì, lương đưa vào P&L |
| 22 Điểm thưởng | 14, 15, 16 | Cộng điểm khi ghi tập, đánh giá; mã giới thiệu đi qua form tập thử |
| 23 Dinh dưỡng/NPS | 16 | NPS thấp đẩy vào danh sách at-risk |
| 24 Hợp đồng/vé ngày | 18 | Kiosk nhận QR vé ngày |
| 25 Lịch rảnh/ngày lễ | 20 | Nghỉ phép trừ vào bảng lương |
| 26 Mật độ/KPI | 16, 20 | Nguồn khách cho KPI, nhân viên bán hàng |
| 28 E2E | 13 | Kịch bản mua gói online |
| 15.1 Nhắn tin | 17 | Đẩy tin nhắn qua SSE |
| 15.1 Hoàn tiền online | 13 | Dựa trên giao dịch VNPay |

Thứ tự an toàn: **12 → 13 → 28 → 17 → 14 → 15 → 16 → 18 → 20 → 22 → 21 → 24 → 25 → 26 → 27 → 19 → 23 → 29 → 30**.

### 16.3 Permission & mã lỗi mới
**Permission** (khai báo trong `consts/permission.go`, seed vào role mặc định)
| Module | Permission | Role mặc định |
|---|---|---|
| Workout | `workout.view`, `workout.manage` | MANAGER, TRAINER |
| Review | `review.view` | OWNER, MANAGER |
| Equipment | `equipment.view`, `equipment.manage`, `equipment.report` | MANAGER (manage); RECEPTIONIST, TRAINER (report) |
| Lead | `lead.view`, `lead.manage`, `lead.convert` | MANAGER, RECEPTIONIST |
| Device | `device.manage` | OWNER, MANAGER |
| Locker | `locker.view`, `locker.manage` | MANAGER, RECEPTIONIST |
| Payroll | `attendance.view`, `payroll.view`, `payroll.manage`, `payroll.lock` | OWNER, MANAGER (lock: OWNER) |
| Finance | `expense.manage`, `expense.approve`, `cash_session.manage`, `installment.manage`, `report.profit_loss` | MANAGER; RECEPTIONIST (cash_session) |
| Loyalty | `loyalty.manage`, `loyalty.adjust` | MANAGER |
| Engagement | `nutrition.manage`, `announcement.manage`, `survey.manage` | MANAGER, TRAINER (nutrition) |
| Contract / pass | `contract.manage`, `day_pass.sell` | MANAGER; RECEPTIONIST (day_pass) |
| Schedule | `availability.manage`, `leave.request`, `leave.approve`, `holiday.manage` | TRAINER (availability, leave.request); MANAGER |
| KPI | `kpi.manage`, `report.cohort` | OWNER, MANAGER |
| Privacy | `member.view_sensitive`, `data_request.process` | OWNER, MANAGER |
| CMS / landing | `cms.manage` | OWNER, MANAGER |
| Hệ thống | `import.legacy`, `setup.wizard` | OWNER |

**Mã lỗi** (đã khai báo sẵn trong `internal/pkg/rescode`, cần thêm các mã còn lại khi cài đặt)
| Mã | Tên | Sprint |
|---|---|---|
| 40104 | `RESET_TOKEN_INVALID` | 12 |
| 40105 | `TOO_MANY_ATTEMPTS` | 12 |
| 40106 | `DEVICE_UNAUTHORIZED` | 18 |
| 40107 / 40108 | `MFA_REQUIRED` / `MFA_INVALID_CODE` | 27 |
| 40901 | `LOCKER_UNAVAILABLE` | 19 |
| 40902 | `PAYROLL_LOCKED` | 20 |
| 40903 / 40904 | `INSUFFICIENT_POINTS` / `REWARD_OUT_OF_STOCK` | 22 |
| 42001 | `CONTRACT_UNSIGNED` | 24 |
| 42002 | `OUTSIDE_ACCESS_HOURS` | 24 |
| 42003 | `INSTALLMENT_OVERDUE` | 21 |
| 42004 | `BRANCH_HOLIDAY_CLOSED` | 25 |
| 44001–44003 | `PAYMENT_INVALID_SIGNATURE` / `PAYMENT_AMOUNT_MISMATCH` / `PAYMENT_TXN_EXPIRED` | 13 |
| 43xxx (mới) | Lịch rảnh HLV / ngoài khung giờ nhận PT | 25 |
| 46xxx (mới) | Thiết bị, tủ đồ | 18–19 |
| 47xxx (mới) | Lương, chi phí, đối soát ca | 20–21 |

### 16.4 Sơ đồ cần bổ sung cho quyển đồ án
- **Use case**: hội viên tự mua gói online; HLV soạn giáo án; lễ tân quản lý lead; quản lý chốt lương.
- **Sequence**: thanh toán VNPay (redirect → IPN → kích hoạt gói); quên mật khẩu; check-in qua kiosk/thiết bị; đăng nhập 2FA.
- **Activity**: vòng đời lead (NEW → CONVERTED/LOST); trả góp (kỳ → quá hạn → chặn check-in).
- **State**: `online_payment_txns`, `leads`, `payrolls`, `data_requests`.
- **ERD**: bổ sung các bảng ở §4.1 (phần giai đoạn 2–4), tách sơ đồ theo nhóm để dễ đọc.

### 16.5 Definition of Done (mỗi sprint)
- [ ] Migration up/down chạy được, có seed demo cho module mới.
- [ ] API có Req/Res + `g.Meta` đầy đủ, hiện trên Swagger; permission kiểm tra ở middleware.
- [ ] Unit test service cho các quy tắc nghiệp vụ; integration test cho transaction/constraint quan trọng.
- [ ] Admin/web có màn hình tương ứng, xử lý trạng thái loading/rỗng/lỗi.
- [ ] Audit log cho thao tác thay đổi dữ liệu quan trọng.
- [ ] CI xanh (lint, typecheck, test, build).
- [ ] Cập nhật `docs/api/<module>.md` và tick tiến độ ở README.

### 16.6 Rủi ro & cách giảm thiểu
| Rủi ro | Ảnh hưởng | Giảm thiểu |
|---|---|---|
| Phạm vi quá lớn so với thời gian | Không kịp bảo vệ | Chỉ cam kết phạm vi §16.1; còn lại ghi vào hướng phát triển |
| VNPay sandbox không ổn định / cần đăng ký | Trễ Sprint 13 | Interface `PaymentGateway` + bản mock để demo; IPN test bằng script ký giả lập |
| Sai lệch tiền (thanh toán, lương, P&L) | Mất niềm tin khi demo | Test đối chiếu tổng; mọi nghiệp vụ tiền chạy trong transaction; idempotency |
| Migration lớn phá dữ liệu demo | Mất thời gian seed lại | Luôn có `down`; backup DB trước khi migrate; seed tái tạo được |
| Rule engine check-in phình to (hợp đồng, khung giờ, trả góp, ngày lễ, vé ngày) | Khó bảo trì, chậm | Tách mỗi rule thành hàm/struct riêng theo interface `CheckinRule`, test từng rule |
| Dữ liệu cá nhân (CCCD, sức khỏe, ảnh) | Rủi ro pháp lý | Sprint 27: mã hóa, che theo quyền, ghi nhận đồng ý, audit log |
| Hiệu năng báo cáo khi dữ liệu lớn | Dashboard chậm | Index theo §4.2, materialized view, giới hạn khoảng thời gian truy vấn |
