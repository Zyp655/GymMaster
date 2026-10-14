-- =====================================================================
-- 000012 – Thanh toán trực tuyến VNPay Sandbox
-- =====================================================================

-- Mở rộng phương thức thanh toán / hoàn tiền thêm VNPAY.
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_check;
ALTER TABLE payments ADD CONSTRAINT payments_method_check
    CHECK (method IN ('CASH', 'TRANSFER', 'CARD', 'VNPAY'));

ALTER TABLE refunds DROP CONSTRAINT IF EXISTS refunds_method_check;
ALTER TABLE refunds ADD CONSTRAINT refunds_method_check
    CHECK (method IN ('CASH', 'TRANSFER', 'CARD', 'VNPAY'));

-- Giao dịch thanh toán trực tuyến qua cổng VNPay.
CREATE TABLE online_payment_txns (
    id               BIGSERIAL PRIMARY KEY,
    invoice_id       BIGINT        NOT NULL REFERENCES invoices (id),
    member_id        BIGINT        NOT NULL REFERENCES members (id),
    txn_ref          VARCHAR(64)   NOT NULL UNIQUE,
    amount           NUMERIC(14,2) NOT NULL CHECK (amount > 0),
    status           VARCHAR(20)   NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'EXPIRED')),
    provider_txn_no  VARCHAR(100)  NOT NULL DEFAULT '',
    bank_code        VARCHAR(50)   NOT NULL DEFAULT '',
    raw_ipn          JSONB,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    paid_at          TIMESTAMPTZ
);
CREATE INDEX idx_online_payment_txns_invoice ON online_payment_txns (invoice_id);
CREATE INDEX idx_online_payment_txns_member ON online_payment_txns (member_id, created_at DESC);
CREATE INDEX idx_online_payment_txns_status ON online_payment_txns (status, created_at);
