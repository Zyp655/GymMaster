-- =====================================================================
-- 000003 – Hội viên và chỉ số cơ thể
-- =====================================================================

-- pgcrypto: gen_random_bytes() cho qr_secret.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SEQUENCE member_code_seq;

CREATE TABLE members (
    id                       BIGSERIAL PRIMARY KEY,
    -- Tài khoản cổng hội viên (role MEMBER). NULL = chưa cấp tài khoản.
    user_id                  BIGINT UNIQUE REFERENCES users (id),
    member_code              VARCHAR(20)  NOT NULL DEFAULT ('GM' || lpad(nextval('member_code_seq')::text, 6, '0')),
    full_name                VARCHAR(150) NOT NULL,
    gender                   VARCHAR(10)  NOT NULL DEFAULT 'OTHER' CHECK (gender IN ('MALE', 'FEMALE', 'OTHER')),
    dob                      DATE,
    phone                    VARCHAR(20)  NOT NULL,
    email                    VARCHAR(150) NOT NULL DEFAULT '',
    id_card                  VARCHAR(20)  NOT NULL DEFAULT '',
    address                  VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url               VARCHAR(500) NOT NULL DEFAULT '',
    emergency_contact_name   VARCHAR(150) NOT NULL DEFAULT '',
    emergency_contact_phone  VARCHAR(20)  NOT NULL DEFAULT '',
    home_branch_id           BIGINT       NOT NULL REFERENCES branches (id),
    source                   VARCHAR(20)  NOT NULL DEFAULT 'WALK_IN'
                             CHECK (source IN ('WALK_IN', 'REFERRAL', 'FACEBOOK', 'WEBSITE', 'OTHER')),
    status                   VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'BANNED')),
    -- Bí mật riêng của từng hội viên để ký QR động (pkg/qrtoken). Đổi = vô hiệu mọi QR cũ.
    qr_secret                VARCHAR(64)  NOT NULL DEFAULT encode(gen_random_bytes(32), 'hex'),
    -- Bị khóa đặt lớp đến thời điểm này (phạt no-show, xem 000007).
    booking_blocked_until    TIMESTAMPTZ,
    note                     TEXT         NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ,
    created_by               BIGINT,
    updated_by               BIGINT
);
CREATE UNIQUE INDEX uq_members_code  ON members (member_code);
CREATE UNIQUE INDEX uq_members_phone ON members (phone) WHERE deleted_at IS NULL;
CREATE INDEX idx_members_branch ON members (home_branch_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_members_name   ON members (lower(full_name));
CREATE INDEX idx_members_deleted_at ON members (deleted_at);
CREATE TRIGGER trg_members_updated_at BEFORE UPDATE ON members FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Chỉ số cơ thể theo thời gian. BMI là cột sinh tự động (GORM: tag `gorm:"->"` – chỉ đọc).
CREATE TABLE body_metrics (
    id              BIGSERIAL PRIMARY KEY,
    member_id       BIGINT       NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    measured_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    weight_kg       NUMERIC(5,2) NOT NULL CHECK (weight_kg > 0),
    height_cm       NUMERIC(5,2) NOT NULL CHECK (height_cm > 0),
    body_fat_pct    NUMERIC(5,2) CHECK (body_fat_pct BETWEEN 0 AND 100),
    muscle_mass_kg  NUMERIC(5,2) CHECK (muscle_mass_kg >= 0),
    waist_cm        NUMERIC(5,2),
    chest_cm        NUMERIC(5,2),
    hip_cm          NUMERIC(5,2),
    bmi             NUMERIC(5,2) GENERATED ALWAYS AS (round(weight_kg / ((height_cm / 100) * (height_cm / 100)), 2)) STORED,
    note            VARCHAR(255) NOT NULL DEFAULT '',
    measured_by     BIGINT REFERENCES users (id),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_body_metrics_member ON body_metrics (member_id, measured_at DESC);

INSERT INTO permissions (code, module, name) VALUES
    ('member.view',    'member', 'Xem hội viên'),
    ('member.create',  'member', 'Thêm hội viên'),
    ('member.update',  'member', 'Sửa hội viên, cấp tài khoản cổng hội viên'),
    ('member.delete',  'member', 'Xóa hội viên'),
    ('member.metric',  'member', 'Ghi chỉ số cơ thể'),
    ('member.import',  'member', 'Nhập hội viên từ Excel');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'member' WHERE r.code IN ('OWNER', 'MANAGER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p
  ON p.code IN ('member.view', 'member.create', 'member.update', 'member.metric')
WHERE r.code = 'RECEPTIONIST';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('member.view', 'member.metric')
WHERE r.code = 'TRAINER';
