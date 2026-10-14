# API – POS bán hàng & tồn kho theo chi nhánh (migration 000009)

Base URL (dev): `http://localhost:8000/v1`. Mọi endpoint cần `Authorization: Bearer <access_token>`.
Envelope, mã lỗi, phân trang: xem `sprint-01-auth-rbac-branch.md`. Tóm tắt:

- Thành công: `{ "code": 0, "message": "OK", "data": ... }`.
- Lỗi: `code` 5 chữ số, HTTP = code/100, `message` tiếng Việt hiển thị thẳng cho người dùng.
  Lỗi validate tag: `code = 51` (HTTP 400). Nghiệp vụ: `40000` (400), `40300` (403), `40400` (404), `40900` (409).
- Ngày `"YYYY-MM-DD"` (giờ Việt Nam). Thời điểm RFC3339. Tiền là `number` VND (giá bán/thu tiền làm tròn đến đồng;
  **giá vốn** `cost_price`/`unit_cost` có thể lẻ 2 chữ số thập phân).
- Danh sách phân trang: `data = {items, total, total_page, current_page, limit}`; query chung `page`, `limit` (≤100), `keyword`, `order_by`, `order_dir`.

**Phạm vi chi nhánh.** OWNER thấy toàn hệ thống. Nhân viên gắn chi nhánh chỉ xem/thao tác kho, bán hàng của chi nhánh mình
(ngoài phạm vi → `40300`). Danh mục, nhà cung cấp, sản phẩm là dữ liệu dùng chung toàn hệ thống.

**Quyền** (seed sẵn): OWNER/MANAGER có tất cả; RECEPTIONIST có `product.view`, `inventory.view`, `pos.sell`.

| Quyền | Ý nghĩa |
|---|---|
| `product.view` | Xem sản phẩm, danh mục, nhà cung cấp |
| `product.manage` | Thêm/sửa/xóa sản phẩm, danh mục, nhà cung cấp |
| `inventory.view` | Xem tồn kho, thẻ kho, phiếu kho, tồn thấp |
| `inventory.import` | Nhập hàng |
| `inventory.transfer` | Điều chuyển giữa chi nhánh |
| `inventory.adjust` | Kiểm kê, điều chỉnh tồn |
| `pos.sell` | Bán hàng tại quầy, trả hàng |

**Audit log** (`audit_logs`): `product` (`CREATE`, `UPDATE`, `DELETE`, `PRICE_CHANGE` khi đổi giá vốn/giá bán/giá hội viên),
`product_category`, `supplier` (`CREATE/UPDATE/DELETE`), `stock_voucher` (`IMPORT`, `TRANSFER`, `ADJUST`; `after.voucher_no` = mã phiếu),
`pos_order` (`SALE`, `RETURN`; `entity_id` = invoices.id), cùng audit `invoice` (`CREATE`, `PAYMENT`, `REFUND`, `VOID`) do InvoiceService ghi.

## Danh sách endpoint

| Method | Path | Quyền | Mô tả |
|---|---|---|---|
| GET | `/admin/product-categories` | `product.view` hoặc `pos.sell` | Danh mục (mảng) |
| POST | `/admin/product-categories` | `product.manage` | Thêm danh mục |
| PUT | `/admin/product-categories/{id}` | `product.manage` | Sửa danh mục |
| DELETE | `/admin/product-categories/{id}` | `product.manage` | Xóa (chặn nếu còn sản phẩm) |
| GET | `/admin/suppliers` | `product.view` | Danh sách NCC (phân trang) |
| GET | `/admin/suppliers/options` | `product.view` hoặc `inventory.import` | NCC cho dropdown |
| GET | `/admin/suppliers/{id}` | `product.view` | Chi tiết NCC |
| POST | `/admin/suppliers` | `product.manage` | Thêm NCC |
| PUT | `/admin/suppliers/{id}` | `product.manage` | Sửa NCC |
| DELETE | `/admin/suppliers/{id}` | `product.manage` | Xóa mềm NCC |
| GET | `/admin/products` | `product.view` | Danh sách sản phẩm (+ tồn tại `branch_id`) |
| GET | `/admin/products/{id}` | `product.view` | Chi tiết + tồn theo chi nhánh |
| POST | `/admin/products` | `product.manage` | Thêm sản phẩm |
| PUT | `/admin/products/{id}` | `product.manage` | Sửa sản phẩm |
| DELETE | `/admin/products/{id}` | `product.manage` | Xóa mềm (chặn nếu còn tồn) |
| POST | `/admin/inventory/imports` | `inventory.import` | Phiếu nhập `NK…` |
| POST | `/admin/inventory/transfers` | `inventory.transfer` | Phiếu điều chuyển `DC…` |
| POST | `/admin/inventory/adjustments` | `inventory.adjust` | Phiếu kiểm kê `KK…` |
| GET | `/admin/inventory/stock` | `inventory.view` | Tồn hiện tại + giá trị tồn |
| GET | `/admin/inventory/transactions` | `inventory.view` | Thẻ kho |
| GET | `/admin/inventory/vouchers/{voucher_no}` | `inventory.view` | Chi tiết phiếu kho |
| GET | `/admin/inventory/low-stock` | `inventory.view` | Các dòng tồn dưới mức tối thiểu |
| GET | `/admin/pos/products` | `pos.sell` | Sản phẩm đang bán + tồn tại chi nhánh |
| GET | `/admin/pos/products/by-barcode/{code}` | `pos.sell` | Tìm theo mã vạch (hoặc SKU) |
| POST | `/admin/pos/orders` | `pos.sell` | Bán hàng (hóa đơn + trừ kho + thu đủ) |
| GET | `/admin/pos/orders` | `pos.sell` hoặc `invoice.view` | Danh sách hóa đơn POS |
| GET | `/admin/pos/orders/{invoice_id}` | `pos.sell` hoặc `invoice.view` | Chi tiết đơn + số lượng còn trả được |
| POST | `/admin/pos/orders/{invoice_id}/returns` | `pos.sell` | Trả hàng (hoàn kho + hoàn tiền) |
| GET | `/admin/pos/summary` | `pos.sell` hoặc `invoice.view` | Tổng kết bán hàng trong ngày |

## Enum

| Trường | Giá trị | Nhãn gợi ý |
|---|---|---|
| `product.status` | `ACTIVE` · `INACTIVE` | Đang kinh doanh · Ngừng kinh doanh |
| `transaction.type` | `IMPORT` · `SALE` · `SALE_RETURN` · `TRANSFER_IN` · `TRANSFER_OUT` · `ADJUST` | Nhập hàng · Bán hàng · Trả hàng/hủy đơn · Nhận điều chuyển · Xuất điều chuyển · Kiểm kê |
| `voucher.kind` | `IMPORT` · `TRANSFER` · `ADJUST` · `SALE` · `SALE_RETURN` | Phiếu nhập · Phiếu điều chuyển · Phiếu kiểm kê · Hóa đơn bán · Phiếu trả hàng |
| `method` | `CASH` · `TRANSFER` · `CARD` | Tiền mặt · Chuyển khoản · Thẻ |

**Mã phiếu**: tiền tố + `yymmdd` (giờ VN) + số thứ tự 4 chữ số trong ngày, ví dụ `NK2610140001`.
`NK` nhập kho · `DC` điều chuyển · `KK` kiểm kê · `TH` trả hàng. Dòng bán hàng dùng `voucher_no` = số hóa đơn (`HD…`),
dòng hoàn kho khi **hủy** hóa đơn cũng dùng số hóa đơn đó.

---

## 1. Danh mục sản phẩm – `/admin/product-categories`

### `Category`
```json
{ "id": 1, "name": "Đồ uống", "sort_order": 1, "product_count": 4,
  "created_at": "2026-10-14T09:56:01+07:00", "updated_at": "2026-10-14T09:56:01+07:00" }
```
- `GET` trả **mảng** (không phân trang), sắp theo `sort_order, name`. Query `keyword` (tùy chọn).
- `POST` / `PUT {id}` body: `{ "name": "Đồ uống", "sort_order": 1 }` (tên bắt buộc ≤100, duy nhất không phân biệt hoa thường).
- `DELETE {id}` → `data: {}`.

| Lỗi | message |
|---|---|
| 40900 | Tên danh mục đã tồn tại |
| 40900 | Danh mục đang có 4 sản phẩm, hãy chuyển hoặc xóa sản phẩm trước |
| 40400 | Không tìm thấy danh mục sản phẩm |

## 2. Nhà cung cấp – `/admin/suppliers`

### `Supplier`
```json
{ "id": 1, "name": "Công ty TNHH Dinh Dưỡng Thể Hình Sài Gòn", "phone": "02838123456",
  "email": "sales@dinhduongthehinh.vn", "address": "45 Nguyễn Thị Minh Khai, Quận 1, TP.HCM",
  "note": "Thực phẩm bổ sung, phụ kiện tập", "created_at": "...", "updated_at": "..." }
```
- `GET /admin/suppliers` – phân trang; `keyword` tìm tên/SĐT/email; `order_by`: `id`, `name`, `created_at`.
- `GET /admin/suppliers/options` → `[{ "id": 1, "name": "..." }]`.
- `POST` / `PUT {id}` body: `{ name*, phone, email, address, note }` (phone `^[0-9+ ]{8,20}$`, email hợp lệ, address ≤255, note ≤255).
- `DELETE {id}` – xóa mềm (thẻ kho cũ vẫn giữ tên NCC).

## 3. Sản phẩm – `/admin/products`

### `Product`
```json
{
  "id": 1, "sku": "AQUA-500", "barcode": "8934588012112", "name": "Nước suối Aquafina 500ml",
  "category_id": 1, "category_name": "Đồ uống", "unit": "chai",
  "cost_price": 5500, "sell_price": 10000, "member_price": 9000,
  "image_url": "http://localhost:8000/uploads/2026/10/....png",
  "min_stock": 24, "status": "ACTIVE",
  "total_stock": 216,
  "stock": 120, "low_stock": false,
  "created_at": "...", "updated_at": "..."
}
```
- `cost_price`: **giá vốn bình quân gia quyền** (tự cập nhật khi nhập hàng – mục 4).
- `member_price`: giá cho hội viên; `null` = hội viên mua theo giá bán.
- `total_stock`: tổng tồn mọi chi nhánh. `stock`, `low_stock` chỉ có khi truyền `branch_id` (ngược lại `null`).
- `low_stock` = `min_stock > 0 && stock < min_stock`.

### `ProductDetail` (`GET /{id}`, response của tạo/sửa) = `Product` +
```json
{ "stocks": [ { "branch_id": 1, "branch_name": "GymMaster Cầu Giấy", "quantity": 120, "low_stock": false },
              { "branch_id": 3, "branch_name": "GymMaster Quận 1", "quantity": 96, "low_stock": false } ] }
```
(chỉ chi nhánh trong phạm vi người dùng và đã từng có dòng tồn.)

### GET `/admin/products`
Query: `category_id`, `status` (`ACTIVE|INACTIVE`), `branch_id` (kèm `stock`/`low_stock`), `low_stock=true` (cần `branch_id`),
`keyword` (tên / SKU / mã vạch), `order_by` (`id`, `sku`, `name`, `sell_price`, `cost_price`, `min_stock`, `status`, `created_at`).

### POST `/admin/products` · PUT `/admin/products/{id}`
```json
{
  "sku": "AQUA-500", "barcode": "8934588012112", "name": "Nước suối Aquafina 500ml",
  "category_id": 1, "unit": "chai", "cost_price": 0, "sell_price": 10000, "member_price": 9000,
  "image_url": "", "min_stock": 24, "status": "ACTIVE"
}
```
| Trường | Bắt buộc | Quy tắc |
|---|---|---|
| `sku` | ✔ | 2–30 ký tự chữ/số/`.`/`-`/`_`, lưu IN HOA, **duy nhất** (trong các sản phẩm chưa xóa) |
| `barcode` | | ≤50, chữ/số/`-` (không bắt buộc duy nhất; POS quét lấy sản phẩm đầu tiên) |
| `name` | ✔ | ≤150 |
| `category_id` | | phải tồn tại; `null` = chưa phân loại |
| `unit` | | ≤20, mặc định `cái` |
| `cost_price` | | ≥0; thường để 0 và để phiếu nhập tính; sửa tay được (ghi audit `PRICE_CHANGE`) |
| `sell_price` | ✔ | ≥0 |
| `member_price` | | `null` hoặc 0 ≤ `member_price` ≤ `sell_price` |
| `image_url` | | URL từ `POST /admin/uploads` (xem `members.md`) |
| `min_stock` | | ≥0, ngưỡng cảnh báo tồn thấp (0 = không cảnh báo) |
| `status` | ✔ | `ACTIVE` / `INACTIVE` (INACTIVE: không hiện trên POS, không bán được) |

### DELETE `/admin/products/{id}` – xóa mềm, chỉ khi tổng tồn mọi chi nhánh = 0.

| Lỗi | message |
|---|---|
| 40000 | Giá hội viên không được lớn hơn giá bán |
| 40000 | Danh mục sản phẩm không tồn tại |
| 40900 | Mã SKU đã tồn tại |
| 40900 | Sản phẩm còn tồn 15 chai, hãy điều chỉnh tồn về 0 trước khi xóa |
| 40400 | Không tìm thấy sản phẩm |

---

## 4. Kho – `/admin/inventory`

### Nguyên tắc chung
- **Mọi biến động tồn** (nhập, bán, trả, điều chuyển, kiểm kê, hủy hóa đơn) đi qua MỘT hàm nội bộ `applyStockChange` trong transaction:
  upsert dòng `stock_levels` + `SELECT … FOR UPDATE` → cộng số lượng (có dấu) → kiểm tra không âm
  (CHECK `quantity >= 0` của DB là chốt chặn cuối, lỗi 23514 cũng trả `40900`) → ghi một dòng `stock_transactions` với `balance_after`.
- Phiếu nhiều dòng khóa các dòng tồn theo thứ tự `(product_id, branch_id)` tăng dần trước khi thay đổi → không deadlock
  (điều chuyển A→B và B→A chạy đồng thời vẫn an toàn).
- Thiếu hàng → `40900 "Sản phẩm \"Whey ON Gold Standard 5lbs\" không đủ hàng tại chi nhánh (còn 3 hộp, cần 4)"`.
- Chi nhánh phải đang `ACTIVE`; chi nhánh thao tác (nhập/kiểm kê, chi nhánh **xuất** khi điều chuyển) phải thuộc phạm vi người dùng.
  Chi nhánh nhận khi điều chuyển có thể ngoài phạm vi.
- **Cảnh báo tồn thấp:** sau mỗi lần **trừ** kho, nếu tồn cũ ≥ `min_stock` và tồn mới < `min_stock` → thông báo in-app
  `LOW_STOCK` cho mọi user có quyền `inventory.view` tại chi nhánh đó (và OWNER), link `/inventory/stock?branch_id=1&low_stock=true`.
  Chỉ báo khi **vượt ngưỡng** (đã thấp rồi bán tiếp không báo lại; nhập lên trên ngưỡng rồi giảm xuống lại thì báo lại).

### Giá vốn bình quân gia quyền (khi nhập hàng)
Tính trên **tổng tồn mọi chi nhánh** của sản phẩm, ngay trước dòng nhập:
```
giá vốn mới = (Tồn_trước × Giá vốn_cũ + SL_nhập × Đơn giá nhập) / (Tồn_trước + SL_nhập)
```
- `Tồn_trước ≤ 0` (hết hàng) → giá vốn mới = đơn giá nhập. Làm tròn 2 chữ số thập phân.
- Phiếu có nhiều dòng cùng sản phẩm: tính lần lượt từng dòng.
- Sản phẩm bị khóa `FOR NO KEY UPDATE` trong phiếu nhập để hai phiếu nhập đồng thời tính tuần tự.
- Ví dụ: tồn 10 (CN1) + 10 (CN3) = 20 chai giá vốn 6.000; nhập 20 chai giá 9.000 → (20×6.000 + 20×9.000)/40 = **7.500**.
- Bán/xuất ghi `unit_cost` = giá vốn hiện tại; trả hàng nhập lại theo `unit_cost` của dòng bán (giá vốn lúc bán) và **không** làm đổi giá vốn bình quân;
  điều chuyển, kiểm kê không đổi giá vốn.

### `Transaction` (dòng thẻ kho)
```json
{
  "id": 3, "product_id": 1, "product_sku": "AQUA-500", "product_name": "Nước suối Aquafina 500ml", "unit": "chai",
  "branch_id": 1, "branch_name": "GymMaster Cầu Giấy",
  "type": "SALE", "quantity": -6, "balance_after": 114, "unit_cost": 5500,
  "reference_type": "INVOICE", "reference_id": 27,
  "supplier_id": null, "supplier_name": "",
  "voucher_no": "HD26101400027", "note": "Bán hàng HD26101400027",
  "created_by": 1, "created_by_name": "Quản trị viên", "created_at": "2026-10-14T09:51:02+07:00"
}
```
`quantity` có dấu (+ nhập, − xuất); `balance_after` = tồn **tại chi nhánh đó** sau biến động.

### `Voucher` (response của nhập/điều chuyển/kiểm kê và `GET /vouchers/{voucher_no}`)
```json
{
  "voucher_no": "DC2610140001", "kind": "TRANSFER",
  "branch_id": 3, "branch_name": "GymMaster Quận 1",
  "to_branch_id": 1, "to_branch_name": "GymMaster Cầu Giấy",
  "supplier_id": null, "supplier_name": "",
  "reference_type": "", "reference_id": null,
  "note": "bổ sung", "total_quantity": 3, "total_value": 16500,
  "created_by": 1, "created_by_name": "Quản trị viên", "created_at": "...",
  "lines": [
    { "...": "Transaction", "type": "TRANSFER_OUT", "quantity": -3, "balance_after": 7, "note": "Chuyển đến GymMaster Cầu Giấy – bổ sung" },
    { "...": "Transaction", "type": "TRANSFER_IN", "quantity": 3, "balance_after": 9, "note": "Nhận từ GymMaster Quận 1 – bổ sung" }
  ]
}
```
- `branch_*`: chi nhánh của phiếu (điều chuyển: chi nhánh **xuất**); `to_branch_*` chỉ có với điều chuyển.
- `total_quantity` = Σ |quantity|, `total_value` = Σ |quantity| × unit_cost (điều chuyển chỉ tính chiều xuất).
- `note`: ghi chú người nhập (phiếu điều chuyển/kiểm kê đã bỏ phần mô tả tự sinh; ghi chú đầy đủ nằm ở từng dòng).

### POST `/admin/inventory/imports` – nhập hàng
```json
{
  "branch_id": 1, "supplier_id": 2, "note": "Tồn đầu kỳ chi nhánh Cầu Giấy",
  "items": [
    { "product_id": 1, "quantity": 120, "unit_cost": 5500 },
    { "product_id": 2, "quantity": 96,  "unit_cost": 8500 }
  ]
}
```
`supplier_id` tùy chọn (phải tồn tại). `quantity` ≥ 1, `unit_cost` ≥ 0. Mỗi dòng → một dòng thẻ kho `IMPORT` (unit_cost = đơn giá nhập)
và cập nhật `products.cost_price` theo công thức trên. Response: `Voucher` (`kind = IMPORT`, `voucher_no = NK…`).

### POST `/admin/inventory/transfers` – điều chuyển
```json
{ "from_branch_id": 3, "to_branch_id": 1, "note": "bổ sung", "items": [ { "product_id": 1, "quantity": 3 } ] }
```
Dòng trùng sản phẩm được gộp. Kiểm tra đủ tồn ở chi nhánh xuất cho **toàn bộ** phiếu trước khi ghi.
Mỗi sản phẩm → `TRANSFER_OUT` (−) ở chi nhánh xuất + `TRANSFER_IN` (+) ở chi nhánh nhận, cùng `voucher_no = DC…`.
Lỗi: `40000 "Chi nhánh nhận phải khác chi nhánh xuất"`, `40900` thiếu hàng.

### POST `/admin/inventory/adjustments` – kiểm kê
```json
{ "branch_id": 1, "note": "Kiểm kê cuối tháng",
  "items": [ { "product_id": 1, "actual_quantity": 8 }, { "product_id": 2, "actual_quantity": 96 } ] }
```
- `actual_quantity` (bắt buộc, ≥0) là số đếm thực tế. Chênh lệch = thực tế − sổ sách (đọc sau khi khóa dòng tồn).
- Dòng **không lệch bị bỏ qua**; dòng lệch ghi `ADJUST` với `quantity` = chênh lệch (±), `unit_cost` = giá vốn hiện tại,
  note `"Kiểm kê: sổ sách 9, thực tế 8 – Kiểm kê cuối tháng"`.
- Không có dòng nào lệch → `40000 "Số lượng thực tế khớp với tồn hiện tại, không có chênh lệch cần điều chỉnh"`.
- Sản phẩm lặp trong phiếu → `40000`.

### GET `/admin/inventory/stock` – tồn hiện tại
Query: `branch_id`, `category_id`, `low_stock=true`, `keyword` (tên/SKU/mã vạch), `page`, `limit`,
`order_by` (`product_id` mặc định tăng dần, `sku`, `name`, `quantity`, `branch_id`, `min_stock`).
- Có `branch_id`: mọi sản phẩm đang kinh doanh (hoặc còn tồn) tại chi nhánh, **kể cả tồn 0**.
- Không có `branch_id`: các dòng tồn (sản phẩm × chi nhánh đã từng phát sinh) của mọi chi nhánh trong phạm vi.
```json
{
  "items": [
    { "product_id": 1, "sku": "AQUA-500", "barcode": "8934588012112", "product_name": "Nước suối Aquafina 500ml",
      "category_id": 1, "category_name": "Đồ uống", "unit": "chai", "image_url": "",
      "branch_id": 1, "branch_name": "GymMaster Cầu Giấy",
      "quantity": 120, "min_stock": 24, "low_stock": false,
      "cost_price": 5500, "stock_value": 660000, "sell_price": 10000 }
  ],
  "total": 24, "total_page": 3, "current_page": 1, "limit": 10,
  "summary": { "total_quantity": 863, "total_value": 38780000, "low_stock_count": 1 }
}
```
`stock_value = quantity × cost_price`. `summary` tính trên toàn bộ bộ lọc.

### GET `/admin/inventory/transactions` – thẻ kho
Query: `product_id`, `branch_id`, `type`, `voucher_no` (chứa), `from`, `to` (ngày VN, tính cả hai đầu), `keyword` (tên/SKU sản phẩm, mã phiếu, ghi chú),
`order_by` (`id` mặc định giảm dần, `created_at`, `quantity`). Response: trang `Transaction`.

### GET `/admin/inventory/vouchers/{voucher_no}` – chi tiết phiếu
Mọi loại phiếu (`NK`, `DC`, `KK`, `TH`, cả số hóa đơn `HD…` để xem dòng xuất bán). Response `Voucher`.
`40400 "Không tìm thấy phiếu kho"`; mọi dòng ngoài phạm vi → `40300`.

### GET `/admin/inventory/low-stock?branch_id=` – tồn thấp
```json
[ { "product_id": 8, "sku": "PRE-C4", "product_name": "Pre-workout Cellucor C4 Original 30 lần dùng", "unit": "hộp",
    "branch_id": 3, "branch_name": "GymMaster Quận 1", "quantity": 1, "min_stock": 2 } ]
```
Chỉ xét dòng tồn đã tồn tại, sản phẩm `ACTIVE`, chi nhánh `ACTIVE`, `min_stock > 0` và `quantity < min_stock`.

**Cho cron/dashboard (Go):** `module.NewStockService(d).ListLowStock(ctx, branchID *uint)` – cùng dữ liệu, `nil` = mọi chi nhánh,
không kiểm tra phạm vi người dùng.

---

## 5. POS – `/admin/pos`

### GET `/admin/pos/products?branch_id=1&keyword=&category_id=&in_stock=`
`branch_id` bắt buộc (phải thuộc phạm vi). Chỉ sản phẩm `ACTIVE`, sắp theo tên, tối đa 500. `in_stock=true` → chỉ còn hàng.
```json
[ { "id": 1, "sku": "AQUA-500", "barcode": "8934588012112", "name": "Nước suối Aquafina 500ml",
    "category_id": 1, "category_name": "Đồ uống", "unit": "chai",
    "sell_price": 10000, "member_price": 9000, "image_url": "",
    "stock": 120, "min_stock": 24, "low_stock": false } ]
```

### GET `/admin/pos/products/by-barcode/{code}?branch_id=1`
Tìm theo `barcode`, không có thì theo SKU (không phân biệt hoa thường). Response: một `PosProduct`.
`40400 "Không tìm thấy sản phẩm có mã …"`, `40000 "Sản phẩm \"…\" đã ngừng kinh doanh"`.

### POST `/admin/pos/orders` – bán hàng
```json
{
  "branch_id": 1,
  "member_id": 359,
  "items": [ { "product_id": 1, "quantity": 2 }, { "product_id": 5, "quantity": 1 } ],
  "promotion_code": "",
  "payments": [
    { "amount": 1000000, "method": "TRANSFER", "reference": "FT2610140001" },
    { "amount": 1000000, "method": "CASH", "reference": "" }
  ],
  "note": ""
}
```
Quy tắc (tất cả trong **một transaction**):
1. `member_id` tùy chọn (null = khách vãng lai). **Giá** mỗi dòng = `member_price` nếu có `member_id`, hội viên đang `ACTIVE`
   và sản phẩm có `member_price`; ngược lại `sell_price`. Dòng trùng sản phẩm được gộp.
2. Sản phẩm phải tồn tại và `ACTIVE`. Khóa các dòng tồn (thứ tự `product_id` tăng dần) và kiểm tra đủ hàng **trước** khi lập hóa đơn
   → thiếu: `40900 "Sản phẩm \"…\" không đủ hàng tại chi nhánh (còn 3 hộp, cần 4)"`.
3. Lập hóa đơn qua `InvoiceService.CreateInvoice` (`source = POS`, dòng `PRODUCT`, `ref_id = product_id`,
   mã khuyến mãi áp với `applies_to = PRODUCT`/`ALL`, phân bổ vào các dòng).
4. **Bắt buộc thanh toán đủ**: Σ `payments.amount` ≥ `total`, nếu thiếu → `40000 "Khách thanh toán chưa đủ: tổng hóa đơn 60.000 đ, đã nhận 50.000 đ (còn thiếu 10.000 đ)"`
   (không lập hóa đơn, không trừ kho). Phần **thừa** chỉ được đến từ tiền mặt (`change_amount` ≤ Σ tiền mặt), ngược lại
   `40000 "Chỉ tiền mặt mới được đưa thừa: số tiền chuyển khoản/thẻ vượt quá tổng hóa đơn"`. Hệ thống chỉ **ghi nhận đúng `total`**:
   các khoản tiền mặt được giảm (từ khoản cuối) bằng tiền thừa; khoản về 0 không ghi. Hóa đơn 0đ (khuyến mãi 100%) không cần thanh toán.
5. Trừ kho từng dòng (`SALE`, `reference_type = INVOICE`, `voucher_no` = số hóa đơn, `unit_cost` = giá vốn hiện tại) và thu tiền
   (`AddPayment` từng phương thức) → hóa đơn `PAID`.

Response:
```json
{
  "invoice": {
    "id": 58, "invoice_no": "HD26101400058", "member_id": 359,
    "member": { "id": 359, "member_code": "GM000359", "full_name": "Nguyễn Văn An", "phone": "0399999001" },
    "branch_id": 1, "branch_name": "GymMaster Cầu Giấy",
    "subtotal": 1868000, "discount": 0, "total": 1868000, "paid_amount": 1868000, "refunded_amount": 0, "due": 0,
    "status": "PAID", "source": "POS", "promotion_id": null, "promotion_code": "", "note": "",
    "issued_by": 1, "issued_by_name": "Quản trị viên", "issued_at": "...", "voided_at": null, "void_reason": "", "created_at": "...",
    "voided_by_name": "",
    "items": [
      { "id": 90, "item_type": "PRODUCT", "ref_id": 1, "description": "Nước suối Aquafina 500ml", "quantity": 2, "unit_price": 9000, "discount": 0, "amount": 18000 },
      { "id": 91, "item_type": "PRODUCT", "ref_id": 5, "description": "Whey ON Gold Standard 100% Whey 5lbs (Double Rich Chocolate)", "quantity": 1, "unit_price": 1850000, "discount": 0, "amount": 1850000 }
    ],
    "payments": [
      { "id": 40, "invoice_id": 58, "amount": 1000000, "method": "TRANSFER", "reference": "FT2610140001", "paid_at": "...", "received_by": 1, "received_by_name": "Quản trị viên", "note": "" },
      { "id": 41, "invoice_id": 58, "amount": 868000, "method": "CASH", "reference": "", "paid_at": "...", "received_by": 1, "received_by_name": "Quản trị viên", "note": "" }
    ],
    "refunds": []
  },
  "received_amount": 2000000,
  "change_amount": 132000,
  "print": {
    "invoice": { "...": "InvoiceDetail như trên" },
    "branch": { "id": 1, "code": "HN01", "name": "GymMaster Cầu Giấy", "address": "144 Xuân Thủy, Cầu Giấy, Hà Nội", "phone": "02437547461", "email": "" },
    "member": { "id": 359, "member_code": "GM000359", "full_name": "Nguyễn Văn An", "phone": "0399999001" },
    "total_in_words": "Một triệu tám trăm sáu mươi tám nghìn đồng",
    "due_in_words": "Không đồng",
    "printed_at": "...", "printed_by_name": "Quản trị viên"
  }
}
```
`invoice` là `InvoiceDetail` (xem `sales.md` mục 3); `print` cùng dạng `GET /admin/invoices/{id}/print`.
Lỗi khác: `40000` sản phẩm ngừng kinh doanh / mã khuyến mãi không hợp lệ, `40400 "Không tìm thấy hội viên"`, `40300` chi nhánh ngoài phạm vi.

### GET `/admin/pos/orders`
Query: `branch_id`, `member_id`, `status`, `from`, `to` (ngày lập), `keyword` (số HĐ, tên/SĐT/mã hội viên), `page`, `limit`, `order_by`
(`id`, `issued_at`, `total`, `paid_amount`, `invoice_no`, `status`). Response: trang `Invoice` (chỉ `source = POS`) – như `GET /admin/invoices`.

### GET `/admin/pos/orders/{invoice_id}` – chi tiết đơn (màn hình trả hàng)
```json
{
  "invoice": { "...": "InvoiceDetail" },
  "lines": [
    { "product_id": 1, "description": "Nước suối Aquafina 500ml", "sold_quantity": 6, "returned_quantity": 2,
      "returnable_quantity": 4, "amount": 60000, "refunded_value": 20000 }
  ]
}
```
`amount` = thành tiền của sản phẩm trên hóa đơn (đã trừ giảm giá/khuyến mãi phân bổ). Hóa đơn không phải POS → `40400`.

### POST `/admin/pos/orders/{invoice_id}/returns` – trả hàng
```json
{ "items": [ { "product_id": 1, "quantity": 2 } ], "refund_method": "CASH", "reason": "Khách đổi ý" }
```
- Chỉ hóa đơn `source = POS`, trạng thái `PAID` (hoặc `REFUNDED` – khi đó không hoàn thêm tiền). Khóa hóa đơn để các lần trả tuần tự.
- Mỗi sản phẩm: số lượng trả ≤ **đã bán − đã trả** → nếu vượt: `40000 "Sản phẩm \"…\" chỉ còn có thể trả 4 (đã bán 6, đã trả 2)"`;
  sản phẩm không có trong hóa đơn → `40000`.
- Hoàn kho `SALE_RETURN` (+) tại chi nhánh của hóa đơn, `unit_cost` = giá vốn lúc bán, `voucher_no = TH…`, `reference = INVOICE`.
- **Số tiền hoàn** theo tỷ lệ thành tiền (đã gồm khuyến mãi), tính lũy kế để trả hết thì tổng hoàn đúng bằng thành tiền:
  `hoàn = round(amount × (đã_trả + SL) / đã_bán) − round(amount × đã_trả / đã_bán)`; không vượt `paid_amount − refunded_amount`.
  Hoàn qua `InvoiceService.Refund` (ghi `refunds`, hóa đơn → `REFUNDED` khi hoàn hết).
```json
{
  "voucher_no": "TH2610140001", "refund_amount": 20000,
  "invoice": { "...": "InvoiceDetail (refunded_amount, refunds đã cập nhật)" },
  "lines": [ { "product_id": 1, "description": "Nước suối Aquafina 500ml", "sold_quantity": 6, "returned_quantity": 2,
               "returnable_quantity": 4, "amount": 60000, "refunded_value": 20000 } ]
}
```

### Hủy hóa đơn POS
`POST /admin/invoices/{id}/void` chỉ cho hóa đơn **chưa thu đồng nào** (đơn POS luôn thu đủ ngay nên thực tế dùng trả hàng).
Module POS đăng ký **VoidHandler `PRODUCT`**: khi một hóa đơn có dòng `PRODUCT` bị hủy, phần đã xuất kho theo hóa đơn mà chưa trả
được nhập lại (`SALE_RETURN`, `voucher_no` = số hóa đơn). Idempotent.

### GET `/admin/pos/summary?branch_id=&date=` – tổng kết ngày
`date` mặc định hôm nay (giờ VN); bỏ `branch_id` = mọi chi nhánh trong phạm vi.
```json
{
  "date": "2026-10-14", "branch_id": 1,
  "order_count": 1, "items_sold": 6,
  "gross_revenue": 60000, "discount": 0, "refund_amount": 20000, "net_revenue": 40000,
  "cost_of_goods": 26000, "gross_profit": 14000,
  "by_method": [ { "method": "CASH", "amount": 40000, "count": 1 }, { "method": "TRANSFER", "amount": 20000, "count": 1 },
                 { "method": "CARD", "amount": 0, "count": 0 } ],
  "top_products": [ { "product_id": 1, "sku": "AQUA-500", "name": "Nước suối Aquafina 500ml", "quantity": 6, "revenue": 60000 } ]
}
```
- `order_count`, `items_sold`, `gross_revenue` (Σ total), `discount`: hóa đơn POS lập trong ngày, trừ hóa đơn `VOID`.
- `refund_amount`: tiền hoàn phát sinh trong ngày cho hóa đơn POS. `net_revenue = gross_revenue − refund_amount`.
- `cost_of_goods` = Σ giá vốn của dòng `SALE` − `SALE_RETURN` trong ngày; `gross_profit = net_revenue − cost_of_goods`.
- `by_method`: tiền thu theo phương thức (luôn đủ 3). `top_products`: top 5 theo số lượng bán.

---

## 6. Gợi ý màn hình (FE admin)
- **Sản phẩm**: bảng + bộ lọc danh mục/trạng thái/chi nhánh (hiện cột tồn + tag "Tồn thấp" khi lọc chi nhánh); form có upload ảnh.
- **POS**: chọn chi nhánh → lưới sản phẩm (`/pos/products`), ô quét mã vạch (`/by-barcode`), chọn hội viên (`/admin/members/options`)
  để hiện giá hội viên, ô mã KM (`POST /admin/promotions/check` với `applies_to = PRODUCT`), nhiều dòng thanh toán, hiển thị tiền thừa
  (`change_amount`), in hóa đơn từ `print`.
- **Kho**: tab Tồn kho (summary giá trị tồn), Thẻ kho, Phiếu nhập / Điều chuyển / Kiểm kê (form nhiều dòng), xem phiếu theo `voucher_no`.
- **Trả hàng**: từ danh sách đơn POS → `GET /pos/orders/{id}` hiển thị `returnable_quantity` → `POST …/returns`.

## 7. Dữ liệu mẫu (demo)
3 danh mục (Đồ uống, Thực phẩm bổ sung, Phụ kiện), 2 nhà cung cấp, 12 sản phẩm, 4 phiếu nhập tồn đầu kỳ cho
chi nhánh 1 (Cầu Giấy) và 3 (Quận 1) – `NK2610140001…0004`. `PRE-C4` tại Quận 1 đang dưới mức tối thiểu (demo cảnh báo).

## 8. Giới hạn hiện tại
- Không có "phiếu xuất hủy" riêng: hàng hỏng/mất xử lý bằng kiểm kê (`ADJUST` âm).
- Trả hàng không làm thay đổi giá vốn bình quân (nhập lại theo giá vốn lúc bán).
- Phiếu nhập không hủy/sửa được sau khi lưu (điều chỉnh bằng kiểm kê).
- Mã phiếu tối đa 9.999 phiếu/loại/ngày.
