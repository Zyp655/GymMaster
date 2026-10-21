-- QA vòng 2 – vận hành: mã thiết bị kiểm soát ra vào chỉ duy nhất trong các thiết bị CHƯA xóa (xóa mềm xong được
-- dùng lại mã cũ). Idempotent (chạy lại an toàn).
ALTER TABLE access_devices DROP CONSTRAINT IF EXISTS access_devices_code_key;
DROP INDEX IF EXISTS access_devices_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_access_devices_code_active
    ON access_devices (UPPER(code)) WHERE deleted_at IS NULL;

-- Lượt check-in bị từ chối lưu thông điệp chi tiết lúc quét (vd "Chi nhánh nghỉ lễ (Tết)", "Ngoài giờ mở cửa ngày lễ …",
-- giờ check-in trước…) để lịch sử hiển thị đúng lý do kể cả khi lịch lễ/giờ mở cửa thay đổi sau đó.
ALTER TABLE checkins ADD COLUMN IF NOT EXISTS deny_message VARCHAR(255) NOT NULL DEFAULT '';
