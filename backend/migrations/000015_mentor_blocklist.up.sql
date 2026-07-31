-- =============================================================================
-- MENTOR BLOCKLIST — doimiy qora ro'yxat (№4: ban tanlovi)
--
-- Redis'dagi dars-darajali ban (room:ban:*) darsning o'zi bilan tugaydi.
-- Mentor "doimiy (mening hamma darslarimdan)" tanlaganda yozuv shu jadvalga
-- tushadi va mentorning KEYINGI darslariga ham amal qiladi.
--
-- Identifikatsiya cheklovi (ongli qaror): o'quvchida akkaunt yo'q, LiveKit
-- identity har join'da yangi ("guest_<uuid>"). Shuning uchun doimiy moslik
-- kaliti — ko'rsatilgan ism (display_name, katta-kichik harf farqsiz).
-- Bu Zoom'ning ro'yxatsiz mehmonlari bilan bir xil darajadagi himoya: ismini
-- o'zgartirgan buzg'unchini faqat qayta chiqarish mumkin. identity esa joriy
-- token amal muddatida (30 min) qaytib kirishni kesish uchun saqlanadi.
-- =============================================================================
CREATE TABLE mentor_blocklist (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    mentor_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    identity     VARCHAR(64)  NOT NULL DEFAULT '',   -- oxirgi ko'rilgan LiveKit identity (audit + qisqa oyna)
    display_name VARCHAR(100) NOT NULL,              -- moslik kaliti (lower bo'yicha)
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Bitta mentor uchun bitta ism bir marta (katta-kichik harf farqsiz).
-- Ismi bo'sh yozuvlar (faqat identity ma'lum) unikallikdan chiqarilgan.
CREATE UNIQUE INDEX idx_mentor_blocklist_name
    ON mentor_blocklist (mentor_id, lower(display_name))
    WHERE display_name <> '';

CREATE INDEX idx_mentor_blocklist_mentor ON mentor_blocklist (mentor_id);

COMMENT ON TABLE mentor_blocklist IS 'Mentor darajasidagi doimiy qora ro''yxat (kick scope=mentor)';
