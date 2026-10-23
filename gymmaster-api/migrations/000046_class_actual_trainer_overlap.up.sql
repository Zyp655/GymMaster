-- QA vòng 4 (C1): chống trùng lịch theo HLV THỰC DẠY của buổi lớp = HLV dạy thay (nếu đã phân công), ngược lại HLV chính.
-- Trước đây EXCLUDE chỉ so trainer_id nên HLV đang dạy thay vẫn bị xếp thêm lớp trùng giờ. Giữ nguyên tên constraint để
-- service dịch lỗi 23P01 sang 409 như cũ.
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS ex_class_trainer_overlap;
ALTER TABLE class_schedules
    ADD CONSTRAINT ex_class_trainer_overlap EXCLUDE USING gist (
        (COALESCE(substitute_trainer_id, trainer_id)) WITH =,
        tstzrange(start_at, end_at) WITH &&
    ) WHERE (status <> 'CANCELLED');

-- Lọc lịch lớp / lịch dạy theo HLV thực dạy.
CREATE INDEX IF NOT EXISTS idx_class_schedules_actual_trainer_time
    ON class_schedules ((COALESCE(substitute_trainer_id, trainer_id)), start_at);
