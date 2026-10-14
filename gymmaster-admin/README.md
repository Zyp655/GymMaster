# GymMaster Admin

Ứng dụng quản trị (SPA) cho hệ thống quản lý trung tâm Fitness **GymMaster** – dành cho Chủ trung tâm, Quản lý, Lễ tân, HLV.
Kiến trúc và giao diện kế thừa `strongbody_admin` (antd + Sider/Header, ActionBar/FilterBar/PageTitle, MobileList…), được hiện đại hóa:

| | strongbody_admin | gymmaster-admin |
|---|---|---|
| Build | CRA (react-scripts) | **Vite 6** |
| Ngôn ngữ | JavaScript | **TypeScript strict** (`erasableSyntaxOnly`, `verbatimModuleSyntax`) |
| Router | react-router v5 | **react-router-dom v6** (lazy + Suspense) |
| State | Redux + thunk cho mọi thứ | **Redux Toolkit** chỉ cho phiên đăng nhập, **TanStack Query v5** cho dữ liệu server |
| HTTP | axios, tự đọc `res.data` | axios instance duy nhất, **unwrap envelope**, `ApiError`, **tự refresh token (single-flight)** |
| Bảng | `useEffect` + state tay | `useTableQuery` + `DataTable`: phân trang/sort/lọc **đồng bộ URL** |
| i18n | react-intl | Chỉ tiếng Việt (chuỗi viết trực tiếp, antd `vi_VN`, dayjs `vi`) |

## Phiên bản chính

React 18.3 · antd 5.29 · @ant-design/icons 5 · @ant-design/charts 2 · react-router-dom 6.30 · @reduxjs/toolkit 2 · react-redux 9 · @tanstack/react-query 5 · axios 1 · dayjs 1.11 · Vite 6 · TypeScript 5.8 · ESLint 9 (flat) · Prettier 3.

## Chạy dự án

```bash
cp .env.example .env          # VITE_API_URL=http://localhost:8000/v1
npm install
npm run dev                   # http://localhost:3001
```

Tài khoản dev: `admin@gymmaster.vn` / `Admin@123` (OWNER).

| Script | Mô tả |
|---|---|
| `npm run dev` | Dev server cổng **3001** |
| `npm run build` | `typecheck` + build production vào `dist/` |
| `npm run preview` | Xem bản build ở cổng 3001 |
| `npm run typecheck` | `tsc` (app + vite config) |
| `npm run lint` | ESLint, không cho phép warning |
| `npm run format` | Prettier (singleQuote, printWidth 100) |
| `npm run verify:api` | Kiểm tra lõi HTTP client: mock server (refresh đồng thời, hết phiên, 5xx) + luồng thật với backend (login → me → CRUD chi nhánh → refresh → logout). Cần Node ≥ 22.18 |

## Docker

```bash
docker build -t gymmaster-admin --build-arg VITE_API_URL=https://api.example.com/v1 .
docker run -p 3001:80 gymmaster-admin
```

Build Vite → `nginx:alpine` (SPA fallback về `index.html`, cache dài cho `/assets/*`). `VITE_API_URL` được nhúng **lúc build**.

## Cấu trúc

```
src/
├── api/              # Mỗi resource một file: branch.ts, role.ts, auth.ts, upload.ts + types.ts
├── assets/css/       # custom.css (bố cục, login, chi tiết nhỏ)
├── components/       # Component dùng chung (xem CLAUDE.md)
├── constants/        # permissions.ts (PERM, ROLE, nhãn module), status.ts
├── container/        # MainApp (khôi phục phiên), AppRoutes (guard + layout), ProtectedRoute
├── hooks/            # useTableQuery, useUrlParams, usePermission, useWorkingBranch, useIsMobile...
├── layouts/          # AppLayout, AppSidebar, AppHeader, AppDrawer, BranchSwitcher, UserMenu...
├── menu/index.tsx    # Cây menu tĩnh + perm
├── routes/           # Trang: auth/Login, home/Dashboard, branches, roles, profile; index.tsx = bảng route
├── store/            # Redux Toolkit: authSlice (user, workingBranchId), session.ts (nối interceptor)
├── utils/            # api.ts / apiClient.ts (HTTP), token.ts, format.ts, errors.ts, feedback.ts
├── App.tsx           # Provider: Redux, QueryClient, ConfigProvider(vi_VN + theme), antd App, Router
├── theme.ts          # Navy #0A3150 + cam #F97316
└── main.tsx
scripts/verify-api-client.ts
```

## Xác thực & token

- `POST /auth/login` → lưu `access_token` (bộ nhớ + localStorage) và `refresh_token` (localStorage); `user` (roles, permissions, branch_ids) vào Redux.
- Khởi động app: `LoadingScreen` → `/auth/me` (refresh trước nếu chỉ còn refresh token).
- Interceptor: gặp **40101** → gọi `/auth/refresh` **một lần duy nhất**, mọi request đang chờ dùng chung promise đó (backend xoay vòng refresh token nên hai lần refresh song song sẽ làm hỏng phiên), rồi thử lại request. **40100/40103** hoặc refresh thất bại → xóa token, về `/login`. Lỗi 5xx/mất mạng → `message.error`.
- Tài khoản chỉ có vai trò `MEMBER` bị từ chối: “Tài khoản hội viên vui lòng dùng cổng hội viên” (refresh token vừa cấp được thu hồi ngay).

**Trade-off lưu token:** localStorage đọc được bằng JavaScript nên nếu trang bị XSS, kẻ tấn công có thể lấy refresh token. Phương án an toàn hơn là refresh token trong cookie `httpOnly; Secure; SameSite=Strict` do backend đặt (FE không chạm vào). Backend hiện trả refresh token trong body và app là SPA thuần (không có BFF) nên chọn localStorage, giảm rủi ro bằng: access token ngắn hạn (15 phút), refresh token xoay vòng + thu hồi khi đăng xuất/đổi mật khẩu, không dùng `dangerouslySetInnerHTML`, CSP ở nginx khi triển khai. Khi backend hỗ trợ cookie, chỉ cần sửa `src/utils/token.ts` và `refreshAccessToken` trong `apiClient.ts`.

## Chi nhánh làm việc

Header có ô chọn **chi nhánh làm việc** (chỉ các chi nhánh trong `branch_ids`; rỗng = toàn hệ thống, có thêm “Tất cả chi nhánh”). Giá trị lưu trong Redux + localStorage, được gửi kèm header `X-Branch-Id`; các trang danh sách lấy bằng `useWorkingBranch()` để lọc theo `branch_id`.

Hướng dẫn thêm module: xem **[CLAUDE.md](./CLAUDE.md)**.
