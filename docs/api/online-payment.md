# API contract – Sprint 13 (Thanh toán trực tuyến VNPay Sandbox)

Base URL (dev): `http://localhost:8000/v1`.

## 1. Cấu hình môi trường (`.env`)

| Biến | Mặc định (dev) | Mô tả |
|---|---|---|
| `VNPAY_TMN_CODE` | `GYMMASTR` | Mã website tại VNPay (`vnp_TmnCode`) |
| `VNPAY_HASH_SECRET` | `GYMMASTER_VNPAY_SANDBOX_SECRET_KEY_32` | Chuỗi bí mật ký HMAC-SHA512 (`vnp_SecureHash`) |
| `VNPAY_PAYMENT_URL` | `https://sandbox.vnpayment.vn/paymentv2/vpcpay.html` | URL cổng thanh toán VNPay Sandbox |
| `VNPAY_RETURN_URL` | `http://localhost:3000/payment/result` | URL điều hướng trình duyệt sau khi thanh toán |

## 2. Mã lỗi nghiệp vụ

| code | HTTP | Ý nghĩa |
|---|---|---|
| 44001 | 440 | `PAYMENT_INVALID_SIGNATURE` – Chữ ký HMAC-SHA512 không hợp lệ |
| 44002 | 440 | `PAYMENT_AMOUNT_MISMATCH` – Số tiền thanh toán không khớp với giao dịch |
| 44003 | 440 | `PAYMENT_TXN_EXPIRED` – Giao dịch thanh toán trực tuyến đã hết hạn |

---

## 3. Cổng hội viên (`/v1/member/*`, role `MEMBER`)

### GET `/v1/member/promotions/check`
Kiểm tra mã khuyến mãi trước khi hội viên tự mua gói tập.
- Query: `code` (bắt buộc), `plan_id`, `amount`.
- Response `data`:
```json
{
  "code": "SUMMER20",
  "name": "Ưu đãi mùa hè 20%",
  "discount": 300000,
  "final_amount": 1200000
}
```

### POST `/v1/member/subscriptions/checkout`
Hội viên tự chọn gói tập → hệ thống lập đăng ký gói (`PENDING`) + hóa đơn (`UNPAID`, `source = PORTAL`) + giao dịch `online_payment_txns` (`PENDING`) và trả URL thanh toán VNPay.

Request:
```json
{
  "plan_id": 2,
  "branch_id": 1,
  "promotion_code": "SUMMER20"
}
```

Response `data`:
```json
{
  "subscription_id": 412,
  "invoice_id": 589,
  "invoice_no": "HD26101400589",
  "amount": 1200000,
  "txn_ref": "VNP26101417300001",
  "payment_url": "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_Amount=120000000&..."
}
```

### POST `/v1/member/invoices/{id}/vnpay`
Tạo giao dịch VNPay mới cho một hóa đơn đang `UNPAID` hoặc `PARTIAL` của hội viên (số tiền = phần còn nợ `due`).

Response `data`:
```json
{
  "txn_ref": "VNP26101417350002",
  "payment_url": "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?...",
  "amount": 1200000,
  "expires_at": "2026-10-14T17:50:00+07:00"
}
```

---

## 4. Callback công khai (`/v1/public/payments/vnpay/*`)

### GET `/v1/public/payments/vnpay/ipn`
Webhook IPN từ VNPay (hoặc giả lập trong môi trường dev/test):
1. Xác thực chữ ký `vnp_SecureHash` bằng HMAC-SHA512.
2. Đối chiếu `vnp_Amount / 100 == txn.Amount`.
3. **Idempotent** theo `vnp_TxnRef`: nếu giao dịch đã `SUCCESS` thì trả thành công ngay, không ghi trùng `payments`.
4. Nếu `vnp_ResponseCode == "00"` và `txn.Status == "PENDING"`: trong 1 transaction ghi `payments` (`method = 'VNPAY'`), cập nhật hóa đơn sang `PAID`, kích hoạt gói tập `ACTIVE` qua `PaidHandler`, và đẩy thông báo `PAYMENT_SUCCESS` cho hội viên.

### GET `/v1/public/payments/vnpay/return`
Chỉ xác thực chữ ký và đọc trạng thái giao dịch để hiển thị trên trang `/payment/result` (không ghi DB).

---

## 5. Quản trị & Cron

- **GET `/v1/admin/online-payments`** (`perm: invoice.view`): danh sách giao dịch `online_payment_txns` có phân trang, lọc theo `status` (`PENDING`, `SUCCESS`, `FAILED`, `EXPIRED`) và từ khóa `search`.
- **Cron `expire_online_txns`** (`*/15 * * * *`): tự động chuyển các giao dịch `PENDING` quá 15 phút sang `EXPIRED`.
