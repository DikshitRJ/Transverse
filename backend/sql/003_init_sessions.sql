-- Practice sessions table: active and historical practice sessions
CREATE TABLE IF NOT EXISTS practice_sessions (
    id                 TEXT PRIMARY KEY,
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode               TEXT NOT NULL DEFAULT 'ADAPTIVE',
    scope              JSONB NOT NULL DEFAULT '{}'::jsonb,
    theta_start        FLOAT NOT NULL DEFAULT 1500,
    theta_current      FLOAT NOT NULL DEFAULT 1500,
    current_problem_id TEXT REFERENCES problems(id) ON DELETE SET NULL,
    responses          JSONB NOT NULL DEFAULT '[]'::jsonb,
    question_count     INT NOT NULL DEFAULT 0,
    status             TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
