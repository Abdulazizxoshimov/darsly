-- =============================================================================
-- VAQT MINTAQASI: default 'UTC' → 'Asia/Tashkent'.
--
-- Mahsulot O'zbekiston uchun va mintaqa endi interfeysda SOZLANMAYDI (mobil va
-- web profilidan maydon olib tashlangan). Default 'UTC' qolsa yangi hisoblar
-- darslarni 5 soat surilgan holda ko'rsatardi.
--
-- Mavjud qatorlar ham tuzatiladi, LEKIN faqat 'UTC' bo'lganlari: kimdir
-- ataylab boshqa mintaqani tanlagan bo'lsa (masalan 'Europe/Moscow'), uni
-- ustiga yozish ma'lumot yo'qotish bo'lardi.
-- =============================================================================
ALTER TABLE users ALTER COLUMN timezone SET DEFAULT 'Asia/Tashkent';

UPDATE users SET timezone = 'Asia/Tashkent' WHERE timezone = 'UTC';

COMMENT ON COLUMN users.timezone IS 'IANA mintaqa; interfeysda sozlanmaydi (default Asia/Tashkent)';
