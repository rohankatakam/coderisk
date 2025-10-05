package treesitter

// CodeEntity represents an extracted code entity (function, class, import, file)
// These entities are used to build Layer 1 of the knowledge graph
type CodeEntity struct {
	Type       string // "function", "class", "import", "file"
	Name       string
	FilePath   string
	StartLine  int
	EndLine    int
	Language   string
	Signature  string // For functions
	ImportPath string // For imports
	Complexity int    // Cyclomatic complexity (optional)
}

// ParseResult contains all entities extracted from a file
type ParseResult struct {
	FilePath string
	Language string
	Entities []CodeEntity
	Error    error
}
