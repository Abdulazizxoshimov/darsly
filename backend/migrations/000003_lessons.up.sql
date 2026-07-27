-- =============================================================================
-- LESSONS — video-darslar
-- =============================================================================
CREATE TABLE lessons (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    mentor_id                UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title                    VARCHAR(255) NOT NULL,
    description              TEXT,
    scheduled_at             TIMESTAMPTZ,
    duration_min             INTEGER      NOT NULL DEFAULT 60,
    recurrence_rule          VARCHAR(255),                 -- iCal RRULE (haftalik takrorlanish)
    join_slug                VARCHAR(32)  NOT NULL UNIQUE,  -- havola: darsly.uz/j/{slug}
    passcode_hash            VARCHAR(255),                  -- bcrypt (nullable)
    is_locked                BOOLEAN      NOT NULL DEFAULT FALSE,
    is_recording_enabled     BOOLEAN      NOT NULL DEFAULT FALSE,
    is_waiting_room_enabled  BOOLEAN      NOT NULL DEFAULT TRUE,
    status                   VARCHAR(16)  NOT NULL DEFAULT 'scheduled'
                             CHECK (status IN ('scheduled', 'live', 'ended', 'cancelled')),
    started_at               TIMESTAMPTZ,
    ended_at                 TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_lessons_join_slug   ON lessons (join_slug);
CREATE INDEX idx_lessons_mentor_id          ON lessons (mentor_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_lessons_status             ON lessons (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_lessons_scheduled_at       ON lessons (scheduled_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_lessons_title_trgm         ON lessons USING GIN (title gin_trgm_ops);

CREATE TRIGGER trg_lessons_updated_at
    BEFORE UPDATE ON lessons
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  lessons IS 'Rejalashtirilgan / jonli video-darslar';
COMMENT ON COLUMN lessons.join_slug IS 'Havola orqali kirish uchun noyob slug';
