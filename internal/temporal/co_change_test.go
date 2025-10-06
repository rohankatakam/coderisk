package temporal

import (
	"testing"
)

func TestCalculateCoChanges(t *testing.T) {
	commits := []Commit{
		{
			SHA: "commit1",
			FilesChanged: []FileChange{
				{Path: "a.ts"},
				{Path: "b.ts"},
			},
		},
		{
			SHA: "commit2",
			FilesChanged: []FileChange{
				{Path: "a.ts"},
				{Path: "b.ts"},
			},
		},
		{
			SHA: "commit3",
			FilesChanged: []FileChange{
				{Path: "a.ts"},
				{Path: "c.ts"},
			},
		},
		{
			SHA: "commit4",
			FilesChanged: []FileChange{
				{Path: "b.ts"},
			},
		},
	}

	// Calculate with 50% minimum frequency
	coChanges := CalculateCoChanges(commits, 0.5)

	// Expected:
	// a.ts <-> b.ts: co_changes=2, a.ts appears 3 times, b.ts appears 3 times, frequency=2/3=0.67 (above 0.5)
	// a.ts <-> c.ts: co_changes=1, a.ts appears 3 times, c.ts appears 1 time, frequency=1/3=0.33 (below 0.5)
	// Should return only a.ts <-> b.ts

	if len(coChanges) != 1 {
		t.Fatalf("expected 1 co-change pair, got %d", len(coChanges))
	}

	result := coChanges[0]
	if result.FileA != "a.ts" || result.FileB != "b.ts" {
		t.Errorf("expected pair a.ts <-> b.ts, got %s <-> %s", result.FileA, result.FileB)
	}
	if result.CoChanges != 2 {
		t.Errorf("expected 2 co-changes, got %d", result.CoChanges)
	}

	// Frequency should be 2/3 = 0.666...
	expectedFreq := 2.0 / 3.0
	if result.Frequency < expectedFreq-0.01 || result.Frequency > expectedFreq+0.01 {
		t.Errorf("expected frequency ~0.67, got %f", result.Frequency)
	}
}

func TestCalculateCoChanges_LowThreshold(t *testing.T) {
	commits := []Commit{
		{
			SHA: "commit1",
			FilesChanged: []FileChange{
				{Path: "x.go"},
				{Path: "y.go"},
			},
		},
		{
			SHA: "commit2",
			FilesChanged: []FileChange{
				{Path: "x.go"},
				{Path: "z.go"},
			},
		},
		{
			SHA: "commit3",
			FilesChanged: []FileChange{
				{Path: "y.go"},
			},
		},
	}

	// With 0.3 threshold (30%)
	coChanges := CalculateCoChanges(commits, 0.3)

	// x.go <-> y.go: co_changes=1, x appears 2 times, y appears 2 times, frequency=1/2=0.5 (above 0.3)
	// x.go <-> z.go: co_changes=1, x appears 2 times, z appears 1 time, frequency=1/2=0.5 (above 0.3)
	// Should return both pairs

	if len(coChanges) != 2 {
		t.Fatalf("expected 2 co-change pairs, got %d", len(coChanges))
	}
}

func TestCalculateCoChanges_SortedByFrequency(t *testing.T) {
	commits := []Commit{
		// a.go and b.go change together 3 times (75% frequency)
		{SHA: "1", FilesChanged: []FileChange{{Path: "a.go"}, {Path: "b.go"}}},
		{SHA: "2", FilesChanged: []FileChange{{Path: "a.go"}, {Path: "b.go"}}},
		{SHA: "3", FilesChanged: []FileChange{{Path: "a.go"}, {Path: "b.go"}}},
		{SHA: "4", FilesChanged: []FileChange{{Path: "a.go"}}},

		// x.go and y.go change together 1 time (50% frequency)
		{SHA: "5", FilesChanged: []FileChange{{Path: "x.go"}, {Path: "y.go"}}},
		{SHA: "6", FilesChanged: []FileChange{{Path: "x.go"}}},
	}

	coChanges := CalculateCoChanges(commits, 0.3)

	if len(coChanges) < 2 {
		t.Fatalf("expected at least 2 co-change pairs, got %d", len(coChanges))
	}

	// First result should have higher frequency than second
	if coChanges[0].Frequency < coChanges[1].Frequency {
		t.Errorf("results not sorted by frequency: %f < %f", coChanges[0].Frequency, coChanges[1].Frequency)
	}
}

func TestCalculateCoChanges_SingleFileCommit(t *testing.T) {
	commits := []Commit{
		{
			SHA: "commit1",
			FilesChanged: []FileChange{
				{Path: "single.ts"},
			},
		},
	}

	coChanges := CalculateCoChanges(commits, 0.3)

	// Single-file commits should not produce co-change pairs
	if len(coChanges) != 0 {
		t.Errorf("expected 0 co-change pairs for single-file commit, got %d", len(coChanges))
	}
}

func TestCalculateCoChanges_WindowDays(t *testing.T) {
	commits := []Commit{
		{
			SHA: "commit1",
			FilesChanged: []FileChange{
				{Path: "a.ts"},
				{Path: "b.ts"},
			},
		},
	}

	coChanges := CalculateCoChanges(commits, 0.3)

	if len(coChanges) != 1 {
		t.Fatalf("expected 1 co-change pair, got %d", len(coChanges))
	}

	// WindowDays should be set to 90
	if coChanges[0].WindowDays != 90 {
		t.Errorf("expected WindowDays=90, got %d", coChanges[0].WindowDays)
	}
}

func TestSplitPairKey(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a.ts|b.ts", []string{"a.ts", "b.ts"}},
		{"file1|file2", []string{"file1", "file2"}},
		{"path/to/a|path/to/b", []string{"path/to/a", "path/to/b"}},
	}

	for _, test := range tests {
		result := splitPairKey(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("splitPairKey(%s): expected %d parts, got %d", test.input, len(test.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != test.expected[i] {
				t.Errorf("splitPairKey(%s): part %d expected %s, got %s", test.input, i, test.expected[i], result[i])
			}
		}
	}
}
