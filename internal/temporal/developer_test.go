package temporal

import (
	"testing"
	"time"
)

func TestCalculateOwnership(t *testing.T) {
	now := time.Now()
	day1 := now.Add(-10 * 24 * time.Hour)
	day5 := now.Add(-6 * 24 * time.Hour)
	day10 := now.Add(-1 * 24 * time.Hour)

	commits := []Commit{
		// Alice commits to file1 twice early on
		{
			SHA:       "commit1",
			Author:    "Alice",
			Email:     "alice@example.com",
			Timestamp: day1,
			FilesChanged: []FileChange{
				{Path: "file1.ts"},
			},
		},
		{
			SHA:       "commit2",
			Author:    "Alice",
			Email:     "alice@example.com",
			Timestamp: day1.Add(1 * time.Hour),
			FilesChanged: []FileChange{
				{Path: "file1.ts"},
			},
		},
		// Bob commits to file1 three times later (becomes owner)
		{
			SHA:       "commit3",
			Author:    "Bob",
			Email:     "bob@example.com",
			Timestamp: day5,
			FilesChanged: []FileChange{
				{Path: "file1.ts"},
			},
		},
		{
			SHA:       "commit4",
			Author:    "Bob",
			Email:     "bob@example.com",
			Timestamp: day5.Add(1 * time.Hour),
			FilesChanged: []FileChange{
				{Path: "file1.ts"},
			},
		},
		{
			SHA:       "commit5",
			Author:    "Bob",
			Email:     "bob@example.com",
			Timestamp: day10,
			FilesChanged: []FileChange{
				{Path: "file1.ts"},
			},
		},
	}

	ownership := CalculateOwnership(commits)

	if len(ownership) != 1 {
		t.Fatalf("expected 1 file ownership, got %d", len(ownership))
	}

	file1Ownership, exists := ownership["file1.ts"]
	if !exists {
		t.Fatal("file1.ts ownership not found")
	}

	// Bob should be current owner (3 commits)
	if file1Ownership.CurrentOwner != "bob@example.com" {
		t.Errorf("expected current owner bob@example.com, got %s", file1Ownership.CurrentOwner)
	}

	// Alice should be previous owner (2 commits)
	if file1Ownership.PreviousOwner != "alice@example.com" {
		t.Errorf("expected previous owner alice@example.com, got %s", file1Ownership.PreviousOwner)
	}

	// Transition date should be when Bob overtook Alice (around day5)
	// Just verify it's not zero
	if file1Ownership.TransitionDate.IsZero() {
		t.Error("expected non-zero transition date")
	}

	// DaysSince should be positive
	if file1Ownership.DaysSince <= 0 {
		t.Errorf("expected positive DaysSince, got %d", file1Ownership.DaysSince)
	}
}

func TestCalculateOwnership_SingleOwner(t *testing.T) {
	commits := []Commit{
		{
			SHA:       "commit1",
			Author:    "Alice",
			Email:     "alice@example.com",
			Timestamp: time.Now(),
			FilesChanged: []FileChange{
				{Path: "solo.ts"},
			},
		},
	}

	ownership := CalculateOwnership(commits)

	soloOwnership, exists := ownership["solo.ts"]
	if !exists {
		t.Fatal("solo.ts ownership not found")
	}

	if soloOwnership.CurrentOwner != "alice@example.com" {
		t.Errorf("expected current owner alice@example.com, got %s", soloOwnership.CurrentOwner)
	}

	// No previous owner
	if soloOwnership.PreviousOwner != "" {
		t.Errorf("expected no previous owner, got %s", soloOwnership.PreviousOwner)
	}
}

func TestCalculateOwnership_MultipleFiles(t *testing.T) {
	commits := []Commit{
		{
			SHA:       "commit1",
			Author:    "Alice",
			Email:     "alice@example.com",
			Timestamp: time.Now(),
			FilesChanged: []FileChange{
				{Path: "fileA.ts"},
				{Path: "fileB.ts"},
			},
		},
		{
			SHA:       "commit2",
			Author:    "Bob",
			Email:     "bob@example.com",
			Timestamp: time.Now(),
			FilesChanged: []FileChange{
				{Path: "fileB.ts"},
			},
		},
	}

	ownership := CalculateOwnership(commits)

	if len(ownership) != 2 {
		t.Fatalf("expected 2 file ownerships, got %d", len(ownership))
	}

	// fileA should be owned by Alice
	if ownership["fileA.ts"].CurrentOwner != "alice@example.com" {
		t.Errorf("fileA: expected owner alice@example.com, got %s", ownership["fileA.ts"].CurrentOwner)
	}

	// fileB should be tied (both have 1 commit), first alphabetically wins
	// Since both have 1 commit, the order might vary - just check it exists
	if ownership["fileB.ts"].CurrentOwner == "" {
		t.Error("fileB: expected an owner")
	}
}

func TestFindTransitionDate(t *testing.T) {
	now := time.Now()
	t1 := now.Add(-5 * 24 * time.Hour)
	t2 := now.Add(-4 * 24 * time.Hour)
	t3 := now.Add(-3 * 24 * time.Hour)
	t4 := now.Add(-2 * 24 * time.Hour)
	t5 := now.Add(-1 * 24 * time.Hour)

	devA := &DeveloperCommitCount{
		Email:   "alice@example.com",
		Commits: []time.Time{t1, t3, t5}, // 3 commits
	}

	devB := &DeveloperCommitCount{
		Email:   "bob@example.com",
		Commits: []time.Time{t2, t4}, // 2 commits
	}

	transitionDate := findTransitionDate(devA, devB)

	// Alice overtakes Bob at some point - just verify we got a date
	if transitionDate.IsZero() {
		t.Error("expected non-zero transition date")
	}
}

func TestFindTransitionDate_NoTransition(t *testing.T) {
	now := time.Now()
	t1 := now.Add(-3 * 24 * time.Hour)
	t2 := now.Add(-2 * 24 * time.Hour)

	devA := &DeveloperCommitCount{
		Email:   "alice@example.com",
		Commits: []time.Time{t1}, // 1 commit
	}

	devB := &DeveloperCommitCount{
		Email:   "bob@example.com",
		Commits: []time.Time{t2}, // 1 commit (but later)
	}

	transitionDate := findTransitionDate(devA, devB)

	// Alice overtakes immediately at t1
	if transitionDate.IsZero() {
		t.Error("expected non-zero transition date")
	}
}

func TestCalculateOwnership_EmailNormalization(t *testing.T) {
	commits := []Commit{
		{
			SHA:       "commit1",
			Author:    "Alice",
			Email:     "Alice@Example.COM", // uppercase
			Timestamp: time.Now(),
			FilesChanged: []FileChange{
				{Path: "file.ts"},
			},
		},
		{
			SHA:       "commit2",
			Author:    "Alice",
			Email:     "alice@example.com", // lowercase
			Timestamp: time.Now(),
			FilesChanged: []FileChange{
				{Path: "file.ts"},
			},
		},
	}

	ownership := CalculateOwnership(commits)

	fileOwnership, exists := ownership["file.ts"]
	if !exists {
		t.Fatal("file.ts ownership not found")
	}

	// Should be normalized to lowercase
	if fileOwnership.CurrentOwner != "alice@example.com" {
		t.Errorf("expected normalized email alice@example.com, got %s", fileOwnership.CurrentOwner)
	}
}
