-- Sprint 20: Chấm công & bảng lương (000019_payroll)

CREATE TABLE IF NOT EXISTS staff_attendances (
    id            BIGSERIAL    PRIMARY KEY,
    staff_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    branch_id     BIGINT       NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    shift_id      BIGINT       REFERENCES shifts(id) ON DELETE SET NULL,
    check_in_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    check_out_at  TIMESTAMPTZ,
    late_minutes  INT          NOT NULL DEFAULT 0,
    early_minutes INT          NOT NULL DEFAULT 0,
    method        VARCHAR(20)  NOT NULL DEFAULT 'MANUAL'
                  CHECK (method IN ('QR', 'MANUAL')),
    note          VARCHAR(255) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_staff_attendances_staff_date ON staff_attendances(staff_id, check_in_at DESC);
CREATE INDEX IF NOT EXISTS idx_staff_attendances_branch_date ON staff_attendances(branch_id, check_in_at DESC);

CREATE TABLE IF NOT EXISTS payrolls (
    id         BIGSERIAL   PRIMARY KEY,
    branch_id  BIGINT      NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    period     CHAR(7)     NOT NULL, -- YYYY-MM
    status     VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
               CHECK (status IN ('DRAFT', 'LOCKED', 'PAID')),
    locked_by  BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    locked_at  TIMESTAMPTZ,
    paid_by    BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    paid_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (branch_id, period)
);

CREATE TABLE IF NOT EXISTS payroll_items (
    id               BIGSERIAL     PRIMARY KEY,
    payroll_id       BIGINT        NOT NULL REFERENCES payrolls(id) ON DELETE CASCADE,
    staff_id         BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_salary      NUMERIC(14,2) NOT NULL DEFAULT 0,
    work_days        NUMERIC(5,2)  NOT NULL DEFAULT 0,
    ot_hours         NUMERIC(5,2)  NOT NULL DEFAULT 0,
    pt_commission    NUMERIC(14,2) NOT NULL DEFAULT 0,
    sales_commission NUMERIC(14,2) NOT NULL DEFAULT 0,
    bonus            NUMERIC(14,2) NOT NULL DEFAULT 0,
    penalty          NUMERIC(14,2) NOT NULL DEFAULT 0,
    total            NUMERIC(14,2) NOT NULL DEFAULT 0,
    note             VARCHAR(255)  NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (payroll_id, staff_id)
);

INSERT INTO permissions (code, module, name) VALUES
    ('payroll.view',   'payroll', 'Xem chấm công & bảng lương'),
    ('payroll.manage', 'payroll', 'Quản lý chấm công, tính lương, khóa & chi lương')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER')
  AND p.code IN ('payroll.view', 'payroll.manage')
ON CONFLICT DO NOTHING;
