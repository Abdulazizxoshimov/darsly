-- =============================================================================
-- RECORDINGS — dars yozuvlari (LiveKit Egress → MinIO)
-- =============================================================================
CREATE TABLE recordings (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id     UUID         NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    egress_id     VARCHAR(64)  NOT NULL,
    object_key    TEXT         NOT NULL,  -- MinIO ichidagi yo'l
    status        VARCHAR(16)  NOT NULL DEFAULT 'recording'
                  CHECK (status IN ('recording', 'processing', 'ready', 'failed')),
    duration_sec  INTEGER      NOT NULL DEFAULT 0,
    size_bytes    BIGINT       NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    ended_at      TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_recordings_egress_id ON recordings (egress_id);
CREATE INDEX idx_recordings_lesson_id        ON recordings (lesson_id);
CREATE INDEX idx_recordings_status           ON recordings (status);

COMMENT ON TABLE recordings IS 'Dars yozuvlari (Egress orqali MinIO ga yoziladi)';
