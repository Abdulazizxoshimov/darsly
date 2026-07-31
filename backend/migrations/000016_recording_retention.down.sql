-- Retention ustunlari va 'expired' holatini qaytarib olish.
-- DIQQAT: 'expired' qatorlar CHECK'ni buzmasligi uchun avval 'failed' ga
-- o'tkaziladi (fayl allaqachon o'chirilgan, ya'ni yuklab bo'lmaydi — 'ready'
-- ga qaytarish yolg'on bo'lardi).
UPDATE recordings SET status = 'failed' WHERE status = 'expired';

ALTER TABLE recordings DROP CONSTRAINT IF EXISTS recordings_status_check;
ALTER TABLE recordings ADD CONSTRAINT recordings_status_check
    CHECK (status IN ('recording', 'processing', 'ready', 'failed'));

DROP INDEX IF EXISTS idx_recordings_retention;

ALTER TABLE recordings
    DROP COLUMN IF EXISTS retention_warned_at,
    DROP COLUMN IF EXISTS deleted_at;
