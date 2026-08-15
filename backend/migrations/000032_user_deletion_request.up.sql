-- Mentor o'z hisobini o'chirishni SO'RAYDI (o'zi o'chira olmaydi); admin tasdiqlab
-- o'chiradi. NULL = so'rov yo'q. Bu yengil "flag" yondashuvi — alohida so'rovlar
-- jadvali o'rniga (bitta mentor = bitta ochiq so'rov, audit uchun yetarli).
ALTER TABLE users ADD COLUMN deletion_requested_at TIMESTAMPTZ;

COMMENT ON COLUMN users.deletion_requested_at IS
    'Mentor hisobni o''chirishni so''ragan vaqt; admin tasdiqlab o''chiradi. NULL = so''rov yo''q.';

-- Admin panelida "o'chirish so'ralgan" mentorlarni tez topish uchun.
CREATE INDEX idx_users_deletion_requested
    ON users (deletion_requested_at)
    WHERE deletion_requested_at IS NOT NULL AND deleted_at IS NULL;
