DELETE FROM permissions WHERE module IN ('plan', 'promotion', 'setting');
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS promotions;
DROP TABLE IF EXISTS pt_packages;
DROP TABLE IF EXISTS membership_plans;
