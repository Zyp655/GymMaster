DROP INDEX IF EXISTS idx_login_attempts_email_time;
ALTER TABLE member_register_otps DROP COLUMN IF EXISTS attempts;
