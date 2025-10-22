# Three-Layer Architecture Integration Tests

These tests verify that all three layers of the CodeRisk graph architecture are functional.

## Prerequisites

1. **Docker services running:**
   ```bash
   cd /Users/rohankatakam/Documents/brain/coderisk
   docker compose up -d
   ```

2. **Test repository:**
   ```bash
   cd /tmp
   rm -rf test-repo
   mkdir test-repo && cd test-repo
   git init

   cat > example.py << 'EOF'
   """Example Python module for testing"""
   import os
   import sys
   from typing import List

   def hello_world():
       """Simple hello world function"""
       print("Hello, World!")
       return True

   def process_data(items: List[str]) -> dict:
       """Process a list of items"""
       result = {}
       for item in items:
           result[item] = len(item)
       return result

   class Calculator:
       """Basic calculator class"""

       def __init__(self):
           self.history = []

       def add(self, a, b):
           """Add two numbers"""
           result = a + b
           self.history.append(('add', a, b, result))
           return result

       def subtract(self, a, b):
           """Subtract b from a"""
           result = a - b
           self.history.append(('subtract', a, b, result))
           return result

       def get_history(self):
           """Return calculation history"""
           return self.history

   if __name__ == "__main__":
       hello_world()
       calc = Calculator()
       print(calc.add(5, 3))
   EOF

   git add . && git commit -m "Initial commit"
   git add . && git commit -m "Add functions and Calculator class"
   ```

## Running the Tests

### Test All Three Layers

```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Layer 1: Structure (Tree-sitter)
go run test/integration/scripts/test_layer1_structure.go

# Layer 2: Temporal (Git History)
go run test/integration/scripts/test_layer2_temporal.go

# Layer 3: Graph (Neo4j)
go run test/integration/scripts/test_layer3_graph.go
```

### Expected Output

#### Layer 1: Structure
```
✅ Layer 1 Test: PASSED
   - Parsed file: /tmp/test-repo/example.py
   - Language: python
   - Entities found: 11
     • Functions: 6
     • Classes: 1
     • Imports: 3
```

#### Layer 2: Temporal
```
✅ Layer 2 Test: PASSED
   - Git history parsed successfully
   - Commits found: 2
   - Unique developers: 1
   - Co-change pairs: 0
```

#### Layer 3: Graph
```
✅ Layer 3 Test: PASSED
   - Connected to Neo4j successfully
   - Created test node (File type)
   - Created CONTAINS edge
   - Test data cleaned up
```

## What Each Test Validates

### Layer 1: Structure (Tree-sitter AST Parsing)
- **File:** `test_layer1_structure.go`
- **Tests:**
  - Tree-sitter can parse Python files
  - Functions are extracted with correct line numbers
  - Classes are extracted with methods
  - Imports are detected
- **Technology:** Tree-sitter Go bindings
- **Package:** `internal/treesitter`

### Layer 2: Temporal (Git History Analysis)
- **File:** `test_layer2_temporal.go`
- **Tests:**
  - Git history can be parsed
  - Developers are extracted from commits
  - Co-change patterns are calculated
  - Ownership tracking works
- **Technology:** Git CLI + Go git parsing
- **Package:** `internal/temporal`

### Layer 3: Graph (Neo4j Integration)
- **File:** `test_layer3_graph.go`
- **Tests:**
  - Neo4j connection works
  - Nodes can be created (File, Function types)
  - Edges can be created (CONTAINS relationships)
  - Queries return results
  - Data can be cleaned up
- **Technology:** Neo4j Go Driver
- **Package:** `internal/graph`

## Troubleshooting

### Neo4j Connection Failed
```
❌ Could not connect to Neo4j: connection refused
```

**Solution:**
```bash
docker compose up -d neo4j
docker logs coderisk-neo4j  # Check for errors
```

### Test Repository Not Found
```
❌ Test file not found: /tmp/test-repo/example.py
```

**Solution:** Create test repo following Prerequisites section above

### Authentication Failed
```
❌ Neo4j authentication failure
```

**Solution:** Check password in `docker-compose.yml`:
```yaml
NEO4J_AUTH: neo4j/CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
```

Update test files to use correct password.

## Integration with CI/CD

These tests can be automated in CI/CD pipelines:

```bash
# Start services
docker compose up -d

# Wait for services to be ready
sleep 15

# Create test repo
./scripts/create_test_repo.sh

# Run tests
go run test/integration/scripts/test_layer1_structure.go || exit 1
go run test/integration/scripts/test_layer2_temporal.go || exit 1
go run test/integration/scripts/test_layer3_graph.go || exit 1

echo "✅ All three layers verified"
```

## Related Documentation

- **Architecture:** [dev_docs/01-architecture/mvp_architecture_overview.md](../../../dev_docs/01-architecture/mvp_architecture_overview.md)
- **MVP Plan:** [dev_docs/00-product/mvp_development_plan.md](../../../dev_docs/00-product/mvp_development_plan.md)
- **Refactoring:** [dev_docs/03-implementation/CORRECTED_REFACTORING_PLAN_FINAL.md](../../../dev_docs/03-implementation/CORRECTED_REFACTORING_PLAN_FINAL.md)
- **Verification:** [dev_docs/03-implementation/REFACTORING_VERIFICATION_REPORT.md](../../../dev_docs/03-implementation/REFACTORING_VERIFICATION_REPORT.md)

---

**Last Updated:** October 21, 2025
**Status:** All tests passing ✅
