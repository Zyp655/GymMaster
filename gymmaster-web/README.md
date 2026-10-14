# gymmaster-web

Ứng dụng **Trang công khai (Landing Page) & Cổng hội viên (`/me/*`)** của hệ thống **GymMaster** (trang quản trị cho nhân viên/chủ trung tâm nằm riêng tại `../gymmaster-admin`).
Kiến trúc dựa trên `strongbody-web`, có cải tiến: HTTP client tập trung (tự refresh token), chỉ dùng TanStack Query, token thiết kế thống nhất trong `@theme`.

## Công nghệ

| Mục       | Thư viện                                                                                             |
| --------- | ---------------------------------------------------------------------------------------------------- |
| Framework | Next.js 16 (App Router, Turbopack), React 19, TypeScript strict                                      |
| UI        | Tailwind CSS v4 (token trong `@theme`), primitives kiểu shadcn/ui trên `radix-ui`, Heroicons, sonner |
| Dữ liệu   | TanStack Query v5, Recharts (biểu đồ chỉ số cơ thể), `qrcode.react` (mã QR động vào cửa)             |
| Form      | react-hook-form + zod                                                                                |
| Xác thực  | next-auth v4 (Credentials, JWT, refresh token xoay vòng)                                             |
| i18n      | next-intl v4 – chỉ tiếng Việt, chuỗi tách theo domain trong `messages/vi/`                           |

## Chạy local

```bash
cp .env.example .env.local   # sửa NEXTAUTH_SECRET
npm install
npm run dev                  # http://localhost:3000
```

Cần backend `gymmaster-api` chạy tại `NEXT_PUBLIC_API_URL` (mặc định `http://localhost:8000/v1`).
Tài khoản hội viên demo (sau khi chạy `make seed`): `hoivien@gymmaster.vn` / `Admin@123`.

| Lệnh                          | Ý nghĩa                                       |
| ----------------------------- | --------------------------------------------- |
| `npm run dev`                 | Chạy môi trường phát triển                    |
| `npm run build` / `npm start` | Build & chạy production (output `standalone`) |
| `npm run typecheck`           | `tsc --noEmit`                                |
| `npm run lint`                | ESLint                                        |
| `npm run format`              | Prettier                                      |

Docker: `docker build --build-arg NEXT_PUBLIC_API_URL=http://localhost:8000/v1 -t gymmaster-web .`
rồi chạy với env `NEXTAUTH_URL`, `NEXTAUTH_SECRET`, `API_URL_INTERNAL`.

## Cấu trúc

```
messages/vi/<domain>.json      # chuỗi giao diện (common, auth, portal)
src/
├── proxy.ts                   # chặn /me/* khi chưa đăng nhập, đẩy người đã đăng nhập khỏi /login về /me
├── env.ts                     # biến môi trường validate bằng zod
├── app/
│   ├── (public)/              # trang chủ giới thiệu trung tâm, gói tập, lịch lớp, chi nhánh
│   ├── (auth)/                # /login, /forgot-password, /reset-password
│   ├── (member)/me/           # Cổng hội viên: Tổng quan, Mã QR, Gói tập, Lớp nhóm, Lịch PT, Tiến độ, Hóa đơn, Thông báo
│   └── api/auth/[...nextauth] # next-auth route handler
├── features/
│   ├── auth/                  # đăng nhập, quên/đặt lại/đổi mật khẩu
│   ├── landing/               # trang chủ công khai (/v1/public/*)
│   └── portal/                # cổng hội viên (/v1/member/* & /v1/notifications/*)
├── components/
│   ├── ui/                    # primitives (button, dialog, select, table, tabs, form…)
│   ├── common/                # PageHeader, ConfirmDialog, EmptyState, LoadError, StatusBadge…
│   └── providers/             # SessionProvider, QueryClient, SessionSync
├── guards/                    # authOptions (next-auth), getCachedSession
├── lib/                       # http.ts (client), http.server.ts, http.core.ts, format, utils
├── hooks/                     # use-auth, use-list-params, use-api-error, use-debounce
├── i18n/  store/  styles/  types/
```

## Luồng xác thực

1. `/login` → `signIn('credentials')` → next-auth gọi `POST /auth/login`, lưu `accessToken`, `refreshToken`, `user` (roles, permissions) trong cookie JWT mã hóa (không dùng localStorage), sau đó chuyển hướng vào `/me`.
2. Callback `jwt` tự gọi `POST /auth/refresh` khi access token còn < 30 giây; các request song song dùng chung một lần refresh (backend xoay vòng refresh token).
3. `apiClient` (`src/lib/http.ts`) gắn Bearer token, bóc envelope `{code, message, data}`, ném `ApiError`. Gặp `40101` → ép refresh rồi gửi lại 1 lần; gặp `40100`/`40103` → đăng xuất.
4. Đăng xuất → `events.signOut` gọi `POST /auth/logout` để thu hồi refresh token.
5. `refreshToken` không bao giờ xuống trình duyệt (callback `session` loại bỏ).

Quy tắc giao diện: xem [DESIGN_SYSTEM.md](./DESIGN_SYSTEM.md) và [AGENTS.md](./AGENTS.md).
