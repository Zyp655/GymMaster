-- 000033 – Toàn vẹn nghiệp vụ vận hành: tủ đồ, lượt vào bằng vé ngày / vé tập thử.

-- 1) Tủ đồ: mỗi tủ chỉ có tối đa MỘT lượt thuê ACTIVE (chặn race khi gán tủ đồng thời).
--    Dọn dữ liệu cũ tất định: giữ lượt ACTIVE sớm nhất (start_at, id nhỏ nhất), các lượt còn lại → ENDED.
WITH ranked AS (
    SELECT id,
           row_number() OVER (PARTITION BY locker_id ORDER BY start_at ASC, id ASC) AS rn
    FROM locker_rentals
    WHERE status = 'ACTIVE'
)
UPDATE locker_rentals lr
SET status     = 'ENDED',
    end_at     = COALESCE(lr.end_at, now()),
    note       = LEFT(CASE WHEN lr.note = '' THEN '' ELSE lr.note || ' | ' END
                      || 'Tự kết thúc: trùng lượt thuê ACTIVE trên cùng tủ (migration 000033)', 255),
    updated_at = now()
FROM ranked r
WHERE r.id = lr.id AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_locker_rentals_locker_active
    ON locker_rentals (locker_id)
    WHERE status = 'ACTIVE';

-- 2) Hóa đơn thuê tủ: đảm bảo invoice_items.item_type chấp nhận LOCKER (đã có từ 000018, giữ idempotent).
ALTER TABLE invoice_items DROP CONSTRAINT IF EXISTS invoice_items_item_type_check;
ALTER TABLE invoice_items
    ADD CONSTRAINT invoice_items_item_type_check
    CHECK (item_type IN ('MEMBERSHIP', 'PT', 'PRODUCT', 'FEE', 'LOCKER', 'OTHER'));

-- 3) Check-in bằng vé ngày / vé khách / vé tập thử được ghi vào checkins (không gắn hội viên).
ALTER TABLE checkins ALTER COLUMN member_id DROP NOT NULL;
ALTER TABLE checkins
    ADD COLUMN IF NOT EXISTS trial_pass_id BIGINT REFERENCES trial_passes(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_checkins_day_pass ON checkins (day_pass_id) WHERE day_pass_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_checkins_trial_pass ON checkins (trial_pass_id) WHERE trial_pass_id IS NOT NULL;
