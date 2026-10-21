-- Sửa lỗi bảo mật xác thực (QA): giới hạn số lần nhập sai OTP đăng ký hội viên
-- và index phục vụ khóa đăng nhập theo email (không phụ thuộc IP).
ALTER TABLE member_register_otps ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_login_attempts_email_time
    ON login_attempts (lower(email), attempted_at DESC);
