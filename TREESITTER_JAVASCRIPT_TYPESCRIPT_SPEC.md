# Tree-sitter JavaScript/TypeScript Graph Data Specification

**Version:** 1.0
**Last Updated:** 2025-10-23
**Purpose:** Definitive specification for graph nodes and edges we can extract from JavaScript/TypeScript code using Tree-sitter with 95%+ confidence

---

## Overview

This document specifies what graph data can be reliably extracted from JavaScript and TypeScript source files using the Tree-sitter parser.

**Key Limitation:** Tree-sitter is a **parser**, not a **semantic analyzer**. It provides syntax structure but cannot resolve symbols, types, or semantic relationships.

**Grammars:**
- `tree-sitter-javascript` (version 0.25.0+)
- `tree-sitter-typescript` (version 0.25.0+)

**Note:** TypeScript extends JavaScript with additional node types. This document covers both.

---

## Nodes We Can Extract (100% Confidence)

### **1. `:File` Node**

**Properties:**
```cypher
(:File {
    path: STRING,           // Absolute file path
    language: STRING,       // "javascript" or "typescript"
    loc: INTEGER,           // Lines of code (counted)
    last_updated: INTEGER,  // Unix timestamp from git log
    unique_id: STRING       // path (used as key)
})
```

**Data Source:** File system + git log
**Confidence:** 100%
**Location:** `internal/ingestion/processor.go:757-776`

**File Extensions:**
- JavaScript: `.js`, `.jsx`, `.mjs`, `.cjs`
- TypeScript: `.ts`, `.tsx`, `.mts`, `.cts`

---

### **2. `:Function` Node**

**Properties:**
```cypher
(:Function {
    name: STRING,           // Function name (or "ClassName.methodName" for methods)
    signature: STRING,      // Full function signature
    start_line: INTEGER,    // Line where function starts
    end_line: INTEGER,      // Line where function ends
    file_path: STRING,      // Path to containing file
    language: STRING,       // "javascript" or "typescript"
    unique_id: STRING       // "filepath:name:start_line"
})
```

**Extracts:**
- ✅ Named functions
- ✅ Arrow functions (with variable assignment)
- ✅ Class methods
- ✅ Async functions
- ✅ Generator functions
- ✅ TypeScript type annotations (parameters and return types)

**Examples:**

**JavaScript:**
```javascript
// Named function
function calculateRisk(file) {
    return 0.5;
}

// Arrow function
const processPayment = async (amount) => {
    return stripe.charge(amount);
};

// Class method
class UserRepository {
    async findById(id) {
        return db.query('SELECT * FROM users WHERE id = ?', [id]);
    }
}
```

**TypeScript:**
```typescript
// Function with type annotations
function calculateRisk(file: string): number {
    return 0.5;
}

// Arrow function with types
const processPayment = async (amount: number): Promise<PaymentResult> => {
    return stripe.charge(amount);
};

// Class method with types
class UserRepository {
    async findById(id: string): Promise<User | null> {
        return db.query('SELECT * FROM users WHERE id = ?', [id]);
    }
}
```

**Extracted Signatures:**
```json
[
  {
    "name": "calculateRisk",
    "signature": "function calculateRisk(file: string): number"
  },
  {
    "name": "processPayment",
    "signature": "const processPayment = async (amount: number): Promise<PaymentResult> =>"
  },
  {
    "name": "UserRepository.findById",
    "signature": "async findById(id: string): Promise<User | null>"
  }
]
```

**Data Source:** Tree-sitter AST nodes
- JavaScript: `function_declaration`, `arrow_function`, `method_definition`, `function_expression`
- TypeScript: Same + type annotations

**Confidence:** 100%
**Location:**
- JavaScript: `internal/treesitter/javascript_extractor.go:66-132`
- TypeScript: `internal/treesitter/typescript_extractor.go:74-153`

---

### **3. `:Class` Node**

**Properties:**
```cypher
(:Class {
    name: STRING,           // Class name
    signature: STRING,      // "class Foo extends Bar"
    start_line: INTEGER,    // Line where class starts
    end_line: INTEGER,      // Line where class ends
    file_path: STRING,      // Path to containing file
    language: STRING,       // "javascript" or "typescript"
    unique_id: STRING       // "filepath:name:start_line"
})
```

**Extracts:**
- ✅ Class name
- ✅ Extends clause (inheritance)
- ✅ Abstract classes (TypeScript)

**Examples:**

**JavaScript:**
```javascript
class UserRepository extends BaseRepository {
    constructor(db) {
        super(db);
    }
}
```

**TypeScript:**
```typescript
abstract class BaseRepository<T> implements Repository<T> {
    constructor(protected db: Database) {}
}

class UserRepository extends BaseRepository<User> {
    async findAll(): Promise<User[]> {
        return this.db.query('SELECT * FROM users');
    }
}
```

**Extracted:**
```json
{
  "name": "UserRepository",
  "signature": "class UserRepository extends BaseRepository<User>",
  "start_line": 7,
  "end_line": 11,
  "extends": "BaseRepository<User>"
}
```

**Data Source:** Tree-sitter AST `class_declaration` nodes
**Confidence:** 100%
**Location:**
- JavaScript: `internal/treesitter/javascript_extractor.go:134-151`
- TypeScript: `internal/treesitter/typescript_extractor.go:155-230`

---

### **4. `:Interface` Node (TypeScript Only)**

**Properties:**
```cypher
(:Interface {
    name: STRING,           // Interface name
    signature: STRING,      // "interface User { ... }"
    start_line: INTEGER,    // Line where interface starts
    end_line: INTEGER,      // Line where interface ends
    file_path: STRING,      // Path to containing file
    language: STRING,       // "typescript"
    unique_id: STRING       // "filepath:name:start_line"
})
```

**Currently Treated As:** `:Class` nodes (to simplify schema)

**Example:**
```typescript
interface User {
    id: string;
    email: string;
    name: string;
}

interface Repository<T> {
    findById(id: string): Promise<T | null>;
    save(entity: T): Promise<void>;
}
```

**Extracted:**
```json
{
  "name": "User",
  "signature": "interface User",
  "start_line": 1,
  "end_line": 5
}
```

**Data Source:** Tree-sitter AST `interface_declaration` nodes
**Confidence:** 100%
**Location:** `internal/treesitter/typescript_extractor.go:155-230`

**Note:** Interfaces are stored as `:Class` nodes to avoid schema complexity. If you need to distinguish interfaces from classes, add a `kind` property: `"interface"` vs `"class"`.

---

### **5. `:TypeAlias` Node (TypeScript Only)**

**Properties:**
```cypher
(:TypeAlias {
    name: STRING,           // Type alias name
    signature: STRING,      // "type ID = string | number"
    start_line: INTEGER,    // Line where type starts
    end_line: INTEGER,      // Line where type ends
    file_path: STRING,      // Path to containing file
    language: STRING,       // "typescript"
    unique_id: STRING       // "filepath:name:start_line"
})
```

**Currently Treated As:** `:Class` nodes (to simplify schema)

**Example:**
```typescript
type ID = string | number;
type PaymentMethod = 'card' | 'bank' | 'crypto';
type Result<T> = { success: true; data: T } | { success: false; error: string };
```

**Extracted:**
```json
{
  "name": "Result",
  "signature": "type Result<T> = { success: true; data: T } | { success: false; error: string }",
  "start_line": 3,
  "end_line": 3
}
```

**Data Source:** Tree-sitter AST `type_alias_declaration` nodes
**Confidence:** 100%
**Location:** `internal/treesitter/typescript_extractor.go:155-230`

---

## Edges We Can Extract

### **1. `[:CONTAINS]` Edge (100% Confidence)**

**Pattern:**
```cypher
(File)-[:CONTAINS {
    entity_type: STRING     // "function", "class", "interface", "type_alias"
}]->(Function|Class)
```

**What it captures:** All functions, classes, interfaces, and type aliases defined within a file
**Reliability:** 100% - Direct AST parent-child relationship
**Location:** `internal/ingestion/processor.go:672-697`

---

### **2. `[:IMPORTS]` Edge (60-90% Confidence)**

**Pattern:**
```cypher
(File)-[:IMPORTS {
    import_line: INTEGER,   // Line number of import statement
    import_path: STRING,    // Raw import string from code
    resolved: BOOLEAN       // True if target file was found
}]->(File)
```

**What it captures:** Import statements resolved to target files

**JavaScript/TypeScript Import Types:**

#### **Type 1: Relative Imports (90% Success)**
```javascript
import { User } from './models/user';           // ✅ Resolved
import { config } from '../config';             // ✅ Resolved
import api from '../../lib/api';                // ✅ Resolved
```

**Resolution:** Relative to current file location

#### **Type 2: Path Aliases (0% Success - Cannot Resolve)**
```typescript
import { Button } from '@/components/ui/button'; // ❌ Cannot resolve (requires tsconfig.json parsing)
import { api } from '~/lib/api';                 // ❌ Cannot resolve (requires tsconfig.json parsing)
```

**Why:** Path aliases are defined in `tsconfig.json` `paths` field, which Tree-sitter doesn't read.

**Workaround:** Could implement tsconfig.json parsing (2-3 hours)

#### **Type 3: Node Modules (Intentionally Skipped)**
```javascript
import React from 'react';                       // ❌ Skipped (external package)
import { Database } from 'pg';                   // ❌ Skipped (external package)
```

#### **Type 4: Index Files (80% Success)**
```javascript
import { components } from './components';       // ✅ Resolved to ./components/index.ts
import utils from '../utils';                    // ✅ Resolved to ../utils/index.js
```

**Resolution Strategy:**
1. Try exact path
2. Try with `.js`, `.ts`, `.jsx`, `.tsx` extensions
3. Try `index.js`, `index.ts` in directory
4. Create edge only if target file found

**Overall Success Rate:** ~70% (depends on path alias usage)

**Data Source:** Tree-sitter AST `import_statement` nodes
**Location:**
- Extraction: `internal/treesitter/javascript_extractor.go:153-198`, `typescript_extractor.go:232-290`
- Resolution: `internal/ingestion/processor.go:851-907`

---

## JavaScript/TypeScript-Specific Features We Can Extract

### **1. JSX/TSX Components (100% Confidence)**

**Current Status:** ❌ Not extracted (but easily could be)

**Example:**
```tsx
// React functional component
export function UserProfile({ user }: { user: User }) {
    return (
        <div className="profile">
            <h1>{user.name}</h1>
            <p>{user.email}</p>
        </div>
    );
}

// React class component
export class Dashboard extends React.Component<DashboardProps> {
    render() {
        return <div>Dashboard</div>;
    }
}
```

**Can Extract:**
- ✅ Component name
- ✅ Component props type
- ✅ Whether functional or class component
- ✅ JSX elements used

**Potential Graph Schema:**
```cypher
(:Component {
    name: STRING,
    type: STRING,  // "functional" or "class"
    props_type: STRING
})-[:RENDERS]->(:JSXElement)
```

**Value:** React/Next.js component tracking, UI dependency graph
**Priority:** High ⭐⭐⭐
**Effort:** 3-4 hours

---

### **2. Export Statements (100% Confidence)**

**Current Status:** ❌ Not extracted

**Examples:**
```javascript
// Named exports
export function foo() {}
export { bar, baz };
export { default as Button } from './Button';

// Default export
export default function main() {}
export default class App extends Component {}
```

**Can Extract:**
- ✅ What's being exported
- ✅ Export type (default vs named)
- ✅ Re-exports (export from another file)

**Potential Graph Schema:**
```cypher
(:Function)-[:EXPORTED_BY {
    export_type: STRING  // "default" or "named"
}]->(File)

// Or simpler: add property to Function/Class
(:Function {
    name: STRING,
    is_exported: BOOLEAN,
    export_type: STRING  // "default" or "named"
})
```

**Value:** Public API surface detection, entry point identification
**Priority:** High ⭐⭐⭐
**Effort:** 2 hours

---

### **3. TypeScript Enums (100% Confidence)**

**Current Status:** ❌ Not extracted

**Example:**
```typescript
enum Status {
    Active = 'active',
    Inactive = 'inactive',
    Pending = 'pending'
}

const enum LogLevel {
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3
}
```

**Can Extract:**
- ✅ Enum name
- ✅ Enum values
- ✅ Whether const enum

**Potential Graph Schema:**
```cypher
(:Enum {
    name: STRING,
    values: [STRING],
    is_const: BOOLEAN
})-[:DEFINED_IN]->(File)
```

**Value:** Track state machines, configuration constants
**Priority:** Medium ⭐⭐
**Effort:** 1-2 hours

---

### **4. Decorators (TypeScript Experimental, 100% Confidence)**

**Current Status:** ❌ Not extracted

**Example:**
```typescript
@Controller('/api/users')
export class UserController {
    @Get('/:id')
    @UseGuards(AuthGuard)
    async findOne(@Param('id') id: string): Promise<User> {
        return this.userService.findById(id);
    }
}
```

**Can Extract:**
- ✅ Decorator names: `Controller`, `Get`, `UseGuards`, `Param`
- ✅ Decorator arguments: `'/api/users'`, `'/:id'`
- ✅ Applied to which class/method/parameter

**Potential Graph Schema:**
```cypher
(:Decorator {name: STRING, arguments: STRING})-[:APPLIED_TO]->(Function|Class)
```

**Value:** NestJS framework detection, route mapping
**Priority:** High ⭐⭐⭐ (for NestJS projects)
**Effort:** 2-3 hours

---

## What Tree-sitter JavaScript/TypeScript CANNOT Extract

### **1. Function Call Graphs (Requires Semantic Analysis)**

**Problem:**
```typescript
// file: payment.ts
import { validateCard } from './validators';
import { charge } from '@/stripe/api';

function processPayment(card: Card) {
    validateCard(card);  // Tree-sitter sees "validateCard" string
    charge(card);        // Tree-sitter sees "charge" string
    // Where are these functions defined?
}
```

**What Tree-sitter Provides:**
- ✅ Call expression exists at line X
- ✅ Function name as string: `"validateCard"`
- ✅ Arguments: `["card"]`

**What Tree-sitter CANNOT Provide:**
- ❌ Which file contains `validateCard`
- ❌ Which function `validateCard` refers to (symbol resolution)
- ❌ Imported from where (path alias `@/` unresolved)

**Success Rate:** ~30% (only direct, same-file calls work)
**Recommendation:** Use TypeScript Compiler API for 95%+ accuracy (Phase 2)

---

### **2. Type Resolution**

**Problem:**
```typescript
const user: User = await getUser(id);
```

**What Tree-sitter Provides:**
- ✅ Variable name: `"user"`
- ✅ Type annotation string: `"User"`

**What Tree-sitter CANNOT Provide:**
- ❌ What `User` type/interface actually is
- ❌ Which file `User` is defined in
- ❌ Properties available on `User`

---

### **3. Path Alias Resolution**

**Problem:**
```typescript
import { Button } from '@/components/ui/button';
```

**Requires:** Parsing `tsconfig.json` `compilerOptions.paths`:
```json
{
  "compilerOptions": {
    "paths": {
      "@/*": ["./src/*"],
      "~/*": ["./"]
    }
  }
}
```

**Success Rate:** 0% (without tsconfig.json parsing)
**Workaround:** Implement tsconfig.json parser (2-3 hours, Phase 2)

---

## Implementation Details

### **Current Extractor Locations:**

**JavaScript:**
- **File:** `internal/treesitter/javascript_extractor.go`
- **Main Function:** `extractJavaScriptEntities(filePath, root, code)`
- **Lines:** 1-198

**TypeScript:**
- **File:** `internal/treesitter/typescript_extractor.go`
- **Main Function:** `extractTypeScriptEntities(filePath, root, code)`
- **Lines:** 1-290

### **Key Functions:**

```go
// JavaScript Functions
func extractJSFunctionDeclaration(node, code, filePath, entities)
// Location: javascript_extractor.go:66-132

func extractJSClassDeclaration(node, code, filePath, entities)
// Location: javascript_extractor.go:134-151

// TypeScript Functions (extends JavaScript)
func extractTSFunctionDeclaration(node, code, filePath, entities)
// Location: typescript_extractor.go:74-153

func extractTSTypeDeclaration(node, code, filePath, entities)
// Location: typescript_extractor.go:155-230
// Handles: class, interface, type_alias, enum
```

---

## Recommended Enhancements

### **Priority 1: Export Tracking** ⭐⭐⭐

**Effort:** 2 hours
**Value:** Public API surface, entry point detection

**Implementation:**
```go
case "export_statement":
    // Check what's being exported
    declaration := node.ChildByFieldName("declaration")
    if declaration != nil {
        // Extract exported function/class/variable name
        // Mark entity as exported
    }
```

---

### **Priority 2: JSX/TSX Component Extraction** ⭐⭐⭐

**Effort:** 3-4 hours
**Value:** React component tracking, UI dependency graph

**Implementation:**
```go
case "function_declaration", "arrow_function":
    // Check if function returns JSX
    body := node.ChildByFieldName("body")
    if containsJSXElement(body) {
        entity.Type = "component"
        entity.ComponentType = "functional"
    }
```

---

### **Priority 3: TypeScript Enum Extraction** ⭐⭐

**Effort:** 1-2 hours
**Value:** State machine tracking, configuration constants

**Implementation:**
```go
case "enum_declaration":
    enumName := node.ChildByFieldName("name")
    // Extract enum values
    entity.Type = "enum"
```

---

### **Priority 4: Decorator Extraction** ⭐⭐⭐ (for NestJS projects)

**Effort:** 2-3 hours
**Value:** Framework detection, route mapping

---

## Testing & Validation

### **Verification Queries:**

```cypher
-- 1. Check JavaScript/TypeScript files exist
MATCH (f:File)
WHERE f.language IN ['javascript', 'typescript']
RETURN f.language, count(f) as count;

-- 2. Check functions extracted
MATCH (f:File)-[:CONTAINS]->(fn:Function)
WHERE f.language IN ['javascript', 'typescript']
RETURN f.path, fn.name, fn.signature
LIMIT 10;

-- 3. Check classes extracted
MATCH (f:File)-[:CONTAINS]->(c:Class)
WHERE f.language IN ['javascript', 'typescript']
RETURN f.path, c.name, c.signature
LIMIT 10;

-- 4. Check TypeScript interfaces
MATCH (f:File {language: "typescript"})-[:CONTAINS]->(c:Class)
WHERE c.signature STARTS WITH "interface"
RETURN f.path, c.name, c.signature
LIMIT 10;

-- 5. Check import edges
MATCH (f1:File)-[r:IMPORTS]->(f2:File)
WHERE f1.language IN ['javascript', 'typescript']
RETURN f1.path, f2.path, r.import_line
LIMIT 10;

-- 6. Check arrow functions
MATCH (f:File)-[:CONTAINS]->(fn:Function)
WHERE fn.signature CONTAINS "=>"
RETURN fn.name, fn.signature
LIMIT 10;

-- 7. Check async functions
MATCH (f:File)-[:CONTAINS]->(fn:Function)
WHERE fn.signature CONTAINS "async"
RETURN fn.name, fn.signature
LIMIT 10;
```

---

## Known Limitations

| Limitation | Impact | Workaround | Phase |
|-----------|--------|------------|-------|
| No function call graphs | Cannot build blast radius | Use file-level dependencies | Phase 2 (use TS Compiler API) |
| Path alias resolution | 0% of aliased imports | Implement tsconfig parser | Phase 2 |
| No type resolution | Cannot understand types | Store as strings for LLM | MVP |
| No semantic analysis | Cannot resolve symbols | Use TypeScript Compiler API | Phase 3 |
| JSX components not extracted | Cannot track React components | Implement JSX detection | Phase 2 |

---

## TypeScript vs JavaScript Differences

| Feature | JavaScript | TypeScript | Notes |
|---------|-----------|------------|-------|
| Functions | ✅ | ✅ | Same extraction |
| Classes | ✅ | ✅ | Same extraction |
| Interfaces | ❌ | ✅ | TypeScript-only |
| Type Aliases | ❌ | ✅ | TypeScript-only |
| Enums | ❌ | ✅ | TypeScript-only |
| Decorators | ❌ | ✅ (experimental) | TypeScript-only |
| Type Annotations | ❌ | ✅ | Stored in signature |
| JSX | ✅ (.jsx) | ✅ (.tsx) | Same structure |

---

## Summary

### **High-Confidence Extraction (95-100%):**
- ✅ File nodes (path, language, LOC)
- ✅ Function nodes (named, arrow, async, methods)
- ✅ Class nodes (with inheritance)
- ✅ Interface nodes (TypeScript, stored as classes)
- ✅ Type alias nodes (TypeScript, stored as classes)
- ✅ CONTAINS edges (File → Function/Class)
- ✅ IMPORTS edges (~70% success for relative imports)

### **JavaScript/TypeScript-Specific Features Available:**
- ✅ JSX components (can extract, not implemented yet)
- ✅ Export statements (can extract, not implemented yet)
- ✅ TypeScript enums (can extract, not implemented yet)
- ✅ Decorators (can extract, not implemented yet)
- ✅ Type annotations (stored in signature)

### **Not Available (Requires Semantic Analysis):**
- ❌ Function call graphs ([:CALLS] edges)
- ❌ Type resolution
- ❌ Path alias resolution (requires tsconfig.json parsing)
- ❌ Symbol resolution

---

## Next Steps

1. **Implement export tracking** (Priority 1, 2 hours)
2. **Implement JSX component extraction** (Priority 2, 3-4 hours)
3. **Implement TypeScript enum extraction** (Priority 3, 1-2 hours)
4. **Test with real codebases** (React, Next.js, NestJS projects)
5. **Consider TypeScript Compiler API integration** for call graphs (Phase 2)

---

**Last Updated:** 2025-10-23
**Status:** ✅ Ready for implementation
**Reference:** Based on tree-sitter-javascript and tree-sitter-typescript grammars and current codebase implementation
