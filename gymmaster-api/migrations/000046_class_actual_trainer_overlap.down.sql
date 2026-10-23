DROP INDEX IF EXISTS idx_class_schedules_actual_trainer_time;
ALTER TABLE class_schedules DROP CONSTRAINT IF EXISTS ex_class_trainer_overlap;
ALTER TABLE class_schedules
    ADD CONSTRAINT ex_class_trainer_overlap EXCLUDE USING gist (trainer_id WITH =, tstzrange(start_at, end_at) WITH &&)
        WHERE (status <> 'CANCELLED');
