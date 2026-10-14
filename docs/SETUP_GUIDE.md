# Hướng dẫn Cài đặt Package và Chạy Hệ thống GymMaster

Tài liệu này hướng dẫn chi tiết cách cài đặt các thư viện (packages), cấu hình biến môi trường, khởi tạo cơ sở dữ liệu và chạy toàn bộ 3 thành phần của hệ thống **GymMaster** (Backend API, Trang quản trị Admin, và Trang công khai + Cổng hội viên).

---

## 1. Yêu cầu hệ thống (Prerequisites)

Trước khi bắt đầu, hãy đảm bảo máy tính đã cài đặt các công cụ sau:

| Công cụ | Phiên bản yêu cầu | Mục đích sử dụng | Kiểm tra phiên bản |
|---|---|---|---|
| **Docker & Docker Compose** | Docker Desktop 4.x+ / Engine 24+ | Chạy PostgreSQL 16, Mailpit (SMTP dev) hoặc chạy toàn bộ hệ thống | `docker -v && docker compose version` |
| **Go (Golang)** | `>= 1.25` | Biên dịch và chạy backend `gymmaster-api` | `go version` |
| **Node.js** | `>= 22.0.0` (khuyến nghị Node 22 LTS hoặc Node 24) | Cài đặt package và chạy `gymmaster-admin` & `gymmaster-web` | `node -v` |
| **npm** | `>= 10.0.0` (đi kèm Node.js) | Quản lý package cho 2 dự án Frontend | `npm -v` |
| **GNU Make** *(tùy chọn)* | `>= 3.81` | Chạy các lệnh tắt trong `gymmaster-api/Makefile` | `make -v` |

---

## 2. Tổng quan kiến trúc & Cổng dịch vụ (Ports)

Dự án gồm 3 thư mục mã nguồn chính và 2 dịch vụ hạ tầng:

| Thành phần | Thư mục | Công nghệ chính | Cổng mặc định | Địa chỉ truy cập |
|---|---|---|---|---|
| **Database** | *(Docker)* | PostgreSQL 16 Alpine (`Asia/Ho_Chi_Minh`) | `5432` | `localhost:5432` |
| **Mailpit (SMTP Dev)** | *(Docker)* | Mailpit (bắt email đặt lại mật khẩu, nhắc hạn...) | `1025` (SMTP) / `8025` (Web UI) | http://localhost:8025 |
| **Backend API** | `gymmaster-api/` | Go 1.25, GoFrame v2, GORM, golang-migrate, JWT, Cron, Excelize | `8000` | http://localhost:8000 (Swagger: `/swagger`) |
| **Trang quản trị (Admin)** | `gymmaster-admin/` | React 18, Vite 6, TypeScript, Ant Design v5, Redux Toolkit, TanStack Query v5 | `3001` | http://localhost:3001 |
| **Landing & Cổng hội viên** | `gymmaster-web/` | Next.js 16 (App Router), React 19, Tailwind CSS v4, NextAuth v4, next-intl, TanStack Query v5 | `3000` | http://localhost:3000 (Cổng HV: `/me`) |

---

## 3. Cách 1: Chạy toàn bộ hệ thống bằng Docker Compose (Nhanh nhất)

Nếu bạn chỉ muốn khởi chạy toàn bộ hệ thống (Database + Mailpit + Migration + Seed dữ liệu mẫu + API + Admin + Web) bằng 1 lệnh duy nhất:

```bash
# Tại thư mục gốc /Users/hieu/Work/GymMaster
docker compose up -d --build
```

- Container `migrate` sẽ tự động đợi PostgreSQL sẵn sàng, chạy toàn bộ migration (`000001` → `000010`) và nạp dữ liệu mẫu (`seed`).
- Xem tiến trình khởi tạo:
  ```bash
  docker compose logs -f migrate api
  ```
- Để dừng hệ thống:
  ```bash
  docker compose down
  ```
- Để xóa sạch dữ liệu DB và khởi tạo lại từ đầu:
  ```bash
  docker compose down -v
  docker compose up -d --build
  ```

---

## 4. Cách 2: Cài đặt Package & Chạy ở chế độ Phát triển (Local Development)

Đây là cách khuyến nghị khi lập trình để có tốc độ biên dịch nhanh và hỗ trợ Hot Reload.

### Bước 1: Khởi chạy PostgreSQL và Mailpit bằng Docker

Tại thư mục gốc của dự án:

```bash
docker compose up -d postgres mailpit
```

Kiểm tra PostgreSQL đã `healthy`:
```bash
docker compose ps
```

---

### Bước 2: Cài đặt & Chạy Backend API (`gymmaster-api`)

#### 2.1. Di chuyển vào thư mục và tạo file môi trường `.env`
```bash
cd gymmaster-api
cp .env.example .env
```

Các biến môi trường quan trọng trong `gymmaster-api/.env`:
- `SERVER_PORT=8000`: Cổng chạy HTTP server.
- `CORS_ORIGINS=http://localhost:3000,http://localhost:3001`: Cho phép `gymmaster-web` và `gymmaster-admin` gọi API.
- `DB_HOST=localhost`, `DB_PORT=5432`, `DB_USER=gymmaster`, `DB_PASSWORD=gymmaster`, `DB_NAME=gymmaster`.
- `JWT_SECRET`: Chuỗi bí mật ký JWT (tối thiểu 32 ký tự).
- `SMTP_HOST=localhost`, `SMTP_PORT=1025`: Kết nối tới Mailpit để bắt email trong môi trường dev.

#### 2.2. Tải các Go module (packages)
```bash
go mod download
```

*(Tùy chọn)* Nếu muốn dùng chế độ tự động tải lại khi sửa code (`make dev`), cài thêm `air`:
```bash
go install github.com/air-verse/air@latest
```

#### 2.3. Chạy Migration và Seed dữ liệu mẫu
```bash
# Áp dụng toàn bộ 10 file SQL migration vào PostgreSQL
make migrate-up
# (Hoặc nếu không dùng make: go run . migrate up)

# Tạo dữ liệu khởi tạo + dữ liệu demo (3 chi nhánh, nhân sự, 501 hội viên, lịch lớp, PT, POS...)
make seed
# (Hoặc nếu không dùng make: go run . seed)
```

> **Lưu ý:** Nếu bạn chỉ muốn tạo tài khoản `OWNER` + 3 chi nhánh + danh mục cơ bản mà **không** sinh 500 hội viên demo, chạy:
> ```bash
> SEED_DEMO=false go run . seed
> ```

#### 2.4. Khởi chạy API Server
```bash
# Chạy bình thường
make run
# (Hoặc: go run . http)

# Hoặc chạy với hot-reload (nếu đã cài air)
make dev
```

Sau khi chạy, kiểm tra tại:
- Healthcheck: http://localhost:8000/v1/health
- Tài liệu Swagger UI: http://localhost:8000/swagger

---

### Bước 3: Cài đặt Package & Chạy Trang quản trị (`gymmaster-admin`)

Mở một cửa sổ terminal mới:

#### 3.1. Cài đặt Node packages
```bash
cd gymmaster-admin
npm install
```

**Các package chính được cài đặt trong `gymmaster-admin`:**
- **UI & Biểu đồ:** `antd` (v5), `@ant-design/icons`, `@ant-design/charts`
- **Trạng thái & Dữ liệu:** `@tanstack/react-query` (v5), `@reduxjs/toolkit`, `react-redux`, `axios`
- **Điều hướng & Tiện ích:** `react-router-dom` (v6), `dayjs`
- **Công cụ phát triển:** `vite` (v6), `typescript` (~5.8), `eslint` (v9), `prettier`

#### 3.2. Tạo file cấu hình `.env`
```bash
cp .env.example .env
```
Nội dung mặc định của `gymmaster-admin/.env`:
```env
VITE_API_URL=http://localhost:8000/v1
```

#### 3.3. Chạy ứng dụng
```bash
# Chạy dev server (cổng 3001)
npm run dev
```
Truy cập Trang quản trị tại: **http://localhost:3001**

**Các lệnh hữu ích khác trong `gymmaster-admin`:**
```bash
npm test              # Chạy unit test (formatters, errors, interceptors, URL mapping, reports)
npm run typecheck     # Kiểm tra kiểu TypeScript (tsc)
npm run lint          # Kiểm tra ESLint (0 warning policy)
npm run build         # Build bản production ra thư mục dist/
npm run preview       # Chạy thử bản build tại http://localhost:3001
```

---

### Bước 4: Cài đặt Package & Chạy Trang công khai + Cổng hội viên (`gymmaster-web`)

Mở một cửa sổ terminal mới:

#### 4.1. Cài đặt Node packages
```bash
cd gymmaster-web
npm install
```

**Các package chính được cài đặt trong `gymmaster-web`:**
- **Framework & Auth:** `next` (v16.4.0 App Router), `react` (v19), `next-auth` (v4), `next-intl` (v4)
- **Giao diện & Icon:** `tailwindcss` (v4), `radix-ui`, `@heroicons/react`, `sonner` (toast), `class-variance-authority`, `tailwind-merge`
- **Biểu mẫu & Xác thực dữ liệu:** `react-hook-form`, `zod`, `@hookform/resolvers`
- **Biểu đồ, Lịch & QR:** `recharts` (biểu đồ chỉ số cơ thể), `qrcode.react` (mã QR động vào cửa), `@fullcalendar/*`, `dayjs`

#### 4.2. Tạo file cấu hình `.env.local`
```bash
cp .env.example .env.local
```
Nội dung mặc định của `gymmaster-web/.env.local`:
```env
NEXT_PUBLIC_API_URL=http://localhost:8000/v1
API_URL_INTERNAL=
NEXTAUTH_URL=http://localhost:3000
NEXTAUTH_SECRET=thay-bang-chuoi-ngau-nhien-it-nhat-32-ky-tu
```

#### 4.3. Chạy ứng dụng
```bash
# Chạy Next.js dev server (cổng 3000)
npm run dev
```
- Trang giới thiệu công khai (Landing Page): **http://localhost:3000**
- Trang đăng nhập hội viên: **http://localhost:3000/login**
- Cổng hội viên (Member Portal): **http://localhost:3000/me**

**Các lệnh hữu ích khác trong `gymmaster-web`:**
```bash
npm test              # Chạy unit test (formatters, http.core, Zod schemas, từ điển i18n)
npm run typecheck     # Kiểm tra kiểu TypeScript (tsc --noEmit)
npm run lint          # Kiểm tra ESLint
npm run build         # Build bản production Next.js
npm run start         # Chạy bản production tại cổng 3000
```

---

## 5. Danh sách Tài khoản Demo

Sau khi chạy `make seed` (hoặc `docker compose up`), hệ thống tạo sẵn các tài khoản sau với **mật khẩu chung là `Admin@123`**:

| Vai trò | Email đăng nhập | Mật khẩu | Chi nhánh | Nơi đăng nhập |
|---|---|---|---|---|
| **Chủ hệ thống (`OWNER`)** | `admin@gymmaster.vn` | `Admin@123` | Toàn hệ thống | http://localhost:3001/login |
| **Quản lý (`MANAGER`)** | `quanly.cg@gymmaster.vn` | `Admin@123` | Cầu Giấy (`HN01`) | http://localhost:3001/login |
| **Quản lý (`MANAGER`)** | `quanly.hbt@gymmaster.vn` | `Admin@123` | Hai Bà Trưng (`HN02`) | http://localhost:3001/login |
| **Quản lý (`MANAGER`)** | `quanly.q1@gymmaster.vn` | `Admin@123` | Quận 1 (`HCM01`) | http://localhost:3001/login |
| **Lễ tân (`RECEPTIONIST`)** | `letan.cg@gymmaster.vn` | `Admin@123` | Cầu Giấy (`HN01`) | http://localhost:3001/login |
| **Huấn luyện viên (`TRAINER`)** | `hlv.cg@gymmaster.vn` | `Admin@123` | Cầu Giấy (`HN01`) | http://localhost:3001/login |
| **Hội viên mẫu (`MEMBER`)** | `hoivien@gymmaster.vn` | `Admin@123` | Cầu Giấy (`HN01`) | http://localhost:3000/login |

---

## 6. Chạy Kiểm thử (Testing) trên toàn dự án

Tại thư mục gốc `/Users/hieu/Work/GymMaster`, bạn có thể chạy kiểm thử cho từng phần:

```bash
# 1. Backend API: Unit tests + Go vet
go -C gymmaster-api test -count=1 ./internal/...
go -C gymmaster-api vet ./...

# 2. Backend API: Integration tests (khi PostgreSQL đang chạy tại localhost:5432)
TEST_DB_DSN="postgres://gymmaster:gymmaster@localhost:5432/gymmaster?sslmode=disable" \
  go -C gymmaster-api test -count=1 ./internal/...

# 3. Admin SPA: Unit tests + Typecheck + ESLint
npm --prefix gymmaster-admin test
npm --prefix gymmaster-admin run typecheck
npm --prefix gymmaster-admin run lint

# 4. Web Portal: Unit tests + Typecheck + ESLint
npm --prefix gymmaster-web test
npm --prefix gymmaster-web run typecheck
npm --prefix gymmaster-web run lint
```

---

## 7. Xử lý sự cố thường gặp (Troubleshooting)

1. **Lỗi `connect: connection refused` tới `localhost:5432` khi chạy `make migrate-up` hoặc `make run`:**
   - Nguyên nhân: Container PostgreSQL chưa được bật.
   - Khắc phục: Chạy `docker compose up -d postgres` tại thư mục gốc.

2. **Lỗi CORS khi gọi API từ trình duyệt:**
   - Kiểm tra biến `CORS_ORIGINS` trong `gymmaster-api/.env` đã bao gồm đúng địa chỉ và cổng của frontend (`http://localhost:3000,http://localhost:3001`) chưa, sau đó khởi động lại `gymmaster-api`.

3. **Cổng `3001` hoặc `3000` hoặc `8000` đang bị chiếm dụng:**
   - Kiểm tra tiến trình đang chiếm cổng trên macOS/Linux:
     ```bash
     lsof -i :8000 -i :3000 -i :3001
     ```
   - Kết thúc tiến trình cũ hoặc đổi cổng trong `.env`.

4. **Muốn làm sạch hoàn toàn cơ sở dữ liệu và nạp lại dữ liệu demo mới:**
   ```bash
   cd gymmaster-api
   make migrate-down n=10
   make migrate-up
   make seed
   ```
