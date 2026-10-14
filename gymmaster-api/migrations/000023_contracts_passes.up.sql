-- Sprint 24: Hợp đồng điện tử, vé ngày & gói off-peak (000023_contracts_passes)

ALTER TABLE membership_plans
    ADD COLUMN IF NOT EXISTS access_hours JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS guest_passes_per_month INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS contract_templates (
    id         BIGSERIAL    PRIMARY KEY,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(150) NOT NULL,
    body_html  TEXT         NOT NULL,
    version    INT          NOT NULL DEFAULT 1,
    status     VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
               CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO contract_templates (code, name, body_html, version, status) VALUES
    ('MEMBER_TERMS_V1', 'Hợp đồng Hội viên & Cam kết An toàn Tập luyện',
     '<h3>HỢP ĐỒNG HỘI VIÊN GYMMASTER</h3><p>Họ và tên hội viên: <strong>{{member_name}}</strong> ({{member_code}})</p><p>Số điện thoại: <strong>{{member_phone}}</strong></p><p>Gói tập đăng ký: <strong>{{plan_name}}</strong></p><p>Tôi xác nhận đủ điều kiện sức khỏe tham gia tập luyện thể thao và đồng ý tuân thủ nội quy phòng tập GymMaster.</p>',
     1, 'ACTIVE')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS member_contracts (
    id               BIGSERIAL    PRIMARY KEY,
    member_id        BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    template_id      BIGINT       NOT NULL REFERENCES contract_templates(id) ON DELETE RESTRICT,
    template_version INT          NOT NULL DEFAULT 1,
    subscription_id  BIGINT       REFERENCES subscriptions(id) ON DELETE SET NULL,
    rendered_html    TEXT         NOT NULL DEFAULT '',
    status           VARCHAR(20)  NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN ('PENDING', 'SIGNED')),
    signed_at        TIMESTAMPTZ,
    signature_url     TEXT         NOT NULL DEFAULT '',
    pdf_url          VARCHAR(500) NOT NULL DEFAULT '',
    ip               VARCHAR(64)  NOT NULL DEFAULT '',
    user_agent       VARCHAR(255) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_member_contracts_member_status ON member_contracts(member_id, status);

CREATE TABLE IF NOT EXISTS day_passes (
    id                BIGSERIAL     PRIMARY KEY,
    branch_id         BIGINT        NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    buyer_name        VARCHAR(150)  NOT NULL,
    phone             VARCHAR(20)   NOT NULL,
    member_id         BIGINT        REFERENCES members(id) ON DELETE SET NULL,
    sponsor_member_id BIGINT        REFERENCES members(id) ON DELETE SET NULL,
    type              VARCHAR(20)   NOT NULL DEFAULT 'DAY'
                      CHECK (type IN ('DAY', 'GUEST')),
    valid_date        DATE          NOT NULL DEFAULT CURRENT_DATE,
    qr_token          VARCHAR(64)   NOT NULL UNIQUE,
    price             NUMERIC(14,2) NOT NULL DEFAULT 0,
    invoice_id        BIGINT        REFERENCES invoices(id) ON DELETE SET NULL,
    used_at           TIMESTAMPTZ,
    created_by        BIGINT        REFERENCES users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_day_passes_qr_token ON day_passes(qr_token);
CREATE INDEX IF NOT EXISTS idx_day_passes_sponsor ON day_passes(sponsor_member_id, created_at DESC);

ALTER TABLE checkins
    ADD COLUMN IF NOT EXISTS day_pass_id BIGINT REFERENCES day_passes(id) ON DELETE SET NULL;

INSERT INTO permissions (code, module, name) VALUES
    ('contract.manage', 'contract', 'Quản lý mẫu hợp đồng điện tử, hợp đồng hội viên & vé ngày')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('OWNER', 'MANAGER', 'RECEPTIONIST')
  AND p.code = 'contract.manage'
ON CONFLICT DO NOTHING;
