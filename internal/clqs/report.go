package clqs

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// FormatReport generates a human-readable text report
func FormatReport(report *CLQSReport, w io.Writer) error {
	// Header
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "  CLQS REPORT: %s\n", report.Repository.FullName)
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════\n\n")

	// Overall score
	fmt.Fprintf(w, "OVERALL SCORE\n")
	fmt.Fprintf(w, "  CLQS:                    %.1f / 100\n", report.Overall.CLQS)
	fmt.Fprintf(w, "  Grade:                   %s\n", report.Overall.Grade)
	fmt.Fprintf(w, "  Rank:                    %s\n", report.Overall.Rank)
	fmt.Fprintf(w, "  Confidence Multiplier:   %.2f\n", report.Overall.ConfidenceMultiplier)
	fmt.Fprintf(w, "\n")

	// Component breakdown
	fmt.Fprintf(w, "COMPONENT BREAKDOWN\n")
	fmt.Fprintf(w, "┌─────────────────────────┬────────┬────────┬──────────────┬──────────┐\n")
	fmt.Fprintf(w, "│ Component               │ Score  │ Weight │ Contribution │ Status   │\n")
	fmt.Fprintf(w, "├─────────────────────────┼────────┼────────┼──────────────┼──────────┤\n")

	for _, comp := range report.Components {
		fmt.Fprintf(w, "│ %-23s │ %5.1f  │  %4.0f%% │    %5.2f     │ %-8s │\n",
			comp.Name, comp.Score, comp.Weight*100, comp.Contribution, comp.Status)
	}

	fmt.Fprintf(w, "└─────────────────────────┴────────┴────────┴──────────────┴──────────┘\n\n")

	// Statistics
	fmt.Fprintf(w, "STATISTICS\n")
	fmt.Fprintf(w, "  Total Closed Issues:     %d\n", report.Statistics.TotalClosedIssues)
	fmt.Fprintf(w, "  Eligible Issues:         %d\n", report.Statistics.EligibleIssues)
	fmt.Fprintf(w, "  Linked Issues:           %d\n", report.Statistics.LinkedIssues)
	fmt.Fprintf(w, "  Total Links:             %d\n", report.Statistics.TotalLinks)
	fmt.Fprintf(w, "  Average Confidence:      %.3f\n", report.Statistics.AvgConfidence)
	fmt.Fprintf(w, "\n")

	// Component details (optional, can be verbose)
	if len(report.Components) > 0 {
		fmt.Fprintf(w, "COMPONENT DETAILS\n")
		for _, comp := range report.Components {
			fmt.Fprintf(w, "\n%s (Score: %.1f/100, Status: %s)\n", comp.Name, comp.Score, comp.Status)
			formatComponentDetails(w, comp)
		}
		fmt.Fprintf(w, "\n")
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintf(w, "RECOMMENDATIONS\n")
		for i, rec := range report.Recommendations {
			impact := strings.ToUpper(rec.ImpactLevel)
			fmt.Fprintf(w, "  %d. [%s] %s\n", i+1, impact, rec.Message)
			fmt.Fprintf(w, "     Action: %s\n", rec.Action)
			fmt.Fprintf(w, "\n")
		}
	}

	// Labeling opportunities
	if report.LabelingOpportunities.LowConfidenceCount > 0 {
		fmt.Fprintf(w, "LABELING OPPORTUNITIES\n")
		fmt.Fprintf(w, "  Low Confidence Links:    %d\n", report.LabelingOpportunities.LowConfidenceCount)
		fmt.Fprintf(w, "  Ambiguous Links:         %d\n", report.LabelingOpportunities.AmbiguousCount)
		fmt.Fprintf(w, "  Potential CLQS Gain:     +%.1f points\n", report.LabelingOpportunities.PotentialCLQSGain)
		fmt.Fprintf(w, "\n")
	}

	// Footer
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "Analyzed at: %s\n", report.Repository.AnalyzedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(w, "CLQS Version: v2.1\n")
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════\n")

	return nil
}

// formatComponentDetails formats detailed information for each component
func formatComponentDetails(w io.Writer, comp *ComponentBreakdown) {
	switch details := comp.Details.(type) {
	case LinkCoverageDetails:
		fmt.Fprintf(w, "  Coverage Rate:      %.1f%% (%d/%d eligible issues linked)\n",
			details.CoverageRate*100, details.LinkedIssues, details.EligibleIssues)
		fmt.Fprintf(w, "  Excluded Issues:    %d (not requiring code fixes)\n", details.ExcludedIssues)
		fmt.Fprintf(w, "  Unlinked Issues:    %d\n", details.UnlinkedIssues)

	case ConfidenceQualityDetails:
		fmt.Fprintf(w, "  Weighted Average:   %.3f\n", details.WeightedAverage)
		fmt.Fprintf(w, "  High (≥0.85):       %d links (%.1f%%, avg: %.3f)\n",
			details.HighConfidenceLinks.Count,
			details.HighConfidenceLinks.Percentage,
			details.HighConfidenceLinks.AvgConfidence)
		fmt.Fprintf(w, "  Medium (0.70-0.84): %d links (%.1f%%, avg: %.3f)\n",
			details.MediumConfidenceLinks.Count,
			details.MediumConfidenceLinks.Percentage,
			details.MediumConfidenceLinks.AvgConfidence)
		fmt.Fprintf(w, "  Low (<0.70):        %d links (%.1f%%, avg: %.3f)\n",
			details.LowConfidenceLinks.Count,
			details.LowConfidenceLinks.Percentage,
			details.LowConfidenceLinks.AvgConfidence)

	case EvidenceDiversityDetails:
		fmt.Fprintf(w, "  Avg Evidence Types: %.1f/6\n", details.AvgEvidenceTypes)
		fmt.Fprintf(w, "  Distribution:       6=%d, 5=%d, 4=%d, 3=%d, 2=%d, 1=%d\n",
			details.Distribution["6_types"],
			details.Distribution["5_types"],
			details.Distribution["4_types"],
			details.Distribution["3_types"],
			details.Distribution["2_types"],
			details.Distribution["1_type"])
		fmt.Fprintf(w, "  Type Usage:\n")
		fmt.Fprintf(w, "    Explicit:         %d links\n", details.EvidenceTypeUsage["explicit"])
		fmt.Fprintf(w, "    Timeline:         %d links\n", details.EvidenceTypeUsage["github_timeline_verified"])
		fmt.Fprintf(w, "    Bidirectional:    %d links\n", details.EvidenceTypeUsage["bidirectional"])
		fmt.Fprintf(w, "    Semantic:         %d links\n", details.EvidenceTypeUsage["semantic"])
		fmt.Fprintf(w, "    Temporal:         %d links\n", details.EvidenceTypeUsage["temporal"])
		fmt.Fprintf(w, "    File Context:     %d links\n", details.EvidenceTypeUsage["file_context"])

	case TemporalPrecisionDetails:
		fmt.Fprintf(w, "  Precision Rate:     %.1f%% (%d/%d links <1 hour)\n",
			details.PrecisionRate*100, details.TightTemporalLinks, details.TotalLinks)
		fmt.Fprintf(w, "  Distribution:\n")
		fmt.Fprintf(w, "    <5 min:           %d links\n", details.Distribution["under_5min"])
		fmt.Fprintf(w, "    5 min - 1 hr:     %d links\n", details.Distribution["5min_to_1hr"])
		fmt.Fprintf(w, "    1 hr - 24 hr:     %d links\n", details.Distribution["1hr_to_24hr"])
		fmt.Fprintf(w, "    >24 hr:           %d links\n", details.Distribution["over_24hr"])

	case SemanticStrengthDetails:
		fmt.Fprintf(w, "  Avg Semantic Score: %.3f\n", details.AvgSemanticScore)
		fmt.Fprintf(w, "  Distribution:\n")
		fmt.Fprintf(w, "    High (≥0.70):     %d links\n", details.Distribution["high_semantic"])
		fmt.Fprintf(w, "    Medium (0.50-0.69): %d links\n", details.Distribution["medium_semantic"])
		fmt.Fprintf(w, "    Low (<0.50):      %d links\n", details.Distribution["low_semantic"])
		fmt.Fprintf(w, "  Avg by Type:\n")
		fmt.Fprintf(w, "    Title:            %.3f\n", details.AvgByType["title"])
		fmt.Fprintf(w, "    Body:             %.3f\n", details.AvgByType["body"])
		fmt.Fprintf(w, "    Comment:          %.3f\n", details.AvgByType["comment"])
	}
}

// FormatJSON generates a JSON report
func FormatJSON(report *CLQSReport, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
