-- Users table: Learner accounts, IRT theta state, Glicko-2 ratings, and psychometric DNA
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    theta         FLOAT NOT NULL DEFAULT 1500,
    glicko_rating FLOAT NOT NULL DEFAULT 1500,
    glicko_rd     FLOAT NOT NULL DEFAULT 350,
    glicko_vol    FLOAT NOT NULL DEFAULT 0.06,
    dna           JSONB NOT NULL DEFAULT '{}'::jsonb,
    password_hash TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);
