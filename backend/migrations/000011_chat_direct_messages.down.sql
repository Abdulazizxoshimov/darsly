DROP INDEX IF EXISTS idx_chat_messages_dm;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS to_identity;
