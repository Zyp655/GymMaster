-- QA4 FIN-PAY-05 / PR-02: hoa hồng PT được đưa vào bảng lương nào (chống trả trùng).
-- Bảng lương chỉ cộng hoa hồng PENDING chưa thuộc bảng lương khác; khi chi lương chỉ đánh dấu PAID đúng các
-- hoa hồng đã cộng vào bảng lương đó.
ALTER TABLE trainer_commissions
    ADD COLUMN IF NOT EXISTS payroll_id BIGINT REFERENCES payrolls (id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_trainer_commissions_payroll ON trainer_commissions (payroll_id);
