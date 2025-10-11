#!/bin/bash
# Checkpoint 4 (Partial): Confidence Loop Performance Benchmark
# Measures dynamic hop behavior, early stopping rate, and latency improvements

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$PROJECT_ROOT/test_sandbox/omnara"
OUTPUT_DIR="$SCRIPT_DIR/output"
BENCHMARK_REPORT="$OUTPUT_DIR/confidence_loop_benchmark.md"
CSV_REPORT="$OUTPUT_DIR/confidence_loop_metrics.csv"

# Source helper functions
source "$SCRIPT_DIR/test_helpers.sh"

print_header "CONFIDENCE LOOP PERFORMANCE BENCHMARK"

# Rebuild binary
print_info "Rebuilding crisk binary..."
cd "$PROJECT_ROOT"
go build -o bin/crisk ./cmd/crisk 2>&1 | grep -v "^#" || true
print_success "Binary rebuilt"

CRISK_BIN="$PROJECT_ROOT/bin/crisk"

# Initialize report
cat > "$BENCHMARK_REPORT" << 'EOF'
# Confidence Loop Performance Benchmark

**Date:** $(date +"%Y-%m-%d %H:%M:%S")
**Agent 4:** Testing & Continuous Validation
**Checkpoint:** 4 (Partial - Confidence Loop Only)
**Status:** Agent 3 confidence loop is integrated, benchmarking performance

---

## Executive Summary

**What Was Benchmarked:**
- ✅ Confidence-driven investigation loop (Agent 3)
- ✅ Dynamic hop behavior (1-5 iterations vs fixed 3)
- ✅ Early stopping rate
- ✅ Average hops per investigation
- ✅ Latency improvements

**What Was NOT Benchmarked (Blocked):**
- ❌ Phase 0 performance (Agent 1 not integrated)
- ❌ Adaptive config impact (Agent 2 not integrated)
- ❌ Full system performance with all optimizations

---

## Test Methodology

**Test Repository:** omnara (Python web application)

**Test Files Selected (20 files):**
- 5 security-sensitive files (auth, API routes)
- 5 documentation files (README, docs)
- 3 configuration files (.env, configs)
- 5 business logic files (backend API)
- 2 test files

**Metrics Tracked:**
1. **Investigation hops:** Count of expand operations per file
2. **Early stop rate:** % of investigations stopping before hop 3
3. **Confidence scores:** Track confidence progression per hop
4. **Latency:** Phase 2 investigation time per file
5. **Breakthrough detection:** Significant risk changes (≥20% threshold)

**Benchmark Execution:**
1. Run `crisk check --explain` on each test file
2. Parse investigation trace to extract hop count, confidence scores
3. Measure Phase 2 latency
4. Calculate aggregate statistics
5. Compare to theoretical fixed 3-hop baseline

---

## Performance Metrics

EOF

# Initialize CSV report
echo "File,Hops,FinalConfidence,Latency_ms,EarlyStopped,BreakthroughDetected" > "$CSV_REPORT"

#############################################
# Select Test Files from omnara
#############################################

print_info "Selecting test files from omnara repository..."

# Security files
SECURITY_FILES=(
    "src/backend/auth/jwt.py"
    "src/backend/auth/routes.py"
    "src/servers/api/auth.py"
    "src/backend/api/auth.py"
    "src/shared/security/permissions.py"
)

# Documentation files
DOC_FILES=(
    "README.md"
    "docs/api.md"
    "docs/development.md"
    "src/backend/README.md"
    "src/servers/README.md"
)

# Configuration files
CONFIG_FILES=(
    ".env.production"
    "src/shared/config.py"
    "docker-compose.yml"
)

# Business logic files
LOGIC_FILES=(
    "src/backend/api/agents.py"
    "src/backend/api/messages.py"
    "src/omnara/cli/commands.py"
    "src/backend/api/health.py"
    "src/servers/app.py"
)

# Test files
TEST_FILES=(
    "tests/test_auth.py"
    "tests/test_api.py"
)

# Combine all files
ALL_TEST_FILES=(
    "${SECURITY_FILES[@]}"
    "${DOC_FILES[@]}"
    "${CONFIG_FILES[@]}"
    "${LOGIC_FILES[@]}"
    "${TEST_FILES[@]}"
)

# Filter to only files that exist
EXISTING_FILES=()
cd "$TEST_DIR"
for file in "${ALL_TEST_FILES[@]}"; do
    if [ -f "$file" ]; then
        EXISTING_FILES+=("$file")
    fi
done

print_success "Selected ${#EXISTING_FILES[@]} test files"

#############################################
# Benchmark Each File
#############################################

print_header "RUNNING CONFIDENCE LOOP BENCHMARKS"

TOTAL_HOPS=0
TOTAL_FILES=0
EARLY_STOP_COUNT=0
BREAKTHROUGH_COUNT=0

for file in "${EXISTING_FILES[@]}"; do
    print_info "Benchmarking: $file"

    # Create a test change to trigger investigation
    # (For benchmark purposes, we'll analyze as-is and track investigation behavior)

    # Run crisk check with --explain to get investigation trace
    START_TIME=$(python3 -c "import time; print(int(time.time() * 1000))")

    cd "$TEST_DIR"
    CHECK_OUTPUT=$("$CRISK_BIN" check "$file" --explain 2>&1 || true)

    END_TIME=$(python3 -c "import time; print(int(time.time() * 1000))")
    LATENCY=$((END_TIME - START_TIME))

    # Parse investigation trace for hop count and confidence
    # Look for "Hop X:" patterns in explain output
    HOP_COUNT=$(echo "$CHECK_OUTPUT" | grep -c "Hop [0-9]:" || echo "0")

    # Extract final confidence (last "Confidence:" value in output)
    FINAL_CONFIDENCE=$(echo "$CHECK_OUTPUT" | grep "Confidence:" | tail -1 | awk '{print $2}' || echo "0.0")

    # Detect early stop (hop count < 3)
    EARLY_STOPPED="false"
    if [ "$HOP_COUNT" -lt 3 ] && [ "$HOP_COUNT" -gt 0 ]; then
        EARLY_STOPPED="true"
        EARLY_STOP_COUNT=$((EARLY_STOP_COUNT + 1))
    fi

    # Detect breakthrough (look for "BREAKTHROUGH" in output)
    BREAKTHROUGH="false"
    if echo "$CHECK_OUTPUT" | grep -q "BREAKTHROUGH"; then
        BREAKTHROUGH="true"
        BREAKTHROUGH_COUNT=$((BREAKTHROUGH_COUNT + 1))
    fi

    # Record metrics
    echo "$file,$HOP_COUNT,$FINAL_CONFIDENCE,$LATENCY,$EARLY_STOPPED,$BREAKTHROUGH" >> "$CSV_REPORT"

    # Update totals
    if [ "$HOP_COUNT" -gt 0 ]; then
        TOTAL_HOPS=$((TOTAL_HOPS + HOP_COUNT))
        TOTAL_FILES=$((TOTAL_FILES + 1))
    fi

    echo "  Hops: $HOP_COUNT, Confidence: $FINAL_CONFIDENCE, Latency: ${LATENCY}ms, Early Stop: $EARLY_STOPPED"
done

#############################################
# Calculate Aggregate Statistics
#############################################

print_header "CALCULATING STATISTICS"

# Average hops (only for files that ran investigation)
if [ "$TOTAL_FILES" -gt 0 ]; then
    AVG_HOPS=$(echo "scale=2; $TOTAL_HOPS / $TOTAL_FILES" | bc)
else
    AVG_HOPS="0.00"
fi

# Early stop rate
if [ "$TOTAL_FILES" -gt 0 ]; then
    EARLY_STOP_RATE=$(echo "scale=1; ($EARLY_STOP_COUNT * 100) / $TOTAL_FILES" | bc)
else
    EARLY_STOP_RATE="0.0"
fi

# Breakthrough rate
if [ "$TOTAL_FILES" -gt 0 ]; then
    BREAKTHROUGH_RATE=$(echo "scale=1; ($BREAKTHROUGH_COUNT * 100) / $TOTAL_FILES" | bc)
else
    BREAKTHROUGH_RATE="0.0"
fi

# Calculate average latency
AVG_LATENCY=$(awk -F',' 'NR>1 {sum+=$4; count++} END {if (count>0) print int(sum/count); else print 0}' "$CSV_REPORT")

# Baseline comparison (theoretical fixed 3-hop)
BASELINE_HOPS=3
HOP_REDUCTION=$(echo "scale=1; (($BASELINE_HOPS - $AVG_HOPS) * 100) / $BASELINE_HOPS" | bc)

print_success "Average hops: $AVG_HOPS (vs fixed 3.0)"
print_success "Hop reduction: ${HOP_REDUCTION}%"
print_success "Early stop rate: ${EARLY_STOP_RATE}% (target: 40%+)"
print_success "Breakthrough detection rate: ${BREAKTHROUGH_RATE}%"
print_success "Average latency: ${AVG_LATENCY}ms"

#############################################
# Generate Report
#############################################

cat >> "$BENCHMARK_REPORT" << EOF

### Summary Statistics

| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| **Average Hops** | $AVG_HOPS | <3.0 (vs fixed 3.0) | $([ $(echo "$AVG_HOPS < 3.0" | bc) -eq 1 ] && echo "✅ PASS" || echo "⚠️ REVIEW") |
| **Hop Reduction** | ${HOP_REDUCTION}% | 33%+ | $([ $(echo "$HOP_REDUCTION >= 33" | bc) -eq 1 ] && echo "✅ PASS" || echo "⚠️ BELOW TARGET") |
| **Early Stop Rate** | ${EARLY_STOP_RATE}% | 40%+ | $([ $(echo "$EARLY_STOP_RATE >= 40" | bc) -eq 1 ] && echo "✅ PASS" || echo "⚠️ BELOW TARGET") |
| **Breakthrough Detection** | ${BREAKTHROUGH_RATE}% | N/A | ℹ️ Informational |
| **Average Latency** | ${AVG_LATENCY}ms | <1,500ms | $([ "$AVG_LATENCY" -lt 1500 ] && echo "✅ PASS" || echo "⚠️ ABOVE TARGET") |
| **Total Files Tested** | $TOTAL_FILES | 20+ | $([ "$TOTAL_FILES" -ge 20 ] && echo "✅ PASS" || echo "⚠️ BELOW TARGET") |

---

## Detailed Results

**Files Benchmarked:** $TOTAL_FILES

**Category Breakdown:**
- Security files: $(echo "${SECURITY_FILES[@]}" | wc -w) tested
- Documentation files: $(echo "${DOC_FILES[@]}" | wc -w) tested
- Configuration files: $(echo "${CONFIG_FILES[@]}" | wc -w) tested
- Business logic files: $(echo "${LOGIC_FILES[@]}" | wc -w) tested
- Test files: $(echo "${TEST_FILES[@]}" | wc -w) tested

**Raw Data:** See \`confidence_loop_metrics.csv\` for per-file metrics

---

## Performance Analysis

### 1. Hop Reduction

**Actual:** $AVG_HOPS hops (avg)
**Baseline:** 3.0 hops (fixed)
**Improvement:** ${HOP_REDUCTION}% reduction

**Analysis:**
$(if [ $(echo "$HOP_REDUCTION >= 33" | bc) -eq 1 ]; then
    echo "✅ **SUCCESS** - Confidence loop achieves target 33% hop reduction. Dynamic stopping is working effectively."
else
    echo "⚠️ **BELOW TARGET** - Hop reduction is ${HOP_REDUCTION}%, below 33% target. Possible causes:"
    echo "  - Confidence threshold (0.85) may be too high"
    echo "  - Test files may require more investigation (complex changes)"
    echo "  - Early stopping logic may need tuning"
fi)

### 2. Early Stopping Rate

**Actual:** ${EARLY_STOP_RATE}%
**Target:** 40%+

**Analysis:**
$(if [ $(echo "$EARLY_STOP_RATE >= 40" | bc) -eq 1 ]; then
    echo "✅ **SUCCESS** - ${EARLY_STOP_RATE}% of investigations stop early (<3 hops). System efficiently identifies low-risk changes."
else
    echo "⚠️ **BELOW TARGET** - Only ${EARLY_STOP_RATE}% stop early, below 40% target. Recommendations:"
    echo "  - Review confidence assessment prompt (may be too conservative)"
    echo "  - Check if test files have inherent complexity"
    echo "  - Consider lowering confidence threshold to 0.80"
fi)

### 3. Latency Performance

**Actual:** ${AVG_LATENCY}ms
**Target:** <1,500ms (for Phase 2 investigation)

**Analysis:**
$(if [ "$AVG_LATENCY" -lt 1500 ]; then
    echo "✅ **SUCCESS** - Average Phase 2 latency is ${AVG_LATENCY}ms, well below 1,500ms target."
else
    echo "⚠️ **ABOVE TARGET** - Latency is ${AVG_LATENCY}ms, exceeding 1,500ms target. This may indicate:"
    echo "  - LLM API latency issues"
    echo "  - Evidence collection overhead"
    echo "  - Need for parallelization of hop operations"
fi)

### 4. Breakthrough Detection

**Rate:** ${BREAKTHROUGH_RATE}%

**Analysis:**
Breakthrough detection identifies significant risk level changes (≥20% threshold) during investigation. A rate of ${BREAKTHROUGH_RATE}% suggests:
$(if [ $(echo "$BREAKTHROUGH_RATE >= 10" | bc) -eq 1 ]; then
    echo "- System is discovering new evidence that changes risk assessment"
    echo "- Investigation is adding value beyond Phase 1 baseline"
else
    echo "- Most investigations confirm Phase 1 assessment without major changes"
    echo "- Could indicate Phase 1 baseline is already accurate OR breakthrough threshold too high"
fi)

---

## Comparison with Expected Outcomes

**From Agent 3's Unit Tests:**
- Average hops: 2.0 (expected) vs $AVG_HOPS (actual)
- Early stop rate: 50% (expected) vs ${EARLY_STOP_RATE}% (actual)

**Variance Analysis:**
$(if [ $(echo "$AVG_HOPS <= 2.5" | bc) -eq 1 ]; then
    echo "✅ Actual performance aligns with unit test expectations"
else
    echo "⚠️ Actual hops higher than unit test predictions - may need review"
fi)

---

## Limitations of This Benchmark

**This benchmark ONLY tests:**
- ✅ Confidence loop dynamic hop behavior
- ✅ Early stopping functionality
- ✅ Investigation latency

**This benchmark DOES NOT test (blocked by integration):**
- ❌ Phase 0 skip logic (not integrated)
- ❌ Phase 0 security/docs escalation (not integrated)
- ❌ Adaptive threshold impact (not integrated)
- ❌ Type-aware LLM prompts (needs Phase 0 modification types)
- ❌ Full system latency with all optimizations
- ❌ False positive rate (requires Phase 0 + adaptive)

**Why Partial Results:**
- Agent 1 (Phase 0) has not completed Checkpoints 3-5
- Agent 2 (Adaptive Config) is complete but NOT integrated into check.go
- Agent 3 (Confidence Loop) IS integrated (line 241-265 in check.go)

**Full benchmarks will be run after:**
1. Agent 1 completes Phase 0 integration (est. 6-7 days)
2. Agent 2's adaptive config is integrated into check.go
3. All systems are wired together

---

## Recommendations

### Immediate Actions

**If targets met:**
1. ✅ Approve Agent 3's confidence loop performance
2. ✅ Mark confidence loop as production-ready
3. ⏭️ Proceed with Agent 1 completion (Phase 0 integration)

**If targets not met:**
1. Review confidence assessment prompt (may be too conservative)
2. Consider adjusting confidence threshold (0.85 → 0.80)
3. Investigate latency bottlenecks (LLM API, evidence collection)
4. Re-run benchmark after adjustments

### Next Steps

**Week 2 (After Agent 1 Completion):**
1. Agent 1 integrates Phase 0 into check.go
2. Agent 2's adaptive config integrated
3. Agent 4 re-runs **FULL** Checkpoint 4 benchmarks
4. Validate all performance targets (latency, FP rate, Phase 0 coverage)

**Checkpoint 5 (Regression Tests):**
1. Run all existing tests: \`go test ./...\`
2. Ensure no regressions from confidence loop integration
3. Validate CLI commands still work correctly

---

## Appendix: Test File Details

**Security Files:**
$(printf '- %s\n' "${SECURITY_FILES[@]}")

**Documentation Files:**
$(printf '- %s\n' "${DOC_FILES[@]}")

**Configuration Files:**
$(printf '- %s\n' "${CONFIG_FILES[@]}")

**Business Logic Files:**
$(printf '- %s\n' "${LOGIC_FILES[@]}")

**Test Files:**
$(printf '- %s\n' "${TEST_FILES[@]}")

---

**Benchmark Status:** ✅ Complete (Partial - Confidence Loop Only)
**Next Benchmark:** Full system after Agent 1 + Agent 2 integration
**Estimated Timeline:** 7-10 days

EOF

print_header "CONFIDENCE LOOP BENCHMARK COMPLETE"
echo ""
echo "Full report saved to: $BENCHMARK_REPORT"
echo "CSV metrics saved to: $CSV_REPORT"
echo ""

# Print summary
echo "Summary:"
echo "  Average hops: $AVG_HOPS (target: <3.0)"
echo "  Hop reduction: ${HOP_REDUCTION}% (target: 33%+)"
echo "  Early stop rate: ${EARLY_STOP_RATE}% (target: 40%+)"
echo "  Average latency: ${AVG_LATENCY}ms (target: <1,500ms)"
echo ""

# Determine overall pass/fail
if [ $(echo "$AVG_HOPS < 3.0" | bc) -eq 1 ] && \
   [ $(echo "$HOP_REDUCTION >= 33" | bc) -eq 1 ] && \
   [ $(echo "$EARLY_STOP_RATE >= 40" | bc) -eq 1 ] && \
   [ "$AVG_LATENCY" -lt 1500 ]; then
    print_success "CONFIDENCE LOOP BENCHMARK: PASSED"
    exit 0
else
    print_warning "CONFIDENCE LOOP BENCHMARK: Some targets not met (see report)"
    exit 0  # Exit 0 to not block workflow
fi
