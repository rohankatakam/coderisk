package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/genai"
)

// HistoryManager intelligently prunes conversation history to stay within token budgets
// while preserving the most valuable context for LLM reasoning
type HistoryManager struct {
	maxTokens    int // Maximum tokens to allocate for history
	recentWindow int // Number of recent hops to always keep in full
	logger       *slog.Logger
}

// NewHistoryManager creates a history manager with default settings
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		maxTokens:    2000, // Leave room for tool definitions (~2K) + system + response
		recentWindow: 2,    // Always keep last 2 hops (4 items) in full
		logger:       slog.Default().With("component", "history_manager"),
	}
}

// PruneHistory intelligently reduces history while preserving valuable context
func (hm *HistoryManager) PruneHistory(history []*genai.Content) []*genai.Content {
	if len(history) == 0 {
		return history
	}

	// Calculate current token count
	currentTokens := hm.estimateTokens(history)

	// If within budget, no pruning needed
	if currentTokens <= hm.maxTokens {
		hm.logger.Debug("history within budget, no pruning needed",
			"items", len(history),
			"tokens", currentTokens,
			"max_tokens", hm.maxTokens)
		return history
	}

	hm.logger.Info("pruning history to fit token budget",
		"items_before", len(history),
		"tokens_before", currentTokens,
		"max_tokens", hm.maxTokens)

	// Strategy: Keep recent window + high-value older items, compress low-value items

	// Calculate recent window boundary (always keep last N hops = 2N items)
	recentBoundary := len(history) - (hm.recentWindow * 2)
	if recentBoundary < 0 {
		recentBoundary = 0
	}

	// Split history into: old items (candidates for pruning) + recent items (always keep)
	oldItems := history[:recentBoundary]
	recentItems := history[recentBoundary:]

	// Score and rank old items by importance
	scoredItems := hm.scoreHistoryItems(oldItems)

	// Build pruned history: keep high-value items, compress low-value items
	prunedOld := []*genai.Content{}
	tokensUsed := hm.estimateTokens(recentItems) // Start with recent items token count

	for _, scored := range scoredItems {
		itemTokens := hm.estimateTokens([]*genai.Content{scored.item})

		// High-value items: keep in full if budget allows
		if scored.score >= 0.7 && tokensUsed+itemTokens <= hm.maxTokens {
			prunedOld = append(prunedOld, scored.item)
			tokensUsed += itemTokens
			hm.logger.Debug("keeping high-value item",
				"score", scored.score,
				"tool", scored.toolName,
				"tokens", itemTokens)
		} else if scored.score >= 0.4 {
			// Medium-value items: try to compress
			compressed := hm.compressHistoryItem(scored.item, scored.toolName)
			compressedTokens := hm.estimateTokens([]*genai.Content{compressed})

			if tokensUsed+compressedTokens <= hm.maxTokens {
				prunedOld = append(prunedOld, compressed)
				tokensUsed += compressedTokens
				hm.logger.Debug("compressed medium-value item",
					"score", scored.score,
					"tool", scored.toolName,
					"tokens_before", itemTokens,
					"tokens_after", compressedTokens)
			} else {
				hm.logger.Debug("skipping medium-value item (budget exceeded)",
					"score", scored.score,
					"tool", scored.toolName)
			}
		} else {
			// Low-value items: skip entirely
			hm.logger.Debug("skipping low-value item",
				"score", scored.score,
				"tool", scored.toolName)
		}
	}

	// Combine: pruned old items + recent items
	result := append(prunedOld, recentItems...)
	finalTokens := hm.estimateTokens(result)

	hm.logger.Info("history pruning complete",
		"items_before", len(history),
		"items_after", len(result),
		"tokens_before", currentTokens,
		"tokens_after", finalTokens,
		"items_kept", len(prunedOld),
		"items_recent", len(recentItems))

	return result
}

// scoredHistoryItem pairs a history item with its importance score
type scoredHistoryItem struct {
	item     *genai.Content
	score    float64
	toolName string
	position int
}

// scoreHistoryItems assigns importance scores to history items and sorts by score (descending)
func (hm *HistoryManager) scoreHistoryItems(items []*genai.Content) []scoredHistoryItem {
	scored := make([]scoredHistoryItem, 0, len(items))

	for i, item := range items {
		toolName := hm.extractToolName(item)
		score := hm.scoreHistoryItem(item, toolName, i, len(items))
		scored = append(scored, scoredHistoryItem{
			item:     item,
			score:    score,
			toolName: toolName,
			position: i,
		})
	}

	// Sort by score descending (highest value first)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	return scored
}

// scoreHistoryItem assigns an importance score (0.0-1.0) to a history item
func (hm *HistoryManager) scoreHistoryItem(item *genai.Content, toolName string, position int, totalItems int) float64 {
	// Factor 1: Tool result value (semantic importance)
	toolScore := hm.getToolValueScore(toolName)

	// Factor 2: Recency bias (more recent = more important)
	// Position 0 = oldest, position N-1 = newest (but already in recent window)
	recencyScore := float64(position) / float64(totalItems)

	// Factor 3: Data density (more content = potentially more valuable)
	densityScore := hm.getDataDensityScore(item)

	// Weighted combination: tool value (50%) + recency (30%) + density (20%)
	finalScore := (toolScore * 0.5) + (recencyScore * 0.3) + (densityScore * 0.2)

	return finalScore
}

// getToolValueScore returns importance score based on tool type
func (hm *HistoryManager) getToolValueScore(toolName string) float64 {
	switch toolName {
	// HIGH VALUE: Critical evidence that drives risk assessment
	case "get_incidents_with_context":
		return 1.0 // Incident history is most critical
	case "finish_investigation":
		return 1.0 // Final assessment reasoning

	// MEDIUM-HIGH VALUE: Important patterns and ownership
	case "get_ownership_timeline":
		return 0.8 // Ownership and staleness patterns
	case "get_cochange_with_explanations":
		return 0.8 // Co-change patterns with commit context
	case "get_blast_radius_analysis":
		return 0.7 // Downstream impact analysis

	// MEDIUM VALUE: Supporting evidence
	case "query_cochange_partners":
		return 0.6 // Basic co-change data (without explanations)
	case "query_recent_commits":
		return 0.5 // Recent activity (can be summarized)

	// LOW VALUE: Basic queries that can be easily re-run or summarized
	case "query_ownership":
		return 0.4 // Simple ownership list
	case "query_blast_radius":
		return 0.4 // Simple dependency list

	default:
		return 0.5 // Unknown tools get medium score
	}
}

// getDataDensityScore estimates value based on content size
func (hm *HistoryManager) getDataDensityScore(item *genai.Content) float64 {
	// Count total characters in all parts
	totalChars := 0
	for _, part := range item.Parts {
		if part.Text != "" {
			totalChars += len(part.Text)
		}
		if part.FunctionResponse != nil {
			// Estimate function response size
			totalChars += 500 // Average response size
		}
	}

	// Score based on size buckets
	if totalChars > 2000 {
		return 1.0 // Large result = high density
	} else if totalChars > 1000 {
		return 0.7 // Medium result
	} else if totalChars > 500 {
		return 0.5 // Small result
	} else {
		return 0.3 // Very small result
	}
}

// extractToolName identifies which tool was called in this history item
func (hm *HistoryManager) extractToolName(item *genai.Content) string {
	if item == nil || len(item.Parts) == 0 {
		return ""
	}

	// Check for function responses (user messages with tool results)
	for _, part := range item.Parts {
		if part.FunctionResponse != nil {
			return part.FunctionResponse.Name
		}
	}

	// Check for function calls (assistant messages requesting tools)
	for _, part := range item.Parts {
		if part.FunctionCall != nil {
			return part.FunctionCall.Name
		}
	}

	return ""
}

// compressHistoryItem reduces the size of a history item by summarizing tool results
func (hm *HistoryManager) compressHistoryItem(item *genai.Content, toolName string) *genai.Content {
	if item == nil {
		return item
	}

	compressed := &genai.Content{
		Role:  item.Role,
		Parts: make([]*genai.Part, 0, len(item.Parts)),
	}

	for _, part := range item.Parts {
		// Compress function responses
		if part.FunctionResponse != nil {
			summary := hm.summarizeToolResult(part.FunctionResponse, toolName)
			compressed.Parts = append(compressed.Parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     part.FunctionResponse.Name,
					Response: map[string]any{"result": summary},
				},
			})
		} else if part.Text != "" {
			// Keep text but truncate if very long
			text := part.Text
			if len(text) > 500 {
				text = text[:500] + "... [truncated for context efficiency]"
			}
			compressed.Parts = append(compressed.Parts, &genai.Part{Text: text})
		} else {
			// Keep other parts as-is (function calls, etc.)
			compressed.Parts = append(compressed.Parts, part)
		}
	}

	return compressed
}

// summarizeToolResult creates a concise summary of tool results
func (hm *HistoryManager) summarizeToolResult(funcResp *genai.FunctionResponse, toolName string) string {
	if funcResp == nil || funcResp.Response == nil {
		return "No result"
	}

	// Extract the result field
	resultRaw, ok := funcResp.Response["result"]
	if !ok {
		return "Empty result"
	}

	// Try to parse as JSON to extract key stats
	resultStr := fmt.Sprintf("%v", resultRaw)

	// Attempt to parse structured data and summarize
	var data interface{}
	if err := json.Unmarshal([]byte(resultStr), &data); err == nil {
		summary := hm.summarizeStructuredData(data, toolName)
		if summary != "" {
			return summary
		}
	}

	// Fallback: truncate the raw result
	if len(resultStr) > 200 {
		return resultStr[:200] + fmt.Sprintf("... [%d more chars, compressed for context efficiency]", len(resultStr)-200)
	}

	return resultStr
}

// summarizeStructuredData creates intelligent summaries of JSON data
func (hm *HistoryManager) summarizeStructuredData(data interface{}, toolName string) string {
	switch v := data.(type) {
	case []interface{}:
		// Array of results
		count := len(v)
		if count == 0 {
			return fmt.Sprintf("%s: No results found", toolName)
		}

		// Try to extract meaningful summary from first item
		if count > 0 {
			switch toolName {
			case "query_recent_commits":
				return fmt.Sprintf("Found %d recent commit(s)", count)
			case "query_ownership":
				return fmt.Sprintf("Found %d developer(s)", count)
			case "query_cochange_partners":
				return fmt.Sprintf("Found %d co-change partner(s)", count)
			case "get_incidents_with_context":
				return fmt.Sprintf("Found %d incident(s) [HIGH VALUE: details preserved in compression]", count)
			default:
				return fmt.Sprintf("Found %d result(s)", count)
			}
		}
	case map[string]interface{}:
		// Single object result
		return fmt.Sprintf("%s: 1 result (details compressed)", toolName)
	}

	return ""
}

// estimateTokens approximates token count for history items
// Uses rough heuristic: 1 token ≈ 4 characters
func (hm *HistoryManager) estimateTokens(history []*genai.Content) int {
	totalChars := 0

	for _, item := range history {
		for _, part := range item.Parts {
			// Count text content
			if part.Text != "" {
				totalChars += len(part.Text)
			}

			// Count function calls
			if part.FunctionCall != nil {
				// Name + args JSON
				totalChars += len(part.FunctionCall.Name) + 100 // Estimate args size
			}

			// Count function responses
			if part.FunctionResponse != nil {
				// Name + response JSON
				totalChars += len(part.FunctionResponse.Name)
				if part.FunctionResponse.Response != nil {
					// Estimate response size by marshaling
					respJSON, _ := json.Marshal(part.FunctionResponse.Response)
					totalChars += len(respJSON)
				}
			}
		}
	}

	// Convert chars to tokens (rough approximation: 4 chars = 1 token)
	tokens := totalChars / 4

	// Add overhead for structure (roles, metadata, etc.)
	tokens += len(history) * 10 // 10 tokens per message overhead

	return tokens
}
