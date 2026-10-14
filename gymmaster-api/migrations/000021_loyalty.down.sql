DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('loyalty.view', 'loyalty.manage'));
DELETE FROM permissions WHERE code IN ('loyalty.view', 'loyalty.manage');

DROP TABLE IF EXISTS challenge_participants;
DROP TABLE IF EXISTS challenges;
DROP TABLE IF EXISTS member_badges;
DROP TABLE IF EXISTS badges;
DROP TABLE IF EXISTS loyalty_rewards;
DROP TABLE IF EXISTS loyalty_ledgers;

ALTER TABLE members
    DROP COLUMN IF EXISTS referred_by,
    DROP COLUMN IF EXISTS referral_code,
    DROP COLUMN IF EXISTS loyalty_points;
