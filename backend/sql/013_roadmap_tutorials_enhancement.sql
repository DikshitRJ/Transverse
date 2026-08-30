-- Migration 013: Roadmap & Tutorial enhancements
CREATE TABLE IF NOT EXISTS user_tutorial_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    tutorial_id UUID NOT NULL,
    completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    UNIQUE(user_id, tutorial_id)
);

CREATE INDEX IF NOT EXISTS idx_tutorials_topic_tags ON tutorials USING GIN(topic_tags);
