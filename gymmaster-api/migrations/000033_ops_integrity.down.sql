DROP INDEX IF EXISTS idx_checkins_trial_pass;
DROP INDEX IF EXISTS idx_checkins_day_pass;
-- Lượt vào bằng vé (member_id NULL) không thể giữ khi khôi phục NOT NULL.
DELETE FROM checkins WHERE member_id IS NULL;
ALTER TABLE checkins DROP COLUMN IF EXISTS trial_pass_id;
ALTER TABLE checkins ALTER COLUMN member_id SET NOT NULL;

DROP INDEX IF EXISTS uq_locker_rentals_locker_active;
