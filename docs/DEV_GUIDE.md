# GymMaster – Hướng dẫn phát triển module

Tài liệu này dành cho người (hoặc agent) thêm một module mới. Đọc kỹ trước khi viết code.

## 0. Nguồn sự thật
- **Schema DB:** `gymmaster-api/migrations/000001…000010` đã chứa **toàn bộ** bảng của hệ thống và đã được áp dụng.
  - **Không sửa** các migration đã có.
  - Nếu thật sự cần thêm cột hoặc bảng, tạo migration mới với số tiếp theo, sau khi kiểm tra trong thư mục rằng số đó chưa có ai dùng.
- **Permission:** dùng hằng số trong `gymmaster-api/internal/consts/permission.go` (backend) và `gymmaster-web/src/lib/permissions.ts` (frontend). Cả hai đã đủ, không cần thêm.
- **Nghiệp vụ:** xem `docs/PLAN.md` §5.
- **Mẫu code tham khảo:** module **branch** (CRUD, phân trang) và **role**.

## 1. Backend (`gymmaster-api`)

### Thêm một module `<m>`
1. **Định nghĩa API.**
   - `api/<m>/v1/<m>.go`: các struct Req/Res. Trong `g.Meta` khai báo `path`, `method`, `tags` (tiếng Việt), `summary` và `perm:"<perm>"`.
     - Endpoint admin luôn có `perm`.
     - Endpoint cổng hội viên dùng `role:"MEMBER"` thay cho `perm`.
   - `api/<m>/<m>.go`: interface `I<M>V1`, viết tay theo đúng mẫu của `api/branch/branch.go`.
   - Validate bằng tag `v:"rule|rule#Thông báo|Thông báo"` (thông báo tiếng Việt). Trường không bắt buộc mà để trống thì GoFrame bỏ qua rule.
   - API danh sách: nhúng `dto.PageReq`, trả `dto.PageRes[*X]`.
2. **Entity:** `internal/entity/<m>.go`.
   - Khớp đúng cột trong migration.
   - Cột GENERATED (ví dụ `body_metrics.bmi`, `subscription_freezes.days`) phải gắn tag `gorm:"->"`.
   - Cột `DATE` map sang `time.Time`; khi trả JSON thì format `"2006-01-02"` (`consts.DateLayout`).
   - Tiền: cột `NUMERIC(14,2)` map sang `float64`.
   - Bảng có `deleted_at` thì nhúng `BaseEntity`.
3. **Repository:** `internal/repository/<m>_repository.go`.
   - Interface + struct private + `NewXRepository(db)`.
   - **Luôn** lấy kết nối bằng `postgres.Conn(ctx, r.db)` để tham gia được transaction.
   - Cần khóa dòng thì dùng `Clauses(clause.Locking{Strength: "UPDATE"})`.
4. **Service:** `internal/service/<m>_service.go`.
   - Interface + struct + `NewXService(...)`.
   - Chỉ phụ thuộc vào interface repository, `context` và các service dùng chung. **Không** import `ghttp`.
   - Lấy người dùng hiện tại: `reqctx.Auth(ctx)` / `reqctx.UserID(ctx)`.
   - **Phạm vi chi nhánh:**
     - `reqctx.AllowedBranches(ctx)` trả `nil` nghĩa là được xem toàn hệ thống.
     - `reqctx.CanAccessBranch(ctx, id)`.
     - `reqctx.ResolveBranchFilter(ctx, req.BranchID)` cho API danh sách.
     - Nhân viên chỉ thấy và thao tác dữ liệu thuộc chi nhánh của mình.
   - **Lỗi:**
     - `rescode.New(rescode.X, "Thông điệp tiếng Việt")`.
     - Không tìm thấy bản ghi (`gorm.ErrRecordNotFound`): trả `rescode.NotFound`.
     - Trùng unique: dùng `mapDuplicate`.
     - Vi phạm EXCLUDE (trùng lịch): lỗi Postgres SQLSTATE `23P01`. Dùng helper `isExclusionViolation(err)` trong `service/errors.go` và trả `rescode.Conflict`.
   - **Transaction:** nghiệp vụ ghi vào nhiều bảng phải bọc trong `deps.Tx.WithTx(ctx, func(ctx) error {...})`.
   - **Service dùng chung** (có sẵn trong `module.Deps`):
     - `Settings` (`SettingService`): đọc cấu hình nghiệp vụ, ví dụ `Settings.Int(ctx, service.SettingBookingAdvanceDays, 7)`.
     - `Audit.Log(ctx, action, entity, id, before, after)`: ghi audit cho thao tác quan trọng (tạo/sửa/xóa, bán, hủy, hoàn tiền…).
     - `Notifier.Notify(...)` / `Notifier.NotifyPermission(...)`: tạo thông báo in-app.
5. **Controller:** `internal/controller/<m>/<m>_new.go` và mỗi action một file `<m>_v1_<action>.go`. Controller phải mỏng, chỉ gọi service.
6. **Lắp ráp:** tạo `internal/module/<m>.go` với hàm `Register<M>(d *Deps, group *ghttp.RouterGroup)`.
   - Thêm **một dòng** vào `internal/router/router.go` tại chỗ đánh dấu `// [admin-modules]`, `// [member-modules]` hoặc `// [public-modules]`.
   - Trước khi sửa file này phải đọc lại nội dung mới nhất, vì có thể người khác vừa thêm dòng.
7. **Test:** `internal/service/<m>_service_test.go` dùng fake repository (xem `auth_service_test.go`). Bắt buộc có test cho các quy tắc nghiệp vụ.

### Quy ước API
- Đường dẫn: `/v1/admin/<resource>` (số nhiều, kebab-case).
  - Hành động nghiệp vụ dùng `POST /v1/admin/<resource>/{id}/<action>`, ví dụ `/subscriptions/{id}/freeze`.
- JSON: snake_case.
- Ngày: `"YYYY-MM-DD"`. Thời điểm: RFC3339.
- Danh sách: `data = {items,total,total_page,current_page,limit}`.
- Danh sách nhỏ dùng cho dropdown có thể trả mảng thường qua endpoint `/options`, ví dụ `GET /v1/admin/rooms/options?branch_id=`.

### Chạy và kiểm tra
- Postgres chạy sẵn tại `localhost:5432` (docker compose). Schema đã ở version 10.
- Chạy API riêng của mình trên một **cổng riêng** để không đụng người khác, ví dụ `SERVER_PORT=8011 go run . http`. **Không** dùng cổng 8000.
- Tài khoản OWNER: `admin@gymmaster.vn` / `Admin@123`.
- Trước khi báo xong phải chạy được: `go build ./... && go vet ./... && go test ./internal/...`.
- Có thể có người khác đang sửa code song song. Nếu build lỗi ở package **không phải của mình**, đợi một chút rồi build lại; **không** sửa code của module khác.

### Thư viện & hạ tầng có sẵn (KHÔNG chạy `go get` / `go mod tidy`)
- excelize (Excel), robfig/cron, gomail, google/uuid, datatypes đã có trong go.mod (giữ bằng `tools.go`).
- Config sẵn: `cfg.Upload` (thư mục `resource/uploads` serve tại `/uploads`), `cfg.Mail` (SMTP → Mailpit), `cfg.Cron.Enabled`.
- Nếu thật sự thiếu thư viện: ghi rõ trong báo cáo cuối, không tự cài.

## 2. Frontend (`gymmaster-web`)
- Đọc trước: `gymmaster-web/README.md`, `DESIGN_SYSTEM.md`, `AGENTS.md`. Next.js 16 có nhiều thay đổi; tra tài liệu trong `node_modules/next/dist/docs/` khi không chắc.
- **Feature folder:** `src/features/<m>/{api.ts,queries.ts,schema.ts,types.ts,components/}`, làm theo mẫu `features/branch`.
- **Trang:** `src/app/(admin)/admin/<route>/page.tsx`, kiểm tra quyền phía server theo mẫu trang branches.
  - Menu sidebar đã khai báo sẵn trong `nav-config.ts`, không cần sửa.
- **Chuỗi giao diện:** ghi vào `messages/vi/<domain>.json`. File đã được tạo sẵn (đang là `{}`) và đã đăng ký trong `src/i18n/messages.ts`.
  - **Chỉ sửa file domain của mình.**
  - Có thể thêm key dùng chung vào `common.json`, nhưng chỉ được **thêm**, không đổi hoặc xóa key có sẵn.
- **Component dùng chung:** `DataTable`, `PageHeader`, `ConfirmDialog`, `EmptyState`, `LoadError`, `PermissionGate`, `StatusBadge`, `SearchInput`, cùng các hook `useListParams` và `useToastApiError`.
  - Tiền tệ, ngày: dùng `src/lib/format.ts`.
  - Nếu cần thêm component dùng chung (ví dụ chọn hội viên `MemberPicker`): đặt trong `src/components/common/` và dùng tên file riêng, không sửa file có sẵn của người khác.
- **Thư viện có sẵn (KHÔNG chạy `npm install`):** recharts (biểu đồ), qrcode.react (hiển thị QR), html5-qrcode (quét QR bằng camera), @fullcalendar/{react,core,timegrid,daygrid,interaction} (lịch tuần), dayjs, TanStack Table v9, react-hook-form + zod, sonner, Heroicons.
- **Kiểm tra:** `npm run typecheck && npm run lint`.
  - **Không** chạy `npm run build`: build dùng chung thư mục `.next` và sẽ được chạy tập trung ở cuối.
  - **Không** chạy `npm run dev` trên cổng 3000.
  - Nếu cần xem giao diện: `npx next dev -p 30xx`, với một cổng riêng.
