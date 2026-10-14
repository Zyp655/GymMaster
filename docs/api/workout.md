# API contract – Sprint 14 (Thư viện bài tập, Giáo án & Nhật ký tập luyện)

Base URL (dev): `http://localhost:8000/v1`.

## 1. Quyền (Permissions)

| Mã quyền | Mô tả | Vai trò mặc định |
|---|---|---|
| `workout.view` | Xem thư viện bài tập & giáo án | `OWNER`, `MANAGER`, `TRAINER`, `RECEPTIONIST` |
| `workout.manage` | Thêm/sửa/xóa bài tập, soạn & nhân bản giáo án | `OWNER`, `MANAGER`, `TRAINER` |

> **Giới hạn phạm vi HLV (`TRAINER`)**: HLV chỉ được tạo/sửa/xóa/clone giáo án do chính mình phụ trách (`trainer_id = user.id`) và cho học viên của mình (hoặc giáo án mẫu `member_id = null`).

---

## 2. Thư viện bài tập (`exercises`)

Nhóm cơ (`muscle_group`): `CHEST` | `BACK` | `SHOULDERS` | `LEGS` | `ARMS` | `CORE` | `CARDIO` | `FULL_BODY`.

| Method | Path | Quyền / Role | Mô tả |
|---|---|---|---|
| GET | `/v1/admin/exercises` | `workout.view,workout.manage` | Danh sách bài tập (phân trang, lọc `muscle_group`, `search`) |
| POST | `/v1/admin/exercises` | `workout.manage` | Thêm bài tập mới |
| PUT | `/v1/admin/exercises/{id}` | `workout.manage` | Cập nhật bài tập |
| DELETE | `/v1/admin/exercises/{id}` | `workout.manage` | Xóa mềm bài tập |
| GET | `/v1/member/exercises` | `role: MEMBER` | Danh sách toàn bộ bài tập cho cổng hội viên |

---

## 3. Giáo án tập luyện (`workout_plans`)

Trạng thái (`status`): `DRAFT` | `ACTIVE` | `COMPLETED` | `ARCHIVED`.

| Method | Path | Quyền / Role | Mô tả |
|---|---|---|---|
| GET | `/v1/admin/workout-plans` | `workout.view,workout.manage` | Danh sách giáo án (lọc `member_id`, `trainer_id`, `status`, `search`) |
| GET | `/v1/admin/workout-plans/{id}` | `workout.view,workout.manage` | Chi tiết giáo án kèm `days` và `items` |
| POST | `/v1/admin/workout-plans` | `workout.manage` | Tạo giáo án mới (kèm danh sách ngày `days` và bài tập `items`) |
| PUT | `/v1/admin/workout-plans/{id}` | `workout.manage` | Cập nhật giáo án và toàn bộ cấu trúc ngày/bài tập |
| DELETE | `/v1/admin/workout-plans/{id}` | `workout.manage` | Xóa mềm giáo án |
| POST | `/v1/admin/workout-plans/{id}/clone` | `workout.manage` | Nhân bản giáo án (từ giáo án mẫu hoặc giáo án có sẵn) cho hội viên |
| GET | `/v1/member/workout-plans` | `role: MEMBER` | Danh sách giáo án của hội viên đang đăng nhập (kèm `days` và `items`) |

Ví dụ body `POST /v1/admin/workout-plans`:
```json
{
  "member_id": 12,
  "trainer_id": 4,
  "name": "Giáo án Hypertrophy 4 buổi/tuần",
  "goal": "Tăng cơ giảm mỡ trong 8 tuần",
  "start_date": "2026-10-14",
  "end_date": "2026-12-14",
  "status": "ACTIVE",
  "days": [
    {
      "day_index": 1,
      "title": "Ngày 1 – Ngực & Tay sau",
      "items": [
        { "exercise_id": 1, "sets": 4, "reps": 10, "weight": 60, "rest_sec": 90, "note": "Hạ thanh đòn chạm ngực nhẹ", "sort": 1 }
      ]
    }
  ]
}
```

---

## 4. Nhật ký tập luyện & Tiến độ mức tạ (`workout_logs`)

### POST `/v1/member/workout-logs` (`role: MEMBER`)
Hội viên tự ghi nhận kết quả thực hiện bài tập.
```json
{
  "plan_item_id": 10,
  "exercise_id": 1,
  "sets": 4,
  "reps": 10,
  "weight": 65,
  "note": "Tăng thêm 5kg so với tuần trước"
}
```

### GET `/v1/member/workout-logs` (`role: MEMBER`)
- Query: `exercise_id` (tùy chọn), `limit` (mặc định 50).
- Response `data`:
```json
{
  "logs": [
    {
      "id": 1,
      "exercise_id": 1,
      "exercise_name": "Barbell Bench Press (Đẩy ngực thanh đòn phẳng)",
      "muscle_group": "CHEST",
      "performed_at": "2026-10-14T18:00:00+07:00",
      "sets": 4,
      "reps": 10,
      "weight": 65,
      "volume": 2600,
      "note": "Tăng thêm 5kg so với tuần trước"
    }
  ],
  "progress": [
    { "date": "2026-10-01", "max_weight": 60, "total_volume": 2400, "total_sets": 4 },
    { "date": "2026-10-14", "max_weight": 65, "total_volume": 2600, "total_sets": 4 }
  ]
}
```
