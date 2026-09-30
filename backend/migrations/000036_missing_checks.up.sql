-- Yetishmayotgan CHECK'lar. NOT VALID + VALIDATE (000022 naqshi): avval yangi qatorlarga
-- qo'llanadi, so'ng mavjudlar tekshiriladi. Tarixiy yomon qator bo'lsa VALIDATE yiqiladi —
-- deploy oldidan tekshiring.
ALTER TABLE recordings ADD CONSTRAINT chk_recordings_transcode_status
    CHECK (transcode_status IN ('pending','running','done','failed','skipped')) NOT VALID;
ALTER TABLE recordings VALIDATE CONSTRAINT chk_recordings_transcode_status;

ALTER TABLE poll_votes ADD CONSTRAINT chk_poll_votes_option_index CHECK (option_index >= 0) NOT VALID;
ALTER TABLE poll_votes VALIDATE CONSTRAINT chk_poll_votes_option_index;

ALTER TABLE lessons ADD CONSTRAINT chk_lessons_duration_min CHECK (duration_min > 0) NOT VALID;
ALTER TABLE lessons VALIDATE CONSTRAINT chk_lessons_duration_min;

ALTER TABLE recordings ADD CONSTRAINT chk_recordings_telegram_attempts CHECK (telegram_attempts >= 0) NOT VALID;
ALTER TABLE recordings VALIDATE CONSTRAINT chk_recordings_telegram_attempts;
