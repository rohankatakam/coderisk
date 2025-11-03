# CodeRisk Backtesting Framework - Quick Start

## Overview

This backtesting framework validates your graph construction and LLM issue-PR linking against ground truth test cases from [test_data/omnara_ground_truth.json](test_data/omnara_ground_truth.json).

## What It Tests

✅ **Temporal Linking** - PRs merged within minutes/hours of issue closure
✅ **Semantic Linking** - Keyword overlap between issue and PR titles
✅ **Explicit Linking** - "Fixes #123" references
✅ **CLQS Benchmark** - Overall codebase linking quality score (0-100)

## Quick Start

### 1. Prerequisites

Ensure your databases are running and populated:

```bash
# Check PostgreSQL has staging data
docker exec coderisk-postgres psql -U coderisk_user -d coderisk -c "SELECT COUNT(*) FROM github_issues;"

# Check Neo4j has graph data
docker exec coderisk-neo4j cypher-shell -u neo4j -p coderisk_password "MATCH (n) RETURN count(n);"
```

### 2. Run Backtesting

```bash
# Run all tests (default)
./scripts/run_backtest.sh

# Run with verbose logging
./scripts/run_backtest.sh --verbose

# Run only temporal validation
./scripts/run_backtest.sh --no-semantic --no-clqs

# Use custom ground truth file
./scripts/run_backtest.sh --ground-truth test_data/custom_ground_truth.json
```

### 3. Review Results

Reports are saved to `test_results/` directory:

```bash
# View summary
cat test_results/backtest_*_summary.json | jq

# Check if targets were met
echo $?  # 0 = success, 1 = failed to meet targets
```

## Expected Output

```
🚀 CodeRisk Backtesting Framework
═══════════════════════════════════

📊 Initializing database connections...
  ✓ Connected to PostgreSQL
  ✓ Connected to Neo4j

📖 Loading ground truth data...
  ✓ Loaded 6 test cases from test_data/omnara_ground_truth.json
    Repository: omnara-ai/omnara
    Pattern distribution: map[internal_fix:1 temporal:3 true_negative:2]

🧪 Running Backtesting Suite...
═══════════════════════════════════

[1/4] Running comprehensive backtest...
  [1/6] Testing Issue #221: [FEATURE] allow user to set default agent
    ✅ PASS (confidence: 0.75, delta: 0.00)
  [2/6] Testing Issue #227: [BUG] Codex version not reflective
    ✅ PASS (true negative)
  ...

[2/4] Running temporal pattern validation...
  Testing Issue #221 (temporal)
    ✅ Matched: [222] (confidence: 0.75)
  ...

[3/4] Running semantic pattern validation...
  Testing Issue #187 (semantic)
    ✅ Matched: [218]
       PR #218: similarity=0.78, confidence=0.65
       Common keywords: [mobile interface sync claude code]
  ...

[4/4] Running CLQS benchmark...
  ✓ Explicit Linking: 85.0%
  ✓ Temporal Correlation: 65.0%
  ✓ Comment Quality: 50.0%
  ✓ Semantic Consistency: 60.0%
  ✓ Bidirectional References: 40.0%
  🎯 Overall CLQS: 72.5 (B - Moderate Quality)

📊 Backtesting Complete:
  Precision: 100.00%
  Recall: 75.00%
  F1 Score: 85.71%
  Accuracy: 83.33%

🎯 Target Metrics Comparison:
  Target Precision: 100.00% | Actual: 100.00% | ✅ PASS
  Target Recall: 75.00% | Actual: 75.00% | ✅ PASS
  Target F1: 86.00% | Actual: 85.71% | ❌ FAIL

✅ All target metrics met!
```

## Understanding the Metrics

| Metric | What It Measures | Target |
|--------|------------------|--------|
| **Precision** | % of detected links that are correct | ≥ 100% |
| **Recall** | % of expected links that were found | ≥ 75% |
| **F1 Score** | Balance of precision & recall | ≥ 86% |
| **CLQS** | Overall codebase linking quality | ≥ 75 |

## Test Cases (Omnara Ground Truth)

### Temporal-Only Links (3 cases)
- **Issue #221 → PR #222** - Feature: default agent (2 min delta)
- **Issue #189 → PR #203** - Bug: Ctrl+Z dead (5 min delta)
- **Issue #187 → PR #218** - Bug: Mobile sync (1 min delta + semantic)

### True Negatives (2 cases)
- **Issue #227** - Closed as "not_planned" (no PR)
- **Issue #219** - Closed without PR

### Expected Miss (1 case)
- **Issue #188** - Internal fix with no GitHub trace

## Interpreting Results

### ✅ All Tests Pass

Your graph construction and linking are working correctly!

**Next Steps:**
1. Review CLQS components to identify improvement areas
2. Add more test cases to increase coverage
3. Integrate into CI/CD pipeline

### ❌ Some Tests Fail

**Common Issues:**

1. **Low Recall (< 75%)**
   - Temporal correlator not running
   - Timestamps missing in staging DB
   - Check: Are PRs in Neo4j graph?

2. **Low Precision (< 100%)**
   - Temporal window too wide (catching unrelated PRs)
   - Semantic threshold too low
   - Check: Review false positives in report

3. **Low CLQS (< 75)**
   - Check component breakdown in CLQS report
   - See [LINKING_QUALITY_SCORE.md](test_data/LINKING_QUALITY_SCORE.md) for improvement tips

## Debugging

### Check Neo4j Graph

```cypher
// View issue-PR links
MATCH (i:Issue {number: 221})-[r]-(target)
RETURN i.number, type(r), target, r.confidence, r.evidence;

// Count total links
MATCH ()-[r:FIXES_ISSUE|ASSOCIATED_WITH|MENTIONS]->()
RETURN type(r), count(*);
```

### Check PostgreSQL Staging

```sql
-- View issue timing
SELECT number, title, closed_at
FROM github_issues
WHERE number IN (221, 189, 187);

-- View PR timing
SELECT number, title, merged_at
FROM github_pull_requests
WHERE number IN (222, 203, 218);
```

### View Detailed Logs

```bash
# Run with verbose mode
./scripts/run_backtest.sh --verbose 2>&1 | tee backtest.log

# Search for specific issue
grep "Issue #221" backtest.log
```

## Report Files

All reports saved to `test_results/backtest_YYYYMMDD_HHMMSS_*.json`:

1. **comprehensive.json** - Full test results with pass/fail status
2. **temporal.json** - Temporal pattern validation details
3. **semantic.json** - Semantic pattern validation details
4. **clqs.json** - Codebase Linking Quality Score breakdown
5. **summary.json** - Aggregated metrics for easy comparison

## Adding Custom Test Cases

See [BACKTESTING_GUIDE.md](test_data/BACKTESTING_GUIDE.md#adding-new-test-cases) for detailed instructions.

Quick template:

```json
{
  "issue_number": 250,
  "title": "Your issue title",
  "expected_links": {
    "associated_prs": [255],
    "fixed_by_commits": []
  },
  "linking_patterns": ["temporal"],
  "primary_evidence": {
    "temporal_delta_seconds": 120,
    "issue_closed_at": "2025-11-01T10:00:00Z",
    "pr_merged_at": "2025-11-01T10:02:00Z"
  },
  "expected_confidence": 0.75,
  "should_detect": true
}
```

## Command-Line Options

```bash
./scripts/run_backtest.sh --help

Options:
  --ground-truth PATH   Path to ground truth JSON file
  --repo-id ID          Repository ID in database (default: 1)
  --output DIR          Output directory for reports
  --verbose             Enable verbose logging
  --no-temporal         Skip temporal validation
  --no-semantic         Skip semantic validation
  --no-clqs             Skip CLQS benchmark
  --help                Show help message
```

## CI/CD Integration

Add to your GitHub Actions workflow:

```yaml
- name: Run Backtests
  run: ./scripts/run_backtest.sh --repo-id=1

- name: Upload Reports
  uses: actions/upload-artifact@v3
  with:
    name: backtest-reports
    path: test_results/
```

Exit code:
- `0` = All targets met ✅
- `1` = Some targets failed ❌

## Documentation

- **Full Guide**: [test_data/BACKTESTING_GUIDE.md](test_data/BACKTESTING_GUIDE.md)
- **Linking Patterns**: [test_data/LINKING_PATTERNS.md](test_data/LINKING_PATTERNS.md)
- **Quality Score**: [test_data/LINKING_QUALITY_SCORE.md](test_data/LINKING_QUALITY_SCORE.md)
- **Data Status**: [DATA_READINESS_STATUS.md](DATA_READINESS_STATUS.md)

## Architecture

```
cmd/backtest/main.go          # Main test runner
├── internal/backtest/
│   ├── backtest.go           # Comprehensive validation
│   ├── temporal_matcher.go   # Temporal pattern tests
│   └── semantic_matcher.go   # Semantic pattern tests
└── internal/graph/
    └── linking_quality_score.go  # CLQS calculator

scripts/run_backtest.sh       # Convenience wrapper
test_data/
├── omnara_ground_truth.json  # Validated test cases
├── LINKING_PATTERNS.md       # Pattern documentation
├── LINKING_QUALITY_SCORE.md  # CLQS methodology
└── BACKTESTING_GUIDE.md      # Full documentation
```

## Success Criteria

Based on Omnara ground truth (6 test cases):

- ✅ **Precision**: 100% (no false positives)
- ✅ **Recall**: 75% (find 3/4 valid links, 1 expected miss)
- ✅ **F1 Score**: 86%
- ✅ **CLQS**: 75+ (High Quality)

If all targets met → Exit 0 ✅
If any target missed → Exit 1 ❌

---

**Ready to test?** Run `./scripts/run_backtest.sh` now!
