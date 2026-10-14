-- =====================================================================
-- 000016 – CRM khách tiềm năng (Lead), vé tập thử & chăm sóc hội viên
-- =====================================================================

CREATE TABLE leads (
    id                BIGSERIAL PRIMARY KEY,
    branch_id         BIGINT       NOT NULL REFERENCES branches (id),
    full_name         VARCHAR(150) NOT NULL,
    phone             VARCHAR(20)  NOT NULL,
    email             VARCHAR(150) NOT NULL DEFAULT '',
    source            VARCHAR(20)  NOT NULL DEFAULT 'WEBSITE'
                      CHECK (source IN ('WEBSITE', 'WALK_IN', 'REFERRAL', 'FACEBOOK', 'OTHER')),
    interest_plan_id  BIGINT       REFERENCES membership_plans (id),
    status            VARCHAR(20)  NOT NULL DEFAULT 'NEW'
                      CHECK (status IN ('NEW', 'CONTACTED', 'TRIAL', 'CONVERTED', 'LOST')),
    assigned_to       BIGINT       REFERENCES users (id),
    trial_date        DATE,
    member_id         BIGINT       REFERENCES members (id),
    note              TEXT         NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    created_by        BIGINT,
    updated_by        BIGINT
);
CREATE INDEX idx_leads_branch_status ON leads (branch_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_leads_phone ON leads (phone) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_leads_updated_at BEFORE UPDATE ON leads FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE lead_activities (
    id          BIGSERIAL PRIMARY KEY,
    lead_id     BIGINT      NOT NULL REFERENCES leads (id) ON DELETE CASCADE,
    type        VARCHAR(20) NOT NULL CHECK (type IN ('CALL', 'SMS', 'VISIT', 'NOTE')),
    content     TEXT        NOT NULL,
    created_by  BIGINT      REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_lead_activities_lead ON lead_activities (lead_id, created_at DESC);

CREATE TABLE trial_passes (
    id          BIGSERIAL PRIMARY KEY,
    lead_id     BIGINT       NOT NULL REFERENCES leads (id) ON DELETE CASCADE,
    branch_id   BIGINT       NOT NULL REFERENCES branches (id),
    valid_date  DATE         NOT NULL,
    qr_token    VARCHAR(100) NOT NULL UNIQUE,
    used_at     TIMESTAMPTZ,
    created_by  BIGINT       REFERENCES users (id),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_trial_passes_lead ON trial_passes (lead_id);
CREATE INDEX idx_trial_passes_token ON trial_passes (qr_token);

INSERT INTO permissions (code, module, name) VALUES
    ('lead.view',   'lead', 'Xem khách tiềm năng (CRM)'),
    ('lead.manage', 'lead', 'Quản lý khách tiềm năng, cấp vé tập thử, chuyển đổi hội viên');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'lead'
WHERE r.code IN ('OWNER', 'MANAGER', 'RECEPTIONIST');
