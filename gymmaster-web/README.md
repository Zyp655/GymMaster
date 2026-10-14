# gymmaster-web

Frontend quản trị của **GymMaster** – hệ thống số hóa quy trình quản lý trung tâm Fitness.
Kiến trúc dựa trên `strongbody-web`, có cải tiến: HTTP client tập trung (tự refresh token), chỉ dùng TanStack Query, bảng dữ liệu dùng chung.

## Công nghệ

| Mục       | Thư viện                                                                                             |
| --------- | ---------------------------------------------------------------------------------------------------- |
| Framework | Next.js 16 (App Router, Turbopack), React 19, TypeScript strict                                      |
| UI        | Tailwind CSS v4 (token trong `@theme`), primitives kiểu shadcn/ui trên `radix-ui`, Heroicons, sonner |
| Dữ liệu   | TanStack Query v5, TanStack Table v9                                                                 |
| Form      | react-hook-form + zod                                                                                |
| Xác thực  | next-auth v4 (Credentials, JWT, refresh token xoay vòng)                                             |
| i18n      | next-intl v4 – chỉ tiếng Việt, chuỗi tách theo domain trong `messages/vi/`                           |
| State UI  | zustand (thu gọn sidebar)                                                                            |

## Chạy local

```bash
cp .env.example .env.local   # sửa NEXTAUTH_SECRET
npm install
npm run dev                  # http://localhost:3000
```

Cần backend `gymmaster-api` chạy tại `NEXT_PUBLIC_API_URL` (mặc định `http://localhost:8000/v1`).
Tài khoản seed: `admin@gymmaster.vn` / `Admin@123`.

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
messages/vi/<domain>.json      # chuỗi giao diện (common, auth, layout, dashboard, branch, role)
src/
├── proxy.ts                   # chặn /admin/* khi chưa đăng nhập, đẩy người đã đăng nhập khỏi /login
├── env.ts                     # biến môi trường validate bằng zod
├── app/
│   ├── (auth)/login/          # trang đăng nhập
│   ├── (admin)/admin/         # layout quản trị + dashboard, branches, roles (+ loading/error)
│   └── api/auth/[...nextauth] # next-auth route handler
├── features/<module>/         # api.ts · queries.ts · schema.ts · types.ts · components/
├── components/
│   ├── ui/                    # primitives (button, dialog, select, table, form…)
│   ├── common/                # DataTable, PageHeader, ConfirmDialog, EmptyState, LoadError, PermissionGate, StatusBadge…
│   ├── layout/                # AdminShell, SidebarNav, UserMenu, nav-config
│   └── providers/             # SessionProvider, QueryClient, SessionSync
├── guards/                    # authOptions (next-auth), getCachedSession
├── lib/                       # http.ts (client), http.server.ts, http.core.ts, permissions, format, utils
├── hooks/                     # use-auth, use-list-params (bộ lọc trên URL), use-api-error, use-debounce
├── i18n/  store/  styles/  types/
```

## Luồng xác thực

1. `/login` → `signIn('credentials')` → next-auth gọi `POST /auth/login`, lưu `accessToken`, `refreshToken`, `user` (roles, permissions) trong cookie JWT mã hóa (không dùng localStorage).
2. Callback `jwt` tự gọi `POST /auth/refresh` khi access token còn < 30 giây; các request song song dùng chung một lần refresh (backend xoay vòng refresh token).
3. `apiClient` (`src/lib/http.ts`) gắn Bearer token, bóc envelope `{code, message, data}`, ném `ApiError`. Gặp `40101` → ép refresh rồi gửi lại 1 lần; gặp `40100`/`40103` → đăng xuất.
4. Đăng xuất → `events.signOut` gọi `POST /auth/logout` để thu hồi refresh token.
5. `refreshToken` không bao giờ xuống trình duyệt (callback `session` loại bỏ).

## Thêm một module mới (checklist)

1. `messages/vi/<module>.json` và đăng ký trong `src/i18n/messages.ts`.
2. `src/features/<module>/{types,api,queries,schema}.ts` + `components/`.
3. Trang `src/app/(admin)/admin/<module>/page.tsx`: kiểm tra quyền phía server (`hasPermission` → `<Forbidden />`).
4. Thêm mã quyền vào `src/lib/permissions.ts`; bật mục menu trong `components/layout/nav-config.ts` (`enabled: true`).
5. Đủ 4 trạng thái loading / empty / error / success; xóa phải qua `ConfirmDialog`.
6. `npm run typecheck && npm run lint`.

Quy tắc giao diện: xem [DESIGN_SYSTEM.md](./DESIGN_SYSTEM.md).
