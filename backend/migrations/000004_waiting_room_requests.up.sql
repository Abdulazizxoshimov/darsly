-- =============================================================================
-- WAITING_ROOM_REQUESTS — kutish xonasi kirish so'rovlari
-- =============================================================================
CREATE TABLE waiting_room_requests (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id      UUID         NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    requester_name VARCHAR(100) NOT NULL,
    guest_identity VARCHAR(64)  NOT NULL,  -- admit bo'lganda LiveKit identity
    status         VARCHAR(16)  NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'admitted', 'rejected')),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    decided_at     TIMESTAMPTZ
);

CREATE INDEX idx_wrr_lesson_pending ON waiting_room_requests (lesson_id) WHERE status = 'pending';
CREATE INDEX idx_wrr_created_at     ON waiting_room_requests (created_at);

COMMENT ON TABLE waiting_room_requests IS 'Kutish xonasidagi kirish so''rovlari (admit/reject)';
