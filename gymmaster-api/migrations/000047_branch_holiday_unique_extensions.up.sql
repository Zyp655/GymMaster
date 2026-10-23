-- QA vòng 4 (C4, HO-05): mỗi chi nhánh (hoặc toàn hệ thống – branch_id NULL) chỉ có MỘT lịch ngày lễ mỗi ngày,
-- và ghi lại các gói đã được cộng bù để xóa ngày lễ thì hoàn tác, không cộng lặp.

-- Gộp bản ghi trùng (chi nhánh, ngày) có sẵn: giữ bản tạo trước (bản đang có hiệu lực theo HolidayOn cũ).
DELETE FROM branch_holidays a
USING branch_holidays b
WHERE COALESCE(a.branch_id, 0) = COALESCE(b.branch_id, 0) AND a.date = b.date AND a.id > b.id;

CREATE UNIQUE INDEX IF NOT EXISTS uq_branch_holidays_branch_date ON branch_holidays ((COALESCE(branch_id, 0)), date);

-- Gói tập đã được cộng bù 1 ngày bởi ngày lễ nào. UNIQUE (subscription_id, holiday_date): một gói chỉ được cộng bù
-- một lần cho mỗi ngày lễ (kể cả khi có cả lịch toàn hệ thống lẫn lịch riêng chi nhánh cùng ngày).
CREATE TABLE IF NOT EXISTS branch_holiday_extensions (
    holiday_id      BIGINT      NOT NULL REFERENCES branch_holidays (id) ON DELETE CASCADE,
    subscription_id BIGINT      NOT NULL REFERENCES subscriptions (id) ON DELETE CASCADE,
    holiday_date    DATE        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (holiday_id, subscription_id),
    CONSTRAINT uq_branch_holiday_extensions_sub_date UNIQUE (subscription_id, holiday_date)
);
