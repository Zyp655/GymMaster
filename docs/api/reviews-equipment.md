# API contract – Sprint 15 (Đánh giá HLV/Lớp học & Quản lý Thiết bị – Bảo trì)

Base URL (dev): `http://localhost:8000/v1`.

## 1. Quyền (Permissions)

| Mã quyền | Mô tả | Vai trò mặc định |
|---|---|---|
| `review.view` | Xem đánh giá HLV & lớp học | `OWNER`, `MANAGER`, `RECEPTIONIST`, `TRAINER` |
| `equipment.view` | Xem thiết bị & lịch sử bảo trì | `OWNER`, `MANAGER`, `RECEPTIONIST`, `TRAINER` |
| `equipment.manage` | Quản lý thiết bị, báo hỏng, cập nhật bảo trì | `OWNER`, `MANAGER` |

---

## 2. Đánh giá HLV & Lớp học (`reviews`)

### Quy tắc nghiệp vụ
1. Hội viên chỉ được đánh giá buổi PT (`target_type = "PT_SESSION"`) có trạng thái `COMPLETED` hoặc buổi lớp nhóm (`target_type = "CLASS_SCHEDULE"`) mà hội viên có lượt đặt chỗ `ATTENDED`.
2. Chỉ được gửi đánh giá trong vòng **7 ngày** kể từ khi buổi tập hoàn thành.
3. Mỗi hội viên chỉ đánh giá **1 lần** cho mỗi buổi (`UNIQUE(member_id, target_type, target_id)`).
4. Trong cùng transaction tạo đánh giá, hệ thống tự động tính lại và cập nhật `trainer_profiles.rating_avg` và `trainer_profiles.rating_count` của HLV.
5. Khi hoàn thành buổi PT hoặc điểm danh lớp học `ATTENDED`, hệ thống gửi thông báo `REVIEW_PROMPT` nhắc hội viên đánh giá buổi tập.

### Endpoints
| Method | Path | Quyền / Role | Mô tả |
|---|---|---|---|
| POST | `/v1/member/reviews` | `role: MEMBER` | Gửi đánh giá 1–5 sao + nhận xét |
| GET | `/v1/member/reviews` | `role: MEMBER` | Danh sách đánh giá đã gửi của hội viên |
| GET | `/v1/admin/reviews` | `review.view` | Danh sách đánh giá (lọc `trainer_id`, `target_type`, `rating`, `branch_id`) |

---

## 3. Quản lý Thiết bị & Bảo trì (`equipments`, `maintenance_logs`)

### Danh mục & Trạng thái
- Danh mục thiết bị (`category`): `CARDIO` | `STRENGTH` | `FREE_WEIGHT` | `FUNCTIONAL` | `ACCESSORY` | `OTHER`.
- Trạng thái thiết bị (`status`): `ACTIVE` | `BROKEN` | `MAINTENANCE` | `RETIRED`.
- Loại bảo trì (`type`): `PERIODIC` (định kỳ) | `REPAIR` (sửa chữa).
- Trạng thái phiếu bảo trì (`status`): `OPEN` | `IN_PROGRESS` | `DONE`.

### Endpoints
| Method | Path | Quyền | Mô tả |
|---|---|---|---|
| GET | `/v1/admin/equipments` | `equipment.view,equipment.manage` | Danh sách thiết bị (lọc `branch_id`, `status`, `category`, `search`) |
| GET | `/v1/admin/equipments/{id}` | `equipment.view,equipment.manage` | Chi tiết thiết bị kèm lịch sử bảo trì |
| POST | `/v1/admin/equipments` | `equipment.manage` | Thêm thiết bị mới |
| PUT | `/v1/admin/equipments/{id}` | `equipment.manage` | Cập nhật thiết bị |
| DELETE | `/v1/admin/equipments/{id}` | `equipment.manage` | Xóa mềm thiết bị |
| POST | `/v1/admin/equipments/{id}/report-issue` | `equipment.view,equipment.manage` | Báo hỏng thiết bị → chuyển thiết bị sang `BROKEN` và mở phiếu `REPAIR` |
| GET | `/v1/admin/maintenance-logs` | `equipment.view,equipment.manage` | Danh sách phiếu bảo trì |
| POST | `/v1/admin/maintenance-logs` | `equipment.manage` | Tạo phiếu bảo trì |
| PUT | `/v1/admin/maintenance-logs/{id}` | `equipment.manage` | Cập nhật phiếu bảo trì (khi `DONE` tự khôi phục thiết bị `ACTIVE` và tính kỳ bảo trì kế tiếp) |
| DELETE | `/v1/admin/maintenance-logs/{id}` | `equipment.manage` | Xóa phiếu bảo trì |
| GET | `/v1/admin/maintenance-logs/summary` | `equipment.view,report.view` | Thống kê chi phí bảo trì theo chi nhánh / tháng |

### Job định kỳ
- **`remind_maintenance`** (`0 7 * * *` – 07:00 hằng ngày): quét các thiết bị `ACTIVE` có `next_maintenance_date <= CURRENT_DATE` và gửi thông báo `MAINTENANCE_DUE` cho nhân sự có quyền `equipment.manage` tại chi nhánh đó.
