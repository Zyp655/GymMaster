-- =====================================================================
-- 000010 – Thông báo, nhật ký thao tác (audit), báo cáo
-- =====================================================================

CREATE TABLE notifications (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- SUBSCRIPTION_EXPIRING | SUBSCRIPTION_EXPIRED | FREEZE_REQUEST | FREEZE_RESULT | BOOKING_PROMOTED
    -- | CLASS_CANCELLED | PT_SCHEDULED | PT_CANCELLED | LOW_STOCK | SYSTEM
    type        VARCHAR(40)  NOT NULL,
    title       VARCHAR(200) NOT NULL,
    body        TEXT         NOT NULL DEFAULT '',
    link        VARCHAR(255) NOT NULL DEFAULT '',
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_user ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE read_at IS NULL;

-- Nhật ký thao tác quan trọng (ai – làm gì – trên đối tượng nào – trước/sau).
CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    BIGINT       REFERENCES users (id),   -- NULL = hệ thống
    action      VARCHAR(50)  NOT NULL,                -- CREATE | UPDATE | DELETE | LOGIN | SELL | VOID | REFUND | ...
    entity      VARCHAR(50)  NOT NULL,                -- branch | member | subscription | invoice | ...
    entity_id   BIGINT,
    before      JSONB,
    after       JSONB,
    ip          VARCHAR(64)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity, entity_id);
CREATE INDEX idx_audit_logs_actor_time ON audit_logs (actor_id, created_at DESC);
CREATE INDEX idx_audit_logs_time ON audit_logs (created_at DESC);

-- Nhật ký các lần chạy job định kỳ (cron) để theo dõi/hiển thị.
CREATE TABLE job_runs (
    id           BIGSERIAL PRIMARY KEY,
    job          VARCHAR(50)  NOT NULL,
    started_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    finished_at  TIMESTAMPTZ,
    status       VARCHAR(10)  NOT NULL DEFAULT 'RUNNING' CHECK (status IN ('RUNNING', 'SUCCESS', 'FAILED')),
    affected     INT          NOT NULL DEFAULT 0,
    message      TEXT         NOT NULL DEFAULT ''
);
CREATE INDEX idx_job_runs_job ON job_runs (job, started_at DESC);

INSERT INTO permissions (code, module, name) VALUES
    ('report.view',   'report', 'Xem báo cáo, dashboard'),
    ('report.export', 'report', 'Xuất báo cáo Excel'),
    ('audit.view',    'system', 'Xem nhật ký thao tác'),
    ('job.view',      'system', 'Xem & chạy thủ công job định kỳ');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module IN ('report', 'system') WHERE r.code = 'OWNER';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('report.view', 'report.export') WHERE r.code = 'MANAGER';
