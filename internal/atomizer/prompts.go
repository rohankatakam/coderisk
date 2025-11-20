package atomizer

// AtomizationPromptTemplate is the LLM prompt for extracting code block changes
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Uses gemini-2.0-flash for fast extraction
// NOTE: Metadata, file paths, and line numbers are NOT extracted by LLM to prevent hallucination
// File paths and line numbers are parsed from diff headers separately
const AtomizationPromptTemplate = `You are a Semantic Event Processor analyzing git commits.

TASK: Extract ALL code block changes from this commit diff.

COMMIT MESSAGE:
%s

DIFF:
%s

OUTPUT SCHEMA: Return ONLY valid JSON (no markdown, no explanations):
{
  "llm_intent_summary": "One sentence summary of the change intent",
  "mentioned_issues_in_msg": ["#123", "#456"],
  "change_events": [
    {
      "behavior": "CREATE_BLOCK",
      "target_block_name": "formatDate",
      "block_type": "function",
      "signature": "(date: Date, format: string): string"
    }
  ]
}

SIGNATURE EXTRACTION (REQUIRED):
- Extract full function signature with parameter types and return type
- Format: "(param1: type1, param2: type2): returnType"
- Examples by language:
  * Go: "(ctx context.Context, id int64): error"
  * Python: "(self, username: str, password: str): bool"
  * TypeScript: "(data: UserData): Promise<void>"
  * Java: "(String username, int userId): void"
  * Ruby: "(username, password)"
- If no parameters: "()"
- If return type unknown: omit or use "void"
- For overloaded functions, signature MUST distinguish them
- Include signature field in ALL CREATE_BLOCK, MODIFY_BLOCK, and RENAME_BLOCK events

RENAME DETECTION (NEW BEHAVIOR):
- Detect when a function is RENAMED (not deleted and recreated)
- Indicators:
  * Function with name A disappears
  * Function with name B appears
  * Same signature (or very similar)
  * Same file location
  * Occurs in same diff hunk
- Output for renames:
  {
    "behavior": "RENAME_BLOCK",
    "old_block_name": "handleLogin",
    "target_block_name": "processAuth",
    "signature": "(username: string, password: string): boolean",
    "block_type": "function",
    "old_version": "function handleLogin(username: string, password: string): boolean { ... }",
    "new_version": "function processAuth(username: string, password: string): boolean { ... }"
  }
- Do NOT output both DELETE + CREATE for renames
- If uncertain, prefer MODIFY over RENAME

BEHAVIOR SELECTION PRIORITY:
1. RENAME_BLOCK - if function name changed but signature same
2. MODIFY_BLOCK - if function body changed but name/signature same
3. CREATE_BLOCK - if function first appears in this commit
4. DELETE_BLOCK - if function removed in this commit

RULES:
- Extract function/method/class level blocks (ignore variables, constants, documentation files)
- Detect import additions/removals from diff
- Summarize the "why" from commit message
- Return empty array if no code blocks changed
- behavior: MUST be one of: CREATE_BLOCK, MODIFY_BLOCK, DELETE_BLOCK, RENAME_BLOCK, ADD_IMPORT, REMOVE_IMPORT
- block_type: MUST be one of: function, method, class, component (REQUIRED for block operations, OMIT for imports)
- target_block_name: ONLY the short name of the function/method/class (max 100 chars, e.g., "formatDate", "UserClass")
- signature: Function signature with parameter types and return type (REQUIRED for CREATE_BLOCK, MODIFY_BLOCK, RENAME_BLOCK)
- old_block_name: Old function name (REQUIRED for RENAME_BLOCK only)
- For ADD_IMPORT/REMOVE_IMPORT, include dependency_path instead of target_block_name
- For MODIFY_BLOCK and RENAME_BLOCK, you can optionally include old_version and new_version (code snippets)
- Extract issue numbers from commit message (e.g., "Fixes #123" → ["#123"])
- Keep llm_intent_summary concise (1-2 sentences max)
- ONLY extract changes to code files (.go, .ts, .js, .py, etc) - ignore .md, .txt, and documentation files
- Return a SINGLE JSON object (not an array of objects)
- DO NOT include file paths or line numbers - these are parsed separately from diff headers
- DO NOT include descriptive text in target_block_name - ONLY the actual function/class name

EXAMPLES:

Example 1 - Function Creation:
{
  "llm_intent_summary": "Add date formatting utility function",
  "mentioned_issues_in_msg": [],
  "change_events": [
    {
      "behavior": "CREATE_BLOCK",
      "target_block_name": "formatDate",
      "block_type": "function",
      "signature": "(date: Date, format: string): string"
    }
  ]
}

Example 2 - Import Addition:
{
  "llm_intent_summary": "Add axios dependency for HTTP requests",
  "mentioned_issues_in_msg": ["#42"],
  "change_events": [
    {
      "behavior": "ADD_IMPORT",
      "dependency_path": "axios"
    }
  ]
}

Example 3 - Multiple Changes:
{
  "llm_intent_summary": "Refactor authentication logic to use JWT tokens",
  "mentioned_issues_in_msg": ["#100", "#102"],
  "change_events": [
    {
      "behavior": "MODIFY_BLOCK",
      "target_block_name": "login",
      "block_type": "function",
      "signature": "(username: string, password: string): Promise<AuthToken>"
    },
    {
      "behavior": "ADD_IMPORT",
      "dependency_path": "jsonwebtoken"
    },
    {
      "behavior": "DELETE_BLOCK",
      "target_block_name": "validateSession",
      "block_type": "function"
    }
  ]
}

Example 4 - Function Rename:
{
  "llm_intent_summary": "Rename authentication handler for clarity",
  "mentioned_issues_in_msg": [],
  "change_events": [
    {
      "behavior": "RENAME_BLOCK",
      "old_block_name": "handleLogin",
      "target_block_name": "processAuth",
      "block_type": "function",
      "signature": "(username: string, password: string): boolean",
      "old_version": "function handleLogin(username: string, password: string): boolean { ... }",
      "new_version": "function processAuth(username: string, password: string): boolean { ... }"
    }
  ]
}

IMPORTANT:
- Return valid JSON only (no markdown code blocks)
- Ensure all required fields are present (llm_intent_summary, mentioned_issues_in_msg, change_events)
- Use empty arrays [] if no data (not null)
- Extract ALL code blocks, not just the first one
- Focus on extracting semantic meaning - commit metadata is handled separately
`
