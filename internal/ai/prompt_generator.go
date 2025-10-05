package ai

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coderisk/coderisk-go/internal/models"
)

// PromptGenerator generates AI-executable prompts for fixes
type PromptGenerator struct{}

// NewPromptGenerator creates a new prompt generator
func NewPromptGenerator() *PromptGenerator {
	return &PromptGenerator{}
}

// GeneratePrompt creates a ready-to-execute prompt for the given issue
func (g *PromptGenerator) GeneratePrompt(issue models.RiskIssue, file models.FileRisk) string {
	switch issue.FixType {
	case "generate_tests":
		return g.generateTestPrompt(issue, file)
	case "add_error_handling":
		return g.generateErrorHandlingPrompt(issue, file)
	case "reduce_coupling":
		return g.generateCouplingPrompt(issue, file)
	case "fix_security":
		return g.generateSecurityPrompt(issue, file)
	default:
		return ""
	}
}

// generateTestPrompt creates a prompt for generating tests
func (g *PromptGenerator) generateTestPrompt(issue models.RiskIssue, file models.FileRisk) string {
	framework := g.detectTestFramework(file.Language)
	testFile := g.testFileName(file.Path, file.Language)

	coverage := 0.0
	if cov, ok := file.Metrics["test_coverage"]; ok {
		coverage = cov.Value
	}

	return fmt.Sprintf(`Generate comprehensive unit and integration tests for %s in %s (lines %d-%d).

Coverage requirements:
- Happy path: Valid inputs, expected outputs
- Edge cases: Boundary conditions, null values
- Error cases: Invalid inputs, exceptions
- Integration: Dependencies and external calls

Framework: Use %s testing framework matching project conventions.

Context:
- File language: %s
- Current coverage: %.1f%%
- Function: %s

Generate tests in separate test file following naming convention: %s`,
		issue.Function,
		file.Path,
		issue.LineStart,
		issue.LineEnd,
		framework,
		file.Language,
		coverage,
		issue.Function,
		testFile,
	)
}

// generateErrorHandlingPrompt creates a prompt for adding error handling
func (g *PromptGenerator) generateErrorHandlingPrompt(issue models.RiskIssue, file models.FileRisk) string {
	retryLib := g.detectRetryLibrary(file.Language)

	return fmt.Sprintf(`Add robust error handling to %s in %s (lines %d-%d).

Requirements:
- Add try/catch or error checks for network calls
- Implement exponential backoff retry logic
- Handle specific errors: TimeoutError, ConnectionError, HTTPError
- Log errors with context for debugging

Use these patterns:
- Library: %s (if available, otherwise implement custom)
- Max retries: 3
- Backoff: 100ms, 200ms, 400ms

Context:
- Language: %s
- Function: %s

Add appropriate error handling while maintaining code readability.`,
		issue.Function,
		file.Path,
		issue.LineStart,
		issue.LineEnd,
		retryLib,
		file.Language,
		issue.Function,
	)
}

// generateCouplingPrompt creates a prompt for reducing coupling
func (g *PromptGenerator) generateCouplingPrompt(issue models.RiskIssue, file models.FileRisk) string {
	return fmt.Sprintf(`Suggest refactoring options to reduce coupling in %s.

Current coupling issues:
- High dependency count
- Strong temporal coupling detected
- Co-change frequency indicates architectural smell

Consider these patterns:
1. Repository pattern - Extract data access interface
2. Dependency injection - Inject dependencies via constructor
3. Interface segregation - Define minimal interfaces

Provide 2-3 refactoring options with:
- Pros/cons of each approach
- Estimated effort (time)
- Risk level

Context:
- Language: %s
- File: %s
- Function: %s (lines %d-%d)`,
		file.Path,
		file.Language,
		file.Path,
		issue.Function,
		issue.LineStart,
		issue.LineEnd,
	)
}

// generateSecurityPrompt creates a prompt for fixing security issues
func (g *PromptGenerator) generateSecurityPrompt(issue models.RiskIssue, file models.FileRisk) string {
	bestPractices := g.securityBestPractices(file.Language)

	return fmt.Sprintf(`Fix security vulnerability in %s (lines %d-%d).

Issue: %s
Severity: %s

Security requirements:
- Input validation: Sanitize all user inputs
- Authentication: Verify user permissions
- Encryption: Use secure protocols (TLS 1.2+)
- Secrets: Use environment variables, not hardcoded

Best practices for %s:
%s

Context:
- File: %s
- Function: %s`,
		file.Path,
		issue.LineStart,
		issue.LineEnd,
		issue.Message,
		issue.Severity,
		file.Language,
		bestPractices,
		file.Path,
		issue.Function,
	)
}

// Helper functions

func (g *PromptGenerator) detectTestFramework(language string) string {
	frameworks := map[string]string{
		"python":     "pytest",
		"javascript": "jest",
		"typescript": "jest",
		"go":         "testing (Go standard)",
		"java":       "JUnit",
		"ruby":       "RSpec",
		"rust":       "cargo test",
	}
	if fw, ok := frameworks[language]; ok {
		return fw
	}
	return "appropriate testing framework"
}

func (g *PromptGenerator) detectRetryLibrary(language string) string {
	libraries := map[string]string{
		"python":     "tenacity",
		"javascript": "axios-retry",
		"typescript": "axios-retry",
		"go":         "go-retryablehttp",
		"java":       "resilience4j",
		"rust":       "backoff crate",
	}
	if lib, ok := libraries[language]; ok {
		return lib
	}
	return "custom retry implementation"
}

func (g *PromptGenerator) testFileName(filePath string, language string) string {
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)

	// Different test naming conventions per language
	switch language {
	case "python":
		// Python: test_filename.py
		dir := filepath.Dir(filePath)
		name := filepath.Base(base)
		return filepath.Join(dir, "test_"+name+ext)
	case "go":
		// Go: filename_test.go
		return base + "_test" + ext
	case "javascript", "typescript":
		// JS/TS: filename.test.js or filename.spec.js
		return base + ".test" + ext
	case "java":
		// Java: FilenameTest.java
		dir := filepath.Dir(filePath)
		name := filepath.Base(base)
		return filepath.Join(dir, name+"Test"+ext)
	default:
		// Default: filename_test.ext
		return base + "_test" + ext
	}
}

func (g *PromptGenerator) securityBestPractices(language string) string {
	practices := map[string]string{
		"python": `- Use parameterized queries (SQLAlchemy)
- Validate with pydantic
- OWASP Top 10 compliance
- Hash passwords with bcrypt`,
		"javascript": `- Sanitize with DOMPurify
- Use helmet.js for headers
- CSRF tokens required
- Validate with joi/zod`,
		"typescript": `- Type-safe input validation
- Use helmet for Express
- CSRF protection
- Content Security Policy`,
		"go": `- Use prepared statements
- bcrypt for passwords
- TLS 1.3 minimum
- Input validation with struct tags`,
		"java": `- OWASP dependency check
- PreparedStatement for SQL
- Spring Security
- Input validation with Bean Validation`,
	}
	if p, ok := practices[language]; ok {
		return p
	}
	return `- Follow OWASP guidelines
- Input validation
- Least privilege principle
- Security headers`
}

// DetermineFixType analyzes an issue and determines the appropriate fix type
func (g *PromptGenerator) DetermineFixType(issue models.RiskIssue, file models.FileRisk) string {
	// Check coverage issues
	if cov, ok := file.Metrics["test_coverage"]; ok && cov.Value < 0.3 {
		return "generate_tests"
	}

	// Check for security keywords
	securityKeywords := []string{"security", "vulnerability", "xss", "injection", "csrf"}
	for _, kw := range securityKeywords {
		if strings.Contains(strings.ToLower(issue.Message), kw) {
			return "fix_security"
		}
	}

	// Check for error handling issues
	errorKeywords := []string{"error", "exception", "timeout", "retry", "network"}
	for _, kw := range errorKeywords {
		if strings.Contains(strings.ToLower(issue.Message), kw) {
			return "add_error_handling"
		}
	}

	// Check for coupling issues
	couplingKeywords := []string{"coupling", "dependency", "co-change"}
	for _, kw := range couplingKeywords {
		if strings.Contains(strings.ToLower(issue.Message), kw) {
			return "reduce_coupling"
		}
	}

	// Default
	return "generate_tests"
}
