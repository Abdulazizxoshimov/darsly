-- =============================================================================
-- ADMIN ROLI — user-management endi faqat `admin` rolida (H-2 tuzatishi).
-- role CHECK constraint'iga 'admin' qo'shiladi; mentor endi global admin EMAS.
-- =============================================================================
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'mentor', 'student'));

COMMENT ON COLUMN users.role IS 'Global rol: admin / mentor / student';
