package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/rohankatakam/coderisk/internal/database"
)

// LogFormatter formats block history output
type LogFormatter struct {
	oneline bool
	verbose bool
}

// NewLogFormatter creates a new log formatter
func NewLogFormatter(oneline, verbose bool) *LogFormatter {
	return &LogFormatter{
		oneline: oneline,
		verbose: verbose,
	}
}

// FormatBlockHistory formats the history of a code block
func (f *LogFormatter) FormatBlockHistory(w io.Writer, block *database.BlockWithHistory, events []database.BlockChangeEvent) error {
	if f.oneline {
		return f.formatOneline(w, events)
	}
	return f.formatStandard(w, block, events)
}

func (f *LogFormatter) formatStandard(w io.Writer, block *database.BlockWithHistory, events []database.BlockChangeEvent) error {
	// Header
	fmt.Fprintf(w, "Block: %s (%s)\n", block.BlockName, block.BlockType)
	fmt.Fprintf(w, "File: %s\n", block.CanonicalFilePath)

	// Rename chain if exists
	if len(block.RenameChain) > 0 {
		chain := strings.Join(block.RenameChain, " ← ")
		fmt.Fprintf(w, "Identity: Tracked across %d renames (%s)\n", len(block.RenameChain), chain)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "─────────────────────────────────────────────────────────────────")

	// Events
	for _, event := range events {
		f.formatEvent(w, event)
	}

	return nil
}

func (f *LogFormatter) formatEvent(w io.Writer, event database.BlockChangeEvent) {
	// Commit header
	shortSHA := event.CommitSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	fmt.Fprintf(w, "[%s] %s (%s <%s>)\n",
		event.CommitDate.Format("2006-01-02"),
		shortSHA,
		event.AuthorName,
		event.AuthorEmail)

	// Incident annotation if present
	if event.IssueNumber != nil {
		severity := "INCIDENT"
		if event.IssueSeverity != nil {
			severity = *event.IssueSeverity
		}
		title := ""
		if event.IssueTitle != nil {
			title = *event.IssueTitle
		}
		fmt.Fprintf(w, "   ⚠️  Fixed %s #%d \"%s\"\n", severity, *event.IssueNumber, title)
	}

	// Change type and stats
	changeType := event.ChangeType
	if event.ChangeType == "RENAMED" && event.OldBlockName != "" {
		changeType = fmt.Sprintf("RENAMED: %s → current name", event.OldBlockName)
	}

	fmt.Fprintf(w, "   %s", changeType)
	if event.LinesAdded > 0 || event.LinesDeleted > 0 {
		fmt.Fprintf(w, " | +%d -%d lines", event.LinesAdded, event.LinesDeleted)
	}
	if event.ComplexityDelta != 0 {
		sign := "+"
		if event.ComplexityDelta < 0 {
			sign = ""
		}
		fmt.Fprintf(w, " | Complexity: %s%d", sign, event.ComplexityDelta)
	}
	fmt.Fprintln(w)

	// Commit message (first line only unless verbose)
	if f.verbose {
		fmt.Fprintf(w, "\n%s\n", indent(event.CommitMessage, "   "))
	} else {
		firstLine := strings.Split(event.CommitMessage, "\n")[0]
		fmt.Fprintf(w, "\n   %s\n", firstLine)
	}

	fmt.Fprintln(w)
}

func (f *LogFormatter) formatOneline(w io.Writer, events []database.BlockChangeEvent) error {
	for _, event := range events {
		shortSHA := event.CommitSHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		date := event.CommitDate.Format("2006-01-02")
		author := event.AuthorName
		if len(author) > 15 {
			author = author[:12] + "..."
		}

		changeType := event.ChangeType
		if event.ChangeType == "RENAMED" {
			changeType = fmt.Sprintf("RENAMED %s", event.OldBlockName)
		}

		incident := ""
		if event.IssueNumber != nil {
			incident = fmt.Sprintf("⚠️ #%d", *event.IssueNumber)
		}

		stats := fmt.Sprintf("+%d -%d", event.LinesAdded, event.LinesDeleted)

		fmt.Fprintf(w, "%s  %s  %-15s  %-15s  %-8s  %s\n",
			shortSHA, date, author, changeType, incident, stats)
	}
	return nil
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
