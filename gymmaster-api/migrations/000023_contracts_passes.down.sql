DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'contract.manage');
DELETE FROM permissions WHERE code = 'contract.manage';

ALTER TABLE checkins DROP COLUMN IF EXISTS day_pass_id;

DROP TABLE IF EXISTS day_passes;
DROP TABLE IF EXISTS member_contracts;
DROP TABLE IF EXISTS contract_templates;

ALTER TABLE membership_plans
    DROP COLUMN IF EXISTS guest_passes_per_month,
    DROP COLUMN IF EXISTS access_hours;
