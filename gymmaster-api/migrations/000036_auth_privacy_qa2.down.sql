DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'job.manage');
DELETE FROM permissions WHERE code = 'job.manage';
UPDATE permissions SET name = 'Xem & chạy thủ công job định kỳ' WHERE code = 'job.view';

-- Gỡ data_request.process đã cấp kèm privacy.manage ở bản up.
DELETE FROM role_permissions rp
USING permissions p
WHERE rp.permission_id = p.id AND p.code = 'data_request.process'
  AND rp.role_id IN (
      SELECT rp2.role_id FROM role_permissions rp2
      JOIN permissions p2 ON p2.id = rp2.permission_id AND p2.code = 'privacy.manage'
  );

DROP INDEX IF EXISTS uq_data_requests_pending;
ALTER TABLE data_requests DROP COLUMN IF EXISTS export_data;
ALTER TABLE user_mfa DROP COLUMN IF EXISTS last_totp_step;
DROP EXTENSION IF EXISTS unaccent;
