DROP INDEX IF EXISTS idx_chat_messages_lesson;
CREATE INDEX idx_chat_messages_lesson ON chat_messages (lesson_id, created_at);

ALTER TABLE chat_messages DROP COLUMN IF EXISTS deleted_by;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS deleted_at;
