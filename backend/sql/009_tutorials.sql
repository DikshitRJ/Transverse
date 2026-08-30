CREATE TYPE tutorial_type_enum AS ENUM ('article', 'video', 'interactive', 'playlist');
CREATE TYPE tutorial_difficulty_enum AS ENUM ('beginner', 'intermediate', 'advanced');

CREATE TABLE tutorials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    source_url TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    topic_id TEXT,
    topic_tags TEXT[],
    type tutorial_type_enum NOT NULL,
    difficulty tutorial_difficulty_enum NOT NULL,
    estimated_minutes INT NOT NULL,
    summary TEXT,
    license_note TEXT,
    thumbnail_url TEXT,
    scraped_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    checksum TEXT
);
