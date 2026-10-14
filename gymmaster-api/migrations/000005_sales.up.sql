-- =====================================================================
-- 000005 – Bán hàng: hóa đơn, thanh toán, hoàn tiền, đăng ký gói tập
-- =====================================================================

CREATE SEQUENCE invoice_no_seq;

-- Hóa đơn dùng chung cho mọi loại bán: gói tập, gói PT, sản phẩm (POS).
CREATE TABLE invoices (
    id            BIGSERIAL PRIMARY KEY,
    invoice_no    VARCHAR(30)   NOT NULL DEFAULT ('HD' || to_char(now() AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYMMDD') || lpad(nextval('invoice_no_seq')::text, 5, '0')),
    member_id     BIGINT        REFERENCES members (id),    -- NULL = khách vãng lai (POS)
    branch_id     BIGINT        NOT NULL REFERENCES branches (id),
    subtotal      NUMERIC(14,2) NOT NULL CHECK (subtotal >= 0),
    discount      NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (discount >= 0),
    total         NUMERIC(14,2) NOT NULL CHECK (total >= 0),
    paid_amount   NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    refunded_amount NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (refunded_amount >= 0),
    status        VARCHAR(20)   NOT NULL DEFAULT 'UNPAID' CHECK (status IN ('UNPAID', 'PARTIAL', 'PAID', 'VOID', 'REFUNDED')),
    source        VARCHAR(20)   NOT NULL DEFAULT 'COUNTER' CHECK (source IN ('COUNTER', 'POS', 'PORTAL')),
    promotion_id  BIGINT        REFERENCES promotions (id),
    note          VARCHAR(500)  NOT NULL DEFAULT '',
    issued_by     BIGINT        REFERENCES users (id),
    issued_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    voided_at     TIMESTAMPTZ,
    voided_by     BIGINT        REFERENCES users (id),
    void_reason   VARCHAR(255)  NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT chk_invoices_total CHECK (total = subtotal - discount),
    CONSTRAINT chk_invoices_paid CHECK (paid_amount <= total),
    CONSTRAINT chk_invoices_refund CHECK (refunded_amount <= paid_amount)
);
CREATE UNIQUE INDEX uq_invoices_no ON invoices (invoice_no);
CREATE INDEX idx_invoices_branch_time ON invoices (branch_id, issued_at);
CREATE INDEX idx_invoices_member ON invoices (member_id);
CREATE TRIGGER trg_invoices_updated_at BEFORE UPDATE ON invoices FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE invoice_items (
    id           BIGSERIAL PRIMARY KEY,
    invoice_id   BIGINT        NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    item_type    VARCHAR(20)   NOT NULL CHECK (item_type IN ('MEMBERSHIP', 'PT', 'PRODUCT', 'FEE', 'OTHER')),
    ref_id       BIGINT,       -- MEMBERSHIP: subscriptions.id · PT: pt_contracts.id · PRODUCT: products.id
    description  VARCHAR(255)  NOT NULL,
    quantity     INT           NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price   NUMERIC(14,2) NOT NULL CHECK (unit_price >= 0),
    discount     NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (discount >= 0),
    amount       NUMERIC(14,2) NOT NULL CHECK (amount >= 0)   -- = quantity * unit_price - discount
);
CREATE INDEX idx_invoice_items_invoice ON invoice_items (invoice_id);
CREATE INDEX idx_invoice_items_ref ON invoice_items (item_type, ref_id);

CREATE TABLE payments (
    id           BIGSERIAL PRIMARY KEY,
    invoice_id   BIGINT        NOT NULL REFERENCES invoices (id),
    amount       NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    method       VARCHAR(20)   NOT NULL CHECK (method IN ('CASH', 'TRANSFER', 'CARD')),
    reference    VARCHAR(100)  NOT NULL DEFAULT '',   -- mã giao dịch chuyển khoản/POS thẻ
    paid_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    received_by  BIGINT        REFERENCES users (id),
    note         VARCHAR(255)  NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_payments_invoice ON payments (invoice_id);
CREATE INDEX idx_payments_paid_at ON payments (paid_at);

CREATE TABLE refunds (
    id           BIGSERIAL PRIMARY KEY,
    invoice_id   BIGINT        NOT NULL REFERENCES invoices (id),
    amount       NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    method       VARCHAR(20)   NOT NULL CHECK (method IN ('CASH', 'TRANSFER', 'CARD')),
    reason       VARCHAR(255)  NOT NULL,
    refunded_by  BIGINT        REFERENCES users (id),
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_refunds_invoice ON refunds (invoice_id);

-- Đăng ký gói tập của hội viên.
--   PENDING   : đã lập hóa đơn, chưa thanh toán đủ
--   ACTIVE    : đã thanh toán đủ (start_date có thể ở tương lai khi nối tiếp gói cũ)
--   FROZEN    : đang bảo lưu
--   EXPIRED / CANCELLED / TRANSFERRED : kết thúc
CREATE TABLE subscriptions (
    id                  BIGSERIAL PRIMARY KEY,
    member_id           BIGINT        NOT NULL REFERENCES members (id),
    plan_id             BIGINT        NOT NULL REFERENCES membership_plans (id),
    branch_id           BIGINT        NOT NULL REFERENCES branches (id),   -- chi nhánh bán / chi nhánh được tập (SINGLE_BRANCH)
    invoice_id          BIGINT        REFERENCES invoices (id),
    start_date          DATE          NOT NULL,
    end_date            DATE          NOT NULL,
    total_sessions      INT,
    remaining_sessions  INT           CHECK (remaining_sessions >= 0),
    price               NUMERIC(14,2) NOT NULL CHECK (price >= 0),
    status              VARCHAR(20)   NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING', 'ACTIVE', 'FROZEN', 'EXPIRED', 'CANCELLED', 'TRANSFERRED')),
    freeze_days_used    INT           NOT NULL DEFAULT 0 CHECK (freeze_days_used >= 0),
    freeze_times_used   INT           NOT NULL DEFAULT 0 CHECK (freeze_times_used >= 0),
    previous_id         BIGINT        REFERENCES subscriptions (id),   -- gói trước (gia hạn/nâng cấp/chuyển nhượng)
    sold_by             BIGINT        REFERENCES users (id),
    note                VARCHAR(500)  NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT chk_subscriptions_dates CHECK (end_date >= start_date),
    CONSTRAINT chk_subscriptions_sessions CHECK (remaining_sessions IS NULL OR remaining_sessions <= total_sessions)
);
CREATE INDEX idx_subscriptions_member ON subscriptions (member_id, status, end_date);
CREATE INDEX idx_subscriptions_status_end ON subscriptions (status, end_date);
CREATE INDEX idx_subscriptions_invoice ON subscriptions (invoice_id);
CREATE TRIGGER trg_subscriptions_updated_at BEFORE UPDATE ON subscriptions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE subscription_freezes (
    id               BIGSERIAL PRIMARY KEY,
    subscription_id  BIGINT       NOT NULL REFERENCES subscriptions (id),
    start_date       DATE         NOT NULL,
    end_date         DATE         NOT NULL,
    days             INT          GENERATED ALWAYS AS (end_date - start_date + 1) STORED,
    reason           VARCHAR(255) NOT NULL DEFAULT '',
    status           VARCHAR(20)  NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'ENDED', 'CANCELLED')),
    requested_by     BIGINT       REFERENCES users (id),
    reviewed_by      BIGINT       REFERENCES users (id),
    reviewed_at      TIMESTAMPTZ,
    review_note      VARCHAR(255) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_freeze_dates CHECK (end_date >= start_date)
);
CREATE INDEX idx_subscription_freezes_sub ON subscription_freezes (subscription_id);
CREATE INDEX idx_subscription_freezes_status ON subscription_freezes (status, start_date);

-- Nhật ký vòng đời gói tập (timeline trên hồ sơ hội viên).
CREATE TABLE subscription_events (
    id               BIGSERIAL PRIMARY KEY,
    subscription_id  BIGINT       NOT NULL REFERENCES subscriptions (id),
    event_type       VARCHAR(30)  NOT NULL
                     CHECK (event_type IN ('CREATED', 'ACTIVATED', 'RENEWED', 'UPGRADED', 'FREEZE_REQUESTED', 'FROZEN',
                                           'UNFROZEN', 'FREEZE_REJECTED', 'TRANSFERRED', 'CANCELLED', 'EXPIRED', 'SESSION_USED')),
    data             JSONB        NOT NULL DEFAULT '{}',
    actor_id         BIGINT       REFERENCES users (id),   -- NULL = hệ thống (cron)
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscription_events_sub ON subscription_events (subscription_id, created_at);

INSERT INTO permissions (code, module, name) VALUES
    ('subscription.view',           'subscription', 'Xem đăng ký gói tập'),
    ('subscription.sell',           'subscription', 'Bán gói tập, gia hạn, nâng cấp'),
    ('subscription.freeze',         'subscription', 'Tạo yêu cầu bảo lưu'),
    ('subscription.approve_freeze', 'subscription', 'Duyệt/từ chối bảo lưu'),
    ('subscription.transfer',       'subscription', 'Chuyển nhượng gói tập'),
    ('subscription.cancel',         'subscription', 'Hủy gói tập'),
    ('invoice.view',                'invoice',      'Xem hóa đơn, thanh toán'),
    ('invoice.collect',             'invoice',      'Thu tiền'),
    ('invoice.void',                'invoice',      'Hủy hóa đơn'),
    ('invoice.refund',              'invoice',      'Hoàn tiền');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.module IN ('subscription', 'invoice') WHERE r.code IN ('OWNER', 'MANAGER');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p
  ON p.code IN ('subscription.view', 'subscription.sell', 'subscription.freeze', 'subscription.transfer', 'invoice.view', 'invoice.collect')
WHERE r.code = 'RECEPTIONIST';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('subscription.view')
WHERE r.code = 'TRAINER';
