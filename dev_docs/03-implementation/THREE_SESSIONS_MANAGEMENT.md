# Managing Three Parallel Sessions (Weeks 2-8)

**Purpose:** Guide for running Sessions A, B, and C simultaneously in separate Claude Code instances.

---

## Quick Reference

| Session | Focus | Package | Duration | Key Deliverable |
|---------|-------|---------|----------|----------------|
| **Session A** | Temporal Analysis | `internal/temporal/` | Weeks 2-3 (1.5-2w) | CO_CHANGED edges, ownership tracking |
| **Session B** | Incident Database | `internal/incidents/` | Weeks 4-5 (1.5-2w) | PostgreSQL FTS, BM25 search |
| **Session C** | LLM Investigation | `internal/agent/` | Weeks 6-8 (2-3w) | Agentic search, risk assessment |

---

## Setup Instructions

### 1. Open Three Separate Claude Code Sessions

**Session A (Temporal Analysis):**
```bash
# In terminal 1
cd /path/to/coderisk-go
claude-code
# Paste: dev_docs/03-implementation/SESSION_A_PROMPT.md contents
```

**Session B (Incident Database):**
```bash
# In terminal 2
cd /path/to/coderisk-go
claude-code
# Paste: dev_docs/03-implementation/SESSION_B_PROMPT.md contents
```

**Session C (LLM Investigation):**
```bash
# In terminal 3
cd /path/to/coderisk-go
claude-code
# Paste: dev_docs/03-implementation/SESSION_C_PROMPT.md contents
```

---

## Execution Timeline

### Weeks 2-3: Session A Only

**Start:** Session A immediately
**Sessions B & C:** On hold

**Session A Tasks:**
1. Create `internal/temporal/` package
2. Implement git history parsing
3. Calculate CO_CHANGED edges
4. Track ownership transitions
5. Integrate with `internal/ingestion/processor.go`
6. **Checkpoint A1:** Git history parsing works
7. **Checkpoint A2:** CO_CHANGED edges in Neo4j
8. **Checkpoint A3:** Ownership tracking complete

**Why sequential:** Sessions B & C depend on Session A's interfaces.

---

### Weeks 4-5: Session B Only (After Session A Complete)

**Start:** Session B after A3 checkpoint passes
**Sessions A & C:** On hold

**Session B Tasks:**
1. Create `internal/incidents/` package
2. Add PostgreSQL schema migration
3. Implement BM25 search with tsvector
4. Build manual incident linking CLI
5. Integrate with Neo4j for CAUSED_BY edges
6. **Checkpoint B1:** PostgreSQL schema + GIN index
7. **Checkpoint B2:** BM25 search working

**Why sequential:** Session C needs incident stats API from Session B.

---

### Weeks 6-8: Session C (After Session B Complete)

**Start:** Session C after B2 checkpoint passes
**Sessions A & B:** Maintenance only

**Session C Tasks:**
1. Create `internal/agent/` package
2. Integrate OpenAI/Anthropic SDKs
3. Implement hop-by-hop navigation
4. Build evidence collector (uses A & B APIs)
5. Implement synthesis and risk scoring
6. Integrate with `cmd/crisk/check.go` for Phase 2
7. **Checkpoint C1:** LLM clients working
8. **Checkpoint C2:** Hop navigation complete
9. **Checkpoint C3:** End-to-end investigation

**Why last:** Requires both temporal data (Session A) and incidents (Session B).

---

## Checkpoint Verification

### Session A Checkpoints

**A1: Git History Parsing ✅**
```bash
go test ./internal/temporal/... -v -run TestParseGitHistory
# Expected: Parses commits from last 90 days, extracts file changes
```

**A2: CO_CHANGED Edges ✅**
```bash
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"
# Expected: >100 edges for omnara repo
```

**A3: Ownership Tracking ✅**
```bash
go run scripts/test_ownership.go /tmp/omnara/src/server.ts
# Expected: Returns current owner, previous owner, transition date
```

### Session B Checkpoints

**B1: PostgreSQL Schema ✅**
```bash
docker exec coderisk-postgres psql -U coderisk -d coderisk -c "\dt"
# Expected: Tables "incidents" and "incident_files" exist
docker exec coderisk-postgres psql -U coderisk -d coderisk -c \
  "SELECT indexname FROM pg_indexes WHERE tablename = 'incidents'"
# Expected: idx_incidents_search (GIN index)
```

**B2: BM25 Search ✅**
```bash
go test ./internal/incidents/... -v -run TestSearchIncidents
# Expected: Returns ranked results, <50ms query time
```

### Session C Checkpoints

**C1: LLM Clients ✅**
```bash
go run scripts/test_llm.go openai "What is 2+2?"
go run scripts/test_llm.go anthropic "What is 2+2?"
# Expected: Valid responses from both providers
```

**C2: Hop Navigation ✅**
```bash
go test ./internal/agent/... -v -run TestHopNavigator
# Expected: 3-hop investigation completes, early exit logic works
```

**C3: End-to-End Investigation ✅**
```bash
./crisk check src/payment_processor.py
# Expected: Shows Risk Level, Summary, Evidence, Investigation details
```

---

## File Ownership (No Collisions)

### Session A Owns
- ✅ **Create:** `internal/temporal/*.go`
- ✅ **Modify:** `internal/ingestion/processor.go` (~30 lines)
- ✅ **Modify:** `internal/graph/builder.go` (~50 lines for Layer 2)

### Session B Owns
- ✅ **Create:** `internal/incidents/*.go`
- ✅ **Create:** `cmd/crisk/link-incident.go`
- ✅ **Modify:** `internal/storage/postgres.go` (~40 lines migration)
- ✅ **Modify:** `internal/graph/builder.go` (~50 lines for Layer 3)

### Session C Owns
- ✅ **Create:** `internal/agent/*.go`
- ✅ **Modify:** `cmd/crisk/check.go` (~80 lines Phase 2)
- ✅ **Read:** `internal/temporal/` (Session A's public API)
- ✅ **Read:** `internal/incidents/` (Session B's public API)

**Collision Prevention:**
- Each session owns its own package directory
- Shared files (`processor.go`, `builder.go`, `check.go`) have clear section boundaries
- Sessions should add code, not modify existing logic

---

## Coordination Between Sessions

### Session A → Session B
**Exports:**
- None (B doesn't depend on A)

**B can start:** Immediately after A completes (or in parallel if desired)

### Session A → Session C
**Exports:**
```go
// Session C imports these:
temporal.GetCoChangedFiles(filePath, minFreq) -> []CoChangeResult
temporal.GetOwnershipHistory(filePath) -> *OwnershipHistory
```

**C must use:** Mock data until A completes, then swap to real API

### Session B → Session C
**Exports:**
```go
// Session C imports these:
incidents.GetIncidentStats(filePath) -> *IncidentStats
incidents.SearchIncidents(query, limit) -> []SearchResult
```

**C must use:** Mock data until B completes, then swap to real API

---

## Conflict Resolution

### If Two Sessions Modify the Same File

**Example:** Both A and B modify `internal/graph/builder.go`

**Resolution:**
1. **Session A:** Add `AddLayer2Temporal()` function at end of file
2. **Session B:** Add `AddLayer3Incidents()` function at end of file
3. **No edits to existing functions**
4. **Git merge:** Should auto-merge (different functions)

### If Git Merge Conflict Occurs

**Strategy:**
1. Pull latest main branch before each session
2. Commit frequently (after each checkpoint)
3. Use clear function boundaries (no shared logic)
4. If conflict: Manual merge, prioritize Session A > B > C

---

## Testing Strategy

### Unit Tests (Each Session)
- Session A: `go test ./internal/temporal/... -v`
- Session B: `go test ./internal/incidents/... -v`
- Session C: `go test ./internal/agent/... -v`

**Target:** >70% coverage per package

### Integration Tests
- Session A: `./test/integration/test_temporal_analysis.sh`
- Session B: `./test/integration/test_incident_search.sh`
- Session C: `./test/integration/test_agent_investigation.sh`

**Run:** After each checkpoint passes

### End-to-End Test (All Sessions)
```bash
# After all three sessions complete
./test/integration/test_full_workflow.sh

# Should test:
# 1. Init repo (creates Layer 1 + Layer 2 from Session A)
# 2. Link incident (Session B)
# 3. Run check (triggers Session C investigation)
# 4. Verify risk assessment uses data from A, B, and C
```

---

## Communication Protocol

### When Session Completes Checkpoint

**Post in shared document:**
```
Session A - Checkpoint A2 ✅
- CO_CHANGED edges: 847
- Top pair: auth.ts <-> database.ts (0.89 frequency)
- Commit: abc123
- Ready for Session B to use GetCoChangedFiles()
```

### When Session Is Blocked

**Post in shared document:**
```
Session C - BLOCKED at C2
- Waiting for Session A's GetOwnershipHistory() API
- ETA: Session A needs 2 more days
- Using mock data for now
```

### When Session Finds Issue in Another Session's Code

**Post in shared document:**
```
Session C - Issue in Session A
- GetCoChangedFiles() returns empty for valid file
- Expected: 10+ results
- Repro: GetCoChangedFiles("src/auth.ts", 0.3)
- Session A: Please investigate
```

---

## Success Criteria (All Sessions Complete)

- [ ] **Session A:** 90-day git history parsed, CO_CHANGED edges in Neo4j, ownership tracking works
- [ ] **Session B:** PostgreSQL schema with BM25 search, manual incident linking CLI, CAUSED_BY edges
- [ ] **Session C:** LLM investigation with 3-hop navigation, risk assessment output, Phase 2 integration
- [ ] **Integration:** All three systems work together in `crisk check` command
- [ ] **Performance:** Full workflow <10s (temporal 5s + incidents 2s + investigation 3s)
- [ ] **Tests:** >70% unit test coverage, all integration tests pass

---

## Example Workflow (After All Sessions Complete)

```bash
# 1. Initialize repository (Session A runs automatically)
cd /tmp/omnara
crisk init-local
# Output: 421 files, 5,103 edges (Layer 1) + 847 CO_CHANGED edges (Layer 2)

# 2. Link an incident (Session B)
crisk link-incident a1b2c3d4 src/payment_processor.py --line 142 --function process_payment
# Output: ✅ Linked incident to payment_processor.py

# 3. Run check (Session C with A & B data)
crisk check src/payment_processor.py
# Output:
# 🔍 RISK ASSESSMENT
# File: src/payment_processor.py
# Risk Level: HIGH (score: 0.78, confidence: 0.85)
#
# Summary:
# This file caused 3 production incidents in the last 30 days.
# It has high temporal coupling with payment_gateway.py (85% co-change).
# Verify payment flow and error handling before merging.
#
# Evidence:
#   1. [incident] 3 incidents in last 90 days (2 critical)
#   2. [co_change] Changes with payment_gateway.py 85% of the time
#   3. [ownership] Ownership transitioned 12 days ago
#
# Investigation: 2 hops, 2,847 tokens
```

---

## Timeline Summary

| Week | Focus | Active Session | Checkpoint | Expected Output |
|------|-------|----------------|------------|----------------|
| 2 | Git parsing | A | A1 ✅ | 1,500+ commits parsed |
| 3 | CO_CHANGED calculation | A | A2 ✅ | 847 edges in Neo4j |
| 3 | Ownership tracking | A | A3 ✅ | Ownership transitions detected |
| 4 | PostgreSQL schema | B | B1 ✅ | incidents table + GIN index |
| 5 | BM25 search | B | B2 ✅ | <50ms full-text search |
| 6 | LLM client | C | C1 ✅ | OpenAI + Anthropic working |
| 7 | Hop navigation | C | C2 ✅ | 3-hop investigation <5s |
| 8 | End-to-end | C | C3 ✅ | Full risk assessment output |

**Total Duration:** 6-7 weeks (parallel work not possible due to dependencies)

---

## Troubleshooting

### Session A Issues

**Problem:** Git log parsing fails
**Solution:** Check git version (`git --version`), ensure `--numstat` flag supported

**Problem:** No CO_CHANGED edges created
**Solution:** Verify commits have >1 file changed, check threshold (min 0.3 frequency)

### Session B Issues

**Problem:** PostgreSQL tsvector not generating
**Solution:** Check `GENERATED ALWAYS AS` syntax, ensure PostgreSQL 12+

**Problem:** BM25 search returns no results
**Solution:** Verify GIN index exists, test with `SELECT * FROM incidents WHERE search_vector @@ to_tsquery('test')`

### Session C Issues

**Problem:** OpenAI API key rejected
**Solution:** Check env var `CODERISK_API_KEY`, test with `curl https://api.openai.com/v1/models`

**Problem:** Hop navigation takes >10s
**Solution:** Reduce token budget, implement early exit, cache graph queries

---

## Next Steps After Completion

1. **Performance optimization** - Reduce investigation time from 5s to <3s
2. **Auto-incident linking** - Use LLM to parse incident descriptions for file paths
3. **Test coverage integration** - Add Layer 4 (test coverage metrics)
4. **Cloud deployment** - Migrate to AWS Neptune and RDS
5. **Team sharing** - Implement multi-user repositories per [team_and_branching.md](../02-operations/team_and_branching.md)

---

**Good luck with your parallel sessions! 🚀**
