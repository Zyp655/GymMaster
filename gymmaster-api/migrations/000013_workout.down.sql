DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE module = 'workout');
DELETE FROM permissions WHERE module = 'workout';

DROP TABLE IF EXISTS workout_logs;
DROP TABLE IF EXISTS workout_plan_items;
DROP TABLE IF EXISTS workout_plan_days;
DROP TABLE IF EXISTS workout_plans;
DROP TABLE IF EXISTS exercises;
