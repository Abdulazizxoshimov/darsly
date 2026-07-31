-- =============================================================================
-- YOZUVLAR RETENTION — 30 kunlik saqlash muddati (PRODUCT.md №5).
--
-- Mahsulot qarori: yozuv 30 kun saqlanadi, keyin avtomatik o'chiriladi. Sabab
-- oddiy — 4 yadroli VPS diski cheksiz emas, 2.5 soatlik dars ~300 MB, va bir
-- mentor haftada bir necha dars o'tadi. Cheksiz saqlash bir necha oyda diskni
-- to'ldirib, YANGI darslarni yozib bo'lmaydigan holatga olib kelardi.
--
-- Ogohlantirish: o'chishiga 3 kun qolganda mentorga bildirishnoma yuboriladi
-- (u kerak yozuvni yuklab olib qo'yadi). `retention_warned_at` — bu
-- ogohlantirish BIR MARTA yuborilishini kafolatlaydigan belgi
-- (`reminder_sent_at` bilan aynan bir xil naqsh: atomik claim).
--
-- `expired` holati: qator O'CHIRILMAYDI, faqat belgilanadi. Sabab — mentor
-- yozuvlar ro'yxatida "bor edi, muddati o'tdi" ni ko'rishi kerak; qator
-- yo'qolsa yozuv "hech qachon bo'lmagan"dek ko'rinardi va bu qo'llab-quvvatlash
-- savollarini ko'paytirardi. `deleted_at` — MinIO'dagi fayl qachon o'chirilgani.
-- =============================================================================
ALTER TABLE recordings
    ADD COLUMN IF NOT EXISTS retention_warned_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_at          TIMESTAMPTZ;

-- Status ro'yxatiga 'expired' qo'shiladi (eski CHECK uni rad etardi).
ALTER TABLE recordings DROP CONSTRAINT IF EXISTS recordings_status_check;
ALTER TABLE recordings ADD CONSTRAINT recordings_status_check
    CHECK (status IN ('recording', 'processing', 'ready', 'failed', 'expired'));

-- Retention ishchisining ikkala so'rovi ham `ready` yozuvlarni tugash vaqti
-- bo'yicha qidiradi. Tor (partial) indeks: jadval o'sgani sayin ham skan
-- faqat hali tirik yozuvlar ustida qoladi.
CREATE INDEX IF NOT EXISTS idx_recordings_retention
    ON recordings (ended_at)
    WHERE status = 'ready';

COMMENT ON COLUMN recordings.retention_warned_at IS 'O''chirish ogohlantirishi yuborilgan vaqt (bir marta)';
COMMENT ON COLUMN recordings.deleted_at IS 'MinIO obyekti o''chirilgan vaqt (retention)';
