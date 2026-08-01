-- Telegram arxivini qaytarib olish.
--
-- DIQQAT: `archived`/`restoring` qatorlar eski CHECK'ni buzmasligi uchun avval
-- ko'chiriladi. `archived` → `expired`: fayl serverda YO'Q, ya'ni `ready` ga
-- qaytarish yolg'on bo'lardi (yuklab olmoqchi bo'lgan mentor 404 olardi).
-- `restoring` → `archived`... eski sxemada u ham yo'q, shuning uchun `expired`.
UPDATE recordings SET status = 'expired' WHERE status IN ('archived', 'restoring');

ALTER TABLE recordings DROP CONSTRAINT IF EXISTS recordings_status_check;
ALTER TABLE recordings ADD CONSTRAINT recordings_status_check
    CHECK (status IN ('recording', 'processing', 'ready', 'failed', 'expired'));

DROP INDEX IF EXISTS idx_recordings_telegram_queue;
DROP INDEX IF EXISTS idx_recordings_cache_expiry;

ALTER TABLE recordings
    DROP COLUMN IF EXISTS telegram_file_id,
    DROP COLUMN IF EXISTS telegram_message_id,
    DROP COLUMN IF EXISTS telegram_chat_id,
    DROP COLUMN IF EXISTS telegram_sent_at,
    DROP COLUMN IF EXISTS telegram_error,
    DROP COLUMN IF EXISTS telegram_attempts,
    DROP COLUMN IF EXISTS telegram_next_attempt_at,
    DROP COLUMN IF EXISTS cached_until;

DROP TABLE IF EXISTS telegram_chats;

DROP INDEX IF EXISTS idx_users_telegram_user_id;
ALTER TABLE users
    DROP COLUMN IF EXISTS telegram_user_id,
    DROP COLUMN IF EXISTS telegram_username,
    DROP COLUMN IF EXISTS telegram_linked_at;
