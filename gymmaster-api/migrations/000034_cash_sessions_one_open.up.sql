-- Mỗi chi nhánh chỉ có tối đa MỘT ca thu ngân đang mở (chặn race khi hai yêu cầu mở ca đồng thời).
-- Dọn dữ liệu cũ (nếu có) một cách tất định: giữ ca OPEN mới nhất (opened_at, id lớn nhất) của mỗi chi nhánh,
-- các ca OPEN còn lại được đóng với số liệu bằng tiền đầu ca và ghi chú lý do.
WITH ranked AS (
    SELECT id,
           row_number() OVER (PARTITION BY branch_id ORDER BY opened_at DESC, id DESC) AS rn
    FROM cash_sessions
    WHERE status = 'OPEN'
)
UPDATE cash_sessions cs
SET status        = 'CLOSED',
    closed_at     = COALESCE(cs.closed_at, now()),
    expected_cash = cs.opening_cash,
    counted_cash  = cs.opening_cash,
    difference    = 0,
    note          = LEFT(CASE WHEN cs.note = '' THEN '' ELSE cs.note || ' | ' END
                         || 'Tự đóng: trùng ca đang mở trong cùng chi nhánh (migration 000034)', 500),
    updated_at    = now()
FROM ranked r
WHERE r.id = cs.id AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_cash_sessions_branch_open
    ON cash_sessions (branch_id)
    WHERE status = 'OPEN';
