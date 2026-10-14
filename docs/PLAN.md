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

---

## 11. Gợi ý cấu trúc quyển đồ án
1. **Mở đầu**: lý do chọn đề tài, mục tiêu, phạm vi, phương pháp.
2. **Cơ sở lý thuyết & công nghệ**: Go/GoFrame, PostgreSQL (transaction, EXCLUDE constraint, isolation), JWT/RBAC, React/Next.js (SSR, App Router), REST, Docker.
3. **Khảo sát & phân tích yêu cầu**: khảo sát quy trình gym thực tế, tác nhân, use case, đặc tả use case, yêu cầu phi chức năng.
4. **Thiết kế hệ thống**: kiến trúc tổng thể, kiến trúc phân lớp BE, FE, ERD, sequence/activity diagram, thiết kế API, thiết kế giao diện, bảo mật.
5. **Cài đặt**: môi trường, cấu trúc mã nguồn, các module tiêu biểu (kèm code trích dẫn), giải pháp cho các bài toán khó (trùng lịch, race condition đặt chỗ, QR chống gian lận, pro-rata).
6. **Kiểm thử & đánh giá**: test case, kết quả, hiệu năng.
7. **Kết luận & hướng phát triển**: mobile app, thiết bị cửa từ/vân tay, AI gợi ý giáo án, multi-tenant SaaS.
