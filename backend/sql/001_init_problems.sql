-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Problems table: multi-platform DSA/CP problem bank
CREATE TABLE IF NOT EXISTS problems (
    id                TEXT PRIMARY KEY,
    source            TEXT NOT NULL,
    name              TEXT NOT NULL,
    url               TEXT NOT NULL,
    slug              TEXT,
    contest_id        TEXT,
    tags              TEXT[] DEFAULT '{}',
    topic             TEXT,
    subtopic          TEXT,
    difficulty_label  TEXT,
    glicko_rating     FLOAT NOT NULL DEFAULT 1500,
    glicko_rd         FLOAT NOT NULL DEFAULT 350,
    glicko_volatility FLOAT NOT NULL DEFAULT 0.06,
    attempt_count     INT NOT NULL DEFAULT 0,
    solve_rate        FLOAT,
    avg_time_ms       INT DEFAULT 0,
    embedding         VECTOR(384),
    embed_text        TEXT,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);
