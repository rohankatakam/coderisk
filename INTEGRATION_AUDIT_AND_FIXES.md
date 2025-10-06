# Integration Audit & Fixes - Sessions A, B, C

**Date:** 2025-10-06
**Purpose:** Eliminate shortcuts, hardcoded values, and incomplete integrations from all three session implementations

---

## Critical Issues Found

### Session A (Temporal Analysis)

**❌ Issue 1: GetCoChangedFiles() Not Implemented**
- Location: `internal/temporal/co_change.go:86-92`
- Problem: Returns `fmt.Errorf("not yet implemented")` instead of querying Neo4j
- Impact: Session C cannot get co-change data for risk assessment
- Fix: Implement Neo4j query for CO_CHANGED edges

**❌ Issue 2: GetOwnershipHistory() Not Implemented**
- Location: `internal/temporal/developer.go:146-152`
- Problem: Returns `fmt.Errorf("not yet implemented")` instead of querying Neo4j
- Impact: Session C cannot get ownership data for risk assessment
- Fix: Implement Neo4j query or direct calculation from commits

**❌ Issue 3: Hardcoded WindowDays**
- Location: `internal/temporal/co_change.go:70`
- Problem: `WindowDays: 90` hardcoded in co-change calculation
- Fix: Make this configurable via function parameter

**❌ Issue 4: CO_CHANGED Edges Not Created in Graph**
- Location: `internal/graph/builder.go` - missing integration
- Problem: Temporal analysis calculates co-changes but doesn't store them in Neo4j
- Impact: Layer 2 edges don't exist in graph
- Fix: Add CO_CHANGED edge creation to graph builder

---

### Session B (Incidents Database)

**✅ Session B Implementation Status: COMPLETE**
- PostgreSQL schema created with BM25 search
- Incident linking to Neo4j working
- CAUSED_BY edges created correctly
- No critical issues found

**Minor Improvement:**
- Linker interface mismatch with graph.Backend (uses custom GraphClient interface)
- Fix: Update to use graph.Backend directly

---

### Session C (LLM Investigation)

**❌ Issue 5: Anthropic References Not Removed**
- Location: `internal/agent/llm_client.go:32-34`
- Problem: Still has Anthropic stub despite instructions to focus on OpenAI only
- Fix: Remove all Anthropic references

**❌ Issue 6: Mock Clients in Production Code**
- Location: `internal/agent/investigator_test.go` - mock clients only
- Problem: No real implementation to connect to Sessions A & B
- Impact: Agent can't actually run investigations
- Fix: Create real TemporalClient and IncidentsClient wrappers

**❌ Issue 7: Phase 2 Integration Missing from check.go**
- Location: `cmd/crisk/check.go:153-159`
- Problem: Comments say "Would escalate to Phase 2" but doesn't actually call investigator
- Impact: Phase 2 never runs, even when risk is high
- Fix: Integrate actual Phase 2 escalation

**❌ Issue 8: GraphClient Interface Not Implemented**
- Location: `internal/agent/evidence.go:24-27`
- Problem: GraphClient interface defined but never implemented
- Fix: Use existing graph.Backend or implement wrapper

---

### Cross-Session Integration Issues

**❌ Issue 9: No Connection Between Sessions**
- Problem: Sessions A, B, C are isolated packages with no integration layer
- Impact: Agent can't access temporal or incidents data
- Fix: Create adapter layer in `internal/agent/adapters.go`

**❌ Issue 10: Hardcoded TODO Values**
- Locations: Multiple files (check.go:131, output/converter.go:13, etc.)
- Problem: Placeholder TODOs that should be implemented
- Fix: Implement all TODOs or remove if not needed

---

## Fix Plan

### Phase 1: Session A Fixes (Temporal)

1. **Implement GetCoChangedFiles with Neo4j Query**
   ```go
   // Query: MATCH (f:File {unique_id: $filePath})-[r:CO_CHANGED]-(other:File)
   //        WHERE r.frequency >= $minFrequency
   //        RETURN other.unique_id, r.frequency, r.co_changes, r.window_days
   ```

2. **Implement GetOwnershipHistory**
   - Option A: Query Neo4j for Developer/Commit nodes
   - Option B: Recalculate from git history on demand
   - **Recommended:** Option B (simpler, no graph dependency)

3. **Make WindowDays Configurable**
   - Add parameter to CalculateCoChanges()
   - Update callers to pass 90 explicitly

4. **Add CO_CHANGED Edge Creation**
   - Modify `internal/graph/builder.go`
   - Add AddLayer2CoChangedEdges() method
   - Call after commit processing

### Phase 2: Session B Fixes (Incidents)

1. **Update Linker to Use graph.Backend**
   - Remove custom GraphClient interface
   - Use graph.Backend directly
   - Update cmd/crisk/incident.go

### Phase 3: Session C Fixes (Agent)

1. **Remove All Anthropic References**
   - Delete anthropic case from llm_client.go
   - Remove TODO comment
   - Update documentation

2. **Create Real Client Adapters**
   ```go
   // internal/agent/adapters.go

   type RealTemporalClient struct {
       gitParser *temporal.GitHistory
   }

   type RealIncidentsClient struct {
       db *incidents.Database
   }

   type RealGraphClient struct {
       backend graph.Backend
   }
   ```

3. **Integrate Phase 2 into check.go**
   - Replace comment with actual investigator call
   - Add OpenAI API key validation
   - Format and display Phase 2 results

4. **Fix Evidence Collection**
   - Replace nil checks with real clients
   - Handle errors properly
   - Add logging

### Phase 4: Cross-Session Integration

1. **Create Integration Layer**
   - New file: `internal/integration/phase2.go`
   - Coordinates temporal, incidents, agent packages
   - Provides clean API for check.go

2. **Fix All TODOs**
   - check.go:131 - Get repo ID from git
   - output/converter.go - Get branch, language, lines from git
   - Remove WindowDays hardcode

3. **End-to-End Testing**
   - Test full workflow: init → link incident → check (Phase 2)
   - Verify evidence collection from all sources
   - Validate LLM investigation

---

## Implementation Order

1. ✅ **Session A: Implement GetCoChangedFiles** (20 min)
2. ✅ **Session A: Implement GetOwnershipHistory** (20 min)
3. ✅ **Session A: Add CO_CHANGED edge creation** (30 min)
4. ✅ **Session B: Update Linker interface** (10 min)
5. ✅ **Session C: Remove Anthropic** (5 min)
6. ✅ **Session C: Create real adapters** (30 min)
7. ✅ **Session C: Integrate Phase 2** (30 min)
8. ✅ **Integration: Create phase2.go** (20 min)
9. ✅ **Cleanup: Fix all TODOs** (20 min)
10. ✅ **Testing: End-to-end validation** (30 min)

**Total Estimated Time:** 3-4 hours

---

## Success Criteria

- [ ] GetCoChangedFiles() returns real data from Neo4j or calculation
- [ ] GetOwnershipHistory() returns real data from git history
- [ ] CO_CHANGED edges created in Neo4j during init-local
- [ ] Linker uses graph.Backend (no custom interface)
- [ ] No Anthropic references in codebase
- [ ] Real TemporalClient and IncidentsClient implementations exist
- [ ] Phase 2 escalation actually runs LLM investigation
- [ ] Evidence collector gets data from all sources (temporal, incidents, graph)
- [ ] No hardcoded TODOs in critical paths
- [ ] End-to-end test passes: init → link → check → Phase 2 result

---

## Testing Checklist

```bash
# 1. Test Session A (Temporal)
cd /tmp/test-repo
git init
# Create some commits
/path/to/crisk init-local
# Verify CO_CHANGED edges exist
docker exec coderisk-neo4j cypher-shell "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"

# 2. Test Session B (Incidents)
crisk incident create "Test incident" "Description" --severity critical
crisk incident link <id> src/file.go
# Verify CAUSED_BY edge
docker exec coderisk-neo4j cypher-shell "MATCH (i:Incident)-[r:CAUSED_BY]->(f:File) RETURN count(r)"

# 3. Test Session C (Agent)
export CODERISK_LLM_PROVIDER=openai
export CODERISK_API_KEY=sk-...
crisk check src/file.go
# Should show Phase 2 investigation if high risk

# 4. Test Integration
# Make a change to a file with incidents
echo "change" >> src/file.go
crisk check src/file.go
# Should:
# - Detect co-change patterns (Session A)
# - Find linked incidents (Session B)
# - Run LLM investigation (Session C)
# - Show risk assessment with evidence
```

---

## Files to Modify

1. `internal/temporal/co_change.go` - Implement GetCoChangedFiles
2. `internal/temporal/developer.go` - Implement GetOwnershipHistory
3. `internal/graph/builder.go` - Add CO_CHANGED edge creation
4. `internal/incidents/linker.go` - Update GraphClient interface
5. `internal/agent/llm_client.go` - Remove Anthropic
6. `internal/agent/adapters.go` - NEW FILE: Real client implementations
7. `internal/integration/phase2.go` - NEW FILE: Integration layer
8. `cmd/crisk/check.go` - Integrate Phase 2 escalation
9. `cmd/crisk/incident.go` - Update linker initialization
10. Various files - Remove TODOs and hardcoded values

---

## Next Steps

After all fixes:
1. Update SESSION_*_COMPLETE.md documents
2. Update PARALLEL_SESSION_PLAN_WEEKS2-8.md
3. Run full test suite
4. Update dev_docs/03-implementation/status.md to 100%
5. Create git commit with comprehensive message
