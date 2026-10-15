-- Sprint 30 & 15.2: Hoàn tiền online, chuyển chi nhánh chính & công cụ import dữ liệu cũ

ALTER TABLE refunds ADD COLUMN IF NOT EXISTS online_txn_id BIGINT REFERENCES online_payment_txns(id);
ALTER TABLE refunds ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'SUCCESS' CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED'));
ALTER TABLE refunds ADD COLUMN IF NOT EXISTS gateway_ref VARCHAR(100) NOT NULL DEFAULT '';

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS source VARCHAR(30) NOT NULL DEFAULT 'ADMIN';
ALTER TABLE pt_contracts ADD COLUMN IF NOT EXISTS source VARCHAR(30) NOT NULL DEFAULT 'ADMIN';

CREATE TABLE IF NOT EXISTS member_branch_transfers (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    from_branch_id BIGINT NOT NULL REFERENCES branches(id),
    to_branch_id BIGINT NOT NULL REFERENCES branches(id),
    subscription_id BIGINT REFERENCES subscriptions(id) ON DELETE SET NULL,
    transfer_fee NUMERIC(14,2) NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'APPROVED' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    approved_by BIGINT REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_branch_transfers_member ON member_branch_transfers(member_id, created_at DESC);

CREATE TABLE IF NOT EXISTS legacy_import_jobs (
    id BIGSERIAL PRIMARY KEY,
    branch_id BIGINT NOT NULL REFERENCES branches(id),
    file_name VARCHAR(255) NOT NULL,
    mode VARCHAR(20) NOT NULL DEFAULT 'DRY_RUN' CHECK (mode IN ('DRY_RUN', 'COMMIT')),
    status VARCHAR(20) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('PENDING', 'COMPLETED', 'FAILED')),
    total_rows INT NOT NULL DEFAULT 0,
    ok_rows INT NOT NULL DEFAULT 0,
    error_rows INT NOT NULL DEFAULT 0,
    summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    errors_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO permissions (code, module, name) VALUES
    ('import.legacy', 'setting', 'Chuyển đổi & nhập dữ liệu hệ thống cũ'),
    ('setup.wizard', 'setting', 'Thiết lập ban đầu toàn hệ thống (Wizard)')
ON CONFLICT (code) DO NOTHING;
