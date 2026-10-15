DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS contact_messages;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS testimonials;
DROP TABLE IF EXISTS trainer_public_profiles;
ALTER TABLE users DROP COLUMN IF EXISTS calendar_token;
