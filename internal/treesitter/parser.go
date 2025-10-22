package treesitter

import (
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Import the language bindings explicitly to ensure proper linking
var _ = tree_sitter_typescript.LanguageTypescript
var _ = tree_sitter_python.Language

// LanguageParser wraps tree-sitter parser with language-specific grammar
// IMPORTANT: Always call Close() to prevent memory leaks (CGO requirement)
type LanguageParser struct {
	parser   *sitter.Parser
	language *sitter.Language
	langName string
}

// NewLanguageParser creates a parser for the specified language
// Supported languages: javascript, typescript, python
// Returns error if language is unsupported
func NewLanguageParser(lang string) (*LanguageParser, error) {
	parser := sitter.NewParser()
	if parser == nil {
		return nil, fmt.Errorf("failed to create tree-sitter parser")
	}

	var language *sitter.Language
	switch lang {
	case "javascript", "jsx":
		language = sitter.NewLanguage(tree_sitter_javascript.Language())
	case "typescript", "tsx":
		language = sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
	case "python":
		language = sitter.NewLanguage(tree_sitter_python.Language())
	default:
		parser.Close()
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	if err := parser.SetLanguage(language); err != nil {
		parser.Close()
		return nil, fmt.Errorf("failed to set language %s: %w", lang, err)
	}

	return &LanguageParser{
		parser:   parser,
		language: language,
		langName: lang,
	}, nil
}

// Close releases parser resources (REQUIRED - CGO memory management)
func (lp *LanguageParser) Close() {
	if lp.parser != nil {
		lp.parser.Close()
	}
}

// Parse parses source code and returns the syntax tree
// Caller must call tree.Close() when done
func (lp *LanguageParser) Parse(code []byte) (*sitter.Tree, error) {
	tree := lp.parser.Parse(code, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse code")
	}
	return tree, nil
}

// ParseFile parses a file and extracts code entities
func ParseFile(filePath string) (*ParseResult, error) {
	// Detect language from file extension
	lang := DetectLanguage(filePath)
	if lang == "" {
		return &ParseResult{
			FilePath: filePath,
			Error:    fmt.Errorf("unsupported file type: %s", filePath),
		}, nil
	}

	// Read file content
	code, err := os.ReadFile(filePath)
	if err != nil {
		return &ParseResult{
			FilePath: filePath,
			Error:    fmt.Errorf("failed to read file: %w", err),
		}, nil
	}

	// Create language parser
	lp, err := NewLanguageParser(lang)
	if err != nil {
		return &ParseResult{
			FilePath: filePath,
			Error:    fmt.Errorf("failed to create parser: %w", err),
		}, nil
	}
	defer lp.Close()

	// Parse code
	tree, err := lp.Parse(code)
	if err != nil {
		return &ParseResult{
			FilePath: filePath,
			Error:    fmt.Errorf("failed to parse: %w", err),
		}, nil
	}
	defer tree.Close()

	// Extract entities based on language
	var entities []CodeEntity
	root := tree.RootNode()

	switch lang {
	case "javascript", "jsx":
		entities, err = extractJavaScriptEntities(filePath, root, code)
	case "typescript", "tsx":
		entities, err = extractTypeScriptEntities(filePath, root, code)
	case "python":
		entities, err = extractPythonEntities(filePath, root, code)
	default:
		return &ParseResult{
			FilePath: filePath,
			Error:    fmt.Errorf("no extractor for language: %s", lang),
		}, nil
	}

	if err != nil {
		return &ParseResult{
			FilePath: filePath,
			Language: lang,
			Error:    err,
		}, nil
	}

	return &ParseResult{
		FilePath: filePath,
		Language: lang,
		Entities: entities,
	}, nil
}

// DetectLanguage returns language identifier from file extension
func DetectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)

	langMap := map[string]string{
		".js":   "javascript",
		".jsx":  "jsx",
		".ts":   "typescript",
		".tsx":  "tsx",
		".mjs":  "javascript",
		".cjs":  "javascript",
		".mts":  "typescript",
		".cts":  "typescript",
		".py":   "python",
		".pyi":  "python",
		".pyw":  "python",
	}

	return langMap[ext]
}

