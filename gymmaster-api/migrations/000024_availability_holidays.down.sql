DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('leave.manage', 'holiday.manage'));
DELETE FROM permissions WHERE code IN ('leave.manage', 'holiday.manage');

DROP TABLE IF EXISTS branch_holidays;

ALTER TABLE pt_sessions DROP COLUMN IF EXISTS original_trainer_id;
ALTER TABLE class_schedules DROP COLUMN IF EXISTS substitute_trainer_id;

DROP TABLE IF EXISTS staff_leave_requests;
DROP TABLE IF EXISTS trainer_availabilities;
