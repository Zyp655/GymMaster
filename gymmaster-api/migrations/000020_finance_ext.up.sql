-- Sprint 21: Chi phí, P&L, đối soát ca & trả góp (000020_finance_ext)

CREATE TABLE IF NOT EXISTS expense_categories (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(150) NOT NULL UNIQUE,
    type       VARCHAR(20)  NOT NULL DEFAULT 'VARIABLE'
               CHECK (type IN ('FIXED', 'VARIABLE')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO expense_categories (name, type) VALUES
    ('Tiền thuê mặt bằng', 'FIXED'),
    ('Điện, nước & Internet', 'VARIABLE'),
    ('Bảo trì & sửa chữa thiết bị', 'VARIABLE'),
    ('Marketing & Quảng cáo', 'VARIABLE'),
    ('Vật tư vệ sinh & Tiện ích', 'VARIABLE')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS expenses (
    id                 BIGSERIAL     PRIMARY KEY,
    branch_id          BIGINT        NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    category_id        BIGINT        NOT NULL REFERENCES expense_categories(id) ON DELETE RESTRICT,
    amount             NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    spent_at           DATE          NOT NULL DEFAULT CURRENT_DATE,
    description        VARCHAR(500)  NOT NULL DEFAULT '',
    attachment_url     VARCHAR(500)  NOT NULL DEFAULT '',
    maintenance_log_id BIGINT        REFERENCES maintenance_logs(id) ON DELETE SET NULL,
    created_by         BIGINT        REFERENCES users(id) ON DELETE SET NULL,
    approved_by        BIGINT        REFERENCES users(id) ON DELETE SET NULL,
    status             VARCHAR(20)   NOT NULL DEFAULT 'PENDING'
                       CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_expenses_branch_spent ON expenses(branch_id, spent_at DESC);

CREATE TABLE IF NOT EXISTS cash_sessions (
    id            BIGSERIAL     PRIMARY KEY,
    branch_id     BIGINT        NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    staff_id      BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    opened_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    closed_at     TIMESTAMPTZ,
    opening_cash  NUMERIC(14,2) NOT NULL DEFAULT 0,
    expected_cash NUMERIC(14,2) NOT NULL DEFAULT 0,
    counted_cash  NUMERIC(14,2) NOT NULL DEFAULT 0,
    difference    NUMERIC(14,2) NOT NULL DEFAULT 0,
    note          VARCHAR(500)  NOT NULL DEFAULT '',
    status        VARCHAR(20)   NOT NULL DEFAULT 'OPEN'
                  CHECK (status IN ('OPEN', 'CLOSED')),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cash_sessions_branch_status ON cash_sessions(branch_id, status);

CREATE TABLE IF NOT EXISTS installment_plans (
    id           BIGSERIAL     PRIMARY KEY,
    invoice_id   BIGINT        NOT NULL UNIQUE REFERENCES invoices(id) ON DELETE CASCADE,
    member_id    BIGINT        NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    total        NUMERIC(14,2) NOT NULL,
    installments INT           NOT NULL CHECK (installments >= 2),
    created_by   BIGINT        REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS installment_schedules (
    id          BIGSERIAL     PRIMARY KEY,
    plan_id     BIGINT        NOT NULL REFERENCES installment_plans(id) ON DELETE CASCADE,
    seq         INT           NOT NULL,
    due_date    DATE          NOT NULL,
    amount      NUMERIC(14,2) NOT NULL,
    paid_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
    status      VARCHAR(20)   NOT NULL DEFAULT 'PENDING'
                CHECK (status IN ('PENDING', 'PAID', 'OVERDUE')),
    paid_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (plan_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_installment_schedules_status_due ON installment_schedules(status, due_date);

INSERT INTO permissions (code, module, name) VALUES
    ('expense.view',        'expense',      'Xem chi phí vận hành & báo cáo lãi lỗ P&L'),
    ('expense.manage',      'expense',      'Quản lý & duyệt chi phí vận hành'),
    ('cash_session.manage', 'cash_session', 'Mở/đóng ca thu ngân & đối soát tiền mặt')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER')
  AND p.code IN ('expense.view', 'expense.manage', 'cash_session.manage')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'RECEPTIONIST'
  AND p.code = 'cash_session.manage'
ON CONFLICT DO NOTHING;
