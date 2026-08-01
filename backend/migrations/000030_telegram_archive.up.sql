-- =============================================================================
-- TELEGRAM ARXIVI — «Dars arxivi va Telegram saqlash» (PRODUCT.md, 2026-08-01)
--
-- Mahsulot qarori: video Telegramda ABADIY qoladi, serverda esa 30 kun.
-- Shu sababli uch narsa kerak:
--
--   1) `users.telegram_user_id` — bot mentorni TANIY olishi uchun. Bot API
--      faqat Telegram user ID beradi; uni bizning `users.id` bilan bog'lamasak
--      «bu tugmani kim bosdi» degan savolga javob yo'q va har kim boshqaning
--      darsini o'z guruhiga yubora olardi.
--
--   2) `telegram_chats` — bot QAYSI guruhlarda borligini eslab qolish uchun.
--      Bot API'da "mening guruhlarim" degan metod YO'Q (Telegram ataylab
--      bermaydi). Yagona manba — bot guruhga qo'shilganda keladigan
--      `my_chat_member` yangilanishi. Uni ushlab shu jadvalga yozamiz; aks
--      holda mentorga tanlash uchun ro'yxat ko'rsata olmaymiz.
--
--   3) `recordings.telegram_*` — yuklash natijasi. `telegram_sent_at` —
--      BUTUN retention siyosatining sharti: server nusxasi FAQAT shu maydon
--      to'lgan bo'lsa (Telegram tasdiqlagan bo'lsa) o'chiriladi. Aks holda
--      yozuv serverda qolaveradi — «video hech qachon yo'qolmaydi» kafolati.
-- =============================================================================

-- ── 1. Mentorni Telegram foydalanuvchisi bilan bog'lash ──────────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS telegram_user_id   BIGINT,
    ADD COLUMN IF NOT EXISTS telegram_username  VARCHAR(64),
    ADD COLUMN IF NOT EXISTS telegram_linked_at TIMESTAMPTZ;

-- UNIQUE: bitta Telegram akkaunti bitta darsly hisobiga bog'lanadi. Bo'lmasa
-- ikki mentor bir Telegram akkauntini ulab, bir-birining tugmalarini bosardi.
-- Partial — NULL'lar (bog'lanmagan foydalanuvchilar) cheklovga tushmaydi.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_telegram_user_id
    ON users (telegram_user_id) WHERE telegram_user_id IS NOT NULL;

COMMENT ON COLUMN users.telegram_user_id IS 'Bog''langan Telegram akkaunt ID (bot /start <kod> orqali)';

-- ── 2. Bot a'zo bo'lgan guruhlar ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS telegram_chats (
    chat_id    BIGINT       PRIMARY KEY,             -- Telegram chat ID (guruhlar manfiy)
    title      TEXT         NOT NULL DEFAULT '',
    type       VARCHAR(32)  NOT NULL DEFAULT 'group', -- group | supergroup | channel
    -- mentor_id — botni guruhga QO'SHGAN mentor. Ro'yxat shu bo'yicha
    -- filtrlanadi: mentor faqat o'zi qo'shgan guruhlarni ko'radi va faqat
    -- ularga yubora oladi.
    mentor_id  UUID         REFERENCES users(id) ON DELETE CASCADE,
    -- is_active — bot guruhdan chiqarilsa FALSE. Qator o'chirilmaydi: qayta
    -- qo'shilganda tarix (kim qo'shgani) saqlanib qolsin.
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    added_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_chats_mentor
    ON telegram_chats (mentor_id) WHERE is_active;

COMMENT ON TABLE telegram_chats IS 'Bot a''zo bo''lgan guruhlar (my_chat_member yangilanishidan)';

-- ── 3. Yozuvning Telegramdagi nusxasi ────────────────────────────────────────
ALTER TABLE recordings
    ADD COLUMN IF NOT EXISTS telegram_file_id     TEXT,
    ADD COLUMN IF NOT EXISTS telegram_message_id  BIGINT,
    ADD COLUMN IF NOT EXISTS telegram_chat_id     BIGINT,
    ADD COLUMN IF NOT EXISTS telegram_sent_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS telegram_error       TEXT,
    -- attempts/next_attempt_at — 3 marta ortib boruvchi oraliqda qayta urinish.
    -- Navbat AYNAN shu ikki ustunda: alohida jadval yoki RabbitMQ ishlatilmadi,
    -- chunki holat baribir `recordings` da va ikki manba bir-biridan ajralib
    -- ketishi (navbatda bor, DB'da yo'q) eng qimmat nosozlik bo'lardi.
    ADD COLUMN IF NOT EXISTS telegram_attempts    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS telegram_next_attempt_at TIMESTAMPTZ,
    -- cached_until — Telegramdan QAYTARIB olingan nusxa qachongacha MinIO'da
    -- turadi (RECORDING_CACHE_TTL_HOURS). NULL = kesh emas, asl nusxa.
    ADD COLUMN IF NOT EXISTS cached_until         TIMESTAMPTZ;

-- Yangi statuslar:
--   archived  — serverda YO'Q, Telegramda BOR (qaytarib olish mumkin);
--   restoring — hozir Telegramdan yuklab olinmoqda (30-60 s).
-- `expired` esa endi FAQAT «butunlay yo'qolgan» degani (Telegramga ham
-- tushmagan va muddati o'tgan holat) — mahsulot qarori bo'yicha bunga yo'l
-- qo'yilmaydi, lekin eski qatorlar uchun status saqlanib qoladi.
ALTER TABLE recordings DROP CONSTRAINT IF EXISTS recordings_status_check;
ALTER TABLE recordings ADD CONSTRAINT recordings_status_check
    CHECK (status IN ('recording', 'processing', 'ready', 'failed', 'expired', 'archived', 'restoring'));

-- Yuklash navbati: `ready`, hali yuborilmagan, urinish vaqti kelgan yozuvlar.
-- Tor (partial) indeks — jadval o'sganda ham skan faqat navbatdagilar ustida.
CREATE INDEX IF NOT EXISTS idx_recordings_telegram_queue
    ON recordings (telegram_next_attempt_at)
    WHERE telegram_sent_at IS NULL AND status IN ('ready', 'archived');

-- Kesh muddati tugaganlarni topish (tiklangan nusxani qayta o'chirish).
CREATE INDEX IF NOT EXISTS idx_recordings_cache_expiry
    ON recordings (cached_until)
    WHERE cached_until IS NOT NULL;

COMMENT ON COLUMN recordings.telegram_sent_at IS 'Telegram TASDIQLAGAN vaqt — server nusxasini o''chirish sharti';
COMMENT ON COLUMN recordings.cached_until IS 'Telegramdan tiklangan nusxa shu vaqtgacha MinIO''da turadi';
