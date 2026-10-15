# Sơ đồ Thiết kế Hệ thống Mở rộng (Giai đoạn 2 – 4, Mục 16.4)

Tài liệu này tổng hợp đầy đủ các sơ đồ **Use Case**, **Sequence**, **Activity**, **State Machine** và **ERD** theo yêu cầu tại Mục 16.4 của `docs/PLAN.md` để đưa vào quyển báo cáo đồ án.

---

## 1. Sơ đồ Use Case Mở rộng

```mermaid
flowchart LR
    subgraph Actors["Tác nhân"]
        M["Hội viên (Member)"]
        R["Lễ tân / Thu ngân (Receptionist)"]
        T["Huấn luyện viên (Trainer)"]
        MG["Quản lý / Chủ phòng tập (Manager / Owner)"]
        K["Thiết bị Kiosk (/kiosk)"]
        VNP["Cổng VNPay Sandbox"]
    end

    subgraph PortalUC["Cổng Hội viên & Tự phục vụ"]
        UC1["Tự mua gói tập & Thanh toán VNPay"]
        UC2["Ký hợp đồng điện tử & Tạo vé khách mời"]
        UC3["Ghi nhật ký tập luyện & Dinh dưỡng"]
        UC4["Đổi điểm Loyalty & Tham gia thử thách"]
        UC5["Nhắn tin với HLV & Đồng bộ lịch .ics"]
    end

    subgraph OpsUC["Vận hành, CRM & Đào tạo"]
        UC6["Quản lý Lead, Cấp vé tập thử & Chuyển đổi HV"]
        UC7["Quét QR Check-in / Kiosk & Gán tủ đồ"]
        UC8["Soạn giáo án mẫu & Nhân bản cho học viên"]
        UC9["Đăng ký lịch rảnh, Xin nghỉ phép & Dạy thay"]
    end

    subgraph FinUC["Nhân sự, Tài chính & Hệ thống"]
        UC10["Tổng hợp công, Hoa hồng & Chốt bảng lương"]
        UC11["Đối soát ca thu ngân, Chi phí & Báo cáo P&L"]
        UC12["Import dữ liệu cũ (Dry-run / Commit) & Setup Wizard"]
    end

    M --> UC1
    M --> UC2
    M --> UC3
    M --> UC4
    M --> UC5
    UC1 --> VNP

    R --> UC6
    R --> UC7
    K --> UC7

    T --> UC8
    T --> UC9
    T --> UC5

    MG --> UC9
    MG --> UC10
    MG --> UC11
    MG --> UC12
```

---

## 2. Sơ đồ Tuần tự (Sequence Diagrams)

### 2.1 Thanh toán trực tuyến VNPay (Redirect → IPN Webhook → Kích hoạt gói)

```mermaid
sequenceDiagram
    autonumber
    actor Member as Hội viên (Web /me)
    participant Web as gymmaster-web
    participant API as gymmaster-api
    participant DB as PostgreSQL
    participant VNPay as VNPay Sandbox

    Member->>Web: Chọn gói tập + mã giảm giá, bấm "Thanh toán VNPay"
    Web->>API: POST /v1/member/subscriptions/checkout
    API->>DB: BEGIN TX: Tạo Subscription(PENDING) + Invoice(UNPAID) + OnlinePaymentTxn(PENDING)
    DB-->>API: COMMIT
    API-->>Web: Trả về payment_url (ký HMAC-SHA512)
    Web->>VNPay: Redirect trình duyệt sang VNPay Gateway
    Member->>VNPay: Nhập thẻ NCB test & OTP xác nhận
    VNPay-->>API: GET /v1/public/payments/vnpay/ipn (Server-to-Server Webhook)
    API->>API: Kiểm tra chữ ký HMAC-SHA512 & khớp số tiền vnp_Amount
    API->>DB: SELECT ... FOR UPDATE online_payment_txns (Kiểm tra Idempotency)
    API->>DB: Cập nhật Txn(SUCCESS) + Tạo Payment(VNPAY) + Invoice(PAID) + Subscription(ACTIVE)
    API->>API: Đẩy thông báo SSE + Gửi email biên lai điện tử
    API-->>VNPay: {"RspCode": "00", "Message": "Confirm Success"}
    VNPay->>Web: Redirect về /payment/result?vnp_TxnRef=...
    Web->>API: GET /v1/public/payments/vnpay/return (Xác thực hiển thị)
    API-->>Web: Kết quả giao dịch thành công
```

### 2.2 Luồng Quên mật khẩu & Đặt lại mật khẩu an toàn

```mermaid
sequenceDiagram
    autonumber
    actor User as Người dùng
    participant FE as Web / Admin
    participant API as gymmaster-api
    participant DB as PostgreSQL
    participant Mail as Mailpit / SMTP

    User->>FE: Nhập email tại /forgot-password
    FE->>API: POST /v1/auth/forgot-password
    API->>DB: Tìm user ACTIVE theo email
    alt Email tồn tại
        API->>API: Sinh rawToken ngẫu nhiên 32 bytes, băm SHA-256
        API->>DB: Lưu password_resets(token_hash, expires_at = +30m)
        API->>Mail: Gửi email chứa link /reset-password?token=<rawToken>
    end
    API-->>FE: Luôn trả thông báo thành công (chống dò email)
    User->>FE: Mở link từ email, nhập mật khẩu mới
    FE->>API: POST /v1/auth/reset-password {token, new_password}
    API->>DB: BEGIN TX: SELECT password_resets FOR UPDATE (used_at IS NULL AND expires_at > NOW())
    API->>DB: Cập nhật users.password_hash + password_resets.used_at = NOW()
    API->>DB: Thu hồi toàn bộ refresh_tokens của user
    DB-->>API: COMMIT
    API-->>FE: Đặt lại mật khẩu thành công
```

### 2.3 Check-in qua Thiết bị Kiosk (`/kiosk`)

```mermaid
sequenceDiagram
    autonumber
    actor Member as Hội viên / Khách
    participant Kiosk as Màn hình Kiosk (/kiosk)
    participant API as gymmaster-api (DeviceService + CheckinService)
    participant Rules as CheckinRule Engine
    participant DB as PostgreSQL

    loop Mỗi 60 giây
        Kiosk->>API: POST /v1/public/device/heartbeat (Header X-Device-Key)
        API->>DB: Cập nhật access_devices(last_seen_at = NOW(), status = 'ACTIVE')
    end

    Member->>Kiosk: Đưa mã QR (GM động / TP-... / DP-...) vào máy quét
    Kiosk->>API: POST /v1/public/device/checkin (X-Device-Key + qr_token)
    API->>DB: Xác thực SHA256(X-Device-Key) -> lấy BranchID của thiết bị
    API->>Rules: Thực thi chuỗi CheckinRule (Status -> Holiday -> Contract -> Installment -> Scope -> OffPeak -> Sessions -> Cooldown)
    alt Hợp lệ (ALLOWED)
        API->>DB: Ghi checkins(ALLOWED, method='DEVICE') + Trừ buổi (nếu gói SESSION)
        API->>API: Phát sự kiện SSE tới màn hình lễ tân
        API-->>Kiosk: {allow: true, result: "ALLOWED", member_name, plan_name}
        Kiosk-->>Member: Hiển thị toàn màn hình XANH "MỜI VÀO CỬA" (tự đóng sau 3s)
    else Bị từ chối (DENIED)
        API->>DB: Ghi checkins(DENIED, deny_reason)
        API-->>Kiosk: {allow: false, result: "DENIED", deny_reason, message}
        Kiosk-->>Member: Hiển thị toàn màn hình ĐỎ "TỪ CHỐI VÀO CỬA" + Lý do
    end
```

### 2.4 Xác thực 2 lớp (2FA TOTP – RFC 6238)

```mermaid
sequenceDiagram
    autonumber
    actor User as Nhân viên / Hội viên
    participant FE as Admin / Portal
    participant API as PrivacyService (cryptoext)
    participant DB as PostgreSQL

    User->>FE: Bấm "Thiết lập 2FA" trong Cài đặt bảo mật
    FE->>API: POST /v1/member/security/mfa/setup
    API->>API: Sinh Secret Base32 20 bytes & chuỗi otpauth://totp/GymMaster:...
    API->>DB: Lưu user_mfa(secret_encrypted, enabled_at = NULL)
    API-->>FE: Trả về secret & otpauth_url (hiển thị mã QR)
    User->>FE: Quét QR bằng Google Authenticator, nhập mã 6 số
    FE->>API: POST /v1/member/security/mfa/enable {code}
    API->>API: VerifyTOTPCode(secret, code, ±1 step 30s)
    API->>DB: Cập nhật enabled_at = NOW() + Sinh 6 mã khôi phục dự phòng (recovery_codes)
    API-->>FE: Trả về danh sách mã khôi phục
```

---

## 3. Sơ đồ Hoạt động (Activity Diagrams)

### 3.1 Vòng đời Khách tiềm năng CRM (`NEW` → `CONVERTED` / `LOST`)

```mermaid
flowchart TD
    A(["Khách gửi form Tập thử trên Landing Page / Lễ tân tạo mới"]) --> B["Trạng thái: NEW (Mới tiếp nhận)"]
    B --> C["Thông báo realtime cho Lễ tân chi nhánh"]
    C --> D["Lễ tân gọi điện / nhắn tin tư vấn (Ghi LeadActivity)"]
    D --> E["Trạng thái: CONTACTED (Đã liên hệ)"]
    E --> F{"Khách muốn đến tập thử?"}
    F -- "Có" --> G["Cấp vé tập thử QR (TP-...) -> Trạng thái: TRIAL"]
    G --> H["Khách quét QR tại quầy / Kiosk -> Vé chuyển USED"]
    F -- "Mua gói ngay" --> I["Bán gói tập -> Chuyển đổi thành Hội viên"]
    H --> J{"Khách đồng ý đăng ký gói?"}
    J -- "Đồng ý" --> I
    I --> K(["Trạng thái: CONVERTED (Đã gắn member_id)"])
    J -- "Từ chối / Không phản hồi" --> L(["Trạng thái: LOST (Thất bại)"])
```

### 3.2 Quy trình Trả góp & Kiểm soát Quá hạn Check-in

```mermaid
flowchart TD
    A(["Hội viên mua gói tập / PT giá trị cao"]) --> B["Tạo Kế hoạch trả góp (2–12 kỳ hàng tháng)"]
    B --> C["Sinh các kỳ installment_schedules (PENDING)"]
    C --> D{"Đến ngày đáo hạn due_date"}
    D -- "Đã thu tiền kỳ trả góp" --> E["Kỳ chuyển sang PAID + Tạo Payment cho Hóa đơn"]
    D -- "Quá hạn chưa thanh toán" --> F["Cron 00:25 đánh dấu kỳ OVERDUE + Nhắc nợ"]
    F --> G["Hội viên quét QR Check-in"]
    G --> H(["CheckinRule chặn: INSTALLMENT_OVERDUE (42003)"])
    H --> I["Hội viên thanh toán kỳ nợ tại quầy / VNPay"]
    I --> E
```

---

## 4. Sơ đồ Trạng thái (State Diagrams)

### 4.1 Giao dịch Thanh toán Trực tuyến (`online_payment_txns`)
```mermaid
stateDiagram-v2
    [*] --> PENDING: Tạo link thanh toán VNPay (TTL 15 phút)
    PENDING --> SUCCESS: Nhận IPN hợp lệ (vnp_ResponseCode = 00)
    PENDING --> FAILED: Nhận IPN thất bại / khách hủy giao dịch
    PENDING --> EXPIRED: Cron quá 15 phút chưa thanh toán
    SUCCESS --> [*]
    FAILED --> [*]
    EXPIRED --> [*]
```

### 4.2 Bảng lương Nhân sự (`payrolls`)
```mermaid
stateDiagram-v2
    [*] --> DRAFT: Tổng hợp công + hoa hồng (Generate)
    DRAFT --> DRAFT: Điều chỉnh lương cơ bản / thưởng / phạt
    DRAFT --> LOCKED: Quản lý / Chủ phòng tập chốt lương (Lock)
    LOCKED --> PAID: Xác nhận chi trả lương & chốt hoa hồng PT (Pay)
    PAID --> [*]
```

### 4.3 Yêu cầu Dữ liệu Cá nhân – Nghị định 13/2023 (`data_requests`)
```mermaid
stateDiagram-v2
    [*] --> PENDING: Hội viên gửi yêu cầu EXPORT / DELETE
    PENDING --> COMPLETED: Admin phê duyệt (Xuất JSON hoặc Ẩn danh hóa PII)
    PENDING --> REJECTED: Admin từ chối (Còn công nợ / nghĩa vụ hợp đồng)
    COMPLETED --> [*]
    REJECTED --> [*]
```

---

## 5. Sơ đồ Thực thể Liên kết Mở rộng (ERD – Giai đoạn 2 đến 4)

```mermaid
erDiagram
    MEMBERS ||--o{ SUBSCRIPTIONS : "đăng ký"
    MEMBERS ||--o{ WORKOUT_PLANS : "nhận giáo án"
    MEMBERS ||--o{ WORKOUT_LOGS : "ghi nhật ký tập"
    MEMBERS ||--o{ REVIEWS : "đánh giá"
    MEMBERS ||--o{ LOYALTY_LEDGERS : "tích/đổi điểm"
    MEMBERS ||--o{ MEMBER_BADGES : "đạt huy hiệu"
    MEMBERS ||--o{ CHALLENGE_PARTICIPANTS : "tham gia"
    MEMBERS ||--o{ FOOD_LOGS : "ghi bữa ăn"
    MEMBERS ||--o{ MEMBER_CONTRACTS : "ký e-contract"
    MEMBERS ||--o{ DAY_PASSES : "mời khách"
    MEMBERS ||--o{ CONSENTS : "đồng ý NĐ13"
    MEMBERS ||--o{ DATA_REQUESTS : "yêu cầu dữ liệu"
    MEMBERS ||--o{ MEMBER_BRANCH_TRANSFERS : "chuyển chi nhánh"

    INVOICES ||--o{ ONLINE_PAYMENT_TXNS : "thanh toán VNPay"
    INVOICES ||--o| INSTALLMENT_PLANS : "trả góp"
    INSTALLMENT_PLANS ||--|{ INSTALLMENT_SCHEDULES : "kỳ thanh toán"

    BRANCHES ||--o{ ACCESS_DEVICES : "thiết bị Kiosk/Cổng"
    BRANCHES ||--o{ LOCKERS : "tủ đồ"
    LOCKERS ||--o{ LOCKER_RENTALS : "lượt thuê"
    BRANCHES ||--o{ EQUIPMENTS : "thiết bị tập"
    EQUIPMENTS ||--o{ MAINTENANCE_LOGS : "bảo trì"
    BRANCHES ||--o{ PAYROLLS : "bảng lương"
    PAYROLLS ||--|{ PAYROLL_ITEMS : "dòng lương NV"
    BRANCHES ||--o{ EXPENSES : "chi phí"
    BRANCHES ||--o{ CASH_SESSIONS : "ca thu ngân"
    BRANCHES ||--o{ BRANCH_HOLIDAYS : "lịch nghỉ lễ"
    BRANCHES ||--o{ SALES_TARGETS : "chỉ tiêu KPI"
    BRANCHES ||--o{ LEGACY_IMPORT_JOBS : "import dữ liệu cũ"

    PT_CONTRACTS ||--o| CONVERSATIONS : "kênh chat PT"
    CONVERSATIONS ||--o{ MESSAGES : "tin nhắn"
    USERS ||--o| USER_MFA : "bảo mật 2FA"
    USERS ||--o| TRAINER_PUBLIC_PROFILES : "hồ sơ HLV công khai"
    USERS ||--o{ NOTIFICATION_PREFERENCES : "tùy chọn thông báo"
```
