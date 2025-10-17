# 🚀 Quick Test Commands

## One-Command Full Test (Recommended)

```bash
./scripts/e2e_clean_test.sh
```

**What it does:**
- ✅ Cleans Docker (containers + volumes)
- ✅ Rebuilds crisk binary
- ✅ Starts fresh services
- ✅ Clones Omnara to `/tmp/coderisk-e2e-test/omnara`
- ✅ Runs `init-local` with full graph construction
- ✅ Validates all edge types (CONTAINS, IMPORTS, CO_CHANGED, CAUSED_BY)
- ✅ Reports success/failure with diagnostic info

**Duration:** ~5-10 minutes

---

## Manual Test Commands

### 1. Clean Everything
```bash
make clean-all
```

### 2. Build Binary
```bash
make build
```

### 3. Start Services
```bash
make start
```

### 4. Test on Omnara Repository
```bash
mkdir -p /tmp/coderisk-test && cd /tmp/coderisk-test
git clone https://github.com/omnara-ai/omnara
cd omnara
~/Documents/brain/coderisk-go/bin/crisk init-local
```

### 5. Validate Results
```bash
cd ~/Documents/brain/coderisk-go
./scripts/validate_graph_edges.sh
```

---

## Expected Results

### ✅ Success Indicators

**During init-local:**
```
INFO co-changes calculated total_pairs=168490 min_frequency=0.3
INFO sample co-change after conversion fileA=/tmp/.../apps/web/... fileB=/tmp/.../apps/web/...
INFO storing CO_CHANGED edges count=336980
DEBUG: Creating 336980 edges. First edge: CO_CHANGED (File:...) -> CO_CHANGED (File:...)
INFO edge verification passed count=336980
```

**During validation:**
```
CO_CHANGED Edge Count: 336980
✅ CO_CHANGED edges created successfully!

Sample CO_CHANGED edges:
| a.name     | b.name     | frequency | co_changes | window_days |
| page.tsx   | layout.tsx | 0.85      | 17         | 90          |
```

### ❌ Failure Indicators

**No CO_CHANGED edges:**
```
CO_CHANGED Edge Count: 0
❌ CRITICAL: No CO_CHANGED edges found!
```

**Path mismatch (old bug):**
```
WARN edge count mismatch expected=336980 actual=0
```

---

## Quick Validation Queries

### Check CO_CHANGED edges exist
```bash
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count"
```
**Expected:** count > 0 (e.g., 336980)

### View sample CO_CHANGED edges
```bash
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (a:File)-[r:CO_CHANGED]->(b:File) RETURN a.name, b.name, r.frequency LIMIT 5"
```

### Verify bidirectional edges
```bash
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (a:File)-[r1:CO_CHANGED]->(b:File), (b)-[r2:CO_CHANGED]->(a) WHERE r1.frequency <> r2.frequency RETURN count(*)"
```
**Expected:** 0 (all frequencies should match)

---

## Service Management Commands

### Start services
```bash
make start
```

### Stop services
```bash
make stop
```

### Check service status
```bash
make status
```

### View logs
```bash
make logs          # All services
make logs-neo4j    # Neo4j only
make logs-postgres # PostgreSQL only
```

### Restart services
```bash
make restart
```

## Cleanup Commands

### Clean Docker (stop and remove volumes)
```bash
make clean-docker
```

### Clean everything (Docker + binaries + temp files)
```bash
make clean-all
```

### Remove test directory
```bash
rm -rf /tmp/coderisk-test /tmp/coderisk-e2e-test
```

---

## Troubleshooting

### Services won't start
```bash
docker compose logs neo4j
docker compose logs postgres
# Check for port conflicts or memory issues
```

### No commits found in git history
```bash
cd /tmp/coderisk-test/omnara
git log --oneline --since="90 days ago" | wc -l
# Should be > 50
```

### Edge creation fails
```bash
# Check DEBUG logs
grep "DEBUG:\|ERROR\|WARN" /tmp/coderisk-e2e-test/init-local.log
```

---

## What Changed (Summary)

### Fixed Files
1. **processor.go** - Converts git relative paths → absolute paths before edge creation
2. **linker.go** - Adds `incident:` and `file:` prefixes to CAUSED_BY edges
3. **neo4j_backend.go** - Enhanced error logging with diagnostic info

### New Files
- `scripts/clean_docker.sh` - Docker cleanup script
- `scripts/validate_graph_edges.sh` - Graph validation script
- `scripts/e2e_clean_test.sh` - Full automated test
- `TESTING_INSTRUCTIONS.md` - Detailed testing guide
- `QUICK_TEST.md` - This file

### Modified Files
- `Makefile` - Added `clean-docker` and `clean-all` targets

---

## Next Steps After Success

1. **Verify CO_CHANGED edges work in risk calculation:**
   ```bash
   cd /tmp/coderisk-test/omnara
   ~/Documents/brain/coderisk-go/bin/crisk check apps/web/src/app/page.tsx
   ```

2. **Test AI mode (requires OpenAI API key):**
   ```bash
   ~/Documents/brain/coderisk-go/bin/crisk check apps/web/src/app/page.tsx --ai-mode
   ```

3. **Test incident linking:**
   ```bash
   ~/Documents/brain/coderisk-go/bin/crisk incident create "Test" "Description" --severity high
   ```

---

**Ready to test!** Run `./scripts/e2e_clean_test.sh` to start.
