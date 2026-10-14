DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('privacy.manage', 'member.view_sensitive'));
DELETE FROM permissions WHERE code IN ('privacy.manage', 'member.view_sensitive');

DROP TABLE IF EXISTS data_requests;
DROP TABLE IF EXISTS consents;

ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS last_used_at,
    DROP COLUMN IF EXISTS device_name;

DROP TABLE IF EXISTS user_mfa;
