ALTER TABLE conversations DROP COLUMN IF EXISTS trainer_joined_at;
DROP INDEX IF EXISTS uq_day_passes_guest_phone_date;
