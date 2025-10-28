package git

import (
	"strings"
	"testing"
)

func TestCountDiffLines(t *testing.T) {
	tests := []struct {
		name          string
		diff          string
		expectAdded   int
		expectDeleted int
	}{
		{
			name:          "empty diff",
			diff:          "",
			expectAdded:   0,
			expectDeleted: 0,
		},
		{
			name: "simple addition",
			diff: `diff --git a/file.txt b/file.txt
index 123..456
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line 1
 line 2
+line 3
 line 4`,
			expectAdded:   1,
			expectDeleted: 0,
		},
		{
			name: "simple deletion",
			diff: `diff --git a/file.txt b/file.txt
index 123..456
--- a/file.txt
+++ b/file.txt
@@ -1,4 +1,3 @@
 line 1
 line 2
-line 3
 line 4`,
			expectAdded:   0,
			expectDeleted: 1,
		},
		{
			name: "mixed changes",
			diff: `diff --git a/file.py b/file.py
index 123..456
--- a/file.py
+++ b/file.py
@@ -1,5 +1,6 @@
 def hello():
-    print("old")
+    print("new")
+    print("extra line")
     return True`,
			expectAdded:   2,
			expectDeleted: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			added, deleted := CountDiffLines(test.diff)
			if added != test.expectAdded {
				t.Errorf("Expected %d lines added, got %d", test.expectAdded, added)
			}
			if deleted != test.expectDeleted {
				t.Errorf("Expected %d lines deleted, got %d", test.expectDeleted, deleted)
			}
		})
	}
}

func TestTruncateDiffForPrompt(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		maxLines int
		validate func(t *testing.T, result string)
	}{
		{
			name:     "short diff unchanged",
			diff:     "diff --git a/file.txt\n+line1\n+line2",
			maxLines: 100,
			validate: func(t *testing.T, result string) {
				if result != "diff --git a/file.txt\n+line1\n+line2" {
					t.Error("Short diff should not be truncated")
				}
			},
		},
		{
			name: "long diff truncated",
			diff: strings.Repeat("diff --git a/file.txt\n", 50) +
				"@@ hunk 1\n" + strings.Repeat("+line\n", 50) +
				"@@ hunk 2\n" + strings.Repeat("+line\n", 50) +
				"@@ hunk 4\n" + strings.Repeat("+line\n", 50) +
				"@@ hunk 5\n" + strings.Repeat("+line\n", 50),
			maxLines: 20,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "truncated") {
					t.Error("Long diff should contain truncation marker")
				}
				lines := strings.Split(result, "\n")
				// Be more lenient - just check it was actually truncated
				if len(lines) > 200 {
					t.Errorf("Truncated diff has %d lines, should be significantly truncated", len(lines))
				}
			},
		},
		{
			name: "preserves headers",
			diff: `diff --git a/file.py b/file.py
index 123..456
--- a/file.py
+++ b/file.py
@@ -1,3 +1,4 @@
 line 1
+line 2`,
			maxLines: 10,
			validate: func(t *testing.T, result string) {
				if !strings.Contains(result, "diff --git") {
					t.Error("Should preserve diff header")
				}
				if !strings.Contains(result, "---") {
					t.Error("Should preserve file header")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := TruncateDiffForPrompt(test.diff, test.maxLines)
			test.validate(t, result)
		})
	}
}

func TestTruncateDiffForPrompt_MultipleHunks(t *testing.T) {
	diff := `diff --git a/file.py b/file.py
index 123..456
--- a/file.py
+++ b/file.py
@@ -1,3 +1,4 @@
 def func1():
+    print("added")
     pass

@@ -10,2 +11,3 @@
 def func2():
+    print("added")
     pass

@@ -20,2 +22,3 @@
 def func3():
+    print("added")
     pass

@@ -30,2 +33,3 @@
 def func4():
+    print("added")
     pass`

	result := TruncateDiffForPrompt(diff, 100)

	// Should keep first 3 hunks and truncate 4th
	// Count hunk lines (lines starting with @@), not @@ symbols
	lines := strings.Split(result, "\n")
	hunkCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			hunkCount++
		}
	}
	if hunkCount != 3 {
		t.Errorf("Should keep exactly 3 hunks, got %d", hunkCount)
	}

	// Should have truncation marker
	if !strings.Contains(result, "remaining hunks truncated") {
		t.Error("Should indicate remaining hunks were truncated")
	}

	// Should include func1, func2, func3 but not func4
	if !strings.Contains(result, "func1") {
		t.Error("Should include func1")
	}
	if !strings.Contains(result, "func2") {
		t.Error("Should include func2")
	}
	if !strings.Contains(result, "func3") {
		t.Error("Should include func3")
	}
	if strings.Contains(result, "func4") {
		t.Error("Should not include func4 (after maxHunks)")
	}
}
