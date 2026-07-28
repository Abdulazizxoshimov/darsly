DROP INDEX IF EXISTS idx_recordings_transcode_queue;

ALTER TABLE recordings
    DROP COLUMN IF EXISTS transcode_started_at,
    DROP COLUMN IF EXISTS original_size_bytes,
    DROP COLUMN IF EXISTS transcode_status;
