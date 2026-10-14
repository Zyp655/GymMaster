-- Sprint 19: Tủ đồ & dịch vụ theo lượt (000018_lockers)

CREATE TABLE IF NOT EXISTS lockers (
    id          BIGSERIAL     PRIMARY KEY,
    branch_id   BIGINT        NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    code        VARCHAR(30)   NOT NULL,
    zone        VARCHAR(50)   NOT NULL DEFAULT 'Khu A',
    size        VARCHAR(20)   NOT NULL DEFAULT 'STANDARD'
                CHECK (size IN ('SMALL', 'STANDARD', 'LARGE')),
    monthly_fee NUMERIC(14,2) NOT NULL DEFAULT 150000,
    status      VARCHAR(20)   NOT NULL DEFAULT 'AVAILABLE'
                CHECK (status IN ('AVAILABLE', 'OCCUPIED', 'RENTED', 'BROKEN')),
    note        VARCHAR(255)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (branch_id, code)
);

CREATE INDEX IF NOT EXISTS idx_lockers_branch_status ON lockers(branch_id, status);

CREATE TABLE IF NOT EXISTS locker_rentals (
    id         BIGSERIAL     PRIMARY KEY,
    locker_id  BIGINT        NOT NULL REFERENCES lockers(id) ON DELETE CASCADE,
    member_id  BIGINT        NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    type       VARCHAR(20)   NOT NULL CHECK (type IN ('DAILY', 'MONTHLY')),
    start_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    end_at     TIMESTAMPTZ,
    invoice_id BIGINT        REFERENCES invoices(id) ON DELETE SET NULL,
    price      NUMERIC(14,2) NOT NULL DEFAULT 0,
    status     VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE'
               CHECK (status IN ('ACTIVE', 'ENDED', 'EXPIRED')),
    note       VARCHAR(255)  NOT NULL DEFAULT '',
    created_by BIGINT        REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_locker_rentals_locker_status ON locker_rentals(locker_id, status);
CREATE INDEX IF NOT EXISTS idx_locker_rentals_member_status ON locker_rentals(member_id, status);

ALTER TABLE invoice_items DROP CONSTRAINT IF EXISTS invoice_items_item_type_check;
ALTER TABLE invoice_items
    ADD CONSTRAINT invoice_items_item_type_check
    CHECK (item_type IN ('MEMBERSHIP', 'PT', 'PRODUCT', 'FEE', 'LOCKER', 'OTHER'));

INSERT INTO permissions (code, module, name) VALUES
    ('locker.view',   'locker', 'Xem sơ đồ tủ đồ & lượt thuê'),
    ('locker.manage', 'locker', 'Quản lý tủ đồ, gán tủ & cho thuê tủ')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER', 'RECEPTIONIST')
  AND p.code IN ('locker.view', 'locker.manage')
ON CONFLICT DO NOTHING;
