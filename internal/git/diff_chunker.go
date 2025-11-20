package git

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Language-specific function detection patterns
var functionPatterns = map[string]*regexp.Regexp{
	"go":         regexp.MustCompile(`^func\s+(\w+|\(\w+\s+\*?\w+\))\s+\w+`),
	"python":     regexp.MustCompile(`^def\s+\w+\s*\(`),
	"javascript": regexp.MustCompile(`^(function\s+\w+|const\s+\w+\s*=\s*(function|\())`),
	"typescript": regexp.MustCompile(`^(function\s+\w+|const\s+\w+\s*=\s*(function|\()|export\s+(async\s+)?function)`),
	"java":       regexp.MustCompile(`^\s*(public|private|protected|static|final)+.*\{$`),
	"ruby":       regexp.MustCompile(`^\s*def\s+\w+`),
	"rust":       regexp.MustCompile(`^\s*(pub\s+)?fn\s+\w+`),
	"c":          regexp.MustCompile(`^\w+\s+\w+\s*\([^\)]*\)\s*\{$`),
	"cpp":        regexp.MustCompile(`^\w+\s+\w+\s*\([^\)]*\)\s*\{$`),
}

const (
	maxChunkSize         = 100 * 1024 // 100KB per chunk
	maxChunksPerFile     = 10         // Budget per file
	defaultLinesPerChunk = 3000       // Fallback line-based chunking
)

// DiffChunk represents a logical chunk from git diff output
type DiffChunk struct {
	FilePath   string   // File path (canonical)
	StartLine  int      // Starting line number in new version
	EndLine    int      // Ending line number in new version
	Content    string   // Raw diff content including @@ headers
	SizeBytes  int      // Size in bytes
	Lines      []string // Lines of content (for new files)
	FileHeader string   // For context (synthesized @@ headers)
}

// DiffChunker extracts manageable chunks from git diff output
type DiffChunker struct {
	maxChunkSize int // Max size in bytes (default 100KB ~= 25K tokens)
}

// NewDiffChunker creates a new diff chunker
func NewDiffChunker(maxChunkSizeBytes int) *DiffChunker {
	if maxChunkSizeBytes == 0 {
		maxChunkSizeBytes = 100 * 1024 // 100KB default
	}
	return &DiffChunker{
		maxChunkSize: maxChunkSizeBytes,
	}
}

// ExtractChunks parses git diff output and extracts chunks
// Uses @@ headers as natural boundaries
func (dc *DiffChunker) ExtractChunks(diffOutput string) ([]DiffChunk, error) {
	var chunks []DiffChunk

	scanner := bufio.NewScanner(strings.NewReader(diffOutput))
	var currentFile string
	var currentChunk strings.Builder
	var chunkStart, chunkEnd int
	var inDiffBlock bool

	for scanner.Scan() {
		line := scanner.Text()

		// Detect file boundary (diff --git a/path b/path)
		if strings.HasPrefix(line, "diff --git") {
			// Save previous chunk if exists
			if currentChunk.Len() > 0 {
				chunks = append(chunks, DiffChunk{
					FilePath:  currentFile,
					StartLine: chunkStart,
					EndLine:   chunkEnd,
					Content:   currentChunk.String(),
					SizeBytes: currentChunk.Len(),
				})
				currentChunk.Reset()
			}

			// Parse new file path
			currentFile = parseFilePath(line)
			inDiffBlock = true
			currentChunk.WriteString(line + "\n")
			continue
		}

		// Skip file if not in diff block yet
		if !inDiffBlock {
			continue
		}

		// Detect chunk boundary (@@ -start,count +start,count @@)
		if strings.HasPrefix(line, "@@") {
			// If current chunk exceeds max size, save it and start new one
			if currentChunk.Len() > dc.maxChunkSize {
				chunks = append(chunks, DiffChunk{
					FilePath:  currentFile,
					StartLine: chunkStart,
					EndLine:   chunkEnd,
					Content:   currentChunk.String(),
					SizeBytes: currentChunk.Len(),
				})
				currentChunk.Reset()
				// Start new chunk with file header
				currentChunk.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", currentFile, currentFile))
			}

			// Parse line numbers from @@ header
			start, end := parseAtHeaders(line)
			if chunkStart == 0 {
				chunkStart = start
			}
			chunkEnd = end
		}

		currentChunk.WriteString(line + "\n")
	}

	// Save final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, DiffChunk{
			FilePath:  currentFile,
			StartLine: chunkStart,
			EndLine:   chunkEnd,
			Content:   currentChunk.String(),
			SizeBytes: currentChunk.Len(),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning diff output: %w", err)
	}

	return chunks, nil
}

// ExtractChunksByFile groups chunks by file path
func (dc *DiffChunker) ExtractChunksByFile(diffOutput string) (map[string][]DiffChunk, error) {
	chunks, err := dc.ExtractChunks(diffOutput)
	if err != nil {
		return nil, err
	}

	byFile := make(map[string][]DiffChunk)
	for _, chunk := range chunks {
		byFile[chunk.FilePath] = append(byFile[chunk.FilePath], chunk)
	}

	return byFile, nil
}

// BatchChunks groups chunks into batches of max N chunks per batch
// Used for multi-LLM distribution when file has >10 chunks
func BatchChunks(chunks []DiffChunk, maxChunksPerBatch int) [][]DiffChunk {
	if maxChunksPerBatch == 0 {
		maxChunksPerBatch = 10 // Default: max 10 chunks per LLM call
	}

	var batches [][]DiffChunk
	for i := 0; i < len(chunks); i += maxChunksPerBatch {
		end := i + maxChunksPerBatch
		if end > len(chunks) {
			end = len(chunks)
		}
		batches = append(batches, chunks[i:end])
	}

	return batches
}

// parseFilePath extracts file path from "diff --git a/path b/path" line
func parseFilePath(line string) string {
	// Format: diff --git a/path/to/file.go b/path/to/file.go
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		// Remove "a/" prefix
		path := parts[2]
		if strings.HasPrefix(path, "a/") {
			return path[2:]
		}
		return path
	}
	return ""
}

// ParseAtHeaders parses @@ -oldStart,oldCount +newStart,newCount @@ header (exported for use in resolution package)
// Returns (newStart, newEnd)
func ParseAtHeaders(line string) (int, int) {
	return parseAtHeaders(line)
}

// parseAtHeaders parses @@ -oldStart,oldCount +newStart,newCount @@ header
// Returns (newStart, newEnd)
func parseAtHeaders(line string) (int, int) {
	// Example: @@ -10,5 +12,7 @@ function name
	re := regexp.MustCompile(`@@ -(\d+),(\d+) \+(\d+),(\d+) @@`)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 5 {
		newStart, _ := strconv.Atoi(matches[3])
		newCount, _ := strconv.Atoi(matches[4])
		newEnd := newStart + newCount - 1
		return newStart, newEnd
	}

	return 0, 0
}

// DiffExcerpt represents a minimal excerpt for entity resolution
type DiffExcerpt struct {
	FilePath      string
	FirstLines    []string // First N lines of changed section
	LastLines     []string // Last N lines of changed section
	MiddleLines   []string // Smart middle section (code-dense)
	TotalLines    int      // Total lines in full diff
	TokenBudget   int      // Estimated tokens
}

// ExtractExcerptForResolution creates minimal context for fuzzy entity resolution
// Uses hybrid context strategy: first 10 + last 5 + smart middle
func ExtractExcerptForResolution(diffContent string, tokenBudget int) *DiffExcerpt {
	lines := strings.Split(diffContent, "\n")

	// Filter to only changed lines (lines starting with +/-)
	var changedLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Exclude diff metadata
			if !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
				changedLines = append(changedLines, line)
			}
		}
	}

	totalLines := len(changedLines)
	if totalLines == 0 {
		return &DiffExcerpt{
			FirstLines:  []string{},
			LastLines:   []string{},
			MiddleLines: []string{},
			TotalLines:  0,
			TokenBudget: 0,
		}
	}

	// If small enough, return all lines
	if totalLines <= 20 {
		return &DiffExcerpt{
			FirstLines:  changedLines,
			LastLines:   []string{},
			MiddleLines: []string{},
			TotalLines:  totalLines,
			TokenBudget: estimateTokens(changedLines),
		}
	}

	// Extract first 10 lines
	firstN := 10
	if totalLines < firstN {
		firstN = totalLines
	}
	firstLines := changedLines[:firstN]

	// Extract last 5 lines
	lastN := 5
	if totalLines < lastN+firstN {
		lastN = totalLines - firstN
	}
	lastLines := changedLines[totalLines-lastN:]

	// Extract smart middle (code-dense sections)
	middleStart := firstN
	middleEnd := totalLines - lastN
	middleLines := selectCodeDenseLines(changedLines[middleStart:middleEnd], tokenBudget-estimateTokens(firstLines)-estimateTokens(lastLines))

	return &DiffExcerpt{
		FirstLines:  firstLines,
		LastLines:   lastLines,
		MiddleLines: middleLines,
		TotalLines:  totalLines,
		TokenBudget: estimateTokens(firstLines) + estimateTokens(lastLines) + estimateTokens(middleLines),
	}
}

// selectCodeDenseLines selects lines with high code density (non-whitespace, non-comments)
func selectCodeDenseLines(lines []string, maxTokens int) []string {
	if len(lines) == 0 {
		return []string{}
	}

	// Score each line by code density
	type scoredLine struct {
		line  string
		score int
	}

	var scored []scoredLine
	for _, line := range lines {
		score := calculateCodeDensity(line)
		scored = append(scored, scoredLine{line, score})
	}

	// Sort by score (descending)
	// Simple bubble sort for small arrays
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Take top lines until token budget
	var selected []string
	currentTokens := 0
	for _, s := range scored {
		lineTokens := len(s.line) / 4 // Rough estimate: 4 chars = 1 token
		if currentTokens+lineTokens > maxTokens {
			break
		}
		selected = append(selected, s.line)
		currentTokens += lineTokens
	}

	return selected
}

// calculateCodeDensity scores a line by code content (higher = more code)
func calculateCodeDensity(line string) int {
	trimmed := strings.TrimSpace(line)

	// Remove +/- prefix
	if strings.HasPrefix(trimmed, "+") || strings.HasPrefix(trimmed, "-") {
		trimmed = strings.TrimSpace(trimmed[1:])
	}

	// Skip empty lines
	if trimmed == "" {
		return 0
	}

	// Skip comment-only lines
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") {
		return 0
	}

	// Score based on code indicators
	score := 0
	score += strings.Count(trimmed, "(") * 2  // Function calls
	score += strings.Count(trimmed, "{") * 2  // Code blocks
	score += strings.Count(trimmed, "=") * 1  // Assignments
	score += strings.Count(trimmed, ".") * 1  // Method calls
	score += len(strings.Fields(trimmed))     // Token count

	return score
}

// estimateTokens estimates token count for lines
func estimateTokens(lines []string) int {
	totalChars := 0
	for _, line := range lines {
		totalChars += len(line)
	}
	return totalChars / 4 // Rough estimate: 4 chars = 1 token
}

// FormatExcerpt formats an excerpt for LLM prompt
func (e *DiffExcerpt) FormatExcerpt() string {
	var b strings.Builder

	b.WriteString("=== First 10 lines ===\n")
	for _, line := range e.FirstLines {
		b.WriteString(line + "\n")
	}

	if len(e.MiddleLines) > 0 {
		b.WriteString("\n... [truncated] ...\n\n")
		b.WriteString("=== Key middle section ===\n")
		for _, line := range e.MiddleLines {
			b.WriteString(line + "\n")
		}
	}

	if len(e.LastLines) > 0 {
		b.WriteString("\n... [truncated] ...\n\n")
		b.WriteString("=== Last 5 lines ===\n")
		for _, line := range e.LastLines {
			b.WriteString(line + "\n")
		}
	}

	b.WriteString(fmt.Sprintf("\n(Total: %d lines, showing %d lines)\n",
		e.TotalLines, len(e.FirstLines)+len(e.MiddleLines)+len(e.LastLines)))

	return b.String()
}

// ExtractChunksForNewFile splits ADD files by function boundaries
func ExtractChunksForNewFile(fileContent, language string, maxChunks int) []DiffChunk {
	lines := strings.Split(fileContent, "\n")

	// Get language-specific pattern
	pattern, ok := functionPatterns[strings.ToLower(language)]
	if !ok {
		// Fallback to line-based splitting
		log.Infof("Using line-based chunking for language: %s", language)
		return splitByLines(lines, defaultLinesPerChunk, maxChunks)
	}

	// Find function boundaries
	functionStarts := []int{0} // Start with beginning of file

	for i, line := range lines {
		// Skip leading whitespace for matching (but preserve in output)
		trimmed := strings.TrimLeft(line, " \t")

		// Match top-level functions only (no leading whitespace)
		if line == trimmed && pattern.MatchString(trimmed) {
			functionStarts = append(functionStarts, i)
		}
	}

	// Add end of file marker
	functionStarts = append(functionStarts, len(lines))

	// Group lines into function blocks
	var chunks []DiffChunk
	currentChunk := DiffChunk{Lines: []string{}}
	currentSize := 0

	for i := 0; i < len(functionStarts)-1; i++ {
		start := functionStarts[i]
		end := functionStarts[i+1]

		functionLines := lines[start:end]
		functionSize := 0
		for _, line := range functionLines {
			functionSize += len(line)
		}

		// If adding this function exceeds chunk size, flush current chunk
		if currentSize+functionSize > maxChunkSize && len(currentChunk.Lines) > 0 {
			currentChunk.EndLine = start - 1
			chunks = append(chunks, currentChunk)
			currentChunk = DiffChunk{Lines: []string{}, StartLine: start}
			currentSize = 0
		}

		// If function itself exceeds max chunk size, split it
		if functionSize > maxChunkSize {
			// Split large function by lines
			subChunks := splitByLines(functionLines, defaultLinesPerChunk, -1)
			for _, subChunk := range subChunks {
				subChunk.StartLine += start
				subChunk.EndLine += start
				chunks = append(chunks, subChunk)
			}
			continue
		}

		// Add function to current chunk
		currentChunk.Lines = append(currentChunk.Lines, functionLines...)
		currentSize += functionSize

		// If we haven't set start line yet, set it now
		if currentChunk.StartLine == 0 {
			currentChunk.StartLine = start
		}
	}

	// Flush last chunk
	if len(currentChunk.Lines) > 0 {
		currentChunk.EndLine = len(lines) - 1
		chunks = append(chunks, currentChunk)
	}

	// Enforce max chunks budget
	if maxChunks > 0 && len(chunks) > maxChunks {
		log.Warnf("File truncated: %d chunks → %d (max budget)", len(chunks), maxChunks)
		chunks = chunks[:maxChunks]
	}

	// Add synthesized @@ headers for LLM context
	for i := range chunks {
		chunks[i].FileHeader = fmt.Sprintf("@@ -%d,%d +%d,%d @@",
			chunks[i].StartLine, len(chunks[i].Lines),
			chunks[i].StartLine, len(chunks[i].Lines))
	}

	return chunks
}

// splitByLines is fallback chunking for unknown languages
func splitByLines(lines []string, linesPerChunk int, maxChunks int) []DiffChunk {
	var chunks []DiffChunk

	for i := 0; i < len(lines); i += linesPerChunk {
		end := i + linesPerChunk
		if end > len(lines) {
			end = len(lines)
		}

		chunk := DiffChunk{
			Lines:     lines[i:end],
			StartLine: i,
			EndLine:   end - 1,
		}
		chunks = append(chunks, chunk)

		// Check max chunks limit
		if maxChunks > 0 && len(chunks) >= maxChunks {
			break
		}
	}

	return chunks
}
