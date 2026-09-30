CREATE INDEX IF NOT EXISTS idx_recordings_retention ON recordings (ended_at) WHERE status = 'ready';
DROP INDEX IF EXISTS idx_recordings_retention_coalesce;
CREATE INDEX IF NOT EXISTS idx_lessons_mentor_id ON lessons (mentor_id) WHERE deleted_at IS NULL;
DROP INDEX IF EXISTS idx_lessons_mentor_created;
CREATE INDEX IF NOT EXISTS idx_chat_messages_dm ON chat_messages (lesson_id, to_identity, created_at) WHERE to_identity IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wrr_created_at ON waiting_room_requests (created_at);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users (is_active) WHERE deleted_at IS NULL;
