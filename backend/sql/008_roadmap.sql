CREATE TYPE roadmap_source_enum AS ENUM ('llm_generated', 'curated');
CREATE TYPE roadmap_status_enum AS ENUM ('active', 'completed', 'abandoned');
CREATE TYPE node_status_enum AS ENUM ('locked', 'unlocked', 'in_progress', 'mastered', 'tested_out');

CREATE TABLE roadmap_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_role TEXT NOT NULL,
    source roadmap_source_enum NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE roadmap_phases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    roadmap_template_id UUID NOT NULL REFERENCES roadmap_templates(id) ON DELETE CASCADE,
    sequence INT NOT NULL,
    title TEXT NOT NULL,
    unlock_rule JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE roadmap_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phase_id UUID NOT NULL REFERENCES roadmap_phases(id) ON DELETE CASCADE,
    topic_id TEXT NOT NULL, -- matches id from topics.json
    sequence INT NOT NULL,
    unlock_rule JSONB NOT NULL DEFAULT '{"type":"no_prerequisite"}'::jsonb,
    tutorial_ids UUID[] DEFAULT '{}',
    practice_topic_ids TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_roadmaps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    roadmap_template_id UUID NOT NULL REFERENCES roadmap_templates(id) ON DELETE CASCADE,
    status roadmap_status_enum NOT NULL DEFAULT 'active',
    current_phase_id UUID REFERENCES roadmap_phases(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

CREATE TABLE user_roadmap_node_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_roadmap_id UUID NOT NULL REFERENCES user_roadmaps(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES roadmap_nodes(id) ON DELETE CASCADE,
    status node_status_enum NOT NULL DEFAULT 'locked',
    unlocked_at TIMESTAMP WITH TIME ZONE,
    mastered_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_roadmap_id, node_id)
);
