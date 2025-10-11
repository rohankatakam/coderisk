package phase0

import (
	"testing"
)

func TestIsDocumentationOnly(t *testing.T) {
	tests := []struct {
		name                    string
		filePath                string
		expectDocumentationOnly bool
		expectSkipAnalysis      bool
		expectDocType           string
		description             string
	}{
		// TRUE POSITIVES - Documentation files that should skip analysis
		{
			name:                    "README.md",
			filePath:                "README.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "README.md should skip all analysis",
		},
		{
			name:                    "README in subdirectory",
			filePath:                "docs/README.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "README in subdirectory should skip analysis",
		},
		{
			name:                    "CHANGELOG.md",
			filePath:                "CHANGELOG.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "CHANGELOG should skip analysis",
		},
		{
			name:                    "CONTRIBUTING.md",
			filePath:                "CONTRIBUTING.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "CONTRIBUTING guide should skip analysis",
		},
		{
			name:                    "LICENSE file",
			filePath:                "LICENSE",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "documentation",
			description:             "LICENSE file should skip analysis",
		},
		{
			name:                    "Plain text documentation",
			filePath:                "docs/architecture.txt",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "txt",
			description:             "Plain text docs should skip analysis",
		},
		{
			name:                    "reStructuredText documentation",
			filePath:                "docs/api.rst",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "rst",
			description:             "reStructuredText docs should skip analysis",
		},
		{
			name:                    "AsciiDoc documentation",
			filePath:                "docs/guide.adoc",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "adoc",
			description:             "AsciiDoc docs should skip analysis",
		},
		{
			name:                    "Markdown variant",
			filePath:                "notes.markdown",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "markdown",
			description:             "Markdown variant extension should skip analysis",
		},
		{
			name:                    "CODE_OF_CONDUCT",
			filePath:                "CODE_OF_CONDUCT.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "Code of conduct should skip analysis",
		},
		{
			name:                    "SECURITY policy",
			filePath:                "SECURITY.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "Security policy should skip analysis",
		},
		{
			name:                    "AUTHORS file",
			filePath:                "AUTHORS",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "documentation",
			description:             "Authors file should skip analysis",
		},

		// TRUE NEGATIVES - Code files that should NOT skip analysis
		{
			name:                    "Go source file",
			filePath:                "internal/handlers/auth.go",
			expectDocumentationOnly: false,
			expectSkipAnalysis:      false,
			expectDocType:           "",
			description:             "Go source should not skip analysis",
		},
		{
			name:                    "Python source file",
			filePath:                "src/services/payment.py",
			expectDocumentationOnly: false,
			expectSkipAnalysis:      false,
			expectDocType:           "",
			description:             "Python source should not skip analysis",
		},
		{
			name:                    "JavaScript source file",
			filePath:                "src/app.js",
			expectDocumentationOnly: false,
			expectSkipAnalysis:      false,
			expectDocType:           "",
			description:             "JavaScript source should not skip analysis",
		},
		{
			name:                    "TypeScript source file",
			filePath:                "src/components/Button.tsx",
			expectDocumentationOnly: false,
			expectSkipAnalysis:      false,
			expectDocType:           "",
			description:             "TypeScript source should not skip analysis",
		},
		{
			name:                    "Configuration file",
			filePath:                "config/database.yaml",
			expectDocumentationOnly: false,
			expectSkipAnalysis:      false,
			expectDocType:           "",
			description:             "Config files should not skip analysis",
		},
		{
			name:                    "Test file",
			filePath:                "internal/handlers/auth_test.go",
			expectDocumentationOnly: false,
			expectSkipAnalysis:      false,
			expectDocType:           "",
			description:             "Test files should not skip analysis",
		},

		// EDGE CASES
		{
			name:                    "Uppercase README",
			filePath:                "README.MD",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "Case-insensitive extension matching",
		},
		{
			name:                    "Lowercase readme",
			filePath:                "readme.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "Lowercase readme should also skip",
		},
		{
			name:                    "Documentation in deep path",
			filePath:                "docs/guides/getting-started/installation.md",
			expectDocumentationOnly: true,
			expectSkipAnalysis:      true,
			expectDocType:           "md",
			description:             "Deep path documentation should skip",
		},
		{
			name:                    "File with .md in name but not extension",
			filePath:                "internal/markdown_parser.go",
			expectDocumentationOnly: false,
			expectSkipAnalysis:      false,
			expectDocType:           "",
			description:             "Extension check should be suffix-based",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDocumentationOnly(tt.filePath)

			if result.IsDocumentationOnly != tt.expectDocumentationOnly {
				t.Errorf("IsDocumentationOnly = %v, want %v\nDescription: %s",
					result.IsDocumentationOnly, tt.expectDocumentationOnly, tt.description)
			}

			if result.SkipAnalysis != tt.expectSkipAnalysis {
				t.Errorf("SkipAnalysis = %v, want %v\nDescription: %s",
					result.SkipAnalysis, tt.expectSkipAnalysis, tt.description)
			}

			if tt.expectDocType != "" && result.DocumentationType != tt.expectDocType {
				t.Errorf("DocumentationType = %v, want %v\nDescription: %s",
					result.DocumentationType, tt.expectDocType, tt.description)
			}

			if result.IsDocumentationOnly && result.Reason == "" {
				t.Error("Expected non-empty reason for documentation-only detection")
			}

			t.Logf("Result: IsDoc=%v, Skip=%v, Type=%s, Reason=%s",
				result.IsDocumentationOnly,
				result.SkipAnalysis,
				result.DocumentationType,
				result.Reason)
		})
	}
}

func TestIsCommentOnlyChange(t *testing.T) {
	tests := []struct {
		name               string
		diff               string
		expectCommentOnly  bool
		description        string
	}{
		// TRUE POSITIVES - Comment-only changes
		{
			name: "Go single-line comment addition",
			diff: `@@ -10,6 +10,7 @@ func Login(username, password string) error {
 	user, err := db.FindUser(username)
+	// Check if user exists
 	if err != nil {
 		return err
 	}`,
			expectCommentOnly: true,
			description:       "Adding Go comment should be comment-only",
		},
		{
			name: "Python comment addition",
			diff: `@@ -5,6 +5,7 @@ def process_payment(amount):
     """Process a payment transaction"""
+    # Validate amount is positive
     if amount <= 0:
         raise ValueError("Amount must be positive")`,
			expectCommentOnly: true,
			description:       "Adding Python comment should be comment-only",
		},
		{
			name: "Multiple comment additions",
			diff: `@@ -10,6 +10,9 @@ func Calculate() int {
 	// Calculate result
 	result := x + y
+	// Log the calculation
+	// TODO: Add proper logging
+	// This is a temporary comment
 	return result
 }`,
			expectCommentOnly: true,
			description:       "Multiple comment additions should be comment-only",
		},
		{
			name: "Block comment addition",
			diff: `@@ -5,6 +5,10 @@ package main

 import "fmt"

+/*
+ * This is a block comment
+ * explaining the package
+ */
 func main() {
 	fmt.Println("Hello")
 }`,
			expectCommentOnly: true,
			description:       "Block comment addition should be comment-only",
		},
		{
			name: "Comment removal",
			diff: `@@ -10,7 +10,6 @@ func Login() error {
 	user := getUser()
-	// This comment is being removed
 	if user == nil {
 		return errors.New("user not found")
 	}`,
			expectCommentOnly: true,
			description:       "Removing comment should be comment-only",
		},

		// TRUE NEGATIVES - Code changes
		{
			name: "Code addition",
			diff: `@@ -10,6 +10,7 @@ func Login() error {
 	user := getUser()
+	user.LastLogin = time.Now()
 	if user == nil {
 		return errors.New("user not found")
 	}`,
			expectCommentOnly: false,
			description:       "Adding code should not be comment-only",
		},
		{
			name: "Mixed comment and code",
			diff: `@@ -10,6 +10,8 @@ func Login() error {
 	user := getUser()
+	// Update last login time
+	user.LastLogin = time.Now()
 	if user == nil {
 		return errors.New("user not found")
 	}`,
			expectCommentOnly: false,
			description:       "Mixed comment and code should not be comment-only",
		},
		{
			name: "Variable declaration",
			diff: `@@ -5,6 +5,7 @@ func Calculate() int {
 	x := 10
 	y := 20
+	z := 30
 	return x + y
 }`,
			expectCommentOnly: false,
			description:       "Variable declaration should not be comment-only",
		},
		{
			name: "Function call",
			diff: `@@ -10,6 +10,7 @@ func Process() {
 	data := fetchData()
+	validate(data)
 	save(data)
 }`,
			expectCommentOnly: false,
			description:       "Function call should not be comment-only",
		},

		// EDGE CASES
		{
			name:              "Empty diff",
			diff:              "",
			expectCommentOnly: false,
			description:       "Empty diff should not be comment-only",
		},
		{
			name: "Only diff metadata",
			diff: `diff --git a/file.go b/file.go
index 1234567..abcdefg 100644
--- a/file.go
+++ b/file.go`,
			expectCommentOnly: false,
			description:       "Only metadata should not be comment-only",
		},
		{
			name: "Empty line additions",
			diff: `@@ -10,6 +10,8 @@ func Login() error {
 	user := getUser()
+
+
 	if user == nil {
 		return errors.New("user not found")
 	}`,
			expectCommentOnly: true,
			description:       "Empty line additions should be comment-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCommentOnlyChange(tt.diff)

			if result != tt.expectCommentOnly {
				t.Errorf("IsCommentOnlyChange = %v, want %v\nDescription: %s\nDiff:\n%s",
					result, tt.expectCommentOnly, tt.description, tt.diff)
			}

			t.Logf("Result: CommentOnly=%v", result)
		})
	}
}

func TestIsDocumentationFile(t *testing.T) {
	// Quick test for the simplified helper function
	tests := []struct {
		filePath string
		expected bool
	}{
		{"README.md", true},
		{"src/main.go", false},
		{"CHANGELOG.txt", true},
		{"config.yaml", false},
		{"LICENSE", true},
		{"internal/auth.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := IsDocumentationFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("IsDocumentationFile(%s) = %v, want %v",
					tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"README.md", "README.md"},
		{"docs/README.md", "README.md"},
		{"path/to/file.txt", "file.txt"},
		{"/absolute/path/file.go", "file.go"},
		{"windows\\path\\file.py", "file.py"},
		{"file", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getFileName(tt.path)
			if result != tt.expected {
				t.Errorf("getFileName(%s) = %s, want %s",
					tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsCommentLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
		language string
	}{
		// Go comments
		{"// This is a comment", true, "Go"},
		{"/* Block comment */", true, "Go"},
		{"* Continuation", true, "Go"},

		// Python comments
		{"# This is a comment", true, "Python"},
		{`"""Docstring"""`, true, "Python"},
		{"'''Docstring'''", true, "Python"},

		// HTML/Markdown comments
		{"<!-- HTML comment -->", true, "HTML"},
		{"-->", true, "HTML"},

		// SQL comments
		{"-- SQL comment", true, "SQL"},

		// Not comments
		{"func main() {", false, "Code"},
		{"x = 10", false, "Code"},
		{"if err != nil {", false, "Code"},
		{"", true, "Empty"},
		{"   ", true, "Whitespace"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := isCommentLine(tt.line)
			if result != tt.expected {
				t.Errorf("isCommentLine(%q) = %v, want %v (%s)",
					tt.line, result, tt.expected, tt.language)
			}
		})
	}
}

// Benchmark documentation detection performance
func BenchmarkIsDocumentationOnly(b *testing.B) {
	paths := []string{
		"README.md",
		"internal/auth/handlers.go",
		"docs/architecture.md",
		"src/services/payment.py",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_ = IsDocumentationOnly(path)
		}
	}
}
