# CLAUDE.md – gymmaster-admin

Hướng dẫn cho người/agent làm việc trong `gymmaster-admin` (SPA quản trị GymMaster). Đọc hết trước khi viết code.

## Tổng quan

- Vite 6 + React 18 + **TypeScript strict** · antd 5 (locale `vi_VN`) · react-router-dom 6 · Redux Toolkit (CHỈ phiên đăng nhập) · TanStack Query 5 (mọi dữ liệu server) · axios · dayjs (locale `vi`).
- Backend: `gymmaster-api` – `VITE_API_URL` (mặc định `http://localhost:8000/v1`). Hợp đồng API: `../docs/api/*.md`, Swagger `http://localhost:8000/swagger`. Mã quyền: `../gymmaster-api/internal/consts/permission.go` (đã sao chép vào `src/constants/permissions.ts` → dùng `PERM.X`).
- Mẫu kiến trúc/giao diện: `strongbody_admin` (Card → [PageTitle | ActionBar] → FilterBar → Table, modal form, MobileList).
- **Trang mẫu để sao chép:** `src/routes/branches/` (list phân trang server + modal form + mobile) và `src/routes/roles/` (list không phân trang + form phức tạp).

## Lệnh

```bash
npm run dev         # cổng 3001
npm run typecheck   # PHẢI sạch
npm run lint        # PHẢI sạch (--max-warnings=0)
npm run build       # typecheck + vite build
npm run verify:api  # kiểm tra interceptor + luồng API thật (cần backend ở 8000)
```

Trước khi báo xong: `npm run typecheck && npm run lint && npm run build`. Không thêm dependency khi chưa thật sự cần (antd/icons/charts/dayjs/query đã đủ cho hầu hết màn hình).

## Quy ước BẮT BUỘC

1. **Chỉ tiếng Việt**, viết chuỗi trực tiếp trong JSX (không i18n). Ngày `DD/MM/YYYY`, giờ `DD/MM/YYYY HH:mm`, tiền `1.500.000 ₫` – dùng `src/utils/format.ts` / `DateText` / `MoneyText` / `MoneyInput`.
2. **Dùng component dùng chung** (bảng dưới) – không tự viết lại thanh lọc, phân trang, xác nhận xóa...
3. **Mọi thao tác xóa/hủy/hoàn tiền phải xác nhận** (`ConfirmButton`, `ActionBar.onDelete`, hoặc `modal.confirm`).
4. **Đủ trạng thái**: loading (Table/Card `loading`, Skeleton), rỗng (`emptyText` tiếng Việt), lỗi (DataTable tự hiện `Result` + "Thử lại"; nơi khác dùng `getErrorMessage`).
5. **Responsive**: trang danh sách có bản điện thoại (`useIsMobile()` → `MobileList`); form dùng `Row/Col` với `xs={24} sm={...}`; Modal không cố định chiều cao.
6. **Quyền**: route + menu khai báo `perm`; nút thêm/sửa/xóa ẩn theo quyền (`addPerm`, `deletePerm`, `perm` của ConfirmButton, `PermissionGate`, `usePermission().can`). Backend vẫn kiểm tra cuối cùng.
7. **Không gọi axios trực tiếp trong component** – luôn qua `src/api/<resource>.ts`. Không `useEffect` + `useState` để tải dữ liệu – dùng TanStack Query.
8. Thông báo trong component: `const { message, modal } = App.useApp()` (không dùng `message` static). Ngoài component: `feedback` (`src/utils/feedback.ts`).
9. Lỗi API: `showError(error, 'Không thể ...')` (tự bỏ qua nếu interceptor đã báo 5xx); lỗi validate/trùng → `setFormError(form, error, { code: /mã/i, ... })` để hiện ngay dưới ô nhập.
10. Màu: navy `#0A3150` (primary – nút thường dùng `type="primary"`), cam `#F97316` cho CTA nổi bật: `<Button color="orange" variant="solid">` (ActionBar đã dùng cho "Thêm mới").
11. Code style: Prettier (singleQuote, printWidth 100), import alias `@/` → `src/`, `import type` cho kiểu (bắt buộc bởi `verbatimModuleSyntax`), không `enum`/namespace (`erasableSyntaxOnly`) → dùng object `as const`.
12. KHÔNG sửa file dùng chung của người khác theo cách phá vỡ API (chỉ thêm prop tùy chọn). Thêm trạng thái mới vào `src/constants/status.ts` (chỉ thêm, không đổi nhãn có sẵn).

## Kiến trúc

```
main.tsx → App.tsx (Redux Provider · QueryClientProvider · ConfigProvider vi_VN+theme · antd <App> · BrowserRouter)
        → container/MainApp.tsx   khôi phục phiên (/auth/me) – LoadingScreen; routes /login, /app/*, / → /app/dashboard
        → container/AppRoutes.tsx chưa đăng nhập → /login; bọc layouts/AppLayout; mỗi route bọc ProtectedRoute(perm) → 403
        → routes/index.tsx        bảng route { path, element (lazy), perm }
```

- **HTTP** – `src/utils/api.ts` (instance duy nhất, cấu hình) dựa trên `src/utils/apiClient.ts` (lõi thuần, test được bằng Node):
  - Kết quả đã **unwrap envelope** → nhận thẳng `data`. Lỗi luôn là `ApiError { code, status, message }` (`isApiError`, `ErrorCode`).
  - 40101 → refresh single-flight rồi thử lại; 40100/40103 → hết phiên → về `/login`; 5xx/mất mạng → toast (tắt bằng `{ silent: true }`).
  - Tùy chọn axios mở rộng: `skipAuthRefresh`, `silent` (xem `src/types/axios.d.ts`).
  - Header `X-Branch-Id` = chi nhánh làm việc (tự gắn).
- **Phiên** – `store/authSlice.ts`: `user: AuthUser | null`, `workingBranchId`, thunk `login`, `logout`, `bootstrapAuth`. Hook: `useAppSelector`, `useAppDispatch`.
- **Quyền** – `usePermission()` → `{ can(perm|perm[]) /* anyOf */, canAll, isOwner, user }`. OWNER luôn `true`.
- **Chi nhánh làm việc** – `useWorkingBranch()` → `{ branchId /* null = tất cả */, setBranchId }`. Danh sách chi nhánh được phép: `useBranchOptions()` / `<BranchSelect />`.
- **Query keys** – mỗi API module có `key` (vd `branchApi.key = 'branches'`). List: `[key, 'list', params]`; mutation xong `invalidateQueries({ queryKey: [key] })` (làm mới cả bản desktop lẫn mobile).

## Component & hook dùng chung (`src/components`, `src/hooks`)

| Tên | API chính |
|---|---|
| `PageTitle` | `title`, `subtitle?`, `back?: boolean \| string`, `onBack?`, `breadcrumb?: {title, path?}[]`, `extra?` (tự đặt `document.title`) |
| `ActionBar` | `onAdd`, `addPerm`, `addText='Thêm mới'` (nút cam) · `onDelete` (gọi SAU khi confirm, có thể async), `deletePerm`, `selectedCount`, `deleteConfirmTitle/Content` · `onRefresh`, `refreshing` · `showAdd/showDelete/showRefresh` · `children` (nút thêm) |
| `FilterBar` | `searchPlaceholder`, `textSearch=true`, `colTextSearch=6`, `filters: { name, placeholder, options:{label,value}[], mode?: 'multiple', col? }[]`, `children`. Ghi thẳng lên URL (`keyword`, `<name>`), về trang 1, có nút "Xóa lọc" |
| `useTableQuery` | `useTableQuery<T, P>({ queryKey, queryFn, filterKeys?, numberKeys?, defaults?: {limit, order_by, order_dir}, extraParams? })` → `{ params, items, total, isFetching, isError, error, refetch, setFilters, setPage, setSort, sortOrderOf }`. URL: `page,limit,keyword,order_by,order_dir` + filterKeys |
| `DataTable` | `table` (từ useTableQuery), `columns` (cột `sorter: true` = sort server theo `dataIndex`/`key`), `rowSelection?`, `emptyText?` + mọi prop Table. Mặc định `size="small"`, `scroll.x`, `rowKey="id"`, phân trang tiếng Việt, trạng thái lỗi có "Thử lại" |
| `useRowSelection<T>()` | `{ keys, rows, clear, rowSelection }` |
| `MobileList` | `title`, `queryKey`, `fetchPage({page,limit,keyword,sort,filters}) → Promise<Page<T>>`, `row: {title, subtitle?, meta?, tags?, thumb?, avatar?, toggle?}`, `onOpen?`, `onAdd?`, `addPerm?`, `actions?(r) → {key,label,icon,danger,onClick}[]`, `details?(r) → [label, value][]`, `sorts?`, `filters?: {name,label,type?: 'select'\|'text'\|'daterange', options?}[]`, `stats?` |
| `useIsMobile()` | `true` khi < 768px |
| `StatusTag` | `status`, `map?` (ghi đè). Preset trong `constants/status.ts` (`STATUS_PRESETS`, `ACTIVE_STATUS_OPTIONS`) |
| `MoneyText` / `MoneyInput` | `value`, `colored?`, `strong?` / InputNumber phân cách nghìn, hậu tố ₫ (dùng trong `Form.Item`) |
| `DateText` | `value`, `withTime?`, `format?`, `relative=true` (tooltip "3 giờ trước") |
| `ImageUpload` | Form control: `value: string \| string[]`, `maxCount=1`, `maxSizeMB=5` – upload `POST /admin/uploads` (field `file`) → lưu `url` |
| `BranchSelect` | Select chi nhánh trong phạm vi user; `allowAll?`, `activeOnly?` + mọi prop Select |
| `PermissionGate` / `usePermission` | `<PermissionGate perm={PERM.X} all? fallback?>` |
| `ConfirmButton` | Button + hộp xác nhận: `confirmTitle`, `confirmContent?`, `onConfirm` (async ok), `perm?`, `danger` |
| `StatCard`, `SectionHeader` | KPI dashboard (`label`, `value`, `currency?`, `caption?`, `delta?`, `progress?`, `icon?`, `onClick?`); token màu ở `StatCard/tokens.ts` |
| `LoadingScreen`, `RouteFallback`, `ComingSoon`, `Forbidden`, `NotFound`, `ErrorBoundary`, `ChangePasswordModal` | |
| `utils/format.ts` | `formatMoney`, `formatNumber`, `formatDate`, `formatDateTime`, `formatPhone`, `toApiDate`, `normalizeText` (tìm không dấu), hằng `DATE_FORMAT`... |
| `utils/errors.ts` | `getErrorMessage`, `showError`, `setFormError`, `isFormError` |
| `hooks/useNotifications` | Nguồn dữ liệu chuông thông báo (đang trả rỗng – module thông báo thay thân hook) |

## Công thức thêm một module (ví dụ: Phòng tập `rooms`, perm `room.view`/`room.manage`)

### 1. Kiểu + API module – `src/api/room.ts`

Thêm kiểu vào `src/api/types.ts` (hoặc trong file module nếu chỉ module đó dùng), khớp đúng JSON backend (snake_case).

```ts
import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

export interface Room { id: number; branch_id: number; code: string; name: string; capacity: number; status: 'ACTIVE' | 'INACTIVE'; created_at: string; updated_at: string }
export type RoomInput = Omit<Room, 'id' | 'created_at' | 'updated_at'>;
export interface RoomListParams extends ListParams { status?: string; branch_id?: number }

class RoomApi {
  readonly key = 'rooms';
  list(params: RoomListParams = {}) { return http.get<Page<Room>>('/admin/rooms', { params }); }
  get(id: number) { return http.get<Room>(`/admin/rooms/${id}`); }
  create(data: RoomInput) { return http.post<Room>('/admin/rooms', data); }
  update(id: number, data: RoomInput) { return http.put<Room>(`/admin/rooms/${id}`, data); }
  remove(id: number) { return http.delete(`/admin/rooms/${id}`); }
  options(branchId?: number) { return http.get<Room[]>('/admin/rooms/options', { params: { branch_id: branchId } }); }
}
export const roomApi = new RoomApi();
export default roomApi;
```

### 2. Mutation – `src/routes/rooms/queries.ts(x)`

Sao chép `routes/branches/queries.tsx`: `useSaveRoom` (create/update + `invalidateQueries({ queryKey: [roomApi.key] })`), `useDeleteRooms` (xóa nhiều bằng `Promise.allSettled`, báo mục lỗi).

### 3. Trang danh sách – `src/routes/rooms/index.tsx`

Sao chép `routes/branches/index.tsx` và giữ nguyên bố cục:

```tsx
const { branchId } = useWorkingBranch();               // nếu dữ liệu thuộc chi nhánh
const table = useTableQuery<Room, RoomListParams>({
  queryKey: [roomApi.key, 'list'],
  queryFn: (p) => roomApi.list(p),
  filterKeys: ['status'],
  defaults: { order_by: 'id', order_dir: 'DESC' },
  extraParams: { branch_id: branchId ?? undefined },   // đổi chi nhánh ở header → tự tải lại
});

<Card variant="borderless" className="page-card">
  <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
    <Col><PageTitle title="Phòng tập" subtitle="..." /></Col>
    <Col><ActionBar onAdd={...} addPerm={PERM.ROOM_MANAGE} onDelete={...} deletePerm={PERM.ROOM_MANAGE}
                    selectedCount={selection.keys.length} onRefresh={() => table.refetch()} refreshing={table.isFetching} /></Col>
  </Row>
  <FilterBar searchPlaceholder="Tìm theo mã, tên..." filters={[{ name: 'status', placeholder: 'Trạng thái', options: ACTIVE_STATUS_OPTIONS }]} />
  <DataTable<Room> table={table} columns={columns} rowSelection={selection.rowSelection} emptyText="Chưa có phòng nào" />
  <RoomForm open={...} record={...} onClose={...} />
</Card>
```

Cột: tên là `Button type="link"` mở modal sửa; cột cuối `fixed: 'right'` gồm nút Sửa + `ConfirmButton` xóa; cột ngày dùng `DateText`; trạng thái dùng `StatusTag`; tiền dùng `MoneyText`. Chỉ đặt `sorter: true` cho cột backend cho phép sắp xếp (whitelist `orderable` trong service).

Export mặc định chọn bản theo thiết bị: `return useIsMobile() ? <RoomsMobile /> : <RoomsDesktop />;` – bản mobile sao chép `BranchesMobile.tsx`.

### 4. Form modal – `src/routes/rooms/RoomForm.tsx`

Sao chép `BranchForm.tsx`: antd `Form` layout vertical trong `Modal` (`forceRender`, `resetFields` + `setFieldsValue` khi mở), **rule validate giống tag `v:` của backend** (đọc `gymmaster-api/api/<m>/v1/*.go`), `readOnly` khi thiếu quyền sửa, submit qua mutation, `onError` → `setFormError(...) || showError(...)`. Ngày/giờ: DatePicker/TimePicker `format` theo `utils/format.ts`, gửi API bằng `toApiDate` hoặc `.format('HH:mm')`.

### 5. Đăng ký route + menu (cùng `perm`)

- `src/routes/index.tsx`: thay dòng `soon('Phòng tập')` bằng
  `const Rooms = lazy(() => import('./rooms'));` và `{ path: 'rooms', element: page(Rooms), perm: [PERM.ROOM_VIEW] }`.
  Trang chi tiết: `{ path: 'members/:id', element: page(MemberDetail), perm: [...] }`.
- `src/menu/index.tsx`: lá menu tương ứng đã có sẵn; kiểm tra `perm` trùng với route. Module mới hoàn toàn → thêm lá vào nhóm phù hợp.

### 6. Kiểm tra

`npm run typecheck && npm run lint && npm run build`, chạy `npm run dev` và thử với backend thật: danh sách, lọc, sort, phân trang (URL thay đổi, F5 giữ nguyên), thêm/sửa/xóa (có confirm), lỗi trùng mã hiện dưới ô, màn hình < 768px.

## Ghi chú

- Danh sách không phân trang (vd `/admin/roles`): dùng antd `Table` + lọc client-side theo `useUrlParams()` (xem `routes/roles/index.tsx`).
- Dropdown phụ thuộc dữ liệu khác: dùng endpoint `/options` của backend qua `useQuery({ queryKey: [xxxApi.key, 'options', ...] })`.
- Không lưu dữ liệu server vào Redux. Không tự gọi refresh token – interceptor lo.
- Chuông thông báo: thay thân `src/hooks/useNotifications.ts`, giữ nguyên interface.
