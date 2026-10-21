-- Bổ sung cột audit cho access_devices (entity nhúng BaseEntity nhưng 000017 thiếu)
ALTER TABLE access_devices
    ADD COLUMN IF NOT EXISTS created_by BIGINT,
    ADD COLUMN IF NOT EXISTS updated_by BIGINT;
