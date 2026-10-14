-- =====================================================================
-- 000011 – Quên mật khẩu, chống brute-force đăng nhập & OTP đăng ký hội viên
-- =====================================================================

-- Token đặt lại mật khẩu (lưu SHA-256, TTL 30 phút, dùng 1 lần).
CREATE TABLE password_resets (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  CHAR(64)     NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL,
    used_at     TIMESTAMPTZ,
    ip          VARCHAR(64)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_password_resets_user ON password_resets (user_id);
CREATE INDEX idx_password_resets_expires ON password_resets (expires_at);

-- Nhật ký thử đăng nhập (đếm số lần sai theo email + IP để khóa tạm).
CREATE TABLE login_attempts (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(150) NOT NULL,
    ip            VARCHAR(64)  NOT NULL DEFAULT '',
    attempted_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    success       BOOLEAN      NOT NULL DEFAULT false
);
CREATE INDEX idx_login_attempts_email_ip_time ON login_attempts (lower(email), ip, attempted_at DESC);

-- OTP xác thực email khi hội viên có sẵn hồ sơ tại quầy tự tạo tài khoản cổng hội viên (Sprint 17).
CREATE TABLE member_register_otps (
    id             BIGSERIAL PRIMARY KEY,
    member_id      BIGINT       NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    email          VARCHAR(150) NOT NULL,
    password_hash  VARCHAR(100) NOT NULL,
    otp_hash       CHAR(64)     NOT NULL,
    expires_at     TIMESTAMPTZ  NOT NULL,
    used_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_member_register_otps_member ON member_register_otps (member_id, created_at DESC);
