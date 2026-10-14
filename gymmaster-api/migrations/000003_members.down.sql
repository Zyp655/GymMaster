DELETE FROM permissions WHERE module = 'member';
DROP TABLE IF EXISTS body_metrics;
DROP TABLE IF EXISTS members;
DROP SEQUENCE IF EXISTS member_code_seq;
