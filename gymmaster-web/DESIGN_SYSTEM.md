# GymMaster – Design System

Quy tắc giao diện cho `gymmaster-web` (dựa trên DESIGN_SYSTEM của strongbody-web, rút gọn cho trang quản trị).
Token định nghĩa trong `src/styles/globals.css` (`@theme`, Tailwind v4 – không có `tailwind.config.js`).

## 1. Nguyên tắc chung

- **Nền xám, card trắng**: trang dùng nền `bg-canvas`; nội dung đặt trong card `rounded-2xl border border-line bg-surface`. Card không có shadow (chỉ viền).
- **Không lồng card có viền trong card**. Nhóm con bên trong card dùng `rounded-xl bg-canvas p-4`.
- **Không dùng giá trị tùy ý** (`text-[15px]`, `bg-[#fff]`, `p-[13px]`, `style={{ color }}`) trong code tính năng. Ngoại lệ duy nhất: primitives trong `src/components/ui/` (VD: `max-h-[calc(100dvh-2rem)]`).
- **Không dùng `dark:`** – ứng dụng chỉ có giao diện sáng.
- **Một font**: Be Vietnam Pro (hỗ trợ đầy đủ tiếng Việt), nạp qua `next/font`.
- **Một bộ icon**: `@heroicons/react` (24/outline cho điều hướng & nút, 20/solid hoặc 16/solid cho icon nhỏ). Cấm `lucide-react` (ESLint chặn).
- **Không hard-code chữ trong component**: mọi chuỗi nằm trong `messages/vi/<domain>.json`.

## 2. Màu

| Nhóm        | Token                                                                         | Dùng cho                                            |
| ----------- | ----------------------------------------------------------------------------- | --------------------------------------------------- |
| Brand (cam) | `brand-25 … brand-900` (chính: `brand-500` #F97316)                           | Nút chính, mục menu đang chọn, điểm nhấn            |
| Ink         | `ink-700 … ink-950`                                                           | Sidebar, panel thương hiệu ở trang đăng nhập        |
| Chữ         | `text-primary` (#111827), `text-secondary`, `text-tertiary`, `text-disabled`  | **`text-primary` là màu đen, không phải màu brand** |
| Nền         | `bg-canvas` (nền trang), `bg-surface` (card), `bg-subtle` (hover, nhóm phụ)   |                                                     |
| Viền        | `border-line`, `border-line-strong` (input), `border-line-subtle` (dòng bảng) |                                                     |
| Trạng thái  | `success`, `warning`, `danger`, `info` + biến thể `-50` làm nền nhạt          | Badge, toast, cảnh báo                              |

## 3. Chữ

Thang chữ ngữ nghĩa: `text-display` 36 · `text-h1` 28 · `text-h2` 22 (tiêu đề trang) · `text-h3` 18 (tiêu đề dialog) · `text-h4` 16 (tiêu đề card) · `text-body-lg` 16 · `text-body` 14 (mặc định) · `text-caption` 12.
Độ đậm: `font-normal`, `font-medium`, `font-semibold`. Không dùng `font-bold`.

## 4. Bo góc & đổ bóng

- Nút: `rounded-button` (10px) · Input/Select: `rounded-xl` · Card/Badge: `rounded-2xl` · Dialog: `rounded-3xl` · Chip/avatar: `rounded-full`.
- Bóng: card **không** bóng; dropdown/popover/dialog `shadow-lg`.

## 5. Thành phần

- **Button** (`components/ui/button.tsx`): `primary` (thêm mới, lưu), `outline` (sửa, hủy), `ghost` (thao tác phụ, icon trong bảng), `danger` (xác nhận xóa), `soft`, `dark`, `link`. Kích thước `sm` h-9, `md` h-10, `lg` h-12, `icon`, `icon-sm`.
- **Quy ước thao tác**: thêm = `primary` + `PlusIcon`; sửa = menu "…" → `PencilSquareIcon`; xóa = mục `danger` trong menu → **luôn mở `ConfirmDialog`**.
- **Badge trạng thái**: dùng `StatusBadge` / `Badge` variant `success|warning|danger|info|neutral|brand`.
- **Bảng**: luôn dùng `DataTable` (phân trang & sắp xếp server, ô tìm kiếm có debounce, bộ lọc trên URL).
- **Form**: react-hook-form + zod (`features/<module>/schema.ts`). Nhãn bắt buộc có dấu `*` (`<FormLabel required>`). Nút submit bị khóa khi đang gửi; form không hợp lệ thì không gửi.

## 6. Trạng thái màn hình (bắt buộc)

Mọi vùng dữ liệu phải có đủ 4 trạng thái:

1. **Loading** – skeleton giống bố cục thật (`Skeleton`, `loading.tsx`).
2. **Empty** – `EmptyState` (phân biệt "chưa có dữ liệu" và "không có kết quả lọc").
3. **Error** – `LoadError` có nút "Thử lại".
4. **Success** – dữ liệu.

Mutation: toast thành công/thất bại (sonner), lỗi map theo mã (`useToastApiError`).

## 7. Responsive

Breakpoints: `sm` 640 · `md` 768 · `lg` 1024 (sidebar hiện từ đây) · `xl` 1280 · `2xl` 1536.
Kiểm tra ở 390px (mobile) và 1440px (desktop). Dùng `min-w-0`, `flex-wrap`, `truncate`/`line-clamp` để chữ tiếng Việt dài không vỡ bố cục.
