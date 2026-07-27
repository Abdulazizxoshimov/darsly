-- =============================================================================
-- NOTIFICATIONS — foydalanuvchi bildirishnomalari
-- =============================================================================
CREATE TABLE notifications (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(32)  NOT NULL,
    title      VARCHAR(255) NOT NULL,
    body       TEXT         NOT NULL DEFAULT '',
    lesson_id  UUID         REFERENCES lessons(id) ON DELETE CASCADE,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_unread ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;
CREATE INDEX idx_notifications_user        ON notifications (user_id, created_at DESC);

COMMENT ON TABLE notifications IS 'Foydalanuvchi bildirishnomalari (WS push + tarix)';

-- Dars eslatmasi bir marta yuborilishi uchun belgi.
ALTER TABLE lessons ADD COLUMN reminder_sent_at TIMESTAMPTZ;
