# CodeRisk Test Cases - Gemini-Only System

## Test Environment
- **Repo**: omnara-ai/omnara (freshly cloned)
- **LLM Provider**: Gemini 2.0 Flash (exclusive)
- **Database**: Clean Neo4j + PostgreSQL
- **Phase 2**: Enabled with GEMINI_API_KEY

---

## Test Categories

### 1. Configuration Files (High Co-change, Build/Deploy Risk)

#### Test 1.1: Python Configuration (pyproject.toml)
```bash
crisk check pyproject.toml --explain
```
**Expected**:
- Risk: MEDIUM-HIGH
- Agent: 5 hops, >60% confidence
- Co-change partners: Multiple related configs
- Incidents: Build/deployment related issues
- Ownership: Core team developers

#### Test 1.2: Package Manager (package.json)
```bash
crisk check apps/web/package.json --explain
```
**Expected**:
- Risk: HIGH (dependency changes affect entire frontend)
- Co-change: yarn.lock, other package.json files
- Incidents: Dependency conflicts, build failures

#### Test 1.3: Environment Config (.env.example)
```bash
crisk check .env.example --explain
```
**Expected**:
- Risk: MEDIUM
- Co-change: docker-compose.yml, deployment configs
- Incidents: Configuration mismatches

---

### 2. Core Application Logic (High Complexity)

#### Test 2.1: Authentication Module
```bash
crisk check apps/web/src/lib/auth.ts --explain
```
**Expected**:
- Risk: CRITICAL-HIGH
- Incidents: Security vulnerabilities, auth failures
- Ownership: Senior engineers only
- Test coverage: Should highlight if insufficient

#### Test 2.2: Database Schema
```bash
crisk check packages/database/prisma/schema.prisma --explain
```
**Expected**:
- Risk: CRITICAL
- Co-change: Migration files, models
- Incidents: Data integrity issues, migration failures
- Blast radius: Affects all services using DB

#### Test 2.3: API Routes
```bash
crisk check apps/web/src/app/api/chat/route.ts --explain
```
**Expected**:
- Risk: HIGH
- Incidents: API errors, rate limiting issues
- Coupling: Frontend components, backend services

---

### 3. UI Components (Medium Complexity, High Co-change)

#### Test 3.1: Dashboard Layout
```bash
crisk check apps/web/src/components/dashboard/SidebarDashboardLayout.tsx --explain
```
**Expected**:
- Risk: MEDIUM
- Co-change: Multiple child components
- Incidents: UI bugs, layout breaks

#### Test 3.2: Chat Component
```bash
crisk check apps/web/src/components/dashboard/chat/ChatMessage.tsx --explain
```
**Expected**:
- Risk: MEDIUM
- Co-change: Chat-related components
- Incidents: Message rendering bugs

#### Test 3.3: Settings Page
```bash
crisk check apps/web/src/app/dashboard/settings/page.tsx --explain
```
**Expected**:
- Risk: LOW-MEDIUM
- Co-change: Settings components
- Incidents: User preference bugs

---

### 4. Infrastructure & DevOps

#### Test 4.1: Docker Compose
```bash
crisk check docker-compose.yml --explain
```
**Expected**:
- Risk: HIGH
- Co-change: Dockerfiles, .env files
- Incidents: Service startup failures

#### Test 4.2: CI/CD Config
```bash
crisk check .github/workflows/ci.yml --explain
```
**Expected**:
- Risk: MEDIUM-HIGH
- Incidents: Build failures, deployment issues
- Co-change: Other workflow files

---

### 5. Edge Cases & Stress Tests

#### Test 5.1: Low-Risk File (README)
```bash
crisk check README.md
```
**Expected**:
- Risk: LOW
- Phase 2: Should NOT escalate
- Fast response (<2s)

#### Test 5.2: New File (No History)
```bash
# Create a new test file first
echo "export const test = 'new';" > /tmp/omnara/apps/web/src/test-new-file.ts
crisk check apps/web/src/test-new-file.ts --explain
```
**Expected**:
- Risk: LOW-MEDIUM
- Incidents: None (new file)
- Ownership: None yet
- Agent should handle gracefully

#### Test 5.3: Large File
```bash
# Find largest TypeScript file
crisk check $(find apps/web/src -name "*.ts" -o -name "*.tsx" | xargs wc -l | sort -rn | head -1 | awk '{print $2}') --explain
```
**Expected**:
- Risk: Depends on file
- Performance: Should complete within timeout
- Agent: May hit token limits, should degrade gracefully

---

### 6. Multi-File Batch Test
```bash
crisk check \
  pyproject.toml \
  apps/web/package.json \
  apps/web/src/lib/auth.ts \
  --explain
```
**Expected**:
- All files analyzed
- Individual risk scores
- No crashes or hangs

---

### 7. Phase 1 Only (No LLM)

#### Test 7.1: Disable Phase 2
```bash
unset PHASE2_ENABLED
crisk check apps/web/src/lib/auth.ts
```
**Expected**:
- Risk: Based on Phase 1 metrics only
- No LLM investigation
- Fast response (<1s)

---

### 8. Error Handling

#### Test 8.1: Non-existent File
```bash
crisk check does-not-exist.ts
```
**Expected**:
- Clear error message
- No crash
- Suggests correct usage

#### Test 8.2: Invalid File Type
```bash
crisk check apps/web/public/logo.png
```
**Expected**:
- Risk: LOW (binary file)
- Graceful handling
- No tree-sitter errors

---

## Verification Checklist

For each test, verify:

- [ ] **Risk Level**: Matches expected range
- [ ] **Confidence**: >60% for Phase 2
- [ ] **Agent Hops**: 3-5 hops for complex files
- [ ] **Response Time**: <30s for Phase 2, <2s for Phase 1
- [ ] **No OpenAI Errors**: Confirms Gemini-only
- [ ] **No Rate Limits**: Gemini has higher limits
- [ ] **Incident Detection**: Finds relevant issues
- [ ] **Ownership**: Returns correct developers
- [ ] **Co-change**: Identifies partner files
- [ ] **Output Format**: Clean, readable, actionable

---

## Success Criteria

**Phase 1 (Baseline)**:
- ✅ All files analyzed without errors
- ✅ Risk scores calculated correctly
- ✅ Co-change detection working

**Phase 2 (LLM Investigation)**:
- ✅ Agent completes 5 hops for complex files
- ✅ Confidence >60% for HIGH/CRITICAL risks
- ✅ Sample incidents retrieved from PostgreSQL
- ✅ No Gemini API errors
- ✅ Recommendations are actionable

**System Integration**:
- ✅ Neo4j queries return data
- ✅ PostgreSQL DORA metrics computed (sample_size > 0)
- ✅ No database connection errors
- ✅ Clean output formatting

---

## Test Execution Notes

1. Run tests **sequentially** to avoid rate limiting
2. Use `--explain` flag for detailed output
3. Save outputs to files for comparison
4. Check logs for any warnings/errors
5. Verify database state between tests if needed

Example batch execution:
```bash
#!/bin/bash
for test in pyproject.toml package.json auth.ts; do
  echo "=== Testing $test ===" | tee -a test_results.txt
  crisk check $test --explain 2>&1 | tee -a test_results.txt
  echo "" | tee -a test_results.txt
done
```
