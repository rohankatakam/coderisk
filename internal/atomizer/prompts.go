package atomizer

// AtomizationPromptTemplate is the LLM prompt for extracting code block changes
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Uses gemini-2.0-flash-exp for fast extraction
const AtomizationPromptTemplate = `You are a Semantic Event Processor analyzing git commits.

TASK: Extract ALL code block changes from this commit diff.

COMMIT MESSAGE:
%s

DIFF:
%s

OUTPUT SCHEMA: Return ONLY valid JSON (no markdown, no explanations):
{
  "commit_sha": "%s",
  "author_email": "%s",
  "timestamp": "%s",
  "llm_intent_summary": "One sentence summary of the change intent",
  "mentioned_issues_in_msg": ["#123", "#456"],
  "change_events": [
    {
      "behavior": "CREATE_BLOCK",
      "target_file": "src/utils.ts",
      "target_block_name": "formatDate",
      "block_type": "function"
    }
  ]
}

RULES:
- Extract function/method/class level blocks (ignore variables, constants, documentation files)
- Detect import additions/removals from diff
- Summarize the "why" from commit message
- Return empty array if no code blocks changed
- behavior: MUST be one of: CREATE_BLOCK, MODIFY_BLOCK, DELETE_BLOCK, ADD_IMPORT, REMOVE_IMPORT
- block_type: MUST be one of: function, method, class, component (REQUIRED for block operations, OMIT for imports)
- For MODIFY_BLOCK, you can optionally include old_version and new_version (code snippets)
- For ADD_IMPORT/REMOVE_IMPORT, include dependency_path instead of target_block_name
- Extract issue numbers from commit message (e.g., "Fixes #123" → ["#123"])
- Keep llm_intent_summary concise (1-2 sentences max)
- ONLY extract changes to code files (.go, .ts, .js, .py, etc) - ignore .md, .txt, and documentation files
- Return a SINGLE JSON object (not an array of objects)

EXAMPLES:

Example 1 - Function Creation:
{
  "commit_sha": "abc123",
  "author_email": "dev@example.com",
  "timestamp": "2024-01-01T00:00:00Z",
  "llm_intent_summary": "Add date formatting utility function",
  "mentioned_issues_in_msg": [],
  "change_events": [
    {
      "behavior": "CREATE_BLOCK",
      "target_file": "src/utils.ts",
      "target_block_name": "formatDate",
      "block_type": "function"
    }
  ]
}

Example 2 - Import Addition:
{
  "commit_sha": "def456",
  "author_email": "dev@example.com",
  "timestamp": "2024-01-01T00:00:00Z",
  "llm_intent_summary": "Add axios dependency for HTTP requests",
  "mentioned_issues_in_msg": ["#42"],
  "change_events": [
    {
      "behavior": "ADD_IMPORT",
      "target_file": "src/api.ts",
      "dependency_path": "axios"
    }
  ]
}

Example 3 - Multiple Changes:
{
  "commit_sha": "ghi789",
  "author_email": "dev@example.com",
  "timestamp": "2024-01-01T00:00:00Z",
  "llm_intent_summary": "Refactor authentication logic to use JWT tokens",
  "mentioned_issues_in_msg": ["#100", "#102"],
  "change_events": [
    {
      "behavior": "MODIFY_BLOCK",
      "target_file": "src/auth.ts",
      "target_block_name": "login",
      "block_type": "function"
    },
    {
      "behavior": "ADD_IMPORT",
      "target_file": "src/auth.ts",
      "dependency_path": "jsonwebtoken"
    },
    {
      "behavior": "DELETE_BLOCK",
      "target_file": "src/auth.ts",
      "target_block_name": "validateSession",
      "block_type": "function"
    }
  ]
}

IMPORTANT:
- Return valid JSON only (no markdown code blocks)
- Ensure all required fields are present
- Use empty arrays [] if no data (not null)
- Extract ALL code blocks, not just the first one
`
