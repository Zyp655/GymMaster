# GymMaster

> **Đồ án tốt nghiệp:** Phát triển hệ thống số hóa quy trình quản lý trung tâm Fitness trên nền tảng Web.

| Thành phần | Công nghệ | Cổng mặc định | Thư mục |
|---|---|---|---|
| **Backend API** | Go 1.25 · GoFrame v2 · GORM · golang-migrate (`gymmaster-api/migrations/`) · JWT · SSE · Cron | `8000` | [`gymmaster-api/`](gymmaster-api/README.md) |
| **Trang quản trị (Admin)** | React 18 · Vite 6 · TypeScript · Ant Design v5 · Redux Toolkit · TanStack Query | `3001` | [`gymmaster-admin/`](gymmaster-admin/README.md) |
| **Trang công khai & Cổng hội viên (Web)** | Next.js 16 (App Router) · React 19 · TypeScript · Tailwind v4 · shadcn/ui · TanStack Query · NextAuth | `3000` | [`gymmaster-web/`](gymmaster-web/README.md) |
| **Database & Mail Dev** | PostgreSQL 16 (`5432`) · Mailpit (`1025` SMTP / `8025` Web UI) | `5432` / `8025` | [`docker-compose.yml`](docker-compose.yml) |
| **Tài liệu & Kiểm thử** | Kế hoạch 30 Sprint, UML, OWASP, Hướng dẫn sử dụng, Playwright E2E, k6, Postman | — | [`docs/`](docs/PLAN.md) · [`e2e/`](e2e/) · [`perf/`](perf/) · [`postman/`](postman/) |

---

## 1. Hướng dẫn Chạy Nhanh bằng Docker Compose (Tất cả trong 1 lệnh)

Chạy toàn bộ hệ thống (PostgreSQL + Mailpit + tự động chạy toàn bộ migrations + nạp dữ liệu mẫu + API + Admin + Web):

```bash
# Khởi chạy toàn bộ hệ thống
docker compose up -d --build

# Xem nhật ký khởi tạo migration/seed và API
docker compose logs -f migrate api

# Dừng toàn bộ hệ thống
docker compose stop

# Tắt và xóa container (giữ lại dữ liệu DB)
docker compose down

# Xóa sạch volume DB để khởi tạo lại từ đầu
docker compose down -v
```

---

## 2. Hướng dẫn Chạy ở Chế độ Phát triển (Local Development – Khuyến nghị)

### Bước 1: Bật PostgreSQL và Mailpit bằng Docker

Tại thư mục gốc dự án (`/Users/hieu/Work/GymMaster`):

```bash
docker compose up -d postgres mailpit
```

### Bước 2: Chạy Backend API (`gymmaster-api` – cổng `8000`)

Mở **Terminal 1**:

```bash
cd gymmaster-api

# 1. Tạo file .env (nếu chưa có)
cp -n .env.example .env

# 2. Chạy toàn bộ migration trong migrations/ (số thứ tự có khoảng trống, vd không có 000039/000040 – golang-migrate chấp nhận)
make migrate-up
# (Hoặc chạy trực tiếp bằng Go: go run . migrate up)

# 3. Nạp dữ liệu mẫu đầy đủ cho Sprint 1–30
make seed
# (Hoặc chạy trực tiếp bằng Go: go run . seed)

# 4. Khởi động HTTP Server tại http://localhost:8000
make run
# (Hoặc: go run . http)
```

### Bước 3: Chạy Trang quản trị (`gymmaster-admin` – cổng `3001`)

Mở **Terminal 2**:

```bash
cd gymmaster-admin
cp -n .env.example .env
npm install
npm run dev
```

### Bước 4: Chạy Trang công khai & Cổng hội viên (`gymmaster-web` – cổng `3000`)

Mở **Terminal 3**:

```bash
cd gymmaster-web
cp -n .env.example .env.local
npm install
npm run dev
```

### Cách đóng/dừng các Server đang chạy

- Trong từng cửa sổ Terminal (`api`, `admin`, `web`): nhấn `Ctrl + C`.
- Dừng PostgreSQL & Mailpit:
  ```bash
  docker compose stop
  ```
- Kiểm tra nhanh xem còn cổng nào đang mở không:
  ```bash
  lsof -iTCP -sTCP:LISTEN -P -n | grep -E '8000|3000|3001|5432|8025'
  ```

---

## 3. Địa chỉ Truy cập & Tài khoản Demo

### Danh sách URL chính

| Thành phần | Địa chỉ | Ghi chú |
|---|---|---|
| **Landing Page công khai** | http://localhost:3000 | Bảng giá, gói PT, lịch lớp, đội ngũ HLV (`/trainers`), Blog (`/blog`), đăng ký tập thử |
| **Cổng Hội viên (Member Portal)** | http://localhost:3000/me | QR động, gói tập, mua gói VNPay, đặt lớp/PT, giáo án, dinh dưỡng, điểm thưởng, thử thách, hợp đồng điện tử, tin nhắn PT & lịch `.ics` |
| **Màn hình Kiosk tự check-in** | http://localhost:3000/kiosk | Giao diện toàn màn hình cho máy tính bảng / đầu đọc QR tại cổng |
| **Trang Quản trị (Admin SPA)** | http://localhost:3001 | Quản trị đa chi nhánh, `Ctrl+K` tìm kiếm nhanh, trợ giúp `?`, chế độ Sáng/Tối |
| **Backend API & Swagger UI** | http://localhost:8000/swagger | Tài liệu OpenAPI đầy đủ (`/health` để kiểm tra trạng thái) |
| **Mailpit (Hộp thư Dev)** | http://localhost:8025 | Xem email quên mật khẩu, mã OTP đăng ký hội viên, chúc mừng sinh nhật |

### Tài khoản Demo (Mật khẩu chung: `Admin@123`)

| Vai trò | Email | Cổng đăng nhập |
|---|---|---|
| Chủ hệ thống (`OWNER`) | `admin@gymmaster.vn` | Trang quản trị (`http://localhost:3001/login`) |
| Quản lý chi nhánh Cầu Giấy (`MANAGER`) | `quanly.cg@gymmaster.vn` | Trang quản trị (`http://localhost:3001/login`) |
| Lễ tân chi nhánh Cầu Giấy (`RECEPTIONIST`) | `letan.cg@gymmaster.vn` | Trang quản trị (`http://localhost:3001/login`) |
| Huấn luyện viên (`TRAINER`) | `hlv.cg@gymmaster.vn` | Trang quản trị (`http://localhost:3001/login`) |
| Hội viên (`MEMBER`) | `hoivien@gymmaster.vn` | Cổng hội viên (`http://localhost:3000/login`) |

---

## 4. Lệnh Kiểm thử & Build (Testing & Verification)

```bash
# 1. Kiểm thử & Build Backend API (Go)
cd gymmaster-api && go test -count=1 ./... && go build ./...

# 2. Kiểm tra kiểu, Unit Test & Build Trang quản trị (Admin)
cd gymmaster-admin && npm run typecheck && npm test && npm run build

# 3. Kiểm tra kiểu, Unit Test & Build Cổng hội viên (Web)
cd gymmaster-web && npm run typecheck && npm test && npm run build

# 4. Chạy E2E Playwright (khi cả 3 dịch vụ đang chạy)
cd e2e && npx playwright test
```

---

## 5. Tài liệu Dự án

- [Hướng dẫn cài đặt & chạy chi tiết](docs/SETUP_GUIDE.md): yêu cầu môi trường, cấu hình `.env`, chạy Docker/Local, xử lý sự cố.
- [Kế hoạch phát triển chi tiết (Sprint 1–30)](docs/PLAN.md): phạm vi, kiến trúc, ERD, nghiệp vụ, lộ trình 5 giai đoạn.
- [Hướng dẫn sử dụng theo vai trò](docs/user-guide/README.md): cẩm nang vận hành cho Chủ phòng tập/Quản lý, Lễ tân – Thu ngân, Huấn luyện viên và Hội viên.
- [Sơ đồ kiến trúc & UML mở rộng](docs/architecture/diagrams-stage2-4.md): Use Case, Sequence, Activity, State Machine, ERD.
- [Báo cáo kiểm thử bảo mật OWASP Top 10 & Phi chức năng](docs/security/owasp-audit.md).
- [Đặc tả API Giai đoạn mở rộng](docs/api/stage5-cms-system.md) & [thư mục `docs/api/`](docs/api/).

---

## 6. Tiến độ Thực hiện (30/30 Sprints Hoàn thành)

- [x] **Sprint 01** – Khởi tạo dự án, đăng nhập/refresh token, RBAC theo permission, CRUD chi nhánh, CRUD vai trò
- [x] **Sprint 02** – Nhân sự, HLV, ca làm, phòng tập
- [x] **Sprint 03** – Hội viên, chỉ số cơ thể
- [x] **Sprint 04–05** – Gói tập, đăng ký/gia hạn/bảo lưu/chuyển nhượng, hóa đơn, thanh toán tại quầy
- [x] **Sprint 06** – Check-in QR động & bộ quy tắc `CheckinRule` độc lập
- [x] **Sprint 07** – Lớp nhóm, đặt chỗ, waitlist tự động đôn chỗ
- [x] **Sprint 08** – Hợp đồng PT, đặt lịch dạy & hoa hồng HLV
- [x] **Sprint 09** – Cổng hội viên (`/me`)
- [x] **Sprint 10** – Báo cáo, thông báo, nhật ký thao tác, job định kỳ
- [x] **Sprint 11** – POS bán hàng + quản lý tồn kho
- [x] **Sprint 12** – Quên mật khẩu & chống brute-force đăng nhập
- [x] **Sprint 13** – Thanh toán online VNPay sandbox & hội viên tự mua gói tập
- [x] **Sprint 14** – Thư viện bài tập, giáo án tập luyện & nhật ký mức tạ
- [x] **Sprint 15** – Đánh giá HLV/lớp học & quản lý thiết bị – bảo trì
- [x] **Sprint 16** – CRM khách tiềm năng (Lead), vé tập thử & chăm sóc hội viên tự động
- [x] **Sprint 17** – Thông báo thời gian thực (SSE), PWA, hội viên tự đăng ký & hoàn thiện demo
- [x] **Sprint 18** – Kiểm soát ra vào bằng thiết bị (Turnstile/Door) & Kiosk tự check-in (`/kiosk`)
- [x] **Sprint 19** – Sơ đồ tủ đồ (Lockers) & dịch vụ thuê tủ theo lượt / theo tháng
- [x] **Sprint 20** – Chấm công nhân viên, tính bảng lương tháng, khóa & chi trả lương
- [x] **Sprint 21** – Chi phí vận hành, báo cáo lãi/lỗ (P&L), đối soát ca thu ngân, trả góp & hóa đơn điện tử
- [x] **Sprint 22** – Điểm thưởng Loyalty, đổi quà/voucher, huy hiệu, thử thách & giới thiệu bạn bè
- [x] **Sprint 23** – Dinh dưỡng & thực đơn (BMR/TDEE), bảng tin chi nhánh & khảo sát NPS
- [x] **Sprint 24** – Hợp đồng điện tử, vé ngày / vé khách mời QR & gói tập khung giờ off-peak
- [x] **Sprint 25** – Lịch rảnh HLV, duyệt đơn nghỉ phép, gán HLV dạy thay & lịch nghỉ lễ phòng tập
- [x] **Sprint 26** – Mật độ phòng tập realtime, chỉ tiêu doanh số KPI & báo cáo Cohort Retention
- [x] **Sprint 27** – Xác thực 2 lớp (2FA TOTP), quản lý phiên đăng nhập & tuân thủ Nghị định 13/2023
- [x] **Sprint 28** – Bộ kiểm thử E2E Playwright (`e2e/`), kiểm thử tải k6 (`perf/`), Postman/Newman (`postman/`) & CI
- [x] **Sprint 29** – Landing mở rộng (`/trainers`, `/blog`, cảm nhận hội viên, liên hệ) + SEO (`sitemap.ts`, `robots.ts`) + Nhắn tin HLV ↔ Hội viên + Lịch `.ics` + Tùy chọn thông báo
- [x] **Sprint 30** – Chuyển đổi dữ liệu cũ (`legacy_import_jobs` Dry-Run & Commit) + Wizard thiết lập ban đầu + Chuyển chi nhánh chính + Hoàn tiền VNPay Online + Tìm kiếm nhanh `Ctrl+K` & Dark Mode Admin + Tài liệu người dùng
