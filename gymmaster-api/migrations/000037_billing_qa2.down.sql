-- Hoàn tác 000037: gộp lại phần xóa nợ vào discount, bỏ cột/trạng thái mới (dữ liệu trạng thái mới quy về gần nhất).
ALTER TABLE subscriptions DROP COLUMN IF EXISTS carried_paid_base;

ALTER TABLE invoices DROP CONSTRAINT IF EXISTS chk_invoices_total;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'invoices' AND column_name = 'write_off_amount') THEN
        UPDATE invoices SET discount = discount + write_off_amount WHERE write_off_amount > 0;
    END IF;
END $$;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_write_off_amount_check;
ALTER TABLE invoices DROP COLUMN IF EXISTS write_off_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS write_off_reason;
ALTER TABLE invoices ADD CONSTRAINT chk_invoices_total CHECK (total = subtotal - discount);

DROP INDEX IF EXISTS idx_online_payment_txns_invoice_status;
UPDATE online_payment_txns SET status = 'SUCCESS' WHERE status IN ('NEEDS_REFUND', 'REFUNDED');
ALTER TABLE online_payment_txns DROP CONSTRAINT IF EXISTS online_payment_txns_status_check;
ALTER TABLE online_payment_txns ADD CONSTRAINT online_payment_txns_status_check
    CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'EXPIRED'));
ALTER TABLE online_payment_txns DROP COLUMN IF EXISTS recorded_amount;
ALTER TABLE online_payment_txns DROP COLUMN IF EXISTS refunded_amount;
ALTER TABLE online_payment_txns DROP COLUMN IF EXISTS note;

UPDATE installment_schedules SET status = 'PENDING' WHERE status IN ('SETTLED', 'CANCELLED');
ALTER TABLE installment_schedules DROP CONSTRAINT IF EXISTS installment_schedules_status_check;
ALTER TABLE installment_schedules ADD CONSTRAINT installment_schedules_status_check
    CHECK (status IN ('PENDING', 'PAID', 'OVERDUE'));
