DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE module = 'review');
DELETE FROM permissions WHERE module = 'review';

DROP TABLE IF EXISTS reviews;

ALTER TABLE trainer_profiles
    DROP COLUMN IF EXISTS rating_count,
    DROP COLUMN IF EXISTS rating_avg;
