-- ADMIN roli constraint'ini qaytarish. DIQQAT: mavjud 'admin' qatorlar bo'lsa bu
-- migratsiya muvaffaqiyatsiz bo'ladi — avval ularni boshqa rolga o'tkazing.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('mentor', 'student'));

COMMENT ON COLUMN users.role IS 'Global rol: mentor / student';
