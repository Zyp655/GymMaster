-- QA vòng 4 – F1 (tiền lõi): sửa dữ liệu cũ theo quy tắc mới.

-- 1) B5: hóa đơn REFUNDED (đã hoàn hết số đã thu) nhưng còn total > paid_amount ("Còn nợ" ảo) → xóa phần nợ.
UPDATE invoices
SET write_off_amount = write_off_amount + (total - paid_amount),
    write_off_reason = CASE WHEN write_off_reason = '' THEN 'Hoàn toàn bộ số đã thu – xóa phần còn nợ (000042)' ELSE write_off_reason END,
    total            = paid_amount,
    updated_at       = NOW()
WHERE status = 'REFUNDED' AND total > paid_amount;

-- 2) B5: gói PENDING kẹt vì hóa đơn đã hoàn toàn bộ → CANCELLED (ghi sự kiện có đánh dấu để hoàn tác được).
INSERT INTO subscription_events (subscription_id, event_type, data)
SELECT s.id, 'CANCELLED', jsonb_build_object('reason', 'Hóa đơn đã hoàn toàn bộ số đã thu', 'invoice_id', i.id, 'migration', '000042')
FROM subscriptions s JOIN invoices i ON i.id = s.invoice_id
WHERE s.status = 'PENDING' AND i.status = 'REFUNDED';
UPDATE subscriptions s SET status = 'CANCELLED', updated_at = NOW()
FROM invoices i
WHERE i.id = s.invoice_id AND s.status = 'PENDING' AND i.status = 'REFUNDED';

-- 3) B6: hóa đơn bán vé tập theo ngày là hóa đơn tại quầy (COUNTER), không phải đơn bán hàng POS.
UPDATE invoices i SET source = 'COUNTER', updated_at = NOW()
FROM day_passes d
WHERE d.invoice_id = i.id AND i.source = 'POS';
