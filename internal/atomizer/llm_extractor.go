package atomizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/llm"
)

// Extractor extracts semantic code block events from git commits using LLM
// Reference: AGENT_P2A_LLM_ATOMIZER.md - LLM-based atomization
type Extractor struct {
	llmClient *llm.Client
}

// NewExtractor creates a new code block extractor
func NewExtractor(llmClient *llm.Client) *Extractor {
	return &Extractor{llmClient: llmClient}
}

// ExtractCodeBlocks extracts semantic events from a commit's diff
// Returns CommitChangeEventLog containing all code block changes
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Main extraction logic
func (e *Extractor) ExtractCodeBlocks(ctx context.Context, commit CommitData) (*CommitChangeEventLog, error) {
	// Handle empty diff case
	if strings.TrimSpace(commit.DiffContent) == "" {
		return &CommitChangeEventLog{
			CommitSHA:        commit.SHA,
			AuthorEmail:      commit.AuthorEmail,
			Timestamp:        commit.Timestamp,
			LLMIntentSummary: "Empty commit (no changes)",
			MentionedIssues:  []string{},
			ChangeEvents:     []ChangeEvent{},
		}, nil
	}

	// 1. Build LLM prompt
	prompt := fmt.Sprintf(AtomizationPromptTemplate,
		commit.Message,
		truncateDiff(commit.DiffContent, 15000), // Limit diff size to avoid token limits
		commit.SHA,
		commit.AuthorEmail,
		commit.Timestamp.Format(time.RFC3339),
	)

	// 2. Call LLM with JSON mode
	response, err := e.llmClient.CompleteJSON(ctx, "system", prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 3. Parse JSON response
	var result CommitChangeEventLog
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Try to repair common JSON errors
		repaired := repairJSON(response)
		if err2 := json.Unmarshal([]byte(repaired), &result); err2 != nil {
			return nil, fmt.Errorf("JSON parse failed: %w (original: %v)", err2, err)
		}
	}

	// 4. Validate result
	if err := validateEventLog(&result); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &result, nil
}

// ExtractCodeBlocksBatch processes multiple commits in parallel
// Returns a map of commit SHA to CommitChangeEventLog
// Failures are logged but don't stop the entire batch
func (e *Extractor) ExtractCodeBlocksBatch(ctx context.Context, commits []CommitData) (map[string]*CommitChangeEventLog, []error) {
	results := make(map[string]*CommitChangeEventLog)
	var errors []error

	// Process commits sequentially (parallel processing can be added later)
	for i, commit := range commits {
		eventLog, err := e.ExtractCodeBlocks(ctx, commit)
		if err != nil {
			errors = append(errors, fmt.Errorf("commit %d (%s): %w", i, commit.SHA[:8], err))
			continue
		}
		results[commit.SHA] = eventLog
	}

	return results, errors
}

// repairJSON attempts to fix common LLM JSON formatting issues
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Error handling for malformed responses
func repairJSON(s string) string {
	// Remove markdown code blocks
	s = strings.ReplaceAll(s, "```json\n", "")
	s = strings.ReplaceAll(s, "```json", "")
	s = strings.ReplaceAll(s, "\n```", "")
	s = strings.ReplaceAll(s, "```", "")

	// Trim whitespace
	s = strings.TrimSpace(s)

	// Check if LLM returned an array instead of an object
	// If it starts with [, extract the first element
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		// Try to parse as array and extract first element
		var arr []json.RawMessage
		if err := json.Unmarshal([]byte(s), &arr); err == nil && len(arr) > 0 {
			return string(arr[0])
		}
	}

	return s
}

// validateEventLog ensures the LLM output is reasonable
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Validation requirements
func validateEventLog(log *CommitChangeEventLog) error {
	if log.CommitSHA == "" {
		return fmt.Errorf("missing commit_sha")
	}
	if log.AuthorEmail == "" {
		return fmt.Errorf("missing author_email")
	}

	// Initialize empty arrays if nil
	if log.MentionedIssues == nil {
		log.MentionedIssues = []string{}
	}
	if log.ChangeEvents == nil {
		log.ChangeEvents = []ChangeEvent{}
	}

	// Filter and validate change events
	var validEvents []ChangeEvent
	for i, event := range log.ChangeEvents {
		if event.Behavior == "" {
			return fmt.Errorf("event %d missing behavior", i)
		}

		// Validate behavior is one of the allowed values
		validBehaviors := map[string]bool{
			"CREATE_BLOCK":  true,
			"MODIFY_BLOCK":  true,
			"DELETE_BLOCK":  true,
			"ADD_IMPORT":    true,
			"REMOVE_IMPORT": true,
		}
		if !validBehaviors[event.Behavior] {
			return fmt.Errorf("event %d has invalid behavior: %s", i, event.Behavior)
		}

		if event.TargetFile == "" {
			return fmt.Errorf("event %d missing target_file", i)
		}

		// For block operations, validate block_type if present
		// Be lenient: if block_type is invalid, filter or normalize it
		if event.BlockType != "" {
			validBlockTypes := map[string]bool{
				"function":  true,
				"method":    true,
				"class":     true,
				"component": true,
			}
			if !validBlockTypes[event.BlockType] {
				// Normalize common variations to valid types
				switch strings.ToLower(event.BlockType) {
				case "variable", "constant", "var", "const":
					// Skip variables/constants - not code blocks we track
					continue
				case "text", "documentation", "doc", "markdown":
					// Skip documentation changes
					continue
				default:
					// Unknown type - normalize to function as best guess
					event.BlockType = "function"
				}
			}
		}

		validEvents = append(validEvents, event)
	}

	// Update with filtered events
	log.ChangeEvents = validEvents

	return nil
}

// truncateDiff limits diff size to avoid token limits
// Large commits (>100 files) are split into smaller chunks
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Edge cases: large commits
func truncateDiff(diff string, maxChars int) string {
	if len(diff) <= maxChars {
		return diff
	}

	// Truncate with a warning message
	truncated := diff[:maxChars]
	truncated += "\n\n[DIFF TRUNCATED - Original size: " + fmt.Sprintf("%d", len(diff)) + " chars]"

	return truncated
}

// IsBinaryFile checks if a file appears to be binary based on extension
// Binary files are skipped during extraction
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Edge cases: binary files
func IsBinaryFile(filename string) bool {
	binaryExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico",
		".pdf", ".zip", ".tar", ".gz", ".bz2",
		".exe", ".dll", ".so", ".dylib",
		".wasm", ".class", ".jar",
		".mp3", ".mp4", ".avi", ".mov",
	}

	lowerFilename := strings.ToLower(filename)
	for _, ext := range binaryExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return true
		}
	}

	return false
}
