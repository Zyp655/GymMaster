-- =====================================================================
-- 000004 – Danh mục bán: gói tập, gói PT, khuyến mãi, cấu hình hệ thống
-- =====================================================================

-- Gói tập.
--   TIME    : tập không giới hạn số buổi trong duration_days ngày.
--   SESSION : được session_count lượt check-in, hết hạn sau duration_days ngày.
CREATE TABLE membership_plans (
    id                BIGSERIAL PRIMARY KEY,
    code              VARCHAR(30)   NOT NULL,
    name              VARCHAR(150)  NOT NULL,
    description       TEXT          NOT NULL DEFAULT '',
    type              VARCHAR(10)   NOT NULL CHECK (type IN ('TIME', 'SESSION')),
    duration_days     INT           NOT NULL CHECK (duration_days > 0),
    session_count     INT           CHECK (session_count > 0),
    price             NUMERIC(14,2) NOT NULL CHECK (price >= 0),
    access_scope      VARCHAR(20)   NOT NULL DEFAULT 'SINGLE_BRANCH' CHECK (access_scope IN ('SINGLE_BRANCH', 'ALL_BRANCHES')),
    max_freeze_days   INT           NOT NULL DEFAULT 0 CHECK (max_freeze_days >= 0),
    max_freeze_times  INT           NOT NULL DEFAULT 0 CHECK (max_freeze_times >= 0),
    includes_classes  BOOLEAN       NOT NULL DEFAULT false,
    transfer_fee      NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (transfer_fee >= 0),
    status            VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    sort_order        INT           NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    created_by        BIGINT,
    updated_by        BIGINT,
    CONSTRAINT chk_plan_session CHECK ((type = 'SESSION') = (session_count IS NOT NULL))
);
CREATE UNIQUE INDEX uq_membership_plans_code ON membership_plans (code) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_membership_plans_updated_at BEFORE UPDATE ON membership_plans FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Gói huấn luyện cá nhân (PT).
CREATE TABLE pt_packages (
    id             BIGSERIAL PRIMARY KEY,
    code           VARCHAR(30)   NOT NULL,
    name           VARCHAR(150)  NOT NULL,
    description    TEXT          NOT NULL DEFAULT '',
    session_count  INT           NOT NULL CHECK (session_count > 0),
    session_minutes INT          NOT NULL DEFAULT 60 CHECK (session_minutes BETWEEN 15 AND 240),
    validity_days  INT           NOT NULL CHECK (validity_days > 0),
    price          NUMERIC(14,2) NOT NULL CHECK (price >= 0),
    status         VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    created_by     BIGINT,
    updated_by     BIGINT
);
CREATE UNIQUE INDEX uq_pt_packages_code ON pt_packages (code) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_pt_packages_updated_at BEFORE UPDATE ON pt_packages FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Mã khuyến mãi.
CREATE TABLE promotions (
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(30)   NOT NULL,
    name          VARCHAR(150)  NOT NULL,
    type          VARCHAR(10)   NOT NULL CHECK (type IN ('PERCENT', 'FIXED')),
    value         NUMERIC(14,2) NOT NULL CHECK (value > 0),
    max_discount  NUMERIC(14,2) CHECK (max_discount > 0),   -- trần giảm cho loại PERCENT
    min_amount    NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (min_amount >= 0),
    applies_to    VARCHAR(20)   NOT NULL DEFAULT 'ALL' CHECK (applies_to IN ('ALL', 'MEMBERSHIP', 'PT', 'PRODUCT')),
    start_at      TIMESTAMPTZ   NOT NULL,
    end_at        TIMESTAMPTZ   NOT NULL,
    usage_limit   INT           CHECK (usage_limit > 0),     -- NULL = không giới hạn
    used_count    INT           NOT NULL DEFAULT 0 CHECK (used_count >= 0),
    status        VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    created_by    BIGINT,
    updated_by    BIGINT,
    CONSTRAINT chk_promotions_time CHECK (end_at > start_at),
    CONSTRAINT chk_promotions_percent CHECK (type <> 'PERCENT' OR value <= 100),
    CONSTRAINT chk_promotions_usage CHECK (usage_limit IS NULL OR used_count <= usage_limit)
);
CREATE UNIQUE INDEX uq_promotions_code ON promotions (upper(code)) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_promotions_updated_at BEFORE UPDATE ON promotions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Cấu hình nghiệp vụ dạng key-value (đọc qua service Settings, có giá trị mặc định trong code).
CREATE TABLE settings (
    key         VARCHAR(100) PRIMARY KEY,
    value       JSONB        NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by  BIGINT
);
INSERT INTO settings (key, value, description) VALUES
    ('checkin.duplicate_window_minutes', '60',  'Không cho check-in lại trong khoảng (phút)'),
    ('checkin.qr_ttl_seconds',           '60',  'Thời gian sống của mã QR động (giây)'),
    ('booking.advance_days',             '7',   'Được đặt lớp trước tối đa (ngày)'),
    ('booking.cancel_before_minutes',    '120', 'Phải hủy lớp trước giờ bắt đầu (phút)'),
    ('booking.no_show_limit',            '3',   'Số lần no-show trong 30 ngày thì bị khóa đặt lớp'),
    ('booking.block_days',               '7',   'Số ngày bị khóa đặt lớp'),
    ('subscription.remind_before_days',  '[7, 1]', 'Nhắc gia hạn trước khi hết hạn (ngày)'),
    ('pt.cancel_before_minutes',         '720', 'Phải hủy buổi PT trước (phút)');

INSERT INTO permissions (code, module, name) VALUES
    ('plan.view',        'plan',      'Xem gói tập, gói PT'),
    ('plan.manage',      'plan',      'Thêm/sửa/xóa gói tập, gói PT'),
    ('promotion.view',   'promotion', 'Xem khuyến mãi'),
    ('promotion.manage', 'promotion', 'Thêm/sửa/xóa khuyến mãi'),
    ('setting.manage',   'setting',   'Cấu hình quy tắc nghiệp vụ');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module IN ('plan', 'promotion', 'setting') WHERE r.code = 'OWNER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('plan.view', 'promotion.view', 'promotion.manage')
WHERE r.code = 'MANAGER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('plan.view', 'promotion.view')
WHERE r.code IN ('RECEPTIONIST', 'TRAINER');
