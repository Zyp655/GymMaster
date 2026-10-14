# GymMaster

> **Đồ án tốt nghiệp:** Phát triển hệ thống số hóa quy trình quản lý trung tâm Fitness trên nền tảng Web.

| Thành phần | Công nghệ | Thư mục |
|---|---|---|
| Backend API | Go 1.25 · GoFrame v2 · GORM · golang-migrate · JWT | [`gymmaster-api/`](gymmaster-api/README.md) |
| Trang quản trị | React 18 · Vite · TypeScript · Ant Design v5 · React Router · Redux Toolkit · TanStack Query (theo `strongbody_admin`) | [`gymmaster-admin/`](gymmaster-admin/README.md) |
| Trang công khai + cổng hội viên | Next.js (App Router) · React · TypeScript · Tailwind v4 · shadcn/ui · TanStack Query · next-auth (theo `strongbody-web`) | [`gymmaster-web/`](gymmaster-web/README.md) |
| Database | PostgreSQL 16 | `docker-compose.yml` |
| Tài liệu | Kế hoạch, API contract | [`docs/`](docs/PLAN.md) |

## Chạy toàn bộ bằng Docker

```bash
docker compose up -d --build
```

Lệnh trên tự áp migration và seed dữ liệu khởi tạo. Sau khi chạy xong:

| Thành phần | Địa chỉ |
|---|---|
| Web (Landing + Cổng hội viên) | http://localhost:3000 |
| Admin (Trang quản trị) | http://localhost:3001 |
| API | http://localhost:8000 |
| Swagger | http://localhost:8000/swagger |
| Mailpit (Xem email dev) | http://localhost:8025 |

### Tài khoản demo (mật khẩu chung: `Admin@123`)

| Vai trò | Email | Ứng dụng sử dụng |
|---|---|---|
| Chủ hệ thống (`OWNER`) | `admin@gymmaster.vn` | Trang quản trị (`http://localhost:3001`) |
| Quản lý chi nhánh Cầu Giấy (`MANAGER`) | `quanly.cg@gymmaster.vn` | Trang quản trị (`http://localhost:3001`) |
| Lễ tân chi nhánh Cầu Giấy (`RECEPTIONIST`) | `letan.cg@gymmaster.vn` | Trang quản trị (`http://localhost:3001`) |
| Huấn luyện viên (`TRAINER`) | `hlv.cg@gymmaster.vn` | Trang quản trị (`http://localhost:3001`) |
| Hội viên (`MEMBER`) | `hoivien@gymmaster.vn` | Cổng hội viên (`http://localhost:3000/me`) |

## Chạy khi phát triển

```bash
docker compose up -d postgres

# Terminal 1 – backend
cd gymmaster-api && cp .env.example .env && make migrate-up && make seed && make run

# Terminal 2 – trang quản trị (admin)
cd gymmaster-admin && cp .env.example .env && npm install && npm run dev

# Terminal 3 – trang công khai + cổng hội viên (web)
cd gymmaster-web && cp .env.example .env.local && npm install && npm run dev
```

## Tài liệu
- [Hướng dẫn cài đặt package & chạy hệ thống](docs/SETUP_GUIDE.md): yêu cầu môi trường, cài đặt thư viện, cấu hình `.env`, chạy Docker/Local, tài khoản demo và kiểm thử.
- [Kế hoạch phát triển chi tiết](docs/PLAN.md): phạm vi, kiến trúc, ERD, nghiệp vụ, lộ trình 16 tuần.
- [API contracts](docs/api/): đặc tả chi tiết tất cả các module.

## Tiến độ
- [x] **Sprint 01** – Khởi tạo dự án, đăng nhập/refresh token, RBAC theo permission, CRUD chi nhánh, CRUD vai trò
- [x] **Sprint 02** – Nhân sự, HLV, ca làm, phòng tập
- [x] **Sprint 03** – Hội viên, chỉ số cơ thể
- [x] **Sprint 04–05** – Gói tập, đăng ký/gia hạn/bảo lưu/chuyển nhượng, hóa đơn, thanh toán tại quầy
- [x] **Sprint 06** – Check-in QR
- [x] **Sprint 07** – Lớp nhóm, đặt chỗ, waitlist
- [x] **Sprint 08** – PT
- [x] **Sprint 09** – Cổng hội viên
- [x] **Sprint 10** – Báo cáo, thông báo, nhật ký thao tác, job định kỳ
- [x] **Sprint 11** – POS bán hàng + tồn kho
