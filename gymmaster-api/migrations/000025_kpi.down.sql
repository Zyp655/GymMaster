DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'kpi.manage');
DELETE FROM permissions WHERE code = 'kpi.manage';

DROP TABLE IF EXISTS sales_targets;
ALTER TABLE branches DROP COLUMN IF EXISTS capacity;
