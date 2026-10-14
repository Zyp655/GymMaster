# API contract – Danh mục: Gói tập, Gói PT, Khuyến mãi, Cấu hình, Trang công khai

Base URL (dev): `http://localhost:8000/v1`. Envelope, mã lỗi và quy ước phân trang: xem `sprint-01-auth-rbac-branch.md`.
Schema: migration `000004_catalog`.

- Tiền là `number` (VND, đã làm tròn đến đồng). Thời điểm là RFC3339 (`2026-10-01T00:00:00+07:00`).
- Lỗi validate tag của GoFrame trả `code = 51` (HTTP 400). Lỗi nghiệp vụ trả `40000` (400), `40400` (404), `40900` (409).
- Mọi thao tác tạo / sửa / đổi trạng thái / xóa đều ghi `audit_logs` (`entity` = `membership_plan` | `pt_package` | `promotion` | `setting`).

| Method | Path | Quyền |
|---|---|---|
| GET | `/admin/membership-plans` | `plan.view` |
| GET | `/admin/membership-plans/options` | `plan.view` hoặc `subscription.sell` |
| GET | `/admin/membership-plans/{id}` | `plan.view` |
| POST | `/admin/membership-plans` | `plan.manage` |
| PUT | `/admin/membership-plans/{id}` | `plan.manage` |
| PUT | `/admin/membership-plans/{id}/status` | `plan.manage` |
| DELETE | `/admin/membership-plans/{id}` | `plan.manage` |
| GET | `/admin/pt-packages` | `plan.view` |
| GET | `/admin/pt-packages/options` | `plan.view` hoặc `pt.sell` |
| GET | `/admin/pt-packages/{id}` | `plan.view` |
| POST | `/admin/pt-packages` | `plan.manage` |
| PUT | `/admin/pt-packages/{id}` | `plan.manage` |
| PUT | `/admin/pt-packages/{id}/status` | `plan.manage` |
| DELETE | `/admin/pt-packages/{id}` | `plan.manage` |
| GET | `/admin/promotions` | `promotion.view` |
| GET | `/admin/promotions/{id}` | `promotion.view` |
| POST | `/admin/promotions` | `promotion.manage` |
| PUT | `/admin/promotions/{id}` | `promotion.manage` |
| DELETE | `/admin/promotions/{id}` | `promotion.manage` |
| POST | `/admin/promotions/check` | `subscription.sell`, `pt.sell`, `pos.sell` hoặc `promotion.view` |
| GET | `/admin/settings` | `setting.manage` |
| PUT | `/admin/settings/{key}` | `setting.manage` |
| GET | `/public/plans` | công khai |
| GET | `/public/branches` | công khai |

---

## 1. Gói tập – `/admin/membership-plans`

### Đối tượng `Plan`
```json
{
  "id": 1,
  "code": "M1",
  "name": "Gói 1 tháng",
  "description": "Tập không giới hạn trong 30 ngày",
  "type": "TIME",
  "duration_days": 30,
  "session_count": null,
  "price": 500000,
  "access_scope": "SINGLE_BRANCH",
  "max_freeze_days": 7,
  "max_freeze_times": 1,
  "includes_classes": true,
  "transfer_fee": 100000,
  "status": "ACTIVE",
  "sort_order": 1,
  "created_at": "2026-10-14T08:58:09.448505+07:00",
  "updated_at": "2026-10-14T08:58:09.448505+07:00"
}
```
| Trường | Ý nghĩa |
|---|---|
| `type` | `TIME`: tập không giới hạn trong `duration_days` ngày. `SESSION`: được `session_count` lượt check-in, hết hạn sau `duration_days` ngày |
| `session_count` | Bắt buộc (> 0) với `SESSION`; luôn `null` với `TIME` |
| `access_scope` | `SINGLE_BRANCH` (chỉ chi nhánh đăng ký) \| `ALL_BRANCHES` |
| `max_freeze_days`, `max_freeze_times` | Tổng số ngày / số lần bảo lưu tối đa. Cùng = 0 nghĩa là không cho bảo lưu |
| `includes_classes` | Gói có gồm lớp tập nhóm |
| `transfer_fee` | Phí chuyển nhượng gói sang hội viên khác |
| `status` | `ACTIVE` (đang bán) \| `INACTIVE` (ngừng bán) |
| `sort_order` | Thứ tự hiển thị (tăng dần) |

### GET `/admin/membership-plans`
Query: `page`, `limit`, `keyword` (mã/tên), `status` (`ACTIVE|INACTIVE`), `type` (`TIME|SESSION`), `order_by` (`id, code, name, type, price, duration_days, status, sort_order, created_at`), `order_dir`.
Mặc định sắp xếp `sort_order ASC`. Trả `{items: Plan[], total, total_page, current_page, limit}`.

### GET `/admin/membership-plans/options`
Mảng `Plan[]` các gói `ACTIVE`, sắp theo `sort_order`, rồi giá. Dùng cho dropdown bán gói.

### GET `/admin/membership-plans/{id}`
Trả `Plan`. Không tồn tại → `40400 "Không tìm thấy gói tập"`.

### POST `/admin/membership-plans` · PUT `/admin/membership-plans/{id}`
Body:
```json
{
  "code": "S12",
  "name": "Gói 12 buổi",
  "description": "Linh hoạt cho người bận rộn",
  "type": "SESSION",
  "duration_days": 60,
  "session_count": 12,
  "price": 600000,
  "access_scope": "SINGLE_BRANCH",
  "max_freeze_days": 0,
  "max_freeze_times": 0,
  "includes_classes": false,
  "transfer_fee": 0,
  "status": "ACTIVE",
  "sort_order": 3
}
```
| Trường | Bắt buộc | Ràng buộc |
|---|---|---|
| `code` | ✔ | `^[A-Z0-9_-]{2,30}$`, duy nhất (trong các gói chưa xóa) |
| `name` | ✔ | ≤ 150 ký tự |
| `description` | | ≤ 2000 ký tự |
| `type` | ✔ | `TIME` \| `SESSION` |
| `duration_days` | ✔ | 1 – 3650 |
| `session_count` | với SESSION | 1 – 10000; với `TIME` bị bỏ qua (lưu `null`) |
| `price` | | ≥ 0 (mặc định 0) |
| `access_scope` | | mặc định `SINGLE_BRANCH` |
| `max_freeze_days`, `max_freeze_times` | | ≥ 0; phải cùng = 0 hoặc cùng > 0; `max_freeze_days` ≤ `duration_days` |
| `transfer_fee` | | ≥ 0 |
| `status` | | mặc định `ACTIVE` |

Trả `Plan`. Lỗi thường gặp:
- `40000 "Gói theo buổi phải có số buổi lớn hơn 0"`
- `40000 "Số ngày bảo lưu và số lần bảo lưu phải cùng bằng 0 (không cho bảo lưu) hoặc cùng lớn hơn 0"`
- `40000 "Số ngày bảo lưu tối đa không được vượt quá thời hạn gói"`
- `40900 "Mã gói tập đã tồn tại"`

> Sửa gói không ảnh hưởng các đăng ký đã bán (subscription lưu giá và hạn riêng).

### PUT `/admin/membership-plans/{id}/status`
Body `{ "status": "INACTIVE" }` → trả `Plan`.

### DELETE `/admin/membership-plans/{id}`
Xóa mềm. Nếu gói đã có `subscriptions` (bất kỳ trạng thái) → `40900 "Gói tập đã có hội viên đăng ký nên không thể xóa, hãy chuyển sang trạng thái Ngừng bán"`.

---

## 2. Gói PT – `/admin/pt-packages`

### Đối tượng `PTPackage`
```json
{
  "id": 1,
  "code": "PT10",
  "name": "PT 10 buổi",
  "description": "Làm quen, xây nền tảng",
  "session_count": 10,
  "session_minutes": 60,
  "validity_days": 45,
  "price": 3500000,
  "price_per_session": 350000,
  "status": "ACTIVE",
  "created_at": "...",
  "updated_at": "..."
}
```
`price_per_session` = `round(price / session_count)` – giá trị tham khảo (module PT dùng để tính hoa hồng).

### GET `/admin/pt-packages`
Query: `page`, `limit`, `keyword`, `status`, `order_by` (`id, code, name, session_count, validity_days, price, status, created_at`), `order_dir`. Mặc định `price ASC`.

### GET `/admin/pt-packages/options`
Mảng `PTPackage[]` đang `ACTIVE`, sắp theo số buổi rồi giá.

### POST / PUT
```json
{
  "code": "PT20",
  "name": "PT 20 buổi",
  "description": "",
  "session_count": 20,
  "session_minutes": 60,
  "validity_days": 90,
  "price": 6400000,
  "status": "ACTIVE"
}
```
| Trường | Bắt buộc | Ràng buộc |
|---|---|---|
| `code` | ✔ | `^[A-Z0-9_-]{2,30}$`, duy nhất |
| `name` | ✔ | ≤ 150 |
| `session_count` | ✔ | 1 – 1000 |
| `session_minutes` | | 15 – 240, mặc định 60 |
| `validity_days` | ✔ | 1 – 3650 |
| `price` | | ≥ 0 |
| `status` | | mặc định `ACTIVE` |

### PUT `/admin/pt-packages/{id}/status` – body `{ "status": "ACTIVE|INACTIVE" }`.

### DELETE `/admin/pt-packages/{id}`
Có `pt_contracts` → `40900 "Gói PT đã có hợp đồng nên không thể xóa, hãy chuyển sang trạng thái Ngừng bán"`.

---

## 3. Khuyến mãi – `/admin/promotions`

### Đối tượng `Promotion`
```json
{
  "id": 1,
  "code": "CHAODONG",
  "name": "Chào đông – giảm 10% gói tập",
  "type": "PERCENT",
  "value": 10,
  "max_discount": 500000,
  "min_amount": 500000,
  "applies_to": "MEMBERSHIP",
  "start_at": "2026-10-01T00:00:00+07:00",
  "end_at": "2026-12-31T23:59:59+07:00",
  "usage_limit": 100,
  "used_count": 0,
  "status": "ACTIVE",
  "validity": "active",
  "created_at": "...",
  "updated_at": "..."
}
```
| Trường | Ý nghĩa |
|---|---|
| `code` | Luôn lưu IN HOA; tra cứu không phân biệt hoa thường |
| `type` / `value` | `PERCENT`: giảm `value`% (0 < value ≤ 100). `FIXED`: giảm `value` đồng |
| `max_discount` | Trần số tiền giảm cho `PERCENT` (`null` = không trần). Luôn `null` với `FIXED` |
| `min_amount` | Giá trị đơn tối thiểu để dùng mã |
| `applies_to` | `ALL` \| `MEMBERSHIP` (gói tập) \| `PT` \| `PRODUCT` (POS) |
| `start_at`, `end_at` | Khoảng hiệu lực, tính cả hai đầu; `end_at > start_at` |
| `usage_limit` / `used_count` | `null` = không giới hạn lượt |
| `validity` | Trạng thái hiệu lực tại thời điểm gọi: `active`, `upcoming` (chưa bắt đầu), `expired` (quá hạn), `exhausted` (hết lượt), `inactive` (status = INACTIVE) |

### GET `/admin/promotions`
Query: `page`, `limit`, `keyword` (mã/tên), `status`, `applies_to`, `validity`, `order_by` (`id, code, name, type, value, start_at, end_at, used_count, status, created_at`), `order_dir`.
`validity`:
- `active`: ACTIVE, đang trong thời gian, còn lượt;
- `upcoming`: ACTIVE, `start_at` > hiện tại;
- `expired`: đã quá `end_at` **hoặc** đã hết lượt;
- `inactive`: status INACTIVE.

### POST / PUT
```json
{
  "code": "pt200k",
  "name": "Giảm 200.000đ gói PT",
  "type": "FIXED",
  "value": 200000,
  "max_discount": null,
  "min_amount": 0,
  "applies_to": "PT",
  "start_at": "2026-10-01T00:00:00+07:00",
  "end_at": "2026-11-30T23:59:59+07:00",
  "usage_limit": null,
  "status": "ACTIVE"
}
```
| Trường | Bắt buộc | Ràng buộc |
|---|---|---|
| `code` | ✔ | `^[A-Za-z0-9_-]{3,30}$`, được chuyển IN HOA, duy nhất (không phân biệt hoa thường) |
| `name` | ✔ | ≤ 150 |
| `type` | ✔ | `PERCENT` \| `FIXED` |
| `value` | ✔ | > 0; PERCENT ≤ 100 |
| `max_discount` | | > 0, chỉ dùng cho PERCENT |
| `min_amount` | | ≥ 0 |
| `applies_to` | | mặc định `ALL` |
| `start_at`, `end_at` | ✔ | `end_at` > `start_at` |
| `usage_limit` | | ≥ 1; khi sửa không được nhỏ hơn `used_count` |
| `status` | | mặc định `ACTIVE` |

Lỗi: `40000 "Phần trăm giảm không được vượt quá 100%"`, `40000 "Thời gian kết thúc phải sau thời gian bắt đầu"`, `40000 "Giới hạn lượt dùng không được nhỏ hơn số lượt đã dùng (n)"`, `40900 "Mã khuyến mãi đã tồn tại"`.

### DELETE `/admin/promotions/{id}`
Mã đã dùng (`used_count > 0`) → `40900 "Mã khuyến mãi đã được sử dụng nên không thể xóa, hãy chuyển sang trạng thái Ngừng áp dụng"`.

### POST `/admin/promotions/check`
Kiểm tra mã ở màn hình bán hàng (không tăng lượt dùng).
```json
{ "code": "chaodong", "applies_to": "MEMBERSHIP", "amount": 1350000 }
```
`applies_to` là loại đơn: `MEMBERSHIP` \| `PT` \| `PRODUCT`. Response `data`:
```json
{
  "promotion": { "...": "Promotion" },
  "discount": 135000,
  "final_amount": 1215000
}
```
Lỗi (đều `40000`, theo thứ tự kiểm tra):
| Thông điệp | Khi |
|---|---|
| `Mã khuyến mãi không tồn tại` | không có mã |
| `Mã khuyến mãi đã ngừng áp dụng` | status INACTIVE |
| `Mã khuyến mãi chưa đến thời gian áp dụng` | now < start_at |
| `Mã khuyến mãi đã hết hạn` | now > end_at |
| `Mã khuyến mãi chỉ áp dụng cho gói tập / gói PT / sản phẩm` | applies_to không khớp |
| `Đơn hàng chưa đạt giá trị tối thiểu 500.000 đ để dùng mã này` | amount < min_amount |
| `Mã khuyến mãi đã hết lượt sử dụng` | used_count ≥ usage_limit |

### Công thức giảm giá
- `PERCENT`: `discount = round(amount × value / 100)`, nếu có `max_discount` thì `discount = min(discount, max_discount)`.
- `FIXED`: `discount = min(value, amount)`.
- Kết quả luôn trong `[0, amount]`, làm tròn đến đồng.

### Dùng trong module bán hàng (Go)
`internal/service/promotion_service.go`:
```go
promotions := service.NewPromotionService(repository.NewPromotionRepository(d.DB), d.Audit)

// Validate kiểm tra mã và tính số tiền giảm cho đơn có giá trị amount thuộc loại appliesTo (MEMBERSHIP|PT|PRODUCT).
Validate(ctx context.Context, code string, appliesTo string, amount float64, now time.Time) (*entity.Promotion, float64, error)
// Consume tăng used_count nguyên tử (chỉ khi còn lượt); 0 dòng → 40000 "Mã khuyến mãi đã hết lượt sử dụng".
Consume(ctx context.Context, promotionID uint) error
```
Luồng chuẩn:
```go
promo, discount, err := promotions.Validate(ctx, req.PromotionCode, entity.AppliesMembership, subtotal, time.Now())
if err != nil { return nil, err }
err = d.Tx.WithTx(ctx, func(ctx context.Context) error {
    // ... tạo invoice với promotion_id = promo.ID, discount = discount ...
    return promotions.Consume(ctx, promo.ID) // trong cùng transaction → hủy đơn thì không mất lượt
})
```
Hàm thuần dùng lại được: `service.CalcPromotionDiscount(p, amount)`, `service.CheckPromotionApplicable(p, appliesTo, amount, now)`, `service.PromotionValidity(p, now)`. Hằng số: `entity.AppliesMembership|AppliesPT|AppliesProduct|AppliesAll`, `entity.PromotionPercent|PromotionFixed`.

---

## 4. Cấu hình – `/admin/settings`

### GET `/admin/settings`
```json
[
  { "key": "booking.advance_days", "value": 7, "description": "Được đặt lớp trước tối đa (ngày)", "updated_at": "..." },
  { "key": "subscription.remind_before_days", "value": [7, 1], "description": "Nhắc gia hạn trước khi hết hạn (ngày)", "updated_at": "..." }
]
```

### PUT `/admin/settings/{key}`
Body `{ "value": 10 }` hoặc `{ "value": [7, 3, 1] }`. Trả `Setting` đã lưu. Ghi audit (`before`/`after` gồm key và value). Cache `SettingService` bị xóa ngay → các module đọc giá trị mới.

| key | Kiểu | Khoảng hợp lệ |
|---|---|---|
| `checkin.duplicate_window_minutes` | số nguyên | 1 – 1440 |
| `checkin.qr_ttl_seconds` | số nguyên | 15 – 600 |
| `booking.advance_days` | số nguyên | 1 – 60 |
| `booking.cancel_before_minutes` | số nguyên | 1 – 10080 |
| `booking.no_show_limit` | số nguyên | 1 – 100 |
| `booking.block_days` | số nguyên | 1 – 365 |
| `subscription.remind_before_days` | mảng số nguyên | 1 – 10 phần tử, mỗi phần tử 1 – 90; được bỏ trùng và sắp giảm dần |
| `pt.cancel_before_minutes` | số nguyên | 1 – 10080 |

Lỗi: `40000 "Giá trị phải là số nguyên"`, `40000 "Giá trị phải từ 1 đến 60"`, `40000 "Giá trị phải là danh sách số nguyên"`, `40400 "Không tìm thấy mục cấu hình"`.

---

## 5. Công khai (không cần đăng nhập)

### GET `/public/plans`
```json
{
  "membership_plans": [
    {
      "id": 1, "code": "M1", "name": "Gói 1 tháng", "description": "...",
      "type": "TIME", "duration_days": 30, "session_count": null, "price": 500000,
      "access_scope": "SINGLE_BRANCH", "max_freeze_days": 7, "includes_classes": true
    }
  ],
  "pt_packages": [
    {
      "id": 1, "code": "PT10", "name": "PT 10 buổi", "description": "...",
      "session_count": 10, "session_minutes": 60, "validity_days": 45, "price": 3500000
    }
  ]
}
```
Chỉ gồm gói đang `ACTIVE`; gói tập sắp theo `sort_order`, gói PT theo số buổi.

### GET `/public/branches`
```json
[
  { "id": 1, "name": "GymMaster Cầu Giấy", "address": "144 Xuân Thủy, Cầu Giấy, Hà Nội",
    "phone": "02437547461", "open_time": "05:30", "close_time": "22:00" }
]
```
Chỉ chi nhánh `ACTIVE` (tối đa 100).
