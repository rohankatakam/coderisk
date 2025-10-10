package ai

// PromptTemplates contains reusable AI prompt templates
// These templates use {{.Variable}} syntax for template expansion
var PromptTemplates = map[string]string{
	"generate_tests": `Generate comprehensive unit and integration tests for {{.Function}} in {{.File}} (lines {{.LineStart}}-{{.LineEnd}}).

Coverage requirements:
- Happy path: Valid inputs, expected outputs
- Edge cases: Boundary conditions, null values
- Error cases: Invalid inputs, exceptions
- Integration: Dependencies and external calls

Framework: Use {{.TestFramework}} matching project conventions.

Context:
- File language: {{.Language}}
- Current coverage: {{.Coverage}}%
- Function: {{.Function}}

Generate tests in: {{.TestFile}}`,

	"add_error_handling": `Add robust error handling to {{.Function}} in {{.File}} (lines {{.LineStart}}-{{.LineEnd}}).

Requirements:
- Add try/catch or error checks for network calls
- Implement exponential backoff retry logic
- Handle specific errors: TimeoutError, ConnectionError, HTTPError
- Log errors with context

Use: {{.RetryLibrary}} (if available, otherwise custom)

Context:
- Language: {{.Language}}
- Max retries: 3
- Backoff: 100ms, 200ms, 400ms`,

	"reduce_coupling": `Suggest refactoring options to reduce coupling in {{.File}}.

Current coupling:
- Co-change frequency: {{.CoChangePercent}}%
- Temporal coupling strength: {{.CouplingStrength}}

Provide 2-3 options with pros/cons.

Context:
- Language: {{.Language}}
- Function: {{.Function}} (lines {{.LineStart}}-{{.LineEnd}})`,

	"fix_security": `Fix security vulnerability in {{.File}} (lines {{.LineStart}}-{{.LineEnd}}).

Issue: {{.Message}}
Severity: {{.Severity}}

Apply security best practices for {{.Language}}.

Context:
- File: {{.File}}
- Function: {{.Function}}`,

	"reduce_complexity": `Refactor {{.Function}} in {{.File}} to reduce complexity.

Current complexity: {{.Complexity}}
Target: < 10

Suggestions:
- Extract methods
- Simplify conditionals
- Remove duplication

Context:
- Language: {{.Language}}
- Lines: {{.LineStart}}-{{.LineEnd}}`,
}

// FixTypeDescriptions provides human-readable descriptions for each fix type
var FixTypeDescriptions = map[string]string{
	"generate_tests":     "Generate comprehensive test coverage",
	"add_error_handling": "Add robust error handling and retry logic",
	"reduce_coupling":    "Refactor to reduce dependencies",
	"fix_security":       "Fix security vulnerability",
	"reduce_complexity":  "Simplify complex code",
}

// FixTypeCategories maps fix types to categories
var FixTypeCategories = map[string]string{
	"generate_tests":     "quality",
	"add_error_handling": "reliability",
	"reduce_coupling":    "architecture",
	"fix_security":       "security",
	"reduce_complexity":  "maintainability",
}
