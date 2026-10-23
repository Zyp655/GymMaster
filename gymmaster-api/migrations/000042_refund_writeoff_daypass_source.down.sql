-- Hoàn tác 000042.
UPDATE invoices i SET source = 'POS', updated_at = NOW()
FROM day_passes d
WHERE d.invoice_id = i.id AND i.source = 'COUNTER' AND i.member_id IS NULL;

UPDATE subscriptions s SET status = 'PENDING', updated_at = NOW()
FROM subscription_events e
WHERE e.subscription_id = s.id AND e.event_type = 'CANCELLED' AND e.data ->> 'migration' = '000042' AND s.status = 'CANCELLED';
DELETE FROM subscription_events WHERE event_type = 'CANCELLED' AND data ->> 'migration' = '000042';

UPDATE invoices
SET total            = total + write_off_amount,
    write_off_amount = 0,
    write_off_reason = '',
    updated_at       = NOW()
WHERE status = 'REFUNDED' AND write_off_reason = 'Hoàn toàn bộ số đã thu – xóa phần còn nợ (000042)';
