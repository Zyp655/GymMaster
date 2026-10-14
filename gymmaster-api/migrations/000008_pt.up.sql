-- =====================================================================
-- 000008 – Huấn luyện cá nhân (PT): hợp đồng, buổi tập, hoa hồng
-- =====================================================================

CREATE TABLE pt_contracts (
    id              BIGSERIAL PRIMARY KEY,
    member_id       BIGINT        NOT NULL REFERENCES members (id),
    trainer_id      BIGINT        NOT NULL REFERENCES users (id),
    pt_package_id   BIGINT        NOT NULL REFERENCES pt_packages (id),
    branch_id       BIGINT        NOT NULL REFERENCES branches (id),
    invoice_id      BIGINT        REFERENCES invoices (id),
    total_sessions  INT           NOT NULL CHECK (total_sessions > 0),
    used_sessions   INT           NOT NULL DEFAULT 0 CHECK (used_sessions >= 0),
    session_minutes INT           NOT NULL DEFAULT 60,
    start_date      DATE          NOT NULL,
    end_date        DATE          NOT NULL,
    price           NUMERIC(14,2) NOT NULL CHECK (price >= 0),
    status          VARCHAR(20)   NOT NULL DEFAULT 'PENDING'
                    CHECK (status IN ('PENDING', 'ACTIVE', 'COMPLETED', 'EXPIRED', 'CANCELLED')),
    sold_by         BIGINT        REFERENCES users (id),
    note            VARCHAR(500)  NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT chk_pt_contracts_used CHECK (used_sessions <= total_sessions),
    CONSTRAINT chk_pt_contracts_dates CHECK (end_date >= start_date)
);
CREATE INDEX idx_pt_contracts_member ON pt_contracts (member_id, status);
CREATE INDEX idx_pt_contracts_trainer ON pt_contracts (trainer_id, status);
CREATE TRIGGER trg_pt_contracts_updated_at BEFORE UPDATE ON pt_contracts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Buổi PT. EXCLUDE: HLV không có 2 buổi PT chồng giờ (trùng với lịch lớp nhóm được kiểm tra ở service).
CREATE TABLE pt_sessions (
    id               BIGSERIAL PRIMARY KEY,
    contract_id      BIGINT       NOT NULL REFERENCES pt_contracts (id),
    trainer_id       BIGINT       NOT NULL REFERENCES users (id),
    member_id        BIGINT       NOT NULL REFERENCES members (id),
    branch_id        BIGINT       NOT NULL REFERENCES branches (id),
    start_at         TIMESTAMPTZ  NOT NULL,
    end_at           TIMESTAMPTZ  NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'COMPLETED', 'CANCELLED', 'NO_SHOW')),
    -- Giáo án / nhận xét buổi tập của HLV
    workout_note     TEXT         NOT NULL DEFAULT '',
    trainer_note     TEXT         NOT NULL DEFAULT '',
    completed_at     TIMESTAMPTZ,
    cancelled_at     TIMESTAMPTZ,
    cancel_reason    VARCHAR(255) NOT NULL DEFAULT '',
    source           VARCHAR(10)  NOT NULL DEFAULT 'COUNTER' CHECK (source IN ('COUNTER', 'PORTAL', 'TRAINER')),
    created_by       BIGINT       REFERENCES users (id),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_pt_sessions_time CHECK (end_at > start_at),
    CONSTRAINT ex_pt_trainer_overlap EXCLUDE USING gist (trainer_id WITH =, tstzrange(start_at, end_at) WITH &&)
        WHERE (status IN ('SCHEDULED', 'COMPLETED'))
);
CREATE INDEX idx_pt_sessions_trainer_time ON pt_sessions (trainer_id, start_at);
CREATE INDEX idx_pt_sessions_member_time ON pt_sessions (member_id, start_at);
CREATE INDEX idx_pt_sessions_contract ON pt_sessions (contract_id);
CREATE TRIGGER trg_pt_sessions_updated_at BEFORE UPDATE ON pt_sessions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Hoa hồng HLV. Sinh tự động khi hoàn thành buổi PT (= giá/buổi × commission_rate)
-- và khi dạy lớp nhóm. UNIQUE(source_type, source_id) chống tính trùng.
CREATE TABLE trainer_commissions (
    id           BIGSERIAL PRIMARY KEY,
    trainer_id   BIGINT        NOT NULL REFERENCES users (id),
    branch_id    BIGINT        NOT NULL REFERENCES branches (id),
    source_type  VARCHAR(20)   NOT NULL CHECK (source_type IN ('PT_SESSION', 'CLASS')),
    source_id    BIGINT        NOT NULL,
    base_amount  NUMERIC(14,2) NOT NULL CHECK (base_amount >= 0),
    rate         NUMERIC(5,2)  NOT NULL CHECK (rate BETWEEN 0 AND 100),
    amount       NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    period       CHAR(7)       NOT NULL,   -- 'YYYY-MM'
    status       VARCHAR(20)   NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PAID')),
    paid_at      TIMESTAMPTZ,
    paid_by      BIGINT        REFERENCES users (id),
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT uq_trainer_commissions_source UNIQUE (source_type, source_id)
);
CREATE INDEX idx_trainer_commissions_trainer_period ON trainer_commissions (trainer_id, period);

INSERT INTO permissions (code, module, name) VALUES
    ('pt.view',           'pt', 'Xem hợp đồng & lịch PT'),
    ('pt.sell',           'pt', 'Bán gói PT'),
    ('pt.schedule',       'pt', 'Đặt/đổi/hủy lịch PT'),
    ('pt.complete',       'pt', 'Xác nhận hoàn thành buổi PT, ghi giáo án'),
    ('pt.cancel_contract','pt', 'Hủy hợp đồng PT'),
    ('commission.view',   'pt', 'Xem hoa hồng HLV'),
    ('commission.pay',    'pt', 'Chi trả hoa hồng HLV');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module = 'pt' WHERE r.code IN ('OWNER', 'MANAGER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('pt.view', 'pt.sell', 'pt.schedule')
WHERE r.code = 'RECEPTIONIST';

-- HLV: chỉ thấy dữ liệu của chính mình (lọc ở service theo reqctx).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('pt.view', 'pt.schedule', 'pt.complete', 'commission.view')
WHERE r.code = 'TRAINER';
