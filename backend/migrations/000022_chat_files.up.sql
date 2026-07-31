-- =============================================================================
-- CHATDA FAYL ULASHISH (№15) — PDF/rasm/ofis hujjati
--
-- Fayl MinIO'da, DB'da faqat KALIT (`file_key`) saqlanadi — havola EMAS.
-- Sabab: presigned havola vaqtinchalik (1 soat) va uni bazaga yozish
-- "eskirgan havola" muammosini tug'diradi; kalitdan esa har o'qishda yangi
-- havola imzolanadi (imzo — mahalliy HMAC, tarmoqqa chiqmaydi, arzon).
--
-- `body` (izoh) fayl bilan birga bo'lishi mumkin, shuning uchun alohida
-- "fayl xabari" turi kiritilmadi: fayl — oddiy chat xabarining ILOVASI.
-- Shu sabab shaxsiy (DM) fayl ham bepul ishlaydi — `to_identity` mantig'i
-- o'zgarishsiz qo'llanadi.
-- =============================================================================
ALTER TABLE chat_messages ADD COLUMN file_key  VARCHAR(255);
ALTER TABLE chat_messages ADD COLUMN file_name VARCHAR(255);
ALTER TABLE chat_messages ADD COLUMN file_size BIGINT;
ALTER TABLE chat_messages ADD COLUMN file_mime VARCHAR(128);

-- Fayl bor bo'lsa metama'lumot ham to'liq bo'lishi SHART: yarim to'ldirilgan
-- qator klientda nomsiz/o'lchamsiz ilova ko'rsatardi va sababi topilmasdi.
ALTER TABLE chat_messages
    ADD CONSTRAINT chk_chat_messages_file_complete
    CHECK (
        file_key IS NULL
        OR (file_name IS NOT NULL AND file_size IS NOT NULL AND file_mime IS NOT NULL)
    );

-- Faylsiz xabarning tanasi bo'sh bo'lolmaydi; fayl bilan kelsa — bo'lishi mumkin
-- (izohsiz ilova). Bu qoida ilgari faqat validatsiya teglarida edi, ya'ni
-- boshqa yozuv yo'li (kelajakdagi import/skript) uni chetlab o'tardi.
--
-- `NOT VALID` — ATAYLAB: cheklov YANGI va YANGILANGAN qatorlarga to'liq
-- qo'llanadi, lekin mavjud jadval skan qilinmaydi. Ikki sabab:
--   1) migratsiya jonli bazada uzoq lock ushlamaydi;
--   2) agar tarixda (validatsiyagacha yozilgan) bo'sh tanali qator qolgan
--      bo'lsa, migratsiya YIQILMAYDI va butun deploy to'xtab qolmaydi.
-- Tozalangach `VALIDATE CONSTRAINT` bilan to'liq kuchga kiritsa bo'ladi.
ALTER TABLE chat_messages
    ADD CONSTRAINT chk_chat_messages_body_or_file
    CHECK (file_key IS NOT NULL OR body <> '') NOT VALID;

COMMENT ON COLUMN chat_messages.file_key IS
    'MinIO obyekt kaliti (chat/<lesson_id>/<uuid><ext>); havola har o''qishda imzolanadi';
