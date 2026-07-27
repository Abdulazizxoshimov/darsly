-- =============================================================================
-- USERS — foydalanuvchilar (mentor / student)
-- =============================================================================
CREATE TABLE users (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email           CITEXT       NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    avatar_url      TEXT,
    color           VARCHAR(7)   NOT NULL DEFAULT '#6366F1',
    role            VARCHAR(32)  NOT NULL DEFAULT 'student'
                    CHECK (role IN ('mentor', 'student')),
    timezone        VARCHAR(64)  NOT NULL DEFAULT 'UTC',
    language        VARCHAR(8)   NOT NULL DEFAULT 'uz',
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_users_email           ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_is_active       ON users (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_full_name_trgm  ON users USING GIN (full_name gin_trgm_ops);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  users IS 'Tizim foydalanuvchilari';
COMMENT ON COLUMN users.role IS 'Global rol: mentor / student';

-- =============================================================================
-- REFRESH_TOKENS — JWT refresh tokenlarni saqlash
-- =============================================================================
CREATE TABLE refresh_tokens (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    VARCHAR(255) NOT NULL UNIQUE,  -- sha256(token)
    user_agent    TEXT,
    ip_address    INET,
    expires_at    TIMESTAMPTZ  NOT NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at) WHERE revoked_at IS NULL;

COMMENT ON TABLE refresh_tokens IS 'Foydalanuvchi sessiyalari (refresh JWT)';

-- =============================================================================
-- PASSWORD_RESETS — parolni tiklash so'rovlari
-- =============================================================================
CREATE TABLE password_resets (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   VARCHAR(255) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ  NOT NULL,
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_resets_user_id    ON password_resets (user_id);
CREATE INDEX idx_password_resets_expires_at ON password_resets (expires_at) WHERE used_at IS NULL;
