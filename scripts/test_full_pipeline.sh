#!/bin/bash
# Full pipeline test orchestrator
# Tests issue linking accuracy across all 3 repos (omnara, supabase, stagehand)

set -e

REPO=${1:-"omnara"}
SKIP_REBUILD=${2:-"false"}

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  FULL GRAPH CONSTRUCTION PIPELINE TEST                       ║"
echo "║  Repository: $(printf '%-46s' "$REPO") ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Step 1: Rebuild entire graph (unless skipped)
if [ "$SKIP_REBUILD" != "true" ]; then
    echo "🔄 Step 1: Rebuilding full graph from Postgres..."
    ./scripts/rebuild_all_layers.sh "$REPO"
else
    echo "⏭️  Step 1: Skipping graph rebuild (using existing graph)"
fi

echo ""

# Step 2: Run validation tests
echo "✅ Step 2: Running validation tests..."
go run cmd/test_full_graph/main.go --repo "$REPO" --skip-rebuild=true

echo ""

# Step 3: Check results and determine go/no-go
echo "📊 Step 3: Checking results..."

REPORT_FILE="test_results/${REPO}_full_pipeline_report.json"

if [ ! -f "$REPORT_FILE" ]; then
    echo "❌ Report not found at $REPORT_FILE"
    exit 1
fi

# Extract F1 score using jq
if ! command -v jq &> /dev/null; then
    echo "⚠️  jq not found - installing via brew..."
    brew install jq
fi

F1_SCORE=$(jq -r '.layer3.f1_score' "$REPORT_FILE")
PRECISION=$(jq -r '.layer3.precision' "$REPORT_FILE")
RECALL=$(jq -r '.layer3.recall' "$REPORT_FILE")
OVERALL_STATUS=$(jq -r '.overall_status' "$REPORT_FILE")
RECOMMENDATION=$(jq -r '.recommendation' "$REPORT_FILE")

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  TEST RESULTS SUMMARY                                        ║"
echo "╠══════════════════════════════════════════════════════════════╣"
echo "║ Repository:      $(printf '%-44s' "$REPO") ║"
echo "║ F1 Score:        $(printf '%-44s' "$F1_SCORE%") ║"
echo "║ Precision:       $(printf '%-44s' "$PRECISION%") ║"
echo "║ Recall:          $(printf '%-44s' "$RECALL%") ║"
echo "║ Status:          $(printf '%-44s' "$OVERALL_STATUS") ║"
echo "╠══════════════════════════════════════════════════════════════╣"
echo "║ DECISION                                                     ║"
echo "╠══════════════════════════════════════════════════════════════╣"

# Determine go/no-go based on F1 score
if (( $(echo "$F1_SCORE >= 75" | bc -l) )); then
    echo "║ ✅ GREEN LIGHT: F1 Score = $F1_SCORE% (PASS)                  ║"
    echo "║                                                              ║"
    echo "║ 🚀 PROCEED TO SUPABASE BACKTESTS                             ║"
    echo "║                                                              ║"
    echo "║ All acceptance criteria met:                                 ║"
    echo "║ • F1 Score ≥ 75%           ✅                                ║"
    echo "║ • Precision ≥ 85%          $([ $(echo "$PRECISION >= 85" | bc -l) -eq 1 ] && echo "✅" || echo "⚠️ ")                                ║"
    echo "║ • Recall ≥ 70%             $([ $(echo "$RECALL >= 70" | bc -l) -eq 1 ] && echo "✅" || echo "⚠️ ")                                ║"
    echo "╚══════════════════════════════════════════════════════════════╝"

    # Print pattern coverage
    echo ""
    echo "📊 Pattern Coverage:"
    jq -r '.pattern_coverage.by_pattern | to_entries[] | "  \(.key): \(.value.detection_rate)%"' "$REPORT_FILE"

    exit 0

elif (( $(echo "$F1_SCORE >= 60" | bc -l) )); then
    echo "║ 🟡 YELLOW LIGHT: F1 Score = $F1_SCORE% (ACCEPTABLE)           ║"
    echo "║                                                              ║"
    echo "║ ⚠️  MORE TUNING NEEDED BEFORE PRODUCTION                     ║"
    echo "║                                                              ║"
    echo "║ Current scores:                                              ║"
    echo "║ • F1 Score: $F1_SCORE% (target: ≥75%)                         ║"
    echo "║ • Precision: $PRECISION%                                      ║"
    echo "║ • Recall: $RECALL%                                            ║"
    echo "╚══════════════════════════════════════════════════════════════╝"

    # Print recommendations
    echo ""
    echo "💡 Recommendations:"
    jq -r '.recommendations[]' "$REPORT_FILE" | while read -r line; do
        echo "  • $line"
    done

    exit 0

else
    echo "║ ❌ RED LIGHT: F1 Score = $F1_SCORE% (FAIL)                    ║"
    echo "║                                                              ║"
    echo "║ ⛔ DO NOT PROCEED TO SUPABASE BACKTESTS                      ║"
    echo "║                                                              ║"
    echo "║ Major gaps detected:                                         ║"
    echo "║ • F1 Score: $F1_SCORE% (target: ≥75%)                         ║"
    echo "║ • Precision: $PRECISION% (target: ≥85%)                       ║"
    echo "║ • Recall: $RECALL% (target: ≥70%)                             ║"
    echo "╚══════════════════════════════════════════════════════════════╝"

    # Print detailed failure analysis
    echo ""
    echo "🔍 Failed Test Cases:"
    jq -r '.layer3.test_cases[] | select(.status == "FAIL") | "  Issue #\(.issue_number): \(.title) - \(.status)"' "$REPORT_FILE"

    echo ""
    echo "💡 Recommendations:"
    jq -r '.recommendations[]' "$REPORT_FILE" | while read -r line; do
        echo "  • $line"
    done

    exit 1
fi
