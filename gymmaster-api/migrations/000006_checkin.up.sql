-- =====================================================================
-- 000006 – Check-in / ra vào
-- =====================================================================

-- Mọi lần quét đều được ghi, kể cả bị từ chối (result = DENIED + deny_reason).
CREATE TABLE checkins (
    id               BIGSERIAL PRIMARY KEY,
    member_id        BIGINT       NOT NULL REFERENCES members (id),
    branch_id        BIGINT       NOT NULL REFERENCES branches (id),
    subscription_id  BIGINT       REFERENCES subscriptions (id),   -- gói được dùng để vào (ALLOWED)
    checked_in_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    checked_out_at   TIMESTAMPTZ,
    method           VARCHAR(10)  NOT NULL CHECK (method IN ('QR', 'MANUAL')),
    result           VARCHAR(10)  NOT NULL CHECK (result IN ('ALLOWED', 'DENIED')),
    -- MEMBER_INACTIVE | NO_ACTIVE_SUBSCRIPTION | WRONG_BRANCH | NO_SESSIONS_LEFT | SUBSCRIPTION_FROZEN
    -- | DUPLICATE | QR_EXPIRED | QR_INVALID | BRANCH_CLOSED
    deny_reason      VARCHAR(40)  NOT NULL DEFAULT '',
    staff_id         BIGINT       REFERENCES users (id),
    note             VARCHAR(255) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_checkins_out CHECK (checked_out_at IS NULL OR checked_out_at >= checked_in_at)
);
CREATE INDEX idx_checkins_branch_time ON checkins (branch_id, checked_in_at);
CREATE INDEX idx_checkins_member_time ON checkins (member_id, checked_in_at DESC);
-- Hội viên đang ở trong phòng tập (chưa checkout) – dùng cho màn hình "đang tập".
CREATE INDEX idx_checkins_open ON checkins (branch_id) WHERE result = 'ALLOWED' AND checked_out_at IS NULL;

INSERT INTO permissions (code, module, name) VALUES
    ('checkin.view', 'checkin', 'Xem lịch sử ra vào'),
    ('checkin.scan', 'checkin', 'Quét QR / check-in thủ công / checkout');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'checkin' WHERE r.code IN ('OWNER', 'MANAGER', 'RECEPTIONIST');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code = 'checkin.view' WHERE r.code = 'TRAINER';
