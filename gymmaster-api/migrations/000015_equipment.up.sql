-- =====================================================================
-- 000015 – Quản lý thiết bị & bảo trì
-- =====================================================================

CREATE TABLE equipments (
    id                         BIGSERIAL PRIMARY KEY,
    branch_id                  BIGINT        NOT NULL REFERENCES branches (id),
    room_id                    BIGINT        REFERENCES rooms (id),
    code                       VARCHAR(30)   NOT NULL,
    name                       VARCHAR(150)  NOT NULL,
    category                   VARCHAR(50)   NOT NULL DEFAULT 'CARDIO'
                               CHECK (category IN ('CARDIO', 'STRENGTH', 'FREE_WEIGHT', 'FUNCTIONAL', 'ACCESSORY', 'OTHER')),
    purchase_date              DATE,
    price                      NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (price >= 0),
    warranty_until             DATE,
    status                     VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE'
                               CHECK (status IN ('ACTIVE', 'BROKEN', 'MAINTENANCE', 'RETIRED')),
    maintenance_interval_days  INT           NOT NULL DEFAULT 90 CHECK (maintenance_interval_days >= 0),
    next_maintenance_date      DATE,
    image_url                  VARCHAR(500)  NOT NULL DEFAULT '',
    note                       TEXT          NOT NULL DEFAULT '',
    created_at                 TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at                 TIMESTAMPTZ,
    created_by                 BIGINT,
    updated_by                 BIGINT
);
CREATE UNIQUE INDEX uq_equipments_code ON equipments (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_equipments_branch_status ON equipments (branch_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_equipments_next_maint ON equipments (next_maintenance_date) WHERE deleted_at IS NULL AND status <> 'RETIRED';
CREATE TRIGGER trg_equipments_updated_at BEFORE UPDATE ON equipments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE maintenance_logs (
    id            BIGSERIAL PRIMARY KEY,
    equipment_id  BIGINT        NOT NULL REFERENCES equipments (id) ON DELETE CASCADE,
    type          VARCHAR(20)   NOT NULL CHECK (type IN ('PERIODIC', 'REPAIR')),
    reported_by   BIGINT        REFERENCES users (id),
    description   TEXT          NOT NULL DEFAULT '',
    cost          NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (cost >= 0),
    status        VARCHAR(20)   NOT NULL DEFAULT 'OPEN'
                  CHECK (status IN ('OPEN', 'IN_PROGRESS', 'DONE')),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_maintenance_logs_equipment ON maintenance_logs (equipment_id, created_at DESC);
CREATE INDEX idx_maintenance_logs_status ON maintenance_logs (status, created_at DESC);
CREATE TRIGGER trg_maintenance_logs_updated_at BEFORE UPDATE ON maintenance_logs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO permissions (code, module, name) VALUES
    ('equipment.view',   'equipment', 'Xem thiết bị & lịch sử bảo trì'),
    ('equipment.manage', 'equipment', 'Quản lý thiết bị, báo hỏng, cập nhật bảo trì');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'equipment'
WHERE r.code IN ('OWNER', 'MANAGER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code = 'equipment.view'
WHERE r.code IN ('RECEPTIONIST', 'TRAINER');
