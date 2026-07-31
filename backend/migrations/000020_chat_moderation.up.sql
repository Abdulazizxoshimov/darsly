-- =============================================================================
-- CHAT MODERATSIYASI (№6) — mentor xabarni o'chira oladi
--
-- QAROR: yumshoq o'chirish (soft-delete), tarixdan BUTUNLAY yashiriladi.
--
-- Ikki variant bor edi: (a) "xabar o'chirildi" degan qabrtosh qoldirish,
-- (b) umuman ko'rsatmaslik. (b) tanlandi:
--   · Moderatsiya sababi odatda haqorat/spam. Qabrtosh o'sha joyni belgilab
--     qo'yadi va "u yerda nima yozilgan edi?" degan muhokamani boshlaydi —
--     ya'ni buzg'unchiga aynan u xohlagan e'tiborni beradi.
--   · Zoom ham o'chirilgan xabarni izsiz olib tashlaydi.
--
-- Qator esa DB'da QOLADI (`deleted_at`/`deleted_by`) — bu moderatsiya izi:
-- keyinchalik shikoyat kelsa nima o'chirilgani va kim o'chirgani ma'lum bo'ladi.
-- Hech bir API bu qatorni qaytarmaydi.
-- =============================================================================
ALTER TABLE chat_messages ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE chat_messages ADD COLUMN deleted_by VARCHAR(128);

-- Tarix so'rovi endi HAR DOIM `deleted_at IS NULL` bilan keladi. Qisman indeks:
-- o'chirilgan xabarlar kam bo'ladi, ular asosiy indeksda o'rin egallamasin.
DROP INDEX IF EXISTS idx_chat_messages_lesson;
CREATE INDEX idx_chat_messages_lesson
    ON chat_messages (lesson_id, created_at)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN chat_messages.deleted_at IS
    'Mentor moderatsiya bilan o''chirgan vaqt; NULL emas → hech bir tarix so''rovida qaytmaydi';
COMMENT ON COLUMN chat_messages.deleted_by IS
    'O''chirgan mentor ID si (moderatsiya izi)';
