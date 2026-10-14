DROP TABLE IF EXISTS online_payment_txns;

ALTER TABLE refunds DROP CONSTRAINT IF EXISTS refunds_method_check;
ALTER TABLE refunds ADD CONSTRAINT refunds_method_check
    CHECK (method IN ('CASH', 'TRANSFER', 'CARD'));

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_check;
ALTER TABLE payments ADD CONSTRAINT payments_method_check
    CHECK (method IN ('CASH', 'TRANSFER', 'CARD'));
