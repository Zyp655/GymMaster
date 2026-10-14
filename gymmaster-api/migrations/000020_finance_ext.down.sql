DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('expense.view', 'expense.manage', 'cash_session.manage'));
DELETE FROM permissions WHERE code IN ('expense.view', 'expense.manage', 'cash_session.manage');

DROP TABLE IF EXISTS installment_schedules;
DROP TABLE IF EXISTS installment_plans;
DROP TABLE IF EXISTS cash_sessions;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS expense_categories;
