-- Sprint 26: Mật độ phòng realtime, KPI doanh số, cohort retention (000025_kpi)

ALTER TABLE branches
    ADD COLUMN IF NOT EXISTS capacity INT NOT NULL DEFAULT 120;

CREATE TABLE IF NOT EXISTS sales_targets (
    id           BIGSERIAL     PRIMARY KEY,
    branch_id    BIGINT        NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    staff_id     BIGINT        REFERENCES users(id) ON DELETE CASCADE,
    period       CHAR(7)       NOT NULL, -- YYYY-MM
    metric       VARCHAR(30)   NOT NULL
                 CHECK (metric IN ('REVENUE', 'NEW_MEMBERS', 'RENEWALS', 'PT_SALES')),
    target_value NUMERIC(14,2) NOT NULL CHECK (target_value >= 0),
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_targets_period ON sales_targets(period, branch_id);

INSERT INTO permissions (code, module, name) VALUES
    ('kpi.manage', 'kpi', 'Thiết lập chỉ tiêu doanh số KPI & xem báo cáo Cohort Retention')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER')
  AND p.code = 'kpi.manage'
ON CONFLICT DO NOTHING;
