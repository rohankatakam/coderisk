package atomizer

import (
	"log"
	"strings"
)

// eventKey is used to group events by file, block name, and signature
type eventKey struct {
	file      string
	blockName string
	signature string
}

// MergeChunkResults deduplicates events from multiple chunks
func MergeChunkResults(events []ChangeEvent) []ChangeEvent {
	// Group events by (target_file, block_name, signature)
	groups := make(map[eventKey][]ChangeEvent)

	for _, event := range events {
		key := eventKey{
			file:      event.TargetFile,
			blockName: event.TargetBlockName,
			signature: NormalizeSignature(event.Signature),
		}
		groups[key] = append(groups[key], event)
	}

	// Merge groups
	var merged []ChangeEvent

	for _, group := range groups {
		if len(group) == 1 {
			merged = append(merged, group[0])
			continue
		}

		// Multiple events for same block - merge them
		mergedEvent := mergeEventGroup(group)
		merged = append(merged, mergedEvent)
	}

	return merged
}

func mergeEventGroup(group []ChangeEvent) ChangeEvent {
	// Check for signature mismatches
	firstSig := group[0].Signature
	for _, event := range group[1:] {
		if event.Signature != firstSig {
			log.Printf("⚠️  WARNING: Signature mismatch detected during merge - file: %s, block_name: %s, signature_1: %s, signature_2: %s, action: using first signature",
				event.TargetFile,
				event.TargetBlockName,
				firstSig,
				event.Signature)
		}
	}

	// Determine priority behavior
	priority := map[string]int{
		"RENAME_BLOCK": 4,
		"MODIFY_BLOCK": 3,
		"CREATE_BLOCK": 2,
		"DELETE_BLOCK": 1,
	}

	highestPriority := group[0]
	for _, event := range group[1:] {
		if priority[event.Behavior] > priority[highestPriority.Behavior] {
			highestPriority = event
		}
	}

	// Merge snippets
	var oldVersions, newVersions []string
	for _, event := range group {
		if event.OldVersion != "" {
			oldVersions = append(oldVersions, event.OldVersion)
		}
		if event.NewVersion != "" {
			newVersions = append(newVersions, event.NewVersion)
		}
	}

	highestPriority.OldVersion = strings.Join(oldVersions, "\n\n// [Chunk boundary]\n\n")
	highestPriority.NewVersion = strings.Join(newVersions, "\n\n// [Chunk boundary]\n\n")

	return highestPriority
}
