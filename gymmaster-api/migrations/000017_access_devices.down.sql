DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('device.view', 'device.manage'));
DELETE FROM permissions WHERE code IN ('device.view', 'device.manage');

ALTER TABLE checkins DROP CONSTRAINT IF EXISTS checkins_method_check;
ALTER TABLE checkins
    ADD CONSTRAINT checkins_method_check CHECK (method IN ('QR', 'MANUAL'));
ALTER TABLE checkins DROP COLUMN IF EXISTS device_id;

DROP TABLE IF EXISTS access_devices;
