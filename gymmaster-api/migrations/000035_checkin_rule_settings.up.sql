-- Cấu hình cho bộ quy tắc check-in (hợp đồng điện tử, trả góp quá hạn)
INSERT INTO settings (key, value, description) VALUES
    ('checkin.require_signed_contract', '0', 'Bắt buộc ký hợp đồng điện tử trước khi check-in (1 = bật, 0 = tắt)'),
    ('checkin.installment_grace_days', '7', 'Số ngày ân hạn kỳ trả góp quá hạn trước khi chặn check-in')
ON CONFLICT (key) DO NOTHING;
