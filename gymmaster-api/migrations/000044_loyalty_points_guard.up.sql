-- QA4 FIN-LOY-03 / LY-03: chặn điểm thưởng âm và cộng điểm trùng.
-- 1) Điểm âm sinh ra do lỗi đổi quà đồng thời: ghi bút toán điều chỉnh (giữ lịch sử) rồi đưa về 0.
INSERT INTO loyalty_ledgers (member_id, points, reason, ref_type, note)
SELECT id, -loyalty_points, 'ADJUST', 'SYSTEM', 'Điều chỉnh điểm âm về 0 (sửa lỗi đổi quà đồng thời)'
FROM members
WHERE loyalty_points < 0;

UPDATE members SET loyalty_points = 0 WHERE loyalty_points < 0;

-- 2) Chốt chặn cuối ở DB: số dư điểm không bao giờ âm.
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_loyalty_points_nonneg;
ALTER TABLE members ADD CONSTRAINT chk_members_loyalty_points_nonneg CHECK (loyalty_points >= 0);

-- 3) Mỗi nguồn cộng điểm (thử thách, check-in, gia hạn, đánh giá, giới thiệu) chỉ ghi nhận một lần cho một hội viên.
--    REDEEM (đổi cùng một quà nhiều lần) và ADJUST (điều chỉnh tay) được lặp. Nếu dữ liệu cũ đã có bản ghi trùng
--    (do lỗi trước đây) thì bỏ qua tạo index – service vẫn chống trùng bằng khóa dòng hội viên.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM loyalty_ledgers
        WHERE ref_id IS NOT NULL AND reason NOT IN ('REDEEM', 'ADJUST')
        GROUP BY member_id, reason, ref_type, ref_id
        HAVING count(*) > 1
    ) THEN
        CREATE UNIQUE INDEX IF NOT EXISTS uq_loyalty_ledgers_earn_once
            ON loyalty_ledgers (member_id, reason, ref_type, ref_id)
            WHERE ref_id IS NOT NULL AND reason NOT IN ('REDEEM', 'ADJUST');
    ELSE
        RAISE NOTICE 'loyalty_ledgers có bút toán cộng điểm trùng – bỏ qua uq_loyalty_ledgers_earn_once';
    END IF;
END $$;
