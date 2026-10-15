# Tài liệu Hướng dẫn Sử dụng Hệ thống GymMaster (Theo Vai trò)

Tài liệu này hướng dẫn chi tiết cách vận hành hệ thống **GymMaster** theo 4 nhóm vai trò chính tại trung tâm Fitness:
1. **Chủ phòng tập & Quản lý (`OWNER` / `MANAGER`)**
2. **Lễ tân & Thu ngân (`RECEPTIONIST`)**
3. **Huấn luyện viên cá nhân & Lớp nhóm (`TRAINER`)**
4. **Hội viên (`MEMBER`)**

---

## 1. Dành cho Chủ phòng tập & Quản lý (`OWNER` / `MANAGER`)

### 1.1 Thiết lập ban đầu (Setup Wizard) & Chuyển đổi dữ liệu cũ
- Truy cập **`/app/system-tools` (Thiết lập & Chuyển đổi)**:
  - **Tab Wizard Thiết lập ban đầu (6 bước)**:
    1. Khai báo thông tin trung tâm (tên thương hiệu, hotline, email, mã số thuế).
    2. Tạo danh sách chi nhánh (`branches`) và sức chứa tối đa (`capacity`).
    3. Thiết lập các phòng tập (`rooms`) tại từng chi nhánh.
    4. Khởi tạo các gói tập mẫu (`membership_plans`) và gói PT (`pt_packages`).
    5. Tạo tài khoản nhân sự & phân quyền (`roles`, `staff`, `trainers`).
    6. Cấu hình giờ mở cửa, thời gian chống quét trùng check-in và quy định hủy lớp/PT.
  - **Tab Chuyển đổi dữ liệu cũ (`Import Legacy`)**:
    - Tải dữ liệu hội viên, gói đang chạy, hợp đồng PT và công nợ từ hệ thống cũ.
    - Luôn bấm **"Chạy thử (Dry-Run)"** trước để kiểm tra các dòng lỗi (trùng SĐT, sai mã gói, sai định dạng ngày).
    - Khi 0 lỗi, bấm **"Ghi chính thức (Commit)"** để hệ thống tạo dữ liệu với nhãn truy vết `source = 'IMPORT'`.

### 1.2 Quản lý Nhân sự, Chấm công & Bảng lương (`/app/payrolls`, `/app/trainer-schedules`)
- **Duyệt nghỉ phép & phân công dạy thay (`/app/trainer-schedules`)**:
  - Xem danh sách đơn xin nghỉ của HLV, bấm **"Xem lịch bị ảnh hưởng"** để kiểm tra các buổi lớp nhóm và PT trùng thời gian nghỉ, sau đó gán HLV dạy thay trước khi duyệt đơn.
  - Khai báo **Lịch nghỉ lễ phòng tập**: nếu chọn *"Đóng cửa & Tự động cộng bù ngày"*, hệ thống sẽ tự cộng thêm số ngày nghỉ lễ vào `end_date` của tất cả gói tập theo thời gian (`TIME`) đang `ACTIVE` tại chi nhánh đó.
- **Tính lương & Khóa bảng lương (`/app/payrolls`)**:
  - Chọn chi nhánh và kỳ lương (`YYYY-MM`), bấm **"Tạo / Tính lại bảng lương"**. Hệ thống tự động tổng hợp:
    - Lương cơ bản theo ngày công thực tế (`staff_attendances`).
    - Hoa hồng dạy PT & lớp nhóm (`trainer_commissions`).
    - Hoa hồng bán gói tập (`2%` doanh thu gói tập do nhân viên chốt).
  - Sau khi kiểm tra và điều chỉnh thưởng/phạt (khi ở trạng thái `DRAFT`), bấm **"Khóa bảng lương"** (`LOCKED`) và **"Xác nhận chi trả"** (`PAID`).

### 1.3 Quản trị Tài chính, Chi phí & Báo cáo P&L (`/app/finance`, `/app/kpi-analytics`)
- **Chi phí vận hành**: Ghi nhận tiền thuê mặt bằng, điện nước, marketing, sửa chữa thiết bị và duyệt chi (`APPROVED`).
- **Báo cáo Lãi / Lỗ (P&L)**: Theo dõi Doanh thu gộp – Hoàn tiền – Giá vốn hàng POS – Chi phí vận hành – Quỹ lương = **Lợi nhuận ròng**.
- **Chỉ tiêu KPI & Cohort Retention (`/app/kpi-analytics`)**: Thiết lập mục tiêu doanh số tháng cho chi nhánh/nhân viên và theo dõi tỷ lệ giữ chân hội viên theo tháng gia nhập.

---

## 2. Dành cho Lễ tân & Thu ngân (`RECEPTIONIST`)

### 2.1 Tìm kiếm nhanh (`Ctrl + K`) & Quầy Check-in (`/app/checkin`)
- Nhấn tổ hợp phím **`Ctrl + K`** (hoặc nút **Tìm kiếm nhanh** trên thanh công cụ) để tra cứu tức thì hội viên theo SĐT/mã `GMxxxxxx`, hóa đơn hoặc sản phẩm POS.
- Tại màn hình **Quầy Check-in (`/app/checkin`)**:
  - Quét mã QR động của hội viên (TTL 60 giây), mã vé tập thử (`TP-...`) hoặc mã vé ngày/vé khách (`DP-...`).
  - Hệ thống kiểm tra tự động qua chuỗi `CheckinRule`: trạng thái hội viên, lịch nghỉ lễ chi nhánh, hợp đồng điện tử, kỳ trả góp quá hạn, phạm vi chi nhánh, khung giờ gói Off-Peak, số buổi còn lại và thời gian chống quét trùng.

### 2.2 Quản lý Khách tiềm năng CRM (`/app/leads`) & Bán gói tập (`/app/subscriptions`)
- Tiếp nhận khách đăng ký tập thử từ Website tại **`/app/leads`** (chế độ Kanban hoặc Bảng).
- Ghi nhận nhật ký gọi điện/tư vấn (`CALL`, `SMS`, `VISIT`), cấp **Vé tập thử QR (`TP-...`)**, và bấm **"Chuyển đổi thành Hội viên"** khi khách đồng ý mua gói.
- Hỗ trợ lập kế hoạch **Trả góp nhiều kỳ** hoặc xuất **Hóa đơn điện tử** tại `/app/finance`.

### 2.3 Sơ đồ Tủ đồ (`/app/lockers`) & Đối soát Ca Thu ngân (`/app/finance`)
- **Sơ đồ tủ đồ (`/app/lockers`)**: Bấm vào ô tủ trống màu xanh để gán tủ theo lượt ngày (`DAILY`) hoặc cho thuê theo tháng (`MONTHLY`); bấm vào tủ đang bận để giải phóng tủ khi hội viên trả chìa/vòng từ.
- **Đối soát ca thu ngân (`/app/finance` → Tab Đối soát ca)**:
  - Đầu ca: Nhập **Tiền mặt đầu ca (`opening_cash`)** để mở ca.
  - Cuối ca: Kiểm đếm tiền mặt thực tế trong két (`counted_cash`), hệ thống tự đối chiếu với `expected_cash` (tiền đầu ca + thu tiền mặt trong ca) và ghi nhận chênh lệch.

---

## 3. Dành cho Huấn luyện viên (`TRAINER`)

### 3.1 Lịch dạy, Điểm danh Lớp & Hoàn thành Buổi PT (`/app/classes`, `/app/pt/sessions`)
- Đăng nhập vào cổng quản trị bằng tài khoản HLV để xem các buổi dạy lớp và buổi PT được phân công.
- Sau mỗi buổi PT, bấm **"Hoàn thành buổi tập"**, nhập ghi chú bài tập & nhận xét cho học viên — hệ thống tự động trừ 1 buổi trong hợp đồng PT và ghi nhận hoa hồng cho HLV.

### 3.2 Soạn Giáo án Tập luyện & Thực đơn Dinh dưỡng (`/app/workout-plans`, `/app/engagement`)
- **Giáo án tập luyện (`/app/workout-plans`)**:
  - Tạo giáo án mẫu hoặc giáo án cá nhân hóa theo từng ngày trong tuần (`Thứ 2` → `Chủ nhật`), chọn bài tập từ thư viện (`/app/exercises`), chỉ định số hiệp (`sets`), số lần lặp (`reps`), mức tạ (`weight`) và thời gian nghỉ (`rest_sec`).
  - Dùng tính năng **"Nhân bản giáo án (Clone)"** để giao nhanh giáo án mẫu cho học viên mới.
- **Thực đơn dinh dưỡng (`/app/engagement`)**: Soạn thực đơn mẫu theo mục tiêu Calo/Macro (Protein – Carbs – Fat) từ thư viện món ăn Việt Nam.

### 3.3 Nhắn tin với Học viên & Đăng ký Lịch rảnh (`/app/cms-marketing`, `/app/trainer-schedules`)
- Trao đổi trực tiếp với học viên có hợp đồng PT đang hiệu lực tại **`/app/cms-marketing` → Tab Tin nhắn HLV ↔ Hội viên**.
- Cập nhật khung giờ rảnh hàng tuần và gửi đơn xin nghỉ phép tại **`/app/trainer-schedules`**.

---

## 4. Dành cho Hội viên (`MEMBER` — Cổng `/me`)

- **Check-in QR (`/me/qr`)**: Mở mã QR tự làm mới mỗi 60 giây (có lưu dự phòng offline PWA khi mất mạng) để quét tại quầy hoặc màn hình Kiosk (`/kiosk`).
- **Mua gói & Thanh toán VNPay (`/me/subscriptions/buy`, `/me/invoices`)**: Tự chọn gói tập, áp mã khuyến mãi, thanh toán qua VNPay Sandbox và theo dõi lịch trả góp.
- **Đặt lớp & Đặt lịch PT (`/me/classes`, `/me/pt`, `/me/bookings`)**: Đặt chỗ lớp học nhóm (hỗ trợ danh sách chờ `WAITLIST` tự động đôn chỗ), đặt lịch PT, xuất lịch **`.ics`** sang Google/Apple Calendar và đánh giá 1–5 sao sau buổi tập.
- **Giáo án, Chỉ số cơ thể & Dinh dưỡng (`/me/workout`, `/me/progress`, `/me/nutrition`)**: Xem giáo án hôm nay, ghi nhận mức tạ thực tế từng hiệp, theo dõi biểu đồ BMI/% mỡ/mức tạ và nhật ký Calo bữa ăn.
- **Điểm thưởng, Thử thách & Hợp đồng điện tử (`/me/rewards`, `/me/challenges`, `/me/contracts`)**: Đổi điểm lấy voucher/quà tặng, tham gia thử thách phòng tập, ký hợp đồng điện tử trên màn hình cảm ứng và tạo vé khách mời (`DP-...`) cho bạn bè.
- **Tin nhắn HLV & Bảo mật tài khoản (`/me/messages`, `/me/account`)**: Nhắn tin trực tiếp với HLV PT, bật xác thực 2 lớp (`2FA TOTP`), quản lý phiên đăng nhập và tùy chỉnh quyền riêng tư theo Nghị định 13/2023/NĐ-CP.
