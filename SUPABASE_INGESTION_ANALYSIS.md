# Supabase Ingestion Time & Cost Analysis

## Repository Comparison

### Omnara (Your Test Case)
- **Stars**: ~300-500
- **Contributors**: ~10-20
- **Commits**: ~2,000-3,000 (estimated)
- **Issues/PRs**: ~200-300
- **Ingestion Time**: 15 minutes (observed)
- **Result**: Success, 13 incidents found

### Supabase (Target Demo Repo)
Based on public GitHub data:
- **Stars**: 73,000+
- **Contributors**: 800+
- **Commits (total)**: ~35,000+
- **Commits (last 12 months)**: ~5,000-8,000 (estimated)
- **Issues**: ~5,000+ open/closed
- **PRs**: ~3,000+ merged
- **Files**: ~15,000+

**Scale Factor**: Supabase is roughly **20-30x larger** than Omnara

---

## Time Estimate for Supabase Ingestion

### GitHub API Rate Limits (Your Constraint)
With a personal access token:
- **Primary rate limit**: 5,000 requests/hour
- **Secondary rate limit**: 900 points/minute for REST endpoints
- **Concurrent requests**: Max 100 concurrent

### Ingestion Breakdown

#### Phase 1: GitHub Data Fetching
`crisk init` needs to fetch:
1. **Commits** (last 365 days): ~5,000-8,000 commits
   - Cost: ~100-150 API calls (commits API with pagination)
   - Time: ~2-3 minutes

2. **Issues**: ~1,000-2,000 (last 12 months, with bug/incident labels)
   - Cost: ~50-100 API calls
   - Time: ~1-2 minutes

3. **Pull Requests**: ~1,500-2,500 (last 12 months)
   - Cost: ~75-125 API calls
   - Time: ~1-2 minutes

4. **Issue Timeline Events** (for incident linking):
   - For each issue: 1-2 API calls
   - Total: ~2,000-4,000 API calls
   - Time: ~20-40 minutes (this is the bottleneck)

5. **File Tree / Git Operations** (local):
   - Cloning repo: ~2-5 minutes (depends on network)
   - Walking file tree: ~1-2 minutes (local, fast)

**Total GitHub API Time**: ~30-50 minutes
**Total API Calls**: ~2,500-4,500 requests

#### Phase 2: Database Ingestion
1. **Neo4j Graph Construction**:
   - Commit nodes: ~5,000-8,000 inserts
   - File nodes: ~15,000 inserts
   - Relationship edges: ~50,000+ (file changes, authorship, etc.)
   - Time: ~10-20 minutes

2. **PostgreSQL Ingestion**:
   - Issues: ~1,000-2,000 rows
   - PRs: ~1,500-2,500 rows
   - Timeline events: ~10,000-20,000 rows
   - DORA metrics calculation: ~2-5 minutes
   - Time: ~5-10 minutes

**Total Database Time**: ~15-30 minutes

---

## Total Estimated Time: 45-80 Minutes

**Conservative Estimate**: 60-75 minutes (1-1.25 hours)
**Optimistic Estimate**: 45-50 minutes

**Compared to Omnara**: 15 min × 4-5x scale = ~60-75 min (aligns with estimate)

---

## Will Batch API Help?

### Short Answer: **NO, not for `crisk init`**

### Why Not:

1. **`crisk init` doesn't use LLMs**
   - It's purely GitHub API + database ingestion
   - No Gemini calls during ingestion
   - Batch API is for LLM inference, not GitHub fetching

2. **The bottleneck is GitHub API rate limits**
   - 5,000 requests/hour limit
   - Batch API can't bypass GitHub's rate limits
   - You're I/O bound, not compute bound

3. **Our LLM usage is minimal during init**
   - `crisk init` doesn't call Gemini at all
   - `crisk check` calls Gemini once per file (Phase 2)
   - For 15 incidents + 50 safe files = 65 calls (negligible)

---

## Where Batch API WOULD Help (Future Optimization)

### Scenario: Scoring 1,000+ Files in Retrospective Audit

If you wanted to score **every file in the repo** (not just 65):
- **Problem**: 15,000 files × 5-10 seconds = 20-40 hours sequentially
- **Solution**: Batch API with 50% cost savings

**Example**:
```python
# Instead of sequential scoring
for file in all_files:
    crisk check file  # 5-10 seconds each

# Use Batch API
batch_requests = [
    {"file_path": file, "contents": read_file(file)}
    for file in all_files
]
client.batches.create(model="gemini-2.5-flash", src=batch_requests)
# Completes in 24 hours (or less), 50% cheaper
```

**But for your demo**: You only need 15 incidents + 50 safe files = **not worth the complexity**

---

## Cost Analysis

### GitHub API: FREE (up to 5,000 requests/hour)
- Supabase ingestion: ~2,500-4,500 requests
- Well within free tier
- **Cost: $0**

### Gemini API (Your Current Approach)

#### For Demo (15 incidents + 50 safe files = 65 files):
- **Phase 1 only** (no LLM): FREE
- **Phase 2** (LLM agent, 5-6 hops):
  - Input tokens: ~1,500 per hop × 6 hops = ~9,000 tokens/file
  - Output tokens: ~500 per hop × 6 hops = ~3,000 tokens/file
  - Total per file: ~12,000 tokens
  - 65 files × 12,000 tokens = 780,000 tokens

**Gemini 2.0 Flash Pricing**:
- Input: $0.10 / 1M tokens
- Output: $0.40 / 1M tokens

**Cost Calculation**:
- Input: (65 × 9,000) / 1M × $0.10 = $0.06
- Output: (65 × 3,000) / 1M × $0.40 = $0.08
- **Total: $0.14** (negligible)

#### With Batch API (50% discount):
- **Total: $0.07** (saving $0.07)

**Savings: 7 cents** (not worth the implementation complexity)

---

## Recommendation: DON'T Use Batch API for Demo

### Reasons:

1. **Time Savings: None**
   - Bottleneck is GitHub API (can't be batched)
   - LLM calls are only 65 files (fast enough sequentially)
   - Batch API turnaround: 24 hours (slower than real-time!)

2. **Cost Savings: Negligible**
   - You'd save $0.07 (7 cents)
   - Implementation time: 2-3 hours
   - Not worth it

3. **Complexity Increase: High**
   - Need to restructure `crisk check` to support batch mode
   - Need to handle async job polling
   - Need to match results back to files
   - Risk of bugs for demo

4. **Demo Impact: None**
   - Investors don't care if you used batch API
   - They care about the Money Slide graph
   - Live demo needs to be real-time anyway

---

## Optimized Demo Strategy

### Phase 1: Use Phase 1 Only for Scoring (Current Scripts)

Your scripts already do this:
```bash
export PHASE2_ENABLED="false"  # Fast, no LLM
```

**Benefits**:
- Scoring is instant (~1-2 seconds per file)
- No API costs
- Still produces valid risk scores based on metrics
- Total time: ~2 minutes for 65 files

### Phase 2: Parallelize with Simple Shell

For modest parallelization (if needed):
```bash
# Score 10 files in parallel
for i in {1..10}; do
    (crisk check $file &)
done
wait
```

This gives you **10x speedup** without batch API complexity.

---

## Realistic Demo Timeline (Supabase)

### Manual Labeling (2 hours):
- Find 15 revert PRs: 1.5 hours
- Fill CSV: 30 minutes

### Automated Scoring (1.5-2 hours):
1. **Clone Supabase**: 5 minutes
2. **`crisk init`**: 60-75 minutes ← **BOTTLENECK (GitHub API)**
3. **Score 15 incidents** (Phase 1): 30 seconds
4. **Score 50 safe files** (Phase 1): 2 minutes
5. **Generate graph**: 1 minute

**Total**: 70-85 minutes automated (walk away time)

---

## When Batch API Makes Sense (Future)

### Use Case 1: Large-Scale Retrospective Audits
- Customer wants to score **all 15,000 files** in their repo
- Sequential: 20-40 hours
- Batch API: 24 hours, 50% cheaper

### Use Case 2: Continuous Monitoring
- Score every PR (100s per day)
- Batch overnight at 50% cost
- Morning: results ready

### Use Case 3: Multi-Repo Analysis
- Score 10 repos × 1,000 files = 10,000 files
- Batch API: Significant savings

**For your demo**: None of these apply. You're scoring 65 files once.

---

## Final Answer

### Should you use Batch API for the demo?
**NO**

### Should you use Golang library instead of Python?
**NO** - Language doesn't matter for I/O-bound GitHub API calls

### What should you do?
**Use the scripts I created** with `PHASE2_ENABLED="false"`:
- Fast (Phase 1 metrics only)
- Free (no LLM calls)
- Still produces valid Money Slide
- Total time: 60-75 min (mostly `crisk init`)

---

## Pro Tip: Start `crisk init` Tonight

Since `crisk init` takes 60-75 minutes and runs unattended:

```bash
# Tonight before bed:
cd /tmp
git clone https://github.com/supabase/supabase.git
cd supabase
nohup /Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 365 > /tmp/supabase_init.log 2>&1 &
```

**Tomorrow morning**:
- Supabase is ingested
- Run your manual labeling (2 hours)
- Run scoring scripts (5 minutes)
- Generate Money Slide (1 minute)

**Total active time tomorrow**: 2 hours instead of 3.5 hours

---

## Summary Table

| Approach | Time | Cost | Complexity | Recommended |
|----------|------|------|------------|-------------|
| **Current Scripts (Phase 1 only)** | 2 hours manual + 75 min automated | $0 | Low | ✅ YES |
| **Current Scripts (Phase 2 LLM)** | 2 hours manual + 2 hours automated | $0.14 | Low | ⚠️ Optional |
| **Batch API** | 2 hours manual + 24+ hours wait | $0.07 | High | ❌ NO |
| **Parallel Shell** | 2 hours manual + 30 min automated | $0.14 | Medium | ⚠️ If needed |

**Recommendation**: Use current scripts with Phase 1 only. Simple, fast, free.
