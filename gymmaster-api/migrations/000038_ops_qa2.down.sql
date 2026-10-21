ALTER TABLE checkins DROP COLUMN IF EXISTS deny_message;
DROP INDEX IF EXISTS uq_access_devices_code_active;
-- Khôi phục UNIQUE(code) toàn bảng khi dữ liệu cho phép (không còn mã trùng giữa thiết bị đã xóa và đang dùng).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'access_devices_code_key')
       AND NOT EXISTS (SELECT code FROM access_devices GROUP BY code HAVING COUNT(*) > 1) THEN
        ALTER TABLE access_devices ADD CONSTRAINT access_devices_code_key UNIQUE (code);
    END IF;
END $$;
