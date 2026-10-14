DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE module = 'lead');
DELETE FROM permissions WHERE module = 'lead';

DROP TABLE IF EXISTS trial_passes;
DROP TABLE IF EXISTS lead_activities;
DROP TABLE IF EXISTS leads;
