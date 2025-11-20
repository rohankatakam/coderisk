package atomizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/llm"
	log "github.com/sirupsen/logrus"
	"google.golang.org/genai"
)

// Extractor extracts semantic code block events from git commits using LLM
// Reference: AGENT_P2A_LLM_ATOMIZER.md - LLM-based atomization
type Extractor struct {
	llmClient *llm.Client
	db        *pgxpool.Pool
	chunker   *git.DiffChunker
}

// NewExtractor creates a new code block extractor
func NewExtractor(llmClient *llm.Client) *Extractor {
	return &Extractor{
		llmClient: llmClient,
		chunker:   git.NewDiffChunker(100 * 1024), // 100KB max chunk size
	}
}

// NewExtractorWithDB creates a new code block extractor with database support
func NewExtractorWithDB(llmClient *llm.Client, db *pgxpool.Pool) *Extractor {
	return &Extractor{
		llmClient: llmClient,
		db:        db,
		chunker:   git.NewDiffChunker(100 * 1024), // 100KB max chunk size
	}
}

// ExtractCodeBlocks extracts semantic events from a commit's diff using structured output with chunking
// Returns CommitChangeEventLog with metadata appended AFTER LLM extraction to prevent hallucination
// Reference: AGENT_P2A_LLM_ATOMIZER.md - Main extraction logic
// Reference: AGENT_4_CHUNKING_SYSTEM.md - Intelligent chunking for large files
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

	// 1. Parse diff to extract file paths and line numbers (BEFORE LLM)
	parsedFiles := ParseDiff(commit.DiffContent)

	// 1b. Filter to code files only (skip docs, config, binary files)
	codeFiles := make(map[string]*DiffFileChange)
	for filePath, change := range parsedFiles {
		if IsCodeFile(filePath) {
			codeFiles[filePath] = change
		}
	}

	// If no code files remain after filtering, return empty event log
	if len(codeFiles) == 0 {
		return &CommitChangeEventLog{
			CommitSHA:        commit.SHA,
			AuthorEmail:      commit.AuthorEmail,
			Timestamp:        commit.Timestamp,
			LLMIntentSummary: "No code file changes detected (only config/docs/binary files)",
			MentionedIssues:  []string{},
			ChangeEvents:     []ChangeEvent{},
		}, nil
	}

	// 2. Process files with chunking logic
	var allEvents []ChangeEvent
	chunksProcessed := 0
	chunksSkipped := 0
	var commitSummary string
	var mentionedIssues []string

	for filePath, fileChange := range codeFiles {
		// Route by change type
		switch fileChange.ChangeType {
		case "deleted":
			// Create DELETE_BLOCK event for all blocks in file
			event := ChangeEvent{
				Behavior:        "DELETE_BLOCK",
				TargetFile:      filePath,
				TargetBlockName: "*", // All blocks in file
			}
			allEvents = append(allEvents, event)

		case "added":
			// Detect language
			lang := detectLanguage(filePath)

			// Extract full file content from diff hunks
			fileContent := extractNewFileContent(fileChange)

			// Extract chunks by function boundaries
			chunks := git.ExtractChunksForNewFile(fileContent, lang, 10) // max 10 chunks per file

			// Process each chunk with LLM
			for _, chunk := range chunks {
				llmEvents, err := e.processChunkWithLLM(ctx, chunk, commit, filePath)
				if err != nil {
					log.Errorf("Failed to process chunk for %s: %v", filePath, err)
					chunksSkipped++
					continue
				}

				// Enrich LLM events with file metadata
				for _, llmEvent := range llmEvents {
					allEvents = append(allEvents, ChangeEvent{
						Behavior:        llmEvent.Behavior,
						TargetFile:      filePath,
						TargetBlockName: llmEvent.TargetBlockName,
						OldBlockName:    llmEvent.OldBlockName,
						Signature:       llmEvent.Signature,
						BlockType:       llmEvent.BlockType,
						StartLine:       chunk.StartLine,
						EndLine:         chunk.EndLine,
						DependencyPath:  llmEvent.DependencyPath,
						OldVersion:      llmEvent.OldVersion,
						NewVersion:      llmEvent.NewVersion,
					})
				}
				chunksProcessed++
			}

		case "modified":
			// Extract chunks by diff boundaries (existing logic)
			chunks, err := e.chunker.ExtractChunks(commit.DiffContent)
			if err != nil {
				log.Errorf("Failed to extract chunks for %s: %v", filePath, err)
				continue
			}

			// Filter chunks for this specific file
			var fileChunks []git.DiffChunk
			for _, chunk := range chunks {
				if chunk.FilePath == filePath {
					fileChunks = append(fileChunks, chunk)
				}
			}

			// Enforce max chunks budget per file
			if len(fileChunks) > 10 {
				log.Warnf("File %s has %d chunks, limiting to 10", filePath, len(fileChunks))
				chunksSkipped += len(fileChunks) - 10
				fileChunks = fileChunks[:10]
			}

			// Process each chunk with LLM
			for _, chunk := range fileChunks {
				llmEvents, err := e.processChunkWithLLM(ctx, chunk, commit, filePath)
				if err != nil {
					log.Errorf("Failed to process chunk for %s: %v", filePath, err)
					chunksSkipped++
					continue
				}

				// Enrich LLM events with file metadata
				for _, llmEvent := range llmEvents {
					allEvents = append(allEvents, ChangeEvent{
						Behavior:        llmEvent.Behavior,
						TargetFile:      filePath,
						TargetBlockName: llmEvent.TargetBlockName,
						OldBlockName:    llmEvent.OldBlockName,
						Signature:       llmEvent.Signature,
						BlockType:       llmEvent.BlockType,
						StartLine:       chunk.StartLine,
						EndLine:         chunk.EndLine,
						DependencyPath:  llmEvent.DependencyPath,
						OldVersion:      llmEvent.OldVersion,
						NewVersion:      llmEvent.NewVersion,
					})
				}
				chunksProcessed++
			}

		case "renamed":
			// File rename handled by file_identity_map (not our concern)
			// If content also changed, process as MODIFIED
			if len(fileChange.Hunks) > 0 {
				// Has content changes, treat as modified
				// (reuse modified logic above - for simplicity, skip for now)
				log.Infof("File %s renamed with content changes (skipping content processing)", filePath)
			}
		}
	}

	// Deduplicate results using Agent 3's chunk_merger
	allEvents = MergeChunkResults(allEvents)

	// Validate and filter events
	validEvents := filterValidEvents(allEvents)

	// Extract commit summary from first chunk's LLM response (if available)
	// For simplicity, generate a basic summary
	if len(validEvents) > 0 {
		commitSummary = fmt.Sprintf("Modified %d code blocks across %d files", len(validEvents), len(codeFiles))
	} else {
		commitSummary = "No code block changes detected"
	}

	// Update chunk metadata in database
	if err := e.updateChunkMetadata(ctx, commit.SHA, chunksProcessed, chunksSkipped); err != nil {
		log.Errorf("Failed to update chunk metadata: %v", err)
	}

	// Build final CommitChangeEventLog
	result := &CommitChangeEventLog{
		CommitSHA:        commit.SHA,
		AuthorEmail:      commit.AuthorEmail,
		Timestamp:        commit.Timestamp,
		LLMIntentSummary: commitSummary,
		MentionedIssues:  mentionedIssues,
		ChangeEvents:     validEvents,
	}

	// Initialize empty arrays if nil
	if result.MentionedIssues == nil {
		result.MentionedIssues = []string{}
	}
	if result.ChangeEvents == nil {
		result.ChangeEvents = []ChangeEvent{}
	}

	return result, nil
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

// IsCodeFile determines if a file should be processed for code block extraction
// Filters out documentation, config files, binary files, and dotfiles
// Returns true only for actual source code files
func IsCodeFile(filename string) bool {
	// Skip binary files
	if IsBinaryFile(filename) {
		return false
	}

	lowerFilename := strings.ToLower(filename)

	// Skip documentation files
	docExtensions := []string{".md", ".mdx", ".txt", ".rst", ".adoc"}
	for _, ext := range docExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return false
		}
	}

	// Skip config files
	configExtensions := []string{
		".json", ".yaml", ".yml", ".toml", ".ini", ".cfg",
		".lock", ".sum", ".mod", ".env",
	}
	for _, ext := range configExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return false
		}
	}

	// Skip dotfiles (like .gitignore, .env, .pre-commit-config.yaml)
	// Extract base filename after last slash
	parts := strings.Split(filename, "/")
	basename := parts[len(parts)-1]
	if strings.HasPrefix(basename, ".") {
		return false
	}

	// Allow known code file extensions
	codeExtensions := []string{
		".go", ".py", ".js", ".ts", ".tsx", ".jsx",
		".java", ".c", ".cpp", ".h", ".hpp", ".cc", ".cxx",
		".rs", ".rb", ".php", ".swift", ".kt", ".kts",
		".scala", ".clj", ".cljs", ".ex", ".exs",
		".sh", ".bash", ".zsh", ".ps1",
		".cs", ".fs", ".vb", ".sql",
	}
	for _, ext := range codeExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return true
		}
	}

	// If no known code extension, default to false (conservative filtering)
	return false
}

// buildExtractionSchema defines the Google ResponseSchema for LLM extraction
// Reference: YC_DEMO_GAP_ANALYSIS.md - Server-side validation prevents hallucination
// NOTE: File paths and line numbers are parsed from diff headers, NOT extracted by LLM
func buildExtractionSchema() *genai.Schema {
	truePtr := boolPtr(true)
	maxBlockNameLength := int64(100)
	maxDependencyPathLength := int64(200)
	maxSummaryLength := int64(500)

	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"llm_intent_summary": {
				Type:        genai.TypeString,
				Description: "One sentence summary of the change intent from the commit message",
				MaxLength:   &maxSummaryLength,
			},
			"mentioned_issues_in_msg": {
				Type:        genai.TypeArray,
				Description: "Issue numbers mentioned in commit message (e.g., #123, #456)",
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
			},
			"change_events": {
				Type:        genai.TypeArray,
				Description: "List of code block changes (file paths and line numbers are parsed separately)",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"behavior": {
							Type:        genai.TypeString,
							Description: "Type of change",
							Enum:        []string{"CREATE_BLOCK", "MODIFY_BLOCK", "DELETE_BLOCK", "ADD_IMPORT", "REMOVE_IMPORT"},
						},
						"target_block_name": {
							Type:        genai.TypeString,
							Description: "Name of the function/method/class (SHORT NAME ONLY, max 100 chars, omit for imports)",
							MaxLength:   &maxBlockNameLength,
							Nullable:    truePtr,
						},
						"block_type": {
							Type:        genai.TypeString,
							Description: "Type of code block",
							Enum:        []string{"function", "method", "class", "component"},
							Nullable:    truePtr,
						},
						"dependency_path": {
							Type:        genai.TypeString,
							Description: "For imports: package/module path (e.g., 'axios', 'lodash')",
							MaxLength:   &maxDependencyPathLength,
							Nullable:    truePtr,
						},
						"old_version": {
							Type:        genai.TypeString,
							Description: "Old code snippet for modifications (optional)",
							Nullable:    truePtr,
						},
						"new_version": {
							Type:        genai.TypeString,
							Description: "New code snippet for modifications (optional)",
							Nullable:    truePtr,
						},
					},
					Required: []string{"behavior"},
				},
			},
		},
		Required: []string{"llm_intent_summary", "mentioned_issues_in_msg", "change_events"},
	}
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// matchEventsToFiles matches LLM-extracted events to parsed file changes
// Enriches events with file paths and line numbers from diff parsing
// Reference: YC_DEMO_GAP_ANALYSIS.md - Prevent file path hallucination
func matchEventsToFiles(llmEvents []LLMChangeEvent, parsedFiles map[string]*DiffFileChange) []ChangeEvent {
	var enrichedEvents []ChangeEvent

	// If only one file in diff, all events belong to that file
	if len(parsedFiles) == 1 {
		var filePath string
		var fileChange *DiffFileChange
		for path, change := range parsedFiles {
			filePath = path
			fileChange = change
			break
		}

		startLine, endLine := GetLineRangeForFile(fileChange)

		for _, llmEvent := range llmEvents {
			enrichedEvents = append(enrichedEvents, ChangeEvent{
				Behavior:        llmEvent.Behavior,
				TargetFile:      filePath,
				TargetBlockName: llmEvent.TargetBlockName,
				BlockType:       llmEvent.BlockType,
				StartLine:       startLine,
				EndLine:         endLine,
				DependencyPath:  llmEvent.DependencyPath,
				OldVersion:      llmEvent.OldVersion,
				NewVersion:      llmEvent.NewVersion,
			})
		}

		return enrichedEvents
	}

	// Multiple files: distribute events evenly across files
	// This is a heuristic - in most cases each file gets one event
	if len(llmEvents) > 0 && len(parsedFiles) > 0 {
		fileList := make([]string, 0, len(parsedFiles))
		for path := range parsedFiles {
			fileList = append(fileList, path)
		}

		for i, llmEvent := range llmEvents {
			// Round-robin distribution
			fileIndex := i % len(fileList)
			filePath := fileList[fileIndex]
			fileChange := parsedFiles[filePath]

			startLine, endLine := GetLineRangeForFile(fileChange)

			enrichedEvents = append(enrichedEvents, ChangeEvent{
				Behavior:        llmEvent.Behavior,
				TargetFile:      filePath,
				TargetBlockName: llmEvent.TargetBlockName,
				BlockType:       llmEvent.BlockType,
				StartLine:       startLine,
				EndLine:         endLine,
				DependencyPath:  llmEvent.DependencyPath,
				OldVersion:      llmEvent.OldVersion,
				NewVersion:      llmEvent.NewVersion,
			})
		}
	}

	return enrichedEvents
}

// filterValidEvents validates and filters change events
// Replaces the old validateEventLog logic but focuses only on event validation
func filterValidEvents(events []ChangeEvent) []ChangeEvent {
	validBehaviors := map[string]bool{
		"CREATE_BLOCK":  true,
		"MODIFY_BLOCK":  true,
		"DELETE_BLOCK":  true,
		"ADD_IMPORT":    true,
		"REMOVE_IMPORT": true,
	}

	validBlockTypes := map[string]bool{
		"function":  true,
		"method":    true,
		"class":     true,
		"component": true,
	}

	var validEvents []ChangeEvent
	for _, event := range events {
		// Skip invalid behaviors (should not happen with schema)
		if !validBehaviors[event.Behavior] {
			continue
		}

		// Skip events without target file (should not happen with schema)
		if event.TargetFile == "" {
			continue
		}

		// Skip events with empty block names (except imports which use dependency_path)
		// This filters out LLM extraction errors where block name was not identified
		if event.Behavior != "ADD_IMPORT" && event.Behavior != "REMOVE_IMPORT" {
			if strings.TrimSpace(event.TargetBlockName) == "" {
				continue
			}
		}

		// Filter or normalize block_type if present
		if event.BlockType != "" && !validBlockTypes[event.BlockType] {
			// Normalize common variations or skip
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

		validEvents = append(validEvents, event)
	}

	return validEvents
}

// extractNewFileContent extracts the full content of a newly added file from diff hunks
func extractNewFileContent(fileChange *DiffFileChange) string {
	var contentLines []string

	for _, hunk := range fileChange.Hunks {
		lines := strings.Split(hunk.Content, "\n")
		for _, line := range lines {
			// Extract lines that start with "+" (new content)
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				// Remove the "+" prefix
				contentLines = append(contentLines, line[1:])
			}
		}
	}

	return strings.Join(contentLines, "\n")
}

// processChunkWithLLM processes a single chunk with the LLM
func (e *Extractor) processChunkWithLLM(ctx context.Context, chunk git.DiffChunk, commit CommitData, filePath string) ([]LLMChangeEvent, error) {
	// Build chunk content (either from Lines field or Content field)
	var chunkContent string
	if len(chunk.Lines) > 0 {
		chunkContent = strings.Join(chunk.Lines, "\n")
	} else {
		chunkContent = chunk.Content
	}

	// Build prompt with chunk header for context
	prompt := fmt.Sprintf(AtomizationPromptTemplate,
		commit.Message,
		chunk.FileHeader+"\n"+chunkContent,
	)

	// Call LLM with rate limiting (handled by gemini_client)
	response, err := e.llmClient.CompleteJSON(ctx, "system", prompt)
	if err != nil {
		return nil, err
	}

	// Handle empty response
	if strings.TrimSpace(response) == "" {
		return []LLMChangeEvent{}, nil
	}

	// Parse response
	var llmResponse LLMExtractionResponse
	if err := json.Unmarshal([]byte(response), &llmResponse); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return llmResponse.ChangeEvents, nil
}

// updateChunkMetadata updates the chunk processing metadata for a commit
func (e *Extractor) updateChunkMetadata(ctx context.Context, commitSHA string, processed, skipped int) error {
	if e.db == nil {
		// If no database connection, skip metadata update
		return nil
	}

	query := `
		UPDATE github_commits
		SET diff_chunks_processed = $1,
		    diff_chunks_skipped = $2
		WHERE sha = $3
	`
	_, err := e.db.Exec(ctx, query, processed, skipped, commitSHA)
	return err
}
