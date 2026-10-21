-- QA vòng 2 – thu tiền/hóa đơn: trả góp đồng bộ hóa đơn, giao dịch VNPay cần hoàn tiền, xóa nợ khi hủy gói,
-- giá trị đã trả mang theo khi chuyển nhượng. Idempotent (chạy lại an toàn).

-- 1) Kỳ trả góp: thêm trạng thái SETTLED (đã tất toán do hóa đơn không còn nợ) và CANCELLED (hóa đơn hủy/hoàn/xóa nợ).
ALTER TABLE installment_schedules DROP CONSTRAINT IF EXISTS installment_schedules_status_check;
ALTER TABLE installment_schedules ADD CONSTRAINT installment_schedules_status_check
    CHECK (status IN ('PENDING', 'PAID', 'OVERDUE', 'SETTLED', 'CANCELLED'));

-- 2) Giao dịch VNPay: NEEDS_REFUND (đã nhận tiền nhưng không/không hết ghi vào hóa đơn – cần hoàn cho khách),
--    REFUNDED (đã hoàn toàn bộ). recorded_amount = phần đã ghi nhận vào hóa đơn (NULL = dữ liệu cũ, coi như toàn bộ);
--    refunded_amount = tổng đã hoàn của giao dịch; note = ghi chú cho nhân viên.
ALTER TABLE online_payment_txns DROP CONSTRAINT IF EXISTS online_payment_txns_status_check;
ALTER TABLE online_payment_txns ADD CONSTRAINT online_payment_txns_status_check
    CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'EXPIRED', 'NEEDS_REFUND', 'REFUNDED'));
ALTER TABLE online_payment_txns ADD COLUMN IF NOT EXISTS recorded_amount NUMERIC(14,2);
ALTER TABLE online_payment_txns ADD COLUMN IF NOT EXISTS refunded_amount NUMERIC(14,2) NOT NULL DEFAULT 0;
ALTER TABLE online_payment_txns ADD COLUMN IF NOT EXISTS note VARCHAR(255) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_online_payment_txns_invoice_status ON online_payment_txns (invoice_id, status);

-- Dữ liệu cũ: phần đã ghi nhận = phiếu thu VNPAY khớp mã giao dịch; hóa đơn đã hủy mà không có phiếu thu → cần hoàn.
UPDATE online_payment_txns t SET recorded_amount = LEAST(t.amount, x.rec)
FROM (
    SELECT t2.id, SUM(p.amount) AS rec
    FROM online_payment_txns t2
    JOIN payments p ON p.invoice_id = t2.invoice_id AND p.method = 'VNPAY' AND p.note LIKE '%' || t2.txn_ref || '%'
    GROUP BY t2.id
) x
WHERE x.id = t.id AND t.status = 'SUCCESS' AND t.recorded_amount IS NULL;

UPDATE online_payment_txns t SET recorded_amount = 0
FROM invoices i
WHERE i.id = t.invoice_id AND t.status = 'SUCCESS' AND t.recorded_amount IS NULL AND i.status = 'VOID';

UPDATE online_payment_txns t SET refunded_amount = x.refd
FROM (SELECT online_txn_id, SUM(amount) AS refd FROM refunds WHERE online_txn_id IS NOT NULL GROUP BY online_txn_id) x
WHERE x.online_txn_id = t.id AND t.refunded_amount <> x.refd;

UPDATE online_payment_txns SET status = 'REFUNDED'
WHERE status IN ('SUCCESS', 'NEEDS_REFUND') AND refunded_amount >= amount;

UPDATE online_payment_txns
SET status = 'NEEDS_REFUND',
    note = CASE WHEN note = '' THEN 'Đã nhận tiền VNPay nhưng hóa đơn đã hủy/đã thu đủ – cần hoàn cho khách' ELSE note END
WHERE status = 'SUCCESS' AND recorded_amount IS NOT NULL AND recorded_amount + refunded_amount < amount;

-- 3) Hóa đơn: tách phần "xóa nợ do hủy gói" khỏi giảm giá (total = subtotal − discount − write_off_amount).
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS write_off_amount NUMERIC(14,2) NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS write_off_reason VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_write_off_amount_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_write_off_amount_check CHECK (write_off_amount >= 0);
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS chk_invoices_total;
-- Dữ liệu cũ: phần xóa nợ đã bị cộng vào discount → tách ra theo sự kiện hủy gói (invoice_written_off).
UPDATE invoices i
SET discount = i.discount - x.w,
    write_off_amount = x.w,
    write_off_reason = COALESCE(NULLIF(x.reason, ''), 'Xóa nợ do hủy gói')
FROM (
    SELECT DISTINCT ON (s.invoice_id) s.invoice_id,
           (e.data ->> 'invoice_written_off')::numeric AS w,
           'Xóa nợ do hủy gói: ' || COALESCE(e.data ->> 'reason', '') AS reason
    FROM subscription_events e
    JOIN subscriptions s ON s.id = e.subscription_id
    WHERE e.event_type = 'CANCELLED' AND e.data ? 'invoice_written_off' AND s.invoice_id IS NOT NULL
    ORDER BY s.invoice_id, e.id DESC
) x
WHERE x.invoice_id = i.id AND i.write_off_amount = 0 AND x.w > 0 AND i.discount >= x.w;
ALTER TABLE invoices ADD CONSTRAINT chk_invoices_total CHECK (total = subtotal - discount - write_off_amount);

-- 4) Gói chuyển nhượng: giá trị đã trả (sau hoàn) mang theo từ gói gốc – gốc khấu trừ khi nâng cấp.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS carried_paid_base NUMERIC(14,2);
UPDATE subscriptions s SET carried_paid_base = s.price
FROM subscription_events e
WHERE e.subscription_id = s.id AND e.event_type = 'CREATED' AND e.data ? 'transfer_from'
  AND s.carried_paid_base IS NULL;

-- 5) Đồng bộ kỳ trả góp với trạng thái hóa đơn (dữ liệu lệch từ trước).
UPDATE installment_schedules s SET status = 'CANCELLED', updated_at = NOW()
FROM installment_plans p JOIN invoices i ON i.id = p.invoice_id
WHERE p.id = s.plan_id AND s.status IN ('PENDING', 'OVERDUE')
  AND (i.status IN ('VOID', 'REFUNDED') OR (i.status = 'PAID' AND i.write_off_amount > 0));

-- Bản sửa cho lần chạy trước của chính migration này: kỳ đã gắn SETTLED nhưng hóa đơn là xóa nợ khi hủy gói.
UPDATE installment_schedules s SET status = 'CANCELLED', updated_at = NOW()
FROM installment_plans p JOIN invoices i ON i.id = p.invoice_id
WHERE p.id = s.plan_id AND s.status = 'SETTLED' AND s.paid_amount = 0
  AND i.status = 'PAID' AND i.write_off_amount > 0;

UPDATE installment_schedules s SET status = 'SETTLED', updated_at = NOW()
FROM installment_plans p JOIN invoices i ON i.id = p.invoice_id
WHERE p.id = s.plan_id AND s.status IN ('PENDING', 'OVERDUE') AND i.status = 'PAID';

-- Hóa đơn còn nợ: chia lại số còn nợ cho các kỳ chưa thu khi tổng các kỳ khác số còn nợ (đã thu trực tiếp).
WITH due AS (
    SELECT p.id AS plan_id, (i.total - i.paid_amount) AS due
    FROM installment_plans p JOIN invoices i ON i.id = p.invoice_id
    WHERE i.status IN ('UNPAID', 'PARTIAL')
), open AS (
    SELECT s.id, s.plan_id, s.paid_amount,
           ROW_NUMBER() OVER (PARTITION BY s.plan_id ORDER BY s.seq) AS rn,
           COUNT(*) OVER (PARTITION BY s.plan_id) AS n,
           SUM(s.amount - s.paid_amount) OVER (PARTITION BY s.plan_id) AS open_sum
    FROM installment_schedules s
    WHERE s.status IN ('PENDING', 'OVERDUE')
), calc AS (
    SELECT o.id, o.paid_amount,
           CASE WHEN o.rn = o.n THEN d.due - FLOOR(d.due / o.n) * (o.n - 1) ELSE FLOOR(d.due / o.n) END AS share
    FROM open o JOIN due d ON d.plan_id = o.plan_id
    WHERE o.open_sum <> d.due AND d.due > 0
)
UPDATE installment_schedules s
SET amount = c.paid_amount + c.share,
    status = CASE WHEN c.share <= 0 THEN 'SETTLED' ELSE s.status END,
    updated_at = NOW()
FROM calc c WHERE c.id = s.id;
