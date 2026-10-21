ALTER TABLE access_devices
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;
