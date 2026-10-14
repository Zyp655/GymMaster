DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE module = 'equipment');
DELETE FROM permissions WHERE module = 'equipment';

DROP TABLE IF EXISTS maintenance_logs;
DROP TABLE IF EXISTS equipments;
