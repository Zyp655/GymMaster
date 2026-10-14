DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('engagement.view', 'engagement.manage'));
DELETE FROM permissions WHERE code IN ('engagement.view', 'engagement.manage');

DROP TABLE IF EXISTS survey_responses;
DROP TABLE IF EXISTS surveys;
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS food_logs;
DROP TABLE IF EXISTS meal_plan_items;
DROP TABLE IF EXISTS meal_plans;
DROP TABLE IF EXISTS foods;
