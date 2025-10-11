#!/bin/bash
# Master Test Runner for Modification Type Scenarios
# Executes all 12 scenarios and generates comparison report

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_FILE="$SCRIPT_DIR/test_report.txt"
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

echo "========================================"
echo "Modification Type Tests - Master Runner"
echo "Started: $TIMESTAMP"
echo "========================================"
echo ""

# Initialize report
cat > "$REPORT_FILE" << EOF
Modification Type Tests - Execution Report
Generated: $TIMESTAMP
========================================

EOF

# Test scenarios
SCENARIOS=(
    "7:scenario_7_security.sh:Security-Sensitive Change (Type 9)"
    "10:scenario_10_docs_only.sh:Documentation-Only Change (Type 6)"
    "6a:scenario_6a_prod_config.sh:Production Config Change (Type 3A)"
)

PASSED=0
FAILED=0
TOTAL=${#SCENARIOS[@]}

# Function to run a single test
run_test() {
    local SCENARIO_NUM=$1
    local SCRIPT_NAME=$2
    local DESCRIPTION=$3

    echo "========================================"
    echo "Running Scenario $SCENARIO_NUM: $DESCRIPTION"
    echo "========================================"
    echo ""

    if [ ! -f "$SCRIPT_DIR/$SCRIPT_NAME" ]; then
        echo "❌ ERROR: Script not found: $SCRIPT_NAME"
        echo "Scenario $SCENARIO_NUM: FAILED (script not found)" >> "$REPORT_FILE"
        ((FAILED++))
        return 1
    fi

    # Make script executable
    chmod +x "$SCRIPT_DIR/$SCRIPT_NAME"

    # Run the test
    if bash "$SCRIPT_DIR/$SCRIPT_NAME"; then
        echo "✅ Scenario $SCENARIO_NUM: PASSED"
        echo "Scenario $SCENARIO_NUM: PASSED" >> "$REPORT_FILE"
        ((PASSED++))
        return 0
    else
        echo "❌ Scenario $SCENARIO_NUM: FAILED"
        echo "Scenario $SCENARIO_NUM: FAILED" >> "$REPORT_FILE"
        ((FAILED++))
        return 1
    fi
}

# Run all tests
for scenario in "${SCENARIOS[@]}"; do
    IFS=':' read -r NUM SCRIPT DESC <<< "$scenario"
    run_test "$NUM" "$SCRIPT" "$DESC" || true
    echo ""
    echo ""
done

# Generate summary
echo "========================================"
echo "TEST EXECUTION SUMMARY"
echo "========================================"
echo "Total scenarios: $TOTAL"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "Success rate: $(( PASSED * 100 / TOTAL ))%"
echo ""

cat >> "$REPORT_FILE" << EOF

========================================
SUMMARY
========================================
Total scenarios: $TOTAL
Passed: $PASSED
Failed: $FAILED
Success rate: $(( PASSED * 100 / TOTAL ))%

Output files generated:
EOF

# List output files
ls -lh "$SCRIPT_DIR"/output_scenario_*.txt 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}' >> "$REPORT_FILE" || echo "  (No output files found)" >> "$REPORT_FILE"

echo "========================================"
echo "VALIDATION ANALYSIS"
echo "========================================"
echo ""

# Analyze each output for key characteristics
for scenario in "${SCENARIOS[@]}"; do
    IFS=':' read -r NUM SCRIPT DESC <<< "$scenario"
    OUTPUT_FILE="$SCRIPT_DIR/output_scenario_${NUM}.txt"

    if [ -f "$OUTPUT_FILE" ]; then
        echo "Scenario $NUM Analysis:"
        echo "  File: $(basename "$OUTPUT_FILE")"
        echo "  Size: $(wc -c < "$OUTPUT_FILE") bytes"

        # Check for key elements
        if grep -q "Risk Level:" "$OUTPUT_FILE"; then
            RISK=$(grep "Risk Level:" "$OUTPUT_FILE" | head -1 | sed 's/.*Risk Level: //')
            echo "  ✅ Risk Level: $RISK"
        else
            echo "  ❌ Risk Level: NOT FOUND"
        fi

        if grep -qi "phase.*2\|escalating\|investigation" "$OUTPUT_FILE"; then
            echo "  ✅ Phase 2 indicators found"
        else
            echo "  ℹ️  No Phase 2 indicators"
        fi

        if grep -q "Recommendations:" "$OUTPUT_FILE"; then
            REC_COUNT=$(grep -c "^  [0-9]" "$OUTPUT_FILE" || echo "0")
            echo "  ✅ Recommendations: ~$REC_COUNT items"
        else
            echo "  ℹ️  No recommendations section"
        fi

        echo ""
    else
        echo "Scenario $NUM: ❌ Output file not found"
        echo ""
    fi
done

echo "========================================"
echo "EXPECTED VS ACTUAL COMPARISON"
echo "========================================"
echo ""
echo "Note: Expected output files not yet created."
echo "To create expected outputs:"
echo "  1. Review actual outputs in test/integration/modification_type_tests/output_scenario_*.txt"
echo "  2. Validate they match the expected behavior from TEST_PLAN.md"
echo "  3. Create expected_scenario_*.txt files with validated outputs"
echo "  4. Re-run this script to compare actual vs expected"
echo ""

# Check if expected files exist
EXPECTED_EXISTS=false
for scenario in "${SCENARIOS[@]}"; do
    IFS=':' read -r NUM SCRIPT DESC <<< "$scenario"
    EXPECTED_FILE="$SCRIPT_DIR/expected_scenario_${NUM}.txt"
    if [ -f "$EXPECTED_FILE" ]; then
        EXPECTED_EXISTS=true
        break
    fi
done

if [ "$EXPECTED_EXISTS" = true ]; then
    echo "Comparing with expected outputs..."
    echo ""

    for scenario in "${SCENARIOS[@]}"; do
        IFS=':' read -r NUM SCRIPT DESC <<< "$scenario"
        OUTPUT_FILE="$SCRIPT_DIR/output_scenario_${NUM}.txt"
        EXPECTED_FILE="$SCRIPT_DIR/expected_scenario_${NUM}.txt"
        DIFF_FILE="$SCRIPT_DIR/diff_scenario_${NUM}.txt"

        if [ -f "$EXPECTED_FILE" ] && [ -f "$OUTPUT_FILE" ]; then
            echo "Scenario $NUM Comparison:"

            if diff -u "$EXPECTED_FILE" "$OUTPUT_FILE" > "$DIFF_FILE" 2>&1; then
                echo "  ✅ Output matches expected"
                rm "$DIFF_FILE"
            else
                DIFF_LINES=$(wc -l < "$DIFF_FILE")
                echo "  ⚠️  Differences found ($DIFF_LINES lines)"
                echo "  See: $DIFF_FILE"
            fi
        fi
    done
fi

echo ""
echo "========================================"
echo "✅ All tests completed"
echo "Report saved to: $REPORT_FILE"
echo "========================================"
echo ""

# Exit with appropriate code
if [ "$FAILED" -gt 0 ]; then
    exit 1
else
    exit 0
fi
