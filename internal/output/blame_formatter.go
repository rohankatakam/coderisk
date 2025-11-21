package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rohankatakam/coderisk/internal/database"
)

// BlameFormatter formats block ownership output
type BlameFormatter struct {
	format string // "table", "json", "csv"
}

// NewBlameFormatter creates a new blame formatter
func NewBlameFormatter(format string) *BlameFormatter {
	return &BlameFormatter{format: format}
}

// FormatFileBlame formats ownership data for a file
func (f *BlameFormatter) FormatFileBlame(w io.Writer, filePath string, blocks []database.BlockWithOwnership) error {
	switch f.format {
	case "json":
		return f.formatJSON(w, filePath, blocks)
	case "csv":
		return f.formatCSV(w, blocks)
	default:
		return f.formatTable(w, filePath, blocks)
	}
}

// FormatDirectoryBlame formats ownership statistics for a directory
func (f *BlameFormatter) FormatDirectoryBlame(w io.Writer, dirPath string, stats database.OwnershipStats) error {
	fmt.Fprintf(w, "Directory: %s (%d functions)\n\n", dirPath, stats.TotalBlocks)

	// Top risky blocks
	if len(stats.TopRiskyBlocks) > 0 {
		fmt.Fprintln(w, "Top 10 Riskiest Functions:")
		for i, block := range stats.TopRiskyBlocks {
			fmt.Fprintf(w, "  %d. %s::%s  (%.0f/100, %d incidents)\n",
				i+1, block.CanonicalFilePath, block.BlockName, block.RiskScore, block.IncidentCount)
		}
		fmt.Fprintln(w)
	}

	// Ownership health
	fmt.Fprintln(w, "Ownership Health:")
	fmt.Fprintf(w, "  🔴 Critical Risk (>70): %d functions\n", stats.CriticalRisk)
	fmt.Fprintf(w, "  🟡 Medium Risk (40-70): %d functions\n", stats.MediumRisk)
	fmt.Fprintf(w, "  🟢 Low Risk (<40): %d functions\n", stats.LowRisk)
	fmt.Fprintf(w, "  ⚠️  Stale Code (>90 days): %d functions\n", stats.StaleBlocks)
	fmt.Fprintf(w, "  ⚠️  Bus Factor Warnings: %d functions\n", stats.BusFactorWarnings)

	return nil
}

func (f *BlameFormatter) formatTable(w io.Writer, filePath string, blocks []database.BlockWithOwnership) error {
	if len(blocks) == 0 {
		fmt.Fprintf(w, "No functions found in %s\n", filePath)
		return nil
	}

	fmt.Fprintf(w, "File: %s (%d functions)\n\n", filePath, len(blocks))

	// Header
	fmt.Fprintf(w, "%-25s %-10s %-25s %10s %6s\n",
		"Function", "Type", "SME", "Staleness", "Risk")
	fmt.Fprintln(w, strings.Repeat("─", 80))

	// Rows
	for _, block := range blocks {
		sme, busFactor, _ := database.CalculateSME(block.FamiliarityMap)
		if sme == "" {
			sme = "<UNKNOWN>"
		} else {
			// Show bus factor warning with SME email
			if busFactor == "CRITICAL" {
				sme = sme + " ⚠️"
			} else if busFactor == "HIGH" {
				sme = sme + " ⚠"
			}
		}

		// Truncate long names
		name := block.BlockName
		if len(name) > 24 {
			name = name[:21] + "..."
		}
		if len(sme) > 24 {
			sme = sme[:21] + "..."
		}

		// Risk emoji
		riskEmoji := "🟢"
		if block.RiskScore > 70 {
			riskEmoji = "🔴"
		} else if block.RiskScore >= 40 {
			riskEmoji = "🟡"
		}

		staleness := fmt.Sprintf("%dd %s", block.StalenessDays, riskEmoji)

		fmt.Fprintf(w, "%-25s %-10s %-25s %10s %5.0f/100\n",
			name, block.BlockType, sme, staleness, block.RiskScore)
	}

	// Summary
	critical, medium, low, stale, busFactorWarnings := 0, 0, 0, 0, 0
	for _, block := range blocks {
		if block.RiskScore > 70 {
			critical++
		} else if block.RiskScore >= 40 {
			medium++
		} else {
			low++
		}
		if block.StalenessDays > 90 {
			stale++
		}
		_, busFactor, _ := database.CalculateSME(block.FamiliarityMap)
		if busFactor == "CRITICAL" || busFactor == "HIGH" {
			busFactorWarnings++
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Summary:")
	fmt.Fprintf(w, "  🔴 %d CRITICAL (risk >70 OR staleness >90d)\n", critical)
	fmt.Fprintf(w, "  🟡 %d MEDIUM\n", medium)
	fmt.Fprintf(w, "  🟢 %d LOW\n", low)
	if busFactorWarnings > 0 {
		fmt.Fprintf(w, "\n  ⚠️  Bus Factor Warnings: %d functions\n", busFactorWarnings)
	}

	return nil
}

func (f *BlameFormatter) formatJSON(w io.Writer, filePath string, blocks []database.BlockWithOwnership) error {
	type OwnershipJSON struct {
		SME            string         `json:"sme"`
		FamiliarityMap map[string]int `json:"familiarity_map"`
		BusFactor      string         `json:"bus_factor"`
		StalenessDays  int            `json:"staleness_days"`
	}

	type RiskJSON struct {
		Score         float64 `json:"score"`
		Level         string  `json:"level"`
		IncidentCount int     `json:"incident_count"`
	}

	type BlockJSON struct {
		Name      string        `json:"name"`
		Type      string        `json:"type"`
		LineRange [2]int        `json:"line_range"`
		Ownership OwnershipJSON `json:"ownership"`
		Risk      RiskJSON      `json:"risk"`
	}

	type OutputJSON struct {
		File   string      `json:"file"`
		Blocks []BlockJSON `json:"blocks"`
	}

	output := OutputJSON{
		File:   filePath,
		Blocks: make([]BlockJSON, 0, len(blocks)),
	}

	for _, block := range blocks {
		sme, busFactor, _ := database.CalculateSME(block.FamiliarityMap)

		riskLevel := "LOW"
		if block.RiskScore > 70 {
			riskLevel = "CRITICAL"
		} else if block.RiskScore >= 40 {
			riskLevel = "MEDIUM"
		}

		blockJSON := BlockJSON{
			Name:      block.BlockName,
			Type:      block.BlockType,
			LineRange: [2]int{block.StartLine, block.EndLine},
			Ownership: OwnershipJSON{
				SME:            sme,
				FamiliarityMap: block.FamiliarityMap,
				BusFactor:      busFactor,
				StalenessDays:  block.StalenessDays,
			},
			Risk: RiskJSON{
				Score:         block.RiskScore,
				Level:         riskLevel,
				IncidentCount: block.IncidentCount,
			},
		}

		output.Blocks = append(output.Blocks, blockJSON)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *BlameFormatter) formatCSV(w io.Writer, blocks []database.BlockWithOwnership) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	if err := writer.Write([]string{
		"function", "type", "sme", "staleness_days", "risk_score", "incident_count", "bus_factor",
	}); err != nil {
		return err
	}

	// Rows
	for _, block := range blocks {
		sme, busFactor, _ := database.CalculateSME(block.FamiliarityMap)

		if err := writer.Write([]string{
			block.BlockName,
			block.BlockType,
			sme,
			fmt.Sprintf("%d", block.StalenessDays),
			fmt.Sprintf("%.2f", block.RiskScore),
			fmt.Sprintf("%d", block.IncidentCount),
			busFactor,
		}); err != nil {
			return err
		}
	}

	return nil
}
