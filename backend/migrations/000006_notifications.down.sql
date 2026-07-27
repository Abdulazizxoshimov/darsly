ALTER TABLE lessons DROP COLUMN IF EXISTS reminder_sent_at;
DROP TABLE IF EXISTS notifications CASCADE;
