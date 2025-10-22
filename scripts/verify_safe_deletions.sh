#!/bin/bash
# Verify Safe Deletions Script
# Checks that files recommended for deletion have no external dependencies

set -e

echo "========================================================================"
echo "Verifying Safe Deletions for Lean Refactoring"
echo "========================================================================"
echo ""

REPO_ROOT="/Users/rohankatakam/Documents/brain/coderisk"
cd "$REPO_ROOT"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check external usage
check_external_usage() {
    local file_pattern="$1"
    local description="$2"

    echo -n "Checking $description... "

    # Search for usage outside internal/graph/
    count=$(grep -r "$file_pattern" --include="*.go" cmd/ internal/ 2>/dev/null | \
            grep -v "^internal/graph/" | \
            grep -v "_test.go" | \
            grep -v "Binary file" | \
            wc -l | tr -d ' ')

    if [ "$count" -eq 0 ]; then
        echo -e "${GREEN}✅ SAFE (0 external usages)${NC}"
        return 0
    else
        echo -e "${RED}❌ UNSAFE ($count external usages found)${NC}"
        grep -r "$file_pattern" --include="*.go" cmd/ internal/ 2>/dev/null | \
            grep -v "^internal/graph/" | \
            grep -v "_test.go" | \
            grep -v "Binary file" | head -3
        return 1
    fi
}

echo "1. Graph Performance Files"
echo "-------------------------------------------"
check_external_usage "PerformanceProfiler\|NewPerformanceProfiler" "performance_profiler.go"
check_external_usage "PoolMonitor\|PoolStats" "pool_monitor.go"
check_external_usage "RoutingStrategy\|ClusterInfo" "routing.go"
check_external_usage "TimeoutMonitor" "timeout_monitor.go"
check_external_usage "LazyQueryIterator\|FetchSizeConfig" "lazy_query.go"
echo ""

echo "2. Output Package Files"
echo "-------------------------------------------"
# Check ai_actions
count=$(grep -r "AIAssistantAction\|ai_actions" --include="*.go" cmd/ internal/ 2>/dev/null | \
        grep -v "^internal/output/" | \
        grep -v "_test.go" | \
        grep -v "// " | \
        wc -l | tr -d ' ')

echo -n "Checking ai_actions.go... "
if [ "$count" -eq 0 ]; then
    echo -e "${GREEN}✅ SAFE (0 external usages)${NC}"
else
    echo -e "${YELLOW}⚠️  WARNING ($count external usages found)${NC}"
    grep -r "AIAssistantAction" --include="*.go" cmd/ internal/ 2>/dev/null | \
        grep -v "^internal/output/" | \
        grep -v "_test.go" | head -3
fi
echo ""

echo "3. Analysis Config Files"
echo "-------------------------------------------"
check_external_usage "AdaptiveThreshold\|CalculatePhase1WithConfig" "adaptive.go"
check_external_usage "DomainInference\|InferDomain" "domain_inference.go"
echo ""

echo "4. File Existence Check"
echo "-------------------------------------------"
FILES=(
    "internal/graph/performance_profiler.go"
    "internal/graph/pool_monitor.go"
    "internal/graph/routing.go"
    "internal/graph/timeout_monitor.go"
    "internal/graph/lazy_query.go"
    "internal/output/ai_actions.go"
    "internal/analysis/config/adaptive.go"
    "internal/analysis/config/domain_inference.go"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✅ EXISTS:${NC} $file"
    else
        echo -e "${RED}❌ MISSING:${NC} $file (already deleted or moved)"
    fi
done
echo ""

echo "5. Proposed Consolidation Files"
echo "-------------------------------------------"
CONSOLIDATE=(
    "internal/output/ai_converter.go|internal/output/ai_mode.go"
    "internal/output/converter.go|internal/output/types.go"
    "internal/output/graph_analysis.go|internal/output/types.go"
    "internal/output/phase2.go|internal/output/standard.go"
    "internal/output/verbosity.go|internal/output/formatter.go"
    "internal/temporal/developer.go|internal/temporal/git_history.go"
)

for pair in "${CONSOLIDATE[@]}"; do
    source="${pair%|*}"
    target="${pair#*|}"

    if [ -f "$source" ] && [ -f "$target" ]; then
        src_lines=$(wc -l < "$source" | tr -d ' ')
        tgt_lines=$(wc -l < "$target" | tr -d ' ')
        total=$((src_lines + tgt_lines))
        echo -e "${GREEN}✅${NC} $source ($src_lines lines) → $target ($tgt_lines lines)"
        echo "   Combined size: $total lines"
    else
        echo -e "${YELLOW}⚠️${NC}  Files not found: $source or $target"
    fi
done
echo ""

echo "6. Current File Counts"
echo "-------------------------------------------"
total=$(find internal -name "*.go" ! -name "*_test.go" | wc -l | tr -d ' ')
graph=$(find internal/graph -name "*.go" ! -name "*_test.go" | wc -l | tr -d ' ')
output=$(find internal/output -name "*.go" ! -name "*_test.go" | wc -l | tr -d ' ')
temporal=$(find internal/temporal -name "*.go" ! -name "*_test.go" | wc -l | tr -d ' ')
analysis=$(find internal/analysis -name "*.go" ! -name "*_test.go" 2>/dev/null | wc -l | tr -d ' ')

echo "Total internal files:    $total"
echo "Graph package:           $graph files"
echo "Output package:          $output files"
echo "Temporal package:        $temporal files"
echo "Analysis package:        $analysis files"
echo ""

echo "7. Projected File Counts After Refactoring"
echo "-------------------------------------------"
graph_after=$((graph - 6))  # Delete 6 performance files
output_after=$((output - 7)) # Delete 1, consolidate 6
temporal_after=$((temporal - 1)) # Consolidate 1
analysis_after=$((analysis - 2)) # Delete 2
total_after=$((total - 16))

echo "Total internal files:    $total → $total_after (-16 files, -19%)"
echo "Graph package:           $graph → $graph_after (-6 files, -43%)"
echo "Output package:          $output → $output_after (-7 files, -54%)"
echo "Temporal package:        $temporal → $temporal_after (-1 file)"
echo "Analysis package:        $analysis → $analysis_after (-2 files)"
echo ""

echo "========================================================================"
echo "Verification Complete"
echo "========================================================================"
echo ""
echo "Next steps:"
echo "1. Review LEAN_REFACTORING_RECOMMENDATIONS.md"
echo "2. Execute Phase 1 (safe deletions)"
echo "3. Run: go test ./..."
echo "4. Execute Phase 2 (consolidations)"
echo "5. Execute Phase 3 (model cleanup)"
echo ""
