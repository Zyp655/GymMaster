DELETE FROM permissions WHERE module IN ('staff', 'room', 'shift');
DROP TABLE IF EXISTS shifts;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS trainer_profiles;
DROP TABLE IF EXISTS staff_profiles;
DROP SEQUENCE IF EXISTS staff_code_seq;
