-- =============================================================================
-- KUTISH XONASI — ustun DEFAULT'i O'CHIQ (PRODUCT.md «Xavfsizlik»)
--
-- Mahsulot qoidasi: «Kutish xonasi: default o'chiq». Amalda uch joyda uch xil
-- qiymat bor edi — DB DEFAULT TRUE, mobil klient false, web klient true.
-- Yagona haqiqat manbai server bo'lishi kerak, shuning uchun ustun default'i
-- ham qoidaga keltiriladi.
--
-- MAVJUD darslarga TEGILMAYDI: ustoz allaqachon yoqib qo'ygan kutish xonasi
-- migratsiya tufayli jimgina o'chib qolmasin (bu xavfsizlik sozlamasi).
-- Faqat BUNDAN KEYIN yaratiladigan qatorlar uchun default o'zgaradi.
--
-- Eslatma: repozitoriy (`postgres/lesson.go`) ustunni HAR DOIM oshkora yozadi,
-- ya'ni API orqali yaratilgan dars uchun qaror `CreateLessonReq` da chiqadi
-- (maydon `bool`, berilmasa false). Bu DEFAULT esa qo'lda SQL / kelajakdagi
-- migratsiya / seed skriptlari uchun zaxira chegara.
-- =============================================================================
ALTER TABLE lessons ALTER COLUMN is_waiting_room_enabled SET DEFAULT FALSE;

COMMENT ON COLUMN lessons.is_waiting_room_enabled IS
    'Kutish xonasi yoqilganmi. Mahsulot default''i — O''CHIQ (PRODUCT.md).';
