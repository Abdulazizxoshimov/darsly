-- =============================================================================
-- POLLS — dars ichidagi so'rovnomalar / viktorinalar
-- =============================================================================
CREATE TABLE polls (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id  UUID         NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    question   VARCHAR(500) NOT NULL,
    options    TEXT[]       NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    closed_at  TIMESTAMPTZ
);

CREATE INDEX idx_polls_lesson ON polls (lesson_id, created_at DESC);

CREATE TABLE poll_votes (
    poll_id        UUID        NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    voter_identity VARCHAR(128) NOT NULL,
    option_index   INTEGER     NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (poll_id, voter_identity)  -- har ishtirokchi bitta ovoz
);

COMMENT ON TABLE polls IS 'Dars so''rovnomalari (real-vaqt natijalar bilan)';
