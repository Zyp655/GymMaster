DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('payroll.view', 'payroll.manage'));
DELETE FROM permissions WHERE code IN ('payroll.view', 'payroll.manage');

DROP TABLE IF EXISTS payroll_items;
DROP TABLE IF EXISTS payrolls;
DROP TABLE IF EXISTS staff_attendances;
