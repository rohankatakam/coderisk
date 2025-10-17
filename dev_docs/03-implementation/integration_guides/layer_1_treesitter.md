# Layer 1: Tree-sitter AST Parsing Integration Guide

**Purpose:** Implementation guide for building Layer 1 (Code Structure) of the CodeRisk knowledge graph using Tree-sitter

**Last Updated:** October 3, 2025

**Prerequisites:**
- Go 1.23+
- Local repository clone available
- Neo4j/Neptune graph database running
- Understanding of [graph_ontology.md](../../01-architecture/graph_ontology.md) Layer 1 schema

**Target:** Priority 7 implementation (after Layers 2 & 3 complete)

---

## Architecture Context

**References:**
- [graph_ontology.md](../../01-architecture/graph_ontology.md) - Layer 1 schema (File, Function, Class nodes)
- [scalability_analysis.md](../../01-architecture/scalability_analysis.md) - Performance targets (<500ms Phase 1)
- [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) - Go bindings for Tree-sitter

**Layer 1 Purpose:**
- Answer "What code depends on what?" (factual, low FP rate ~1%)
- Enables Phase 1 baseline checks (<500ms, no LLM)
- Foundation for Layers 2 & 3 (MODIFIES edges link commits → files)

**Performance Targets:**
- omnara-ai/omnara (~1K files): <10s parsing
- kubernetes/kubernetes (~50K files): <5min parsing

---

## Overview

Layer 1 construction follows a 4-phase pipeline:

```
Phase 1: Repository Clone & File Discovery
   ↓ (git clone, file tree walk)
Phase 2: Language Detection & Parser Selection
   ↓ (file extension → tree-sitter grammar)
Phase 3: AST Parsing & Entity Extraction
   ↓ (parse file → extract functions, classes, imports)
Phase 4: Graph Construction
   ↓ (create File/Function/Class nodes + CALLS/IMPORTS edges)
```

**Total Time:** ~10s (omnara) to ~5min (kubernetes)

---

## Tree-sitter Fundamentals

### How Tree-sitter Works

**Input:**
- Source code as `[]byte` (single file)
- Language grammar (loaded separately)

**Output:**
- Abstract Syntax Tree (AST) with typed nodes
- Node properties: `Type()`, `Content()`, `StartPoint()`, `EndPoint()`, `ChildCount()`
- Tree structure allows traversal

**Example:**
```go
package main

import (
    tree_sitter "github.com/tree-sitter/go-tree-sitter"
    tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func parseGoFile(code []byte) (*tree_sitter.Tree, error) {
    parser := tree_sitter.NewParser()
    defer parser.Close()

    // Load Go grammar
    parser.SetLanguage(tree_sitter.NewLanguage(tree_sitter_go.Language()))

    // Parse source code
    tree := parser.Parse(code, nil)
    return tree, nil
}
```

### Supported Languages for v1.0

**Priority 1 (Most common):**
- Go: `github.com/tree-sitter/tree-sitter-go`
- Python: `github.com/tree-sitter/tree-sitter-python`
- JavaScript/TypeScript: `github.com/tree-sitter/tree-sitter-javascript`, `tree-sitter-typescript`
- Java: `github.com/tree-sitter/tree-sitter-java`

**Priority 2 (Add later):**
- Rust, C/C++, Ruby, PHP, etc.

**Unsupported (treat as opaque files):**
- Binary files, images, PDFs
- Generated code (`.pb.go`, `.d.ts`, etc.) - detect and skip

---

## Phase 1: Repository Clone & File Discovery

### Step 1: Clone Repository

**Implementation: `internal/ingestion/clone.go`**

```go
package ingestion

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

// CloneRepository performs shallow clone for Layer 1 parsing
func CloneRepository(ctx context.Context, url string) (string, error) {
    // Generate unique hash for storage
    hash := generateRepoHash(url)
    repoPath := filepath.Join(os.Getenv("HOME"), ".coderisk", "repos", hash)

    // Check if already cloned
    if _, err := os.Stat(repoPath); err == nil {
        return repoPath, nil // Already exists
    }

    // Shallow clone (--depth 1) for speed
    cmd := exec.CommandContext(ctx, "git", "clone",
        "--depth", "1",
        "--single-branch",
        url,
        repoPath,
    )

    if output, err := cmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("git clone failed: %w, output: %s", err, output)
    }

    return repoPath, nil
}
```

### Step 2: Walk File Tree

**Filter Strategy:**
- ✅ Include: Source files (`.go`, `.py`, `.js`, `.ts`, `.java`)
- ❌ Exclude: `.git/`, `node_modules/`, `vendor/`, `venv/`, `__pycache__/`
- ❌ Exclude: Generated files (`.pb.go`, `.generated.ts`, `.min.js`)
- ❌ Exclude: Binary files, archives

```go
// WalkSourceFiles walks repository and yields source files
func WalkSourceFiles(repoPath string) (<-chan string, error) {
    files := make(chan string, 100)

    go func() {
        defer close(files)

        filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
            if err != nil {
                return err
            }

            // Skip excluded directories
            if d.IsDir() && shouldSkipDir(d.Name()) {
                return filepath.SkipDir
            }

            // Only process files with supported extensions
            if !d.IsDir() && isSupportedFile(path) {
                files <- path
            }

            return nil
        })
    }()

    return files, nil
}

func shouldSkipDir(name string) bool {
    excludeDirs := []string{
        ".git", "node_modules", "vendor", "venv", "__pycache__",
        ".next", "dist", "build", "target",
    }

    for _, exclude := range excludeDirs {
        if name == exclude {
            return true
        }
    }
    return false
}

func isSupportedFile(path string) bool {
    ext := filepath.Ext(path)
    supported := []string{".go", ".py", ".js", ".ts", ".java"}

    for _, s := range supported {
        if ext == s {
            return !isGeneratedFile(path) // Skip generated files
        }
    }
    return false
}

func isGeneratedFile(path string) bool {
    generatedPatterns := []string{
        ".pb.go",        // Protocol buffers
        ".generated.ts", // Generated TypeScript
        ".min.js",       // Minified JS
        ".d.ts",         // TypeScript declarations (optional: could include)
    }

    for _, pattern := range generatedPatterns {
        if strings.HasSuffix(path, pattern) {
            return true
        }
    }
    return false
}
```

---

## Phase 2: Language Detection & Parser Selection

### GitHub API Language Detection (Before Parsing)

**Optimization:** Detect repository languages via GitHub API to load only required parsers.

```go
package ingestion

import (
    "context"
    "github.com/google/go-github/v57/github"
)

// DetectRepoLanguages calls GitHub API to get language distribution
func DetectRepoLanguages(ctx context.Context, client *github.Client, owner, repo string) ([]string, error) {
    // Call /repos/{owner}/{repo}/languages
    languages, _, err := client.Repositories.ListLanguages(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("failed to get languages: %w", err)
    }

    // Extract language names (sorted by bytes descending)
    var langs []string
    for lang := range languages {
        langs = append(langs, strings.ToLower(lang))
    }

    return langs, nil
}

// Example output for omnara-ai/omnara:
// ["typescript", "javascript"] - only load these 2 parsers, skip python/go/java
```

**Benefits:**
- Load 1-2 parsers instead of 5 (memory efficient)
- Faster initialization
- Accurate language reporting

### Language Grammar Registry

```go
package parser

import (
    tree_sitter "github.com/tree-sitter/go-tree-sitter"
    tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
    tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
    tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
    tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
    tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

type LanguageParser struct {
    parser   *tree_sitter.Parser
    language string
}

// NewLanguageParser creates parser for given language
func NewLanguageParser(lang string) (*LanguageParser, error) {
    parser := tree_sitter.NewParser()

    var grammar unsafe.Pointer
    switch lang {
    case "go":
        grammar = tree_sitter_go.Language()
    case "python":
        grammar = tree_sitter_python.Language()
    case "javascript":
        grammar = tree_sitter_javascript.Language()
    case "typescript":
        grammar = tree_sitter_typescript.Language()
    case "java":
        grammar = tree_sitter_java.Language()
    default:
        parser.Close()
        return nil, fmt.Errorf("unsupported language: %s", lang)
    }

    parser.SetLanguage(tree_sitter.NewLanguage(grammar))

    return &LanguageParser{
        parser:   parser,
        language: lang,
    }, nil
}

func (lp *LanguageParser) Close() {
    lp.parser.Close()
}

// DetectLanguage returns language from file extension
func DetectLanguage(filePath string) string {
    ext := filepath.Ext(filePath)

    langMap := map[string]string{
        ".go":   "go",
        ".py":   "python",
        ".js":   "javascript",
        ".ts":   "typescript",
        ".tsx":  "typescript",
        ".jsx":  "javascript",
        ".java": "java",
    }

    return langMap[ext]
}
```

---

## Phase 3: AST Parsing & Entity Extraction

### Generic Entity Extraction

```go
type CodeEntity struct {
    Type       string // "function", "class", "import", "file"
    Name       string
    FilePath   string
    StartLine  int
    EndLine    int
    Language   string
    Signature  string // For functions
    ImportPath string // For imports
}

// ParseFile parses a file and extracts entities
func ParseFile(filePath string, code []byte) ([]CodeEntity, error) {
    lang := DetectLanguage(filePath)
    if lang == "" {
        return nil, fmt.Errorf("unsupported file type: %s", filePath)
    }

    lp, err := NewLanguageParser(lang)
    if err != nil {
        return nil, err
    }
    defer lp.Close()

    tree := lp.parser.Parse(code, nil)
    defer tree.Close()

    root := tree.RootNode()

    // Extract entities based on language
    switch lang {
    case "go":
        return extractGoEntities(filePath, root, code)
    case "python":
        return extractPythonEntities(filePath, root, code)
    case "javascript", "typescript":
        return extractJSEntities(filePath, root, code)
    case "java":
        return extractJavaEntities(filePath, root, code)
    default:
        return nil, fmt.Errorf("unsupported language: %s", lang)
    }
}
```

### Go-Specific Extraction

```go
// extractGoEntities extracts functions, methods, structs, and imports from Go AST
func extractGoEntities(filePath string, root *tree_sitter.Node, code []byte) ([]CodeEntity, error) {
    entities := []CodeEntity{}

    // Add file entity
    entities = append(entities, CodeEntity{
        Type:     "file",
        Name:     filepath.Base(filePath),
        FilePath: filePath,
        Language: "go",
    })

    // Traverse AST
    cursor := tree_sitter.NewTreeCursor(root)
    defer cursor.Close()

    for {
        node := cursor.CurrentNode()
        nodeType := node.Type()

        switch nodeType {
        case "function_declaration":
            // Extract function
            funcName := getChildByFieldName(node, "name", code)
            params := getChildByFieldName(node, "parameters", code)

            entities = append(entities, CodeEntity{
                Type:      "function",
                Name:      funcName,
                FilePath:  filePath,
                StartLine: int(node.StartPoint().Row) + 1,
                EndLine:   int(node.EndPoint().Row) + 1,
                Language:  "go",
                Signature: fmt.Sprintf("func %s%s", funcName, params),
            })

        case "method_declaration":
            // Extract method (function with receiver)
            receiver := getChildByFieldName(node, "receiver", code)
            funcName := getChildByFieldName(node, "name", code)
            params := getChildByFieldName(node, "parameters", code)

            entities = append(entities, CodeEntity{
                Type:      "function",
                Name:      funcName,
                FilePath:  filePath,
                StartLine: int(node.StartPoint().Row) + 1,
                EndLine:   int(node.EndPoint().Row) + 1,
                Language:  "go",
                Signature: fmt.Sprintf("func %s %s%s", receiver, funcName, params),
            })

        case "type_declaration":
            // Extract struct/interface/type
            typeName := getChildByFieldName(node, "name", code)

            entities = append(entities, CodeEntity{
                Type:      "class", // Using "class" for consistency across languages
                Name:      typeName,
                FilePath:  filePath,
                StartLine: int(node.StartPoint().Row) + 1,
                EndLine:   int(node.EndPoint().Row) + 1,
                Language:  "go",
            })

        case "import_declaration":
            // Extract imports
            importPath := getImportPath(node, code)

            entities = append(entities, CodeEntity{
                Type:       "import",
                Name:       importPath,
                FilePath:   filePath,
                Language:   "go",
                ImportPath: importPath,
            })
        }

        // Traverse to next node
        if !cursor.GotoNextSibling() {
            if !cursor.GotoParent() {
                break
            }
        }
    }

    return entities, nil
}

// Helper: Get child node content by field name
func getChildByFieldName(node *tree_sitter.Node, fieldName string, code []byte) string {
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.FieldName() == fieldName {
            return string(code[child.StartByte():node.EndByte()])
        }
    }
    return ""
}
```

**Note:** Python, JavaScript, and Java extractors follow similar pattern but with language-specific node types.

---

## Phase 4: Graph Construction (Neo4j/Neptune) - Branch-Aware

### Branch Properties on All Nodes

**All Layer 1 nodes must include branch context:**

```go
type CodeEntity struct {
    Type       string // "function", "class", "import", "file"
    Name       string
    FilePath   string
    StartLine  int
    EndLine    int
    Language   string
    Signature  string // For functions
    ImportPath string // For imports
    Branch     string // NEW: Current branch name
    GitSHA     string // NEW: Commit SHA (source of truth)
}
```

**How to get branch and SHA:**
```go
func getCurrentBranch() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func getCurrentSHA() (string, error) {
    cmd := exec.Command("git", "rev-parse", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}
```

### Dual Query Generation

Since we support both **Neo4j (local)** and **Neptune (cloud)**, we need dual query generation:

```go
type GraphBackend string

const (
    Neo4j   GraphBackend = "neo4j"
    Neptune GraphBackend = "neptune"
)

// GenerateNodeQuery creates backend-specific query for node creation
func GenerateNodeQuery(entity CodeEntity, backend GraphBackend) string {
    switch backend {
    case Neo4j:
        return generateCypherNode(entity)
    case Neptune:
        return generateGremlinNode(entity)
    default:
        return ""
    }
}
```

### Neo4j (Cypher) Queries

```go
// generateCypherNode creates Cypher query for node creation (branch-aware)
func generateCypherNode(entity CodeEntity) string {
    switch entity.Type {
    case "file":
        return fmt.Sprintf(`
            MERGE (f:File {path: "%s", branch: "%s"})
            SET f.language = "%s",
                f.name = "%s",
                f.git_sha = "%s"
        `, entity.FilePath, entity.Branch, entity.Language, entity.Name, entity.GitSHA)

    case "function":
        return fmt.Sprintf(`
            MERGE (f:Function {name: "%s", file: "%s"})
            SET f.signature = "%s",
                f.start_line = %d,
                f.end_line = %d,
                f.language = "%s"
        `, entity.Name, entity.FilePath, entity.Signature,
           entity.StartLine, entity.EndLine, entity.Language)

    case "class":
        return fmt.Sprintf(`
            MERGE (c:Class {name: "%s", file: "%s"})
            SET c.start_line = %d,
                c.end_line = %d,
                c.language = "%s"
        `, entity.Name, entity.FilePath,
           entity.StartLine, entity.EndLine, entity.Language)

    case "import":
        return fmt.Sprintf(`
            MERGE (i:Import {path: "%s"})
        `, entity.ImportPath)
    }
    return ""
}

// generateCypherEdge creates Cypher query for relationship
func generateCypherEdge(from, to CodeEntity, edgeType string) string {
    switch edgeType {
    case "CONTAINS":
        // File CONTAINS Function/Class
        return fmt.Sprintf(`
            MATCH (file:File {path: "%s"})
            MATCH (entity {name: "%s", file: "%s"})
            MERGE (file)-[:CONTAINS]->(entity)
        `, from.FilePath, to.Name, to.FilePath)

    case "IMPORTS":
        // File IMPORTS ImportPath
        return fmt.Sprintf(`
            MATCH (file:File {path: "%s"})
            MATCH (import:Import {path: "%s"})
            MERGE (file)-[:IMPORTS]->(import)
        `, from.FilePath, to.ImportPath)

    case "CALLS":
        // Function CALLS Function (requires call graph analysis)
        return fmt.Sprintf(`
            MATCH (caller:Function {name: "%s", file: "%s"})
            MATCH (callee:Function {name: "%s"})
            MERGE (caller)-[:CALLS]->(callee)
        `, from.Name, from.FilePath, to.Name)
    }
    return ""
}
```

### Neptune (Gremlin) Queries

```go
// generateGremlinNode creates Gremlin query for node creation
func generateGremlinNode(entity CodeEntity) string {
    switch entity.Type {
    case "file":
        return fmt.Sprintf(`
            g.V().has('File', 'path', '%s').
              fold().
              coalesce(
                unfold(),
                addV('File').
                  property('path', '%s').
                  property('language', '%s').
                  property('name', '%s')
              )
        `, entity.FilePath, entity.FilePath, entity.Language, entity.Name)

    case "function":
        return fmt.Sprintf(`
            g.V().has('Function', 'name', '%s').has('file', '%s').
              fold().
              coalesce(
                unfold(),
                addV('Function').
                  property('name', '%s').
                  property('file', '%s').
                  property('signature', '%s').
                  property('start_line', %d).
                  property('end_line', %d).
                  property('language', '%s')
              )
        `, entity.Name, entity.FilePath, entity.Name, entity.FilePath,
           entity.Signature, entity.StartLine, entity.EndLine, entity.Language)

    // Similar for class and import...
    }
    return ""
}
```

### Batch Execution

```go
// BatchInsertEntities inserts entities in batches (100 per request)
func BatchInsertEntities(entities []CodeEntity, backend GraphBackend) error {
    batchSize := 100

    for i := 0; i < len(entities); i += batchSize {
        end := i + batchSize
        if end > len(entities) {
            end = len(entities)
        }

        batch := entities[i:end]

        // Generate queries
        var queries []string
        for _, entity := range batch {
            query := GenerateNodeQuery(entity, backend)
            queries = append(queries, query)
        }

        // Execute batch
        if err := executeBatch(queries, backend); err != nil {
            return fmt.Errorf("batch %d-%d failed: %w", i, end, err)
        }
    }

    return nil
}

func executeBatch(queries []string, backend GraphBackend) error {
    switch backend {
    case Neo4j:
        return executeCypherBatch(queries)
    case Neptune:
        return executeGremlinBatch(queries)
    default:
        return fmt.Errorf("unsupported backend: %s", backend)
    }
}
```

---

## Performance Optimization

### Parallel Parsing

```go
// ParseRepository parses all files in parallel
func ParseRepository(repoPath string, workers int) ([]CodeEntity, error) {
    files, err := WalkSourceFiles(repoPath)
    if err != nil {
        return nil, err
    }

    // Worker pool
    results := make(chan []CodeEntity, workers)
    errors := make(chan error, workers)

    var wg sync.WaitGroup

    // Start workers
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()

            for filePath := range files {
                code, err := os.ReadFile(filePath)
                if err != nil {
                    errors <- err
                    continue
                }

                entities, err := ParseFile(filePath, code)
                if err != nil {
                    errors <- err
                    continue
                }

                results <- entities
            }
        }()
    }

    // Collect results
    go func() {
        wg.Wait()
        close(results)
        close(errors)
    }()

    allEntities := []CodeEntity{}
    for entities := range results {
        allEntities = append(allEntities, entities...)
    }

    // Check for errors
    select {
    case err := <-errors:
        return allEntities, err
    default:
        return allEntities, nil
    }
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestParseGoFile(t *testing.T) {
    code := []byte(`
package main

import "fmt"

func Hello(name string) string {
    return fmt.Sprintf("Hello, %s!", name)
}

type User struct {
    Name string
}
`)

    entities, err := ParseFile("test.go", code)
    assert.NoError(t, err)

    // Verify entities
    assert.Len(t, entities, 4) // file, import, function, struct

    // Check function
    funcEntity := findEntity(entities, "function", "Hello")
    assert.NotNil(t, funcEntity)
    assert.Equal(t, "func Hello(name string) string", funcEntity.Signature)
}
```

### Integration Tests

1. Parse omnara-ai/omnara codebase
2. Verify node counts match expected
3. Check relationships (CONTAINS, IMPORTS)
4. Validate query performance (<10s total)

---

## Estimated Performance

| Repository | Files | Parsing Time | Nodes | Edges | Graph Insert |
|------------|-------|--------------|-------|-------|--------------|
| omnara | ~1,000 | 5s | 10K | 20K | 2s |
| kubernetes | ~50,000 | 4min | 500K | 1M | 30s |

**Total Layer 1 Init Time:**
- omnara: ~7s ✅
- kubernetes: ~4.5min ✅

---

## Next Steps

1. ✅ **Design complete** - Tree-sitter integration strategy validated
2. ⏭️ **Implement parsers** - `internal/parser/` for each language
3. ⏭️ **Implement graph builder** - `internal/graph/layer1.go`
4. ⏭️ **Test with omnara** - Validate end-to-end
5. ⏭️ **Optimize for kubernetes** - Parallel processing, batching

---

## References

- **go-tree-sitter:** https://github.com/tree-sitter/go-tree-sitter
- **Graph Ontology:** [graph_ontology.md](../../01-architecture/graph_ontology.md)
- **Local Deployment:** [local_deployment.md](local_deployment.md)
- **Test Data:** [test_data/github_api/omnara-ai-omnara/](../../../test_data/github_api/omnara-ai-omnara/)
