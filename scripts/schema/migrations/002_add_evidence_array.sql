-- Migration: Add evidence array field to github_issue_commit_refs
-- This enables storing multiple evidence tags per link (e.g., ["temporal_match_5min", "semantic_high"])
-- Reference: LINKING_PATTERNS.md and LINKING_QUALITY_SCORE.md

-- Add evidence column
ALTER TABLE github_issue_commit_refs
ADD COLUMN IF NOT EXISTS evidence TEXT[] DEFAULT '{}';

-- Create index for evidence array queries
CREATE INDEX IF NOT EXISTS idx_issue_commit_refs_evidence
ON github_issue_commit_refs USING GIN (evidence);

-- Add comment
COMMENT ON COLUMN github_issue_commit_refs.evidence IS
'Array of evidence tags (e.g., ["temporal_match_5min", "semantic_high", "explicit"])';
