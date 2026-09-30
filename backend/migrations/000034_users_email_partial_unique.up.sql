-- users.email UNIQUE endi FAQAT faol (o'chirilmagan) qatorlar uchun: soft-delete'dan
-- keyin email qayta ro'yxatdan o'tkazilishi mumkin.
-- Partial unique jadval-darajali UNIQUE'ning qismi bo'lgani uchun CREATE mavjud
-- ma'lumotda yiqilmaydi. (CONCURRENTLY ishlatilmadi: golang-migrate bitta tranzaksiyada yuritadi.)
CREATE UNIQUE INDEX IF NOT EXISTS users_email_active ON users (email) WHERE deleted_at IS NULL;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
-- idx_users_email (000002) users_email_key / users_email_active bilan takror.
DROP INDEX IF EXISTS idx_users_email;
