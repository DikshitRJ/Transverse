-- User problem statistics: tracks historical attempts, successes, and time per problem
CREATE TABLE IF NOT EXISTS user_problem_stats (
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    problem_id     TEXT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    attempt_count  INT NOT NULL DEFAULT 0,
    correct_count  INT NOT NULL DEFAULT 0,
    total_time_ms  BIGINT NOT NULL DEFAULT 0,
    last_attempted TIMESTAMPTZ,
    PRIMARY KEY (user_id, problem_id)
);

-- Topic mastery statistics: per-user per-topic mastery scores and psychometrics
CREATE TABLE IF NOT EXISTS topic_stats (
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic         TEXT NOT NULL,
    theta         FLOAT NOT NULL DEFAULT 1500,
    mastery_score FLOAT NOT NULL DEFAULT 0.0,
    glicko_rating FLOAT NOT NULL DEFAULT 1500,
    attempt_count INT NOT NULL DEFAULT 0,
    correct_count INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, topic)
);
