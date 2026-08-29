-- HNSW index for cosine ANN search (the concept-similarity engine)
CREATE INDEX IF NOT EXISTS problems_embedding_idx 
    ON problems USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Topic + rating index for scoped queries
CREATE INDEX IF NOT EXISTS problems_topic_rating_idx ON problems (topic, glicko_rating);
CREATE INDEX IF NOT EXISTS problems_source_rating_idx ON problems (source, glicko_rating);

-- Session indexes
CREATE INDEX IF NOT EXISTS sessions_user_status_idx ON practice_sessions (user_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS sessions_user_active_idx 
    ON practice_sessions (user_id) WHERE status = 'ACTIVE';

-- Stats indexes  
CREATE INDEX IF NOT EXISTS topic_stats_user_idx ON topic_stats (user_id);
