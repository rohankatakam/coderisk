# Tree-sitter Library Evaluation for CodeRisk Layer 1

**Date:** October 3, 2025
**Status:** Recommended for implementation
**Decision:** Proceed with `github.com/tree-sitter/go-tree-sitter` for AST parsing

---

## Executive Summary

Tree-sitter is the **industry-standard** incremental parsing library for building abstract syntax trees (ASTs). It is the **correct choice** for CodeRisk Layer 1 implementation based on:

✅ **Performance:** Designed to parse on every keystroke in text editors (sub-millisecond updates)
✅ **Robustness:** Handles syntax errors gracefully, providing useful results even with incomplete code
✅ **Language Support:** Official parsers for Go, Python, JavaScript, TypeScript, Java, and 40+ languages
✅ **Production Battle-tested:** Used by GitHub (code navigation), Neovim, Atom, and major IDEs
✅ **Query System:** Powerful S-expression query language for extracting specific code patterns

---

## What is Tree-sitter?

**Official Definition:** "A parser generator tool and an incremental parsing library"

**Core Capabilities:**
1. **Concrete Syntax Trees:** Builds detailed ASTs representing complete code structure
2. **Incremental Parsing:** Updates trees efficiently as code changes (not needed for v1.0)
3. **Error Recovery:** Continues parsing even with syntax errors (critical for real-world code)
4. **Language-Agnostic:** Single API works across all programming languages

**Technical Design:**
- Runtime library: Pure C11 (fast, embeddable, dependency-free)
- Language parsers: Separate packages (modular, only import what you need)
- Go bindings: `github.com/tree-sitter/go-tree-sitter` (official, well-maintained)

---

## How Tree-sitter Works

### Basic Workflow

```go
// 1. Create parser
parser := tree_sitter.NewParser()
defer parser.Close()

// 2. Load language grammar
parser.SetLanguage(tree_sitter.NewLanguage(tree_sitter_javascript.Language()))

// 3. Parse source code
tree := parser.Parse(code, nil)
defer tree.Close()

// 4. Traverse syntax tree
root := tree.RootNode()
for i := 0; i < root.ChildCount(); i++ {
    child := root.Child(i)
    processNode(child)
}
```

### Node Structure

Each `Node` provides:
- **Type Information:** `Kind()` returns node type (e.g., "function_declaration", "class_definition")
- **Position:** `StartByte()`, `EndByte()`, `StartPosition()`, `EndPosition()`
- **Content:** Extract source text via byte offsets
- **Relationships:** `Parent()`, `Child(i)`, `ChildCount()`, `NextSibling()`
- **Metadata:** `IsNamed()`, `IsError()`, `IsMissing()`

### Query System

Tree-sitter includes a powerful query language (S-expressions) for pattern matching:

```scheme
;; Find all function definitions
(function_declaration
  name: (identifier) @function.name
  parameters: (formal_parameters) @function.params)

;; Find all import statements
(import_statement
  source: (string) @import.path)
```

**Go API:**
```go
query, _ := tree_sitter.NewQuery(language, querySource)
cursor := tree_sitter.NewQueryCursor()
defer cursor.Close()

cursor.Exec(query, rootNode)
for {
    match, ok := cursor.NextMatch()
    if !ok {
        break
    }
    // Process captures
    for _, capture := range match.Captures {
        node := capture.Node
        // Extract function name, parameters, etc.
    }
}
```

---

## Language Support

### Official Parsers (v1.0 Priority)

| Language | Package | Repository | Use Case |
|----------|---------|------------|----------|
| **Go** | `tree-sitter/tree-sitter-go` | https://github.com/tree-sitter/tree-sitter-go | Backend services |
| **Python** | `tree-sitter/tree-sitter-python` | https://github.com/tree-sitter/tree-sitter-python | ML/Data pipelines |
| **JavaScript** | `tree-sitter/tree-sitter-javascript` | https://github.com/tree-sitter/tree-sitter-javascript | Frontend apps |
| **TypeScript** | `tree-sitter/tree-sitter-typescript` | https://github.com/tree-sitter/tree-sitter-typescript | Modern frontend |
| **Java** | `tree-sitter/tree-sitter-java` | https://github.com/tree-sitter/tree-sitter-java | Enterprise apps |

**Usage:**
```bash
go get github.com/tree-sitter/go-tree-sitter@latest
go get github.com/tree-sitter/tree-sitter-go@latest
go get github.com/tree-sitter/tree-sitter-python@latest
go get github.com/tree-sitter/tree-sitter-javascript@latest
go get github.com/tree-sitter/tree-sitter-typescript/tree_sitter_typescript@latest
go get github.com/tree-sitter/tree-sitter-java@latest
```

### Future Languages (v2.0)

- Rust: `tree-sitter/tree-sitter-rust`
- C/C++: `tree-sitter/tree-sitter-c`, `tree-sitter/tree-sitter-cpp`
- Ruby: `tree-sitter/tree-sitter-ruby`
- PHP: `tree-sitter/tree-sitter-php`

**Total Available:** 40+ language parsers maintained by Tree-sitter community

---

## Memory Management (CRITICAL)

**⚠️ Important:** Go bindings use CGO, requiring explicit cleanup

**Must call `Close()` on:**
- `Parser`
- `Tree`
- `TreeCursor`
- `Query`
- `QueryCursor`
- `LookaheadIterator`

**Reason:** Bug in `runtime.SetFinalizer` with CGO prevents automatic garbage collection

**Pattern:**
```go
parser := tree_sitter.NewParser()
defer parser.Close()  // REQUIRED

tree := parser.Parse(code, nil)
defer tree.Close()    // REQUIRED

// ... use tree
```

---

## Performance Characteristics

### Parsing Speed
- **Design Goal:** Parse on every keystroke in text editor
- **Real-world:** 10-50ms for typical files (1K-5K lines)
- **Large files:** 100-200ms for 50K lines
- **Incremental:** Sub-millisecond updates (not using in v1.0)

### Memory Usage
- **Small files (<1K lines):** ~1MB per file
- **Large files (10K lines):** ~10MB per file
- **Optimization:** Parse and discard trees immediately (don't keep all trees in memory)

### CodeRisk Targets (from scalability_analysis.md)
- **omnara-ai/omnara (~1K files):** <10s total parsing → **Achievable** (10ms/file avg)
- **kubernetes/kubernetes (~50K files):** <5min total parsing → **Achievable** (6ms/file avg)

### Bottlenecks
- **Not parsing:** File I/O, tree traversal, graph writes dominate
- **Solution:** Concurrent parsing (worker pool pattern)

---

## Alternatives Considered

### 1. Language-Specific Tools
**Go:** `go/ast`, `go/parser` (standard library)
- ✅ Native Go, no CGO
- ❌ Only parses Go (need 5+ languages)
- ❌ Inconsistent APIs across languages

**Python:** `ast` module
- ✅ Built-in
- ❌ Only parses Python
- ❌ Requires embedding Python interpreter in Go

**Decision:** Rejected due to multi-language requirement

### 2. Universal Parsers
**srcml** (http://www.srcml.org)
- ✅ Supports multiple languages
- ❌ XML output (slow to parse)
- ❌ No Go bindings

**Decision:** Rejected due to poor Go integration

### 3. LSP (Language Server Protocol)
- ✅ Semantic understanding (type info, references)
- ❌ Requires language-specific servers (Go: gopls, Python: pyright, etc.)
- ❌ Complex setup, process management
- ❌ Overkill for v1.0 (only need structure, not semantics)

**Decision:** Deferred to v2.0 for enhanced analysis

---

## Recommendation: Tree-sitter is the Right Choice

### Strengths
1. ✅ **Industry Standard:** Used by GitHub, Neovim, Atom, major IDEs
2. ✅ **Multi-language:** Single API for 40+ languages
3. ✅ **Performance:** Designed for real-time parsing
4. ✅ **Robustness:** Handles syntax errors gracefully
5. ✅ **Query System:** Powerful pattern matching for extracting entities
6. ✅ **Go Bindings:** Official, well-maintained
7. ✅ **Production-Ready:** Battle-tested in GitHub code navigation

### Weaknesses (Mitigated)
1. ⚠️ **CGO Dependency:** Requires explicit `Close()` calls
   - **Mitigation:** Use `defer` pattern religiously (enforced by code review)

2. ⚠️ **Grammar Learning Curve:** Each language has unique AST structure
   - **Mitigation:** Use existing integration guide examples (layer_1_treesitter.md)

3. ⚠️ **No Semantic Analysis:** Only syntax structure (no type info, cross-file references)
   - **Mitigation:** Not needed for v1.0 Layer 1 (structure only)
   - **Future:** Add LSP in v2.0 for semantic analysis

### Comparison to Requirements

| Requirement | Tree-sitter | Alternative |
|-------------|-------------|-------------|
| Multi-language support | ✅ 40+ languages | ❌ Language-specific |
| Performance (<10s for 1K files) | ✅ ~10ms/file | ✅ Similar |
| Error handling | ✅ Graceful | ❌ Strict parsing |
| Go integration | ✅ Official bindings | ❌ Poor/none |
| Production-ready | ✅ GitHub, IDEs | ❌ Experimental |
| Query system | ✅ Powerful S-expressions | ❌ Manual traversal |

**Verdict:** Tree-sitter meets all requirements with no viable alternatives

---

## Implementation Plan

### Phase 1: Dependencies (5 min)
```bash
go get github.com/tree-sitter/go-tree-sitter@latest
go get github.com/tree-sitter/tree-sitter-go@latest
go get github.com/tree-sitter/tree-sitter-python@latest
go get github.com/tree-sitter/tree-sitter-javascript@latest
go get github.com/tree-sitter/tree-sitter-typescript/tree_sitter_typescript@latest
go get github.com/tree-sitter/tree-sitter-java@latest
```

### Phase 2: Core Parsing (2 hours)
- `internal/treesitter/parser.go`: Multi-language parser factory
- `internal/treesitter/extractors/`: Language-specific entity extractors

### Phase 3: Entity Extraction (4 hours)
- Go: Functions, methods, structs, interfaces, imports
- Python: Functions, classes, methods, imports
- JS/TS: Functions, classes, imports, exports
- Java: Classes, methods, imports

### Phase 4: Graph Construction (2 hours)
- File nodes
- Function/Class nodes
- CALLS edges (function → function)
- IMPORTS edges (file → file)

### Phase 5: Testing & Validation (2 hours)
- Unit tests with sample files
- Integration test with omnara-ai/omnara
- Performance validation (<10s target)

**Total Estimate:** ~1 day (8 hours)

---

## References

- **Official Docs:** https://tree-sitter.github.io/tree-sitter/
- **Go Bindings:** https://github.com/tree-sitter/go-tree-sitter
- **Language Parsers:** https://github.com/tree-sitter
- **Query Syntax:** https://tree-sitter.github.io/tree-sitter/using-parsers#pattern-matching-with-queries
- **Integration Guide:** [dev_docs/03-implementation/integration_guides/layer_1_treesitter.md](../03-implementation/integration_guides/layer_1_treesitter.md)

---

## Next Steps

1. ✅ **Decision Made:** Use Tree-sitter for Layer 1
2. ⏭️ **Implementation:** Follow [layer_1_treesitter.md](../03-implementation/integration_guides/layer_1_treesitter.md)
3. ⏭️ **Testing:** Validate on omnara-ai/omnara
4. ⏭️ **Production:** Deploy Layer 1 → enable Phase 1 metrics
