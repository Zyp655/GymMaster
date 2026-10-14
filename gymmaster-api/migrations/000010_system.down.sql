DELETE FROM permissions WHERE module IN ('report', 'system');
DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS notifications;
