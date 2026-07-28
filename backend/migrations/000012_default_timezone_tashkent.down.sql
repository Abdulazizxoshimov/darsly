-- Default'ni qaytaramiz. Qatorlardagi qiymatlar TEGILMAYDI: qaysi biri
-- migratsiyada o'zgargani, qaysi biri foydalanuvchi tanlagani endi ma'lum emas —
-- hammasini 'UTC' ga qaytarish to'g'ri ma'lumotni ham buzardi.
ALTER TABLE users ALTER COLUMN timezone SET DEFAULT 'UTC';

COMMENT ON COLUMN users.timezone IS NULL;
