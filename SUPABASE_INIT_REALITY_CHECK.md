# Supabase `crisk init` - Reality Check

## You're Right: LLM IS Used During Init!

I apologize for the incomplete analysis. After reviewing the actual code, here's what **really** happens:

---

## The Truth About `crisk init`

### Stage 3.5: LLM Extraction Phase (I Missed This!)

From `/cmd/crisk/init.go` line 326-370:

```go
// Stage 1.5: Extract issue-commit-PR relationships using LLM
fmt.Printf("\n[3.5/5] Extracting issue-commit-PR relationships (LLM analysis)...\n")

llmClient, err := llm.NewClient(ctx, cfg)

if llmClient.IsEnabled() {
    // Create extractors
    issueExtractor := github.NewIssueExtractor(llmClient, stagingDB)
    commitExtractor := github.NewCommitExtractor(llmClient, stagingDB)

    // Extract from Issues (LLM CALL!)
    issueRefs, err := issueExtractor.ExtractReferences(ctx, repoID)

    // Extract from Commits (LLM CALL!)
    commitRefs, err := commitExtractor.ExtractReferences(ctx, repoID)

    // Extract from PRs (LLM CALL!)
    prRefs, err := prExtractor.ExtractReferences(ctx, repoID)
}
```

### What This Actually Does

**IssueExtractor** (`internal/github/issue_extractor.go`):
- Fetches up to 1,000 unprocessed issues
- **Batches them 20 at a time**
- Each batch = **1 LLM call** to extract commit/PR references
- Looks for patterns like "fixes #123", "closed by abc123def", etc.

**CommitExtractor** (similar):
- Fetches all commits
- **Batches 20 at a time**
- Each batch = **1 LLM call** to extract issue/PR references from commit messages

**PRExtractor** (similar):
- Fetches all PRs
- **Batches 20 at a time**
- Each batch = **1 LLM call** to extract issue references from PR descriptions

---

## Supabase LLM Cost Calculation (Corrected!)

### Data Volume Estimate

Based on Supabase scale:
- **Issues**: ~1,000-2,000 (last 12 months with bug labels)
- **Commits**: ~5,000-8,000
- **PRs**: ~1,500-2,500

### LLM Calls Required

**Batch size**: 20 items per call

- Issues: 1,000 / 20 = **50 LLM calls**
- Commits: 6,000 / 20 = **300 LLM calls**
- PRs: 2,000 / 20 = **100 LLM calls**

**Total**: ~450 LLM calls

### Token Estimation Per Call

Looking at the prompts (`issue_extractor.go` line 79-110):

**System prompt**: ~200 tokens
**User prompt per batch**:
- 20 issues/commits/PRs
- Each ~50-150 tokens (title + body/message)
- Average: 100 tokens × 20 = 2,000 tokens

**Total per call**:
- Input: ~2,200 tokens
- Output: ~500 tokens (JSON references)

**Total for Supabase init**:
- Input: 450 calls × 2,200 = 990,000 tokens (~1M)
- Output: 450 calls × 500 = 225,000 tokens (~0.23M)

---

## Cost Analysis (CORRECTED)

### Gemini 2.0 Flash Pricing
- Input: $0.10 / 1M tokens
- Output: $0.40 / 1M tokens

### Supabase Init Cost
- Input: 1M × $0.10 = **$0.10**
- Output: 0.23M × $0.40 = **$0.09**
- **Total: $0.19** (~20 cents)

### With Batch API (50% discount)
- **Total: $0.095** (~10 cents)
- **Savings: 9 cents**

---

## Time Analysis (CORRECTED)

### GitHub API Phase (unchanged)
- 2,500-4,500 API calls
- Time: ~30-50 minutes
- **Bottleneck: GitHub rate limits**

### LLM Extraction Phase (NEW!)
- 450 LLM calls
- Average latency: ~2-3 seconds per call
- Sequential: 450 × 2.5s = **~18-20 minutes**
- **This is significant!**

### Database Ingestion (unchanged)
- Neo4j + PostgreSQL writes
- Time: ~15-30 minutes

---

## Total Supabase Init Time: 60-100 Minutes

**Breakdown**:
1. GitHub API: 30-50 min (I/O bound)
2. **LLM Extraction: 18-20 min** ← **I MISSED THIS**
3. Database writes: 15-30 min (CPU bound)

**Total**: 63-100 minutes (1-1.7 hours)

**Conservative estimate**: 75-90 minutes

---

## Would Batch API Help NOW?

### For the Extraction Phase: YES, But...

**Problem**: 450 sequential LLM calls taking ~20 minutes

**Solution**: Batch API could parallelize these

**But**:
1. **Turnaround time**: 24 hours (target) - way slower than 20 minutes
2. **Implementation complexity**: High
   - Restructure extractors to use batch mode
   - Convert synchronous calls to async job submissions
   - Poll for completion
   - Parse JSONL results
   - Handle failures gracefully
3. **Cost savings**: 9 cents (not worth the dev time)
4. **Demo timeline**: You want to run this in ~2 hours, not 24+ hours

---

## Realistic Options

### Option 1: Use Current System (Recommended)
**Time**: 75-90 minutes
**Cost**: $0.19
**Complexity**: Zero (just run it)
**Works for**: Demo

### Option 2: Simple Parallelization (Easy Win!)
**Idea**: Run extractors in parallel instead of sequential

Current code (sequential):
```go
issueRefs, _ := issueExtractor.ExtractReferences(ctx, repoID)  // 7 min
commitRefs, _ := commitExtractor.ExtractReferences(ctx, repoID) // 10 min
prRefs, _ := prExtractor.ExtractReferences(ctx, repoID)        // 5 min
```

Parallel version:
```go
var wg sync.WaitGroup
wg.Add(3)
go func() { issueExtractor.ExtractReferences(ctx, repoID); wg.Done() }()
go func() { commitExtractor.ExtractReferences(ctx, repoID); wg.Done() }()
go func() { prExtractor.ExtractReferences(ctx, repoID); wg.Done() }()
wg.Wait()
```

**Time savings**: ~12-15 minutes (run all 3 in parallel instead of sequential)
**New total**: 50-75 minutes (down from 75-90 min)
**Implementation**: ~30 minutes of coding

### Option 3: Increase Batch Size
Current: 20 items/batch
Increase to: 50 items/batch

**LLM calls**: 450 → 180 (60% reduction)
**Time savings**: ~10-12 minutes
**Trade-off**: Larger prompts, higher per-call cost (but total cost same)

### Option 4: Batch API (Not Recommended)
**Time**: 24+ hours
**Cost savings**: 9 cents
**Complexity**: 4-6 hours dev time
**Not worth it for demo**

---

## My Recommendation: Quick Fix (Option 2)

Let me implement simple parallelization in the init command. This takes 30 minutes to code and saves you 12-15 minutes every time you run init.

**Do you want me to:**
1. ✅ **Just run it as-is** (75-90 min, works fine)
2. ⚡ **Parallelize extractors** (30 min to code, saves 12-15 min)
3. 📦 **Increase batch size** (10 min to code, saves 10 min)

---

## Updated Timeline for Supabase Demo

### Tonight (Recommended):
```bash
# Start this before bed
cd /tmp
git clone https://github.com/supabase/supabase.git
cd supabase
nohup /Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --days 365 > /tmp/supabase_init.log 2>&1 &
```

**Runtime**: 75-90 minutes (runs while you sleep)

### Tomorrow Morning:
- ✅ Supabase already ingested
- Manual labeling: 2 hours
- Scoring: 5 minutes
- Graph generation: 1 minute

**Total active time**: 2 hours

---

## Cost Breakdown (Final)

| Phase | API Calls | Cost | Time |
|-------|-----------|------|------|
| **GitHub API** | 2,500-4,500 | $0 (free tier) | 30-50 min |
| **LLM Extraction** | 450 | $0.19 | 18-20 min |
| **Database Writes** | N/A | $0 | 15-30 min |
| **Demo Scoring** (15+50 files, Phase 1) | 0 | $0 | 3 min |
| **TOTAL** | - | **$0.19** | **63-100 min** |

**With Batch API**: $0.10 (saves 9 cents, takes 24+ hours)

---

## Bottom Line

**You were right** - `crisk init` DOES use LLMs for incident extraction.

**Good news**: It's only ~$0.20 and takes 75-90 minutes.

**Better news**: Run it tonight, it'll be done by morning.

**Best news**: No need for Batch API complexity for a 20-cent cost.

---

## Should We Modify The Code?

**Quick wins available**:
1. Parallelize extractors (saves 12-15 min) - 30 min to implement
2. Increase batch size (saves 10 min) - 10 min to implement

**Not worth it**:
1. Batch API (saves 9 cents, takes 24+ hours, 6 hours to implement)

**Your call**: We can implement #1 or #2 if you want, or just run it as-is tonight.
