# Backtesting Quick Start Guide

## 🚀 TL;DR - Run This Now

```bash
cd /Users/rohankatakam/Documents/brain/coderisk
./scripts/run_backtest.sh --verbose
```

Expected runtime: **~30 seconds**

## ✅ What Gets Tested

| Pattern | Test Cases | What It Validates |
|---------|------------|-------------------|
| **Temporal** | 3 cases | PRs merged within minutes of issue closure |
| **Semantic** | 1 case | Keyword overlap between issue/PR titles |
| **True Negatives** | 2 cases | Correctly identifying no-link cases |
| **Expected Miss** | 1 case | Known limitations (internal fixes) |

**Total:** 6 validated test cases from Omnara repository

## 📊 Success Criteria

| Metric | Target | What It Means |
|--------|--------|---------------|
| Precision | ≥ 100% | No false positives (all detected links are correct) |
| Recall | ≥ 75% | Find 3 out of 4 valid links (1 expected miss) |
| F1 Score | ≥ 86% | Balanced measure of precision & recall |
| CLQS | ≥ 75 | Overall codebase linking quality |

## 📁 Files Created

### Core Framework
- [internal/backtest/backtest.go](internal/backtest/backtest.go) - Main backtesting engine
- [internal/backtest/temporal_matcher.go](internal/backtest/temporal_matcher.go) - Temporal validation
- [internal/backtest/semantic_matcher.go](internal/backtest/semantic_matcher.go) - Semantic validation
- [cmd/backtest/main.go](cmd/backtest/main.go) - CLI test runner
- [scripts/run_backtest.sh](scripts/run_backtest.sh) - Shell wrapper

### Documentation
- [BACKTESTING_README.md](BACKTESTING_README.md) - Quick start guide ⭐
- [test_data/BACKTESTING_GUIDE.md](test_data/BACKTESTING_GUIDE.md) - Complete guide
- [test_data/BACKTESTING_ARCHITECTURE.md](test_data/BACKTESTING_ARCHITECTURE.md) - System architecture
- [BACKTESTING_IMPLEMENTATION_SUMMARY.md](BACKTESTING_IMPLEMENTATION_SUMMARY.md) - Implementation details

## 🎯 Example Output

### ✅ Success Case
```
🧪 Running Backtesting Suite...

[1/4] Running comprehensive backtest...
  [1/6] Testing Issue #221: [FEATURE] allow user to set default agent
    ✅ PASS (confidence: 0.75, delta: 0.00)
  [2/6] Testing Issue #227: [BUG] Codex version
    ✅ PASS (true negative)
  [3/6] Testing Issue #189: [BUG] Ctrl + Z
    ✅ PASS (confidence: 0.70, delta: 0.00)
  [4/6] Testing Issue #187: [BUG] Mobile interface sync
    ✅ PASS (confidence: 0.90, delta: 0.00)
  [5/6] Testing Issue #219: [BUG] Prompts from subagents
    ✅ PASS (true negative)
  [6/6] Testing Issue #188: [BUG] Git diff view
    ⏭️  EXPECTED_MISS (internal fix)

📊 Backtesting Complete:
  Precision: 100.00% ✅
  Recall: 75.00% ✅
  F1 Score: 85.71% ✅
  Accuracy: 83.33%

🎯 Target Metrics Comparison:
  Target Precision: 100.00% | Actual: 100.00% | ✅ PASS
  Target Recall: 75.00% | Actual: 75.00% | ✅ PASS
  Target F1: 86.00% | Actual: 85.71% | ✅ PASS

✅ All target metrics met!
```

### ❌ Failure Case (Temporal Not Implemented)
```
[1/6] Testing Issue #221: [FEATURE] allow user to set default agent
  ❌ FAIL
     - Expected link not found

📊 Backtesting Complete:
  Precision: N/A
  Recall: 0.00% ❌
  F1 Score: 0.00% ❌

🎯 Target Metrics Comparison:
  Target Recall: 75.00% | Actual: 0.00% | ❌ FAIL

❌ Some target metrics not met

Next Steps:
  1. Implement temporal correlator
  2. Run graph construction with temporal matching
  3. Re-run backtest
```

## 📄 Output Reports

All reports saved to `test_results/backtest_YYYYMMDD_HHMMSS_*.json`:

1. **comprehensive.json** - All test results + metrics
2. **temporal.json** - Temporal pattern analysis
3. **semantic.json** - Semantic pattern analysis
4. **clqs.json** - Codebase quality score
5. **summary.json** - Aggregated summary

## 🔧 Command Options

```bash
# Default (all tests)
./scripts/run_backtest.sh

# Verbose logging
./scripts/run_backtest.sh --verbose

# Only temporal validation
./scripts/run_backtest.sh --no-semantic --no-clqs

# Custom ground truth
./scripts/run_backtest.sh --ground-truth test_data/custom.json

# Help
./scripts/run_backtest.sh --help
```

## 🐛 Troubleshooting

### Database Connection Errors
```bash
# Check PostgreSQL
docker ps | grep postgres

# Check Neo4j
docker ps | grep neo4j

# Restart if needed
docker-compose restart
```

### No Links Found (Recall = 0%)
```bash
# Check if graph has data
docker exec coderisk-neo4j cypher-shell -u neo4j -p password \
  "MATCH (i:Issue)-[r]->(p:PR) RETURN count(r);"

# If count = 0, rebuild graph
go run cmd/crisk/main.go build-graph --repo-id=1
```

### False Positives (Precision < 100%)
```bash
# Review what was incorrectly linked
cat test_results/backtest_*_comprehensive.json | \
  jq '.results[] | select(.status == "FAIL")'
```

## 📚 Documentation Index

| Topic | File | Description |
|-------|------|-------------|
| **Quick Start** | [BACKTESTING_README.md](BACKTESTING_README.md) | Start here! |
| **Full Guide** | [test_data/BACKTESTING_GUIDE.md](test_data/BACKTESTING_GUIDE.md) | Complete documentation |
| **Architecture** | [test_data/BACKTESTING_ARCHITECTURE.md](test_data/BACKTESTING_ARCHITECTURE.md) | System design |
| **Implementation** | [BACKTESTING_IMPLEMENTATION_SUMMARY.md](BACKTESTING_IMPLEMENTATION_SUMMARY.md) | What we built |
| **Patterns** | [test_data/LINKING_PATTERNS.md](test_data/LINKING_PATTERNS.md) | Linking patterns reference |
| **Quality Score** | [test_data/LINKING_QUALITY_SCORE.md](test_data/LINKING_QUALITY_SCORE.md) | CLQS methodology |
| **Ground Truth** | [test_data/omnara_ground_truth.json](test_data/omnara_ground_truth.json) | Test cases |

## 🎓 Key Concepts

### Temporal Linking
Matches issues to PRs based on timing:
- PR merged < 5 min after issue closed → 75% confidence
- PR merged < 1 hr after issue closed → 65% confidence
- PR merged < 24 hr after issue closed → 55% confidence

### Semantic Linking
Matches issues to PRs based on keyword overlap:
- Jaccard similarity ≥ 70% → High match
- Jaccard similarity ≥ 50% → Medium match
- Jaccard similarity ≥ 30% → Low match

### CLQS (Codebase Linking Quality Score)
Weighted score (0-100) based on:
- Explicit Linking (40%)
- Temporal Correlation (25%)
- Comment Quality (20%)
- Semantic Consistency (10%)
- Bidirectional References (5%)

Grades:
- 90-100: A (World-Class)
- 75-89: B (High Quality)
- 60-74: C (Moderate)
- <60: D/F (Below Average/Poor)

## 🔄 CI/CD Integration

Add to GitHub Actions:

```yaml
- name: Run Backtests
  run: ./scripts/run_backtest.sh

- name: Check Exit Code
  if: failure()
  run: echo "Backtesting failed - targets not met"
```

Exit codes:
- `0` = All targets met ✅
- `1` = Some targets failed ❌

## 📞 Next Steps

1. **Run the backtest**: `./scripts/run_backtest.sh --verbose`
2. **Review results**: Check `test_results/` directory
3. **If tests fail**: See [BACKTESTING_GUIDE.md](test_data/BACKTESTING_GUIDE.md#troubleshooting)
4. **Add more tests**: See [BACKTESTING_GUIDE.md](test_data/BACKTESTING_GUIDE.md#adding-new-test-cases)
5. **Integrate CI/CD**: Add to your workflow

---

**Ready?** Run `./scripts/run_backtest.sh` now! 🚀
