ALTER TABLE recordings DROP CONSTRAINT IF EXISTS chk_recordings_telegram_attempts;
ALTER TABLE lessons DROP CONSTRAINT IF EXISTS chk_lessons_duration_min;
ALTER TABLE poll_votes DROP CONSTRAINT IF EXISTS chk_poll_votes_option_index;
ALTER TABLE recordings DROP CONSTRAINT IF EXISTS chk_recordings_transcode_status;
