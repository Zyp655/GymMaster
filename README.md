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
| Web | http://localhost:3000 |
| API | http://localhost:8000 |
| Swagger | http://localhost:8000/swagger |

Tài khoản demo: `admin@gymmaster.vn` / `Admin@123`.

## Chạy khi phát triển

```bash
docker compose up -d postgres

# Terminal 1 – backend
cd gymmaster-api && cp .env.example .env && make migrate-up && make seed && make run

# Terminal 2 – frontend
cd gymmaster-web && cp .env.example .env.local && npm install && npm run dev
```

## Tài liệu
- [Kế hoạch phát triển chi tiết](docs/PLAN.md): phạm vi, kiến trúc, ERD, nghiệp vụ, lộ trình 16 tuần.
- [API contract Sprint 01](docs/api/sprint-01-auth-rbac-branch.md): Auth, RBAC, Chi nhánh.

## Tiến độ
- [x] **Sprint 01** – Khởi tạo dự án, đăng nhập/refresh token, RBAC theo permission, CRUD chi nhánh, CRUD vai trò
- [ ] Sprint 02 – Nhân sự, HLV, ca làm, phòng tập
- [ ] Sprint 03 – Hội viên, chỉ số cơ thể
- [ ] Sprint 04–05 – Gói tập, đăng ký/gia hạn/bảo lưu/chuyển nhượng, hóa đơn, thanh toán tại quầy
- [ ] Sprint 06 – Check-in QR
- [ ] Sprint 07 – Lớp nhóm, đặt chỗ, waitlist
- [ ] Sprint 08 – PT
- [ ] Sprint 09 – Cổng hội viên
- [ ] Sprint 10 – Báo cáo, thông báo
- [ ] Sprint 11 – POS bán hàng + tồn kho
