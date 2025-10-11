#!/bin/bash
# E2E Test Helper Functions
# Shared utilities for Phase 0, Adaptive Config, and Confidence Loop testing

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test state tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Performance tracking (using arrays instead of associative arrays for bash 3.2 compatibility)
TEST_NAMES=()
TEST_LATENCIES=()
TEST_RESULTS=()

# Function: Print section header
print_header() {
    local title=$1
    echo ""
    echo "========================================"
    echo "$title"
    echo "========================================"
    echo ""
}

# Function: Print success message
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function: Print error message
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function: Print warning message
print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Function: Print info message
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Function: Find crisk binary
find_crisk_binary() {
    if [ -f "./bin/crisk" ]; then
        echo "./bin/crisk"
    elif [ -f "./crisk" ]; then
        echo "./crisk"
    else
        print_error "crisk binary not found. Please run: make build"
        exit 1
    fi
}

# Function: Ensure git is clean
ensure_git_clean() {
    local test_dir=$1

    cd "$test_dir" || exit 1

    if [ -n "$(git status --porcelain)" ]; then
        print_error "Git working directory not clean in $test_dir"
        git status --short
        exit 1
    fi

    print_success "Git working directory clean"
}

# Function: Restore git state
restore_git_state() {
    local test_dir=$1
    local files=("${@:2}")

    cd "$test_dir" || exit 1

    for file in "${files[@]}"; do
        git restore "$file" 2>/dev/null || true
        rm -f "${file}.bak" 2>/dev/null || true
    done

    # Clean untracked files created during test
    git clean -fd 2>/dev/null || true

    if [ -n "$(git status --porcelain)" ]; then
        print_warning "Git working directory not fully restored"
        git status --short
    else
        print_success "Git working directory restored"
    fi
}

# Function: Measure execution time
measure_time() {
    local start_time=$1
    # Get current time in seconds and extract milliseconds
    local end_time=$(python3 -c "import time; print(int(time.time() * 1000))")
    echo $((end_time - start_time))
}

# Function: Extract field from output
extract_field() {
    local output_file=$1
    local field_name=$2
    local default_value=${3:-"NOT FOUND"}

    if grep -q "$field_name:" "$output_file"; then
        grep "$field_name:" "$output_file" | head -1 | sed "s/.*$field_name: //"
    else
        echo "$default_value"
    fi
}

# Function: Check if output contains pattern
check_pattern() {
    local output_file=$1
    local pattern=$2
    local pattern_name=$3

    if grep -qi "$pattern" "$output_file"; then
        print_success "$pattern_name found in output"
        return 0
    else
        print_warning "$pattern_name not found in output"
        return 1
    fi
}

# Function: Validate risk level
validate_risk_level() {
    local output_file=$1
    local expected_risk=$2

    local actual_risk=$(extract_field "$output_file" "Risk Level" "UNKNOWN")

    if [[ "$actual_risk" == "$expected_risk" ]]; then
        print_success "Risk level matches expected: $expected_risk"
        return 0
    else
        print_error "Risk level mismatch. Expected: $expected_risk, Actual: $actual_risk"
        return 1
    fi
}

# Function: Validate modification type (Phase 0 feature)
validate_modification_type() {
    local output_file=$1
    local expected_type=$2

    if ! grep -q "Modification Type:" "$output_file"; then
        print_info "Modification Type not in output (Phase 0 not yet implemented)"
        return 2  # Not implemented yet
    fi

    local actual_type=$(extract_field "$output_file" "Modification Type")

    if [[ "$actual_type" == *"$expected_type"* ]]; then
        print_success "Modification type matches expected: $expected_type"
        return 0
    else
        print_error "Modification type mismatch. Expected: $expected_type, Actual: $actual_type"
        return 1
    fi
}

# Function: Validate Phase 0 skip behavior
validate_phase0_skip() {
    local output_file=$1

    # Check for Phase 0 skip indicators
    if grep -qi "skipped\|documentation-only\|zero runtime impact" "$output_file"; then
        print_success "Phase 0 skip logic detected"
        return 0
    else
        print_warning "Phase 0 skip logic not detected (may not be implemented)"
        return 1
    fi
}

# Function: Validate Phase 2 escalation
validate_phase2_escalation() {
    local output_file=$1
    local should_escalate=$2  # true or false

    if grep -qi "phase 2\|escalating\|investigation\|llm" "$output_file"; then
        if [ "$should_escalate" = "true" ]; then
            print_success "Phase 2 escalation detected (expected)"
            return 0
        else
            print_error "Phase 2 escalation detected (should not escalate)"
            return 1
        fi
    else
        if [ "$should_escalate" = "false" ]; then
            print_success "Phase 2 not triggered (expected)"
            return 0
        else
            print_error "Phase 2 not triggered (should escalate)"
            return 1
        fi
    fi
}

# Function: Validate latency target
validate_latency() {
    local latency_ms=$1
    local target_ms=$2
    local test_name=$3

    if [ "$latency_ms" -le "$target_ms" ]; then
        print_success "$test_name latency: ${latency_ms}ms (target: <${target_ms}ms)"
        return 0
    else
        print_warning "$test_name latency: ${latency_ms}ms (exceeds target: ${target_ms}ms)"
        return 1
    fi
}

# Function: Generate test report
generate_test_report() {
    local report_file=$1

    print_header "TEST REPORT SUMMARY"

    echo "Total Tests: $TESTS_TOTAL" | tee -a "$report_file"
    echo "Passed: $TESTS_PASSED" | tee -a "$report_file"
    echo "Failed: $TESTS_FAILED" | tee -a "$report_file"

    if [ $TESTS_TOTAL -gt 0 ]; then
        local success_rate=$(( TESTS_PASSED * 100 / TESTS_TOTAL ))
        echo "Success Rate: ${success_rate}%" | tee -a "$report_file"
    fi

    echo "" | tee -a "$report_file"

    # Latency statistics
    if [ ${#TEST_NAMES[@]} -gt 0 ]; then
        echo "Performance Metrics:" | tee -a "$report_file"
        for i in "${!TEST_NAMES[@]}"; do
            echo "  ${TEST_NAMES[$i]}: ${TEST_LATENCIES[$i]}ms" | tee -a "$report_file"
        done
        echo "" | tee -a "$report_file"
    fi

    # Test results detail
    echo "Test Results Detail:" | tee -a "$report_file"
    for i in "${!TEST_NAMES[@]}"; do
        echo "  ${TEST_NAMES[$i]}: ${TEST_RESULTS[$i]}" | tee -a "$report_file"
    done
}

# Function: Record test result
record_test_result() {
    local test_name=$1
    local passed=$2  # true or false
    local latency_ms=${3:-0}

    ((TESTS_TOTAL++))

    TEST_NAMES+=("$test_name")
    TEST_LATENCIES+=("$latency_ms")

    if [ "$passed" = "true" ]; then
        ((TESTS_PASSED++))
        TEST_RESULTS+=("PASSED")
    else
        ((TESTS_FAILED++))
        TEST_RESULTS+=("FAILED")
    fi
}

# Function: Compare with baseline
compare_with_baseline() {
    local current_value=$1
    local baseline_value=$2
    local metric_name=$3
    local improvement_direction=${4:-"lower"}  # "lower" or "higher"

    local diff=$((current_value - baseline_value))
    local percent_change=0

    if [ "$baseline_value" -gt 0 ]; then
        percent_change=$(( (diff * 100) / baseline_value ))
    fi

    if [ "$improvement_direction" = "lower" ]; then
        # Lower is better (e.g., latency, false positives)
        if [ "$current_value" -lt "$baseline_value" ]; then
            print_success "$metric_name: ${current_value} (baseline: ${baseline_value}, improved by ${percent_change}%)"
        else
            print_warning "$metric_name: ${current_value} (baseline: ${baseline_value}, worsened by ${percent_change}%)"
        fi
    else
        # Higher is better (e.g., accuracy, coverage)
        if [ "$current_value" -gt "$baseline_value" ]; then
            print_success "$metric_name: ${current_value} (baseline: ${baseline_value}, improved by ${percent_change}%)"
        else
            print_warning "$metric_name: ${current_value} (baseline: ${baseline_value}, worsened by ${percent_change}%)"
        fi
    fi
}

# Export functions for use in other scripts
export -f print_header
export -f print_success
export -f print_error
export -f print_warning
export -f print_info
export -f find_crisk_binary
export -f ensure_git_clean
export -f restore_git_state
export -f measure_time
export -f extract_field
export -f check_pattern
export -f validate_risk_level
export -f validate_modification_type
export -f validate_phase0_skip
export -f validate_phase2_escalation
export -f validate_latency
export -f generate_test_report
export -f record_test_result
export -f compare_with_baseline
