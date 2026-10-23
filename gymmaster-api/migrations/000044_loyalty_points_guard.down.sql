DROP INDEX IF EXISTS uq_loyalty_ledgers_earn_once;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_loyalty_points_nonneg;
