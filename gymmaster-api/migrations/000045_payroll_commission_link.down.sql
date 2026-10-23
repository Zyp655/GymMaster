DROP INDEX IF EXISTS idx_trainer_commissions_payroll;
ALTER TABLE trainer_commissions DROP COLUMN IF EXISTS payroll_id;
