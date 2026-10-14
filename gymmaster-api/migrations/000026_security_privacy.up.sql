-- Sprint 27: 2FA, quản lý phiên, tuân thủ Nghị định 13/2023 (000026_security_privacy)

CREATE TABLE IF NOT EXISTS user_mfa (
    user_id             BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    totp_secret_enc     VARCHAR(255) NOT NULL,
    enabled_at          TIMESTAMPTZ,
    recovery_codes_hash JSONB       NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS device_name VARCHAR(150) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS consents (
    id         BIGSERIAL   PRIMARY KEY,
    member_id  BIGINT      NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    type       VARCHAR(30) NOT NULL
               CHECK (type IN ('TERMS', 'MARKETING', 'PHOTO')),
    granted    BOOLEAN     NOT NULL DEFAULT TRUE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip         VARCHAR(64) NOT NULL DEFAULT '',
    UNIQUE (member_id, type)
);

CREATE TABLE IF NOT EXISTS data_requests (
    id         BIGSERIAL    PRIMARY KEY,
    member_id  BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    type       VARCHAR(20)  NOT NULL CHECK (type IN ('EXPORT', 'DELETE')),
    status     VARCHAR(20)  NOT NULL DEFAULT 'PENDING'
               CHECK (status IN ('PENDING', 'PROCESSING', 'DONE', 'REJECTED')),
    reason     VARCHAR(500) NOT NULL DEFAULT '',
    file_url   VARCHAR(500) NOT NULL DEFAULT '',
    handled_by BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    note       VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    done_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_data_requests_status ON data_requests(status, created_at DESC);

INSERT INTO permissions (code, module, name) VALUES
    ('privacy.manage',        'privacy', 'Xử lý yêu cầu xuất/xóa dữ liệu cá nhân (Nghị định 13/2023)'),
    ('member.view_sensitive', 'member',  'Xem dữ liệu nhạy cảm của hội viên (SĐT/CCCD đầy đủ)')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER')
  AND p.code IN ('privacy.manage', 'member.view_sensitive')
ON CONFLICT DO NOTHING;
