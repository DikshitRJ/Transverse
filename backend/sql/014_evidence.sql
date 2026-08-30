-- Migration 014: Evidence pipeline (resume/codebase uploads + connector-sourced signal)
-- user_id is intentionally NOT a foreign key into users(id): BYPASS_AUTH injects a
-- synthetic "dev-user-001" identity that may not have a corresponding users row, mirroring
-- the same convention already used by user_tutorial_progress in migration 013.
CREATE TABLE IF NOT EXISTS evidence_sources (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL,
    kind          TEXT NOT NULL,
    external_ref  TEXT,
    object_key    TEXT,
    status        TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at  TIMESTAMPTZ,
    purge_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_evidence_sources_user_id ON evidence_sources(user_id);

CREATE TABLE IF NOT EXISTS evidence_extracts (
    id                  TEXT PRIMARY KEY,
    evidence_source_id  TEXT NOT NULL REFERENCES evidence_sources(id) ON DELETE CASCADE,
    extracted_json      JSONB NOT NULL DEFAULT '{}'::jsonb,
    confidence          DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_evidence_extracts_source_id ON evidence_extracts(evidence_source_id);
