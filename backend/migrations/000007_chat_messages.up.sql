-- =============================================================================
-- CHAT_MESSAGES — dars ichidagi chat tarixi
-- =============================================================================
CREATE TABLE chat_messages (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id       UUID         NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    sender_identity VARCHAR(128) NOT NULL,
    sender_name     VARCHAR(128) NOT NULL,
    body            TEXT         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_messages_lesson ON chat_messages (lesson_id, created_at);

COMMENT ON TABLE chat_messages IS 'Dars chat tarixi (real-vaqt LiveKit data-channel orqali)';
