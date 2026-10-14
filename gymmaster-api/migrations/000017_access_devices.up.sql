-- Sprint 18: Kiểm soát ra vào bằng thiết bị & Kiosk (000017_access_devices)

CREATE TABLE IF NOT EXISTS access_devices (
    id           BIGSERIAL    PRIMARY KEY,
    branch_id    BIGINT       NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    code         VARCHAR(50)  NOT NULL UNIQUE,
    name         VARCHAR(150) NOT NULL,
    type         VARCHAR(20)  NOT NULL DEFAULT 'KIOSK'
                 CHECK (type IN ('TURNSTILE', 'DOOR', 'KIOSK')),
    api_key_hash CHAR(64)     NOT NULL,
    last_seen_at TIMESTAMPTZ,
    status       VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                 CHECK (status IN ('ACTIVE', 'OFFLINE', 'DISABLED')),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_access_devices_branch_id ON access_devices(branch_id);
CREATE INDEX IF NOT EXISTS idx_access_devices_key_hash ON access_devices(api_key_hash);

ALTER TABLE checkins
    ADD COLUMN IF NOT EXISTS device_id BIGINT REFERENCES access_devices(id) ON DELETE SET NULL;

ALTER TABLE checkins DROP CONSTRAINT IF EXISTS checkins_method_check;
ALTER TABLE checkins
    ADD CONSTRAINT checkins_method_check CHECK (method IN ('QR', 'MANUAL', 'DEVICE'));

INSERT INTO permissions (code, module, name) VALUES
    ('device.view',   'device', 'Xem thiết bị kiểm soát ra vào & Kiosk'),
    ('device.manage', 'device', 'Quản lý thiết bị kiểm soát ra vào & Kiosk')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER')
  AND p.code IN ('device.view', 'device.manage')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'RECEPTIONIST'
  AND p.code = 'device.view'
ON CONFLICT DO NOTHING;
