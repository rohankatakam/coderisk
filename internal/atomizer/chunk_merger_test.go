package atomizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeChunkResults(t *testing.T) {
	// Test 1: Duplicate CREATE events
	t.Run("Duplicate CREATE events", func(t *testing.T) {
		events := []ChangeEvent{
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
		}
		result := MergeChunkResults(events)
		assert.Len(t, result, 1, "Should deduplicate to 1 event")
		assert.Equal(t, "CREATE_BLOCK", result[0].Behavior)
	})

	// Test 2: Conflicting behaviors (CREATE + MODIFY)
	t.Run("Conflicting behaviors - MODIFY takes priority", func(t *testing.T) {
		events := []ChangeEvent{
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
			{Behavior: "MODIFY_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
		}
		result := MergeChunkResults(events)
		assert.Len(t, result, 1)
		assert.Equal(t, "MODIFY_BLOCK", result[0].Behavior, "Should keep MODIFY (higher priority)")
	})

	// Test 3: Different signatures (should NOT merge)
	t.Run("Different signatures should NOT merge", func(t *testing.T) {
		events := []ChangeEvent{
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:string)"},
		}
		result := MergeChunkResults(events)
		assert.Len(t, result, 2, "Different signatures should NOT merge")
	})

	// Test 4: RENAME takes highest priority
	t.Run("RENAME takes highest priority", func(t *testing.T) {
		events := []ChangeEvent{
			{Behavior: "MODIFY_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
			{Behavior: "RENAME_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
		}
		result := MergeChunkResults(events)
		assert.Len(t, result, 1)
		assert.Equal(t, "RENAME_BLOCK", result[0].Behavior)
	})

	// Test 5: Merge snippets from multiple chunks
	t.Run("Merge snippets from multiple chunks", func(t *testing.T) {
		events := []ChangeEvent{
			{
				Behavior:        "MODIFY_BLOCK",
				TargetFile:      "a.go",
				TargetBlockName: "foo",
				Signature:       "(x:int)",
				OldVersion:      "old chunk 1",
				NewVersion:      "new chunk 1",
			},
			{
				Behavior:        "MODIFY_BLOCK",
				TargetFile:      "a.go",
				TargetBlockName: "foo",
				Signature:       "(x:int)",
				OldVersion:      "old chunk 2",
				NewVersion:      "new chunk 2",
			},
		}
		result := MergeChunkResults(events)
		assert.Len(t, result, 1)
		assert.Contains(t, result[0].OldVersion, "old chunk 1")
		assert.Contains(t, result[0].OldVersion, "old chunk 2")
		assert.Contains(t, result[0].OldVersion, "[Chunk boundary]")
		assert.Contains(t, result[0].NewVersion, "new chunk 1")
		assert.Contains(t, result[0].NewVersion, "new chunk 2")
	})

	// Test 6: Different files should NOT merge
	t.Run("Different files should NOT merge", func(t *testing.T) {
		events := []ChangeEvent{
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
			{Behavior: "CREATE_BLOCK", TargetFile: "b.go", TargetBlockName: "foo", Signature: "(x:int)"},
		}
		result := MergeChunkResults(events)
		assert.Len(t, result, 2)
	})

	// Test 7: Different block names should NOT merge
	t.Run("Different block names should NOT merge", func(t *testing.T) {
		events := []ChangeEvent{
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "foo", Signature: "(x:int)"},
			{Behavior: "CREATE_BLOCK", TargetFile: "a.go", TargetBlockName: "bar", Signature: "(x:int)"},
		}
		result := MergeChunkResults(events)
		assert.Len(t, result, 2)
	})

	// Test 8: Empty events list
	t.Run("Empty events list", func(t *testing.T) {
		events := []ChangeEvent{}
		result := MergeChunkResults(events)
		assert.Len(t, result, 0)
	})
}
