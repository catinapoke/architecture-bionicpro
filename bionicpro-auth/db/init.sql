CREATE TABLE IF NOT EXISTS user_profiles (
    subject     TEXT PRIMARY KEY,
    username    TEXT,
    email       TEXT,
    name        TEXT,
    issuer      TEXT,
    provider    TEXT,
    raw_claims  TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_profiles_provider ON user_profiles (provider);
CREATE INDEX IF NOT EXISTS idx_user_profiles_email ON user_profiles (email);
