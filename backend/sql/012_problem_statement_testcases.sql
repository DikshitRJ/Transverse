-- Add statement and test_cases columns to problems table
ALTER TABLE problems ADD COLUMN IF NOT EXISTS statement TEXT DEFAULT '';
ALTER TABLE problems ADD COLUMN IF NOT EXISTS test_cases JSONB DEFAULT '[]'::jsonb;
