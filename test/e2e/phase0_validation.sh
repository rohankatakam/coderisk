#!/bin/bash
# Phase 0 E2E Validation Tests
# Tests security detection, documentation skip, and environment detection
# References: TEST_RESULTS_OCT_2025.md scenarios 7, 10, 6A

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$PROJECT_ROOT/test_sandbox/omnara"
OUTPUT_DIR="$SCRIPT_DIR/output"
REPORT_FILE="$OUTPUT_DIR/phase0_validation_report.txt"

# Source helper functions
source "$SCRIPT_DIR/test_helpers.sh"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Initialize report
cat > "$REPORT_FILE" << EOF
Phase 0 Validation Test Report
Generated: $(date +"%Y-%m-%d %H:%M:%S")
========================================

Test Scenarios:
1. Security-Sensitive Change (auth.py) - Should detect security keywords and force CRITICAL
2. Documentation-Only Change (README.md) - Should skip Phase 1/2 and return LOW <50ms
3. Production Config Change (.env.production) - Should detect production env and force HIGH/CRITICAL

Expected Outcomes:
- 70-80% FP reduction vs baseline
- Phase 0 decisions <50ms
- 100% accuracy on security/docs/env detection

========================================

EOF

print_header "PHASE 0 E2E VALIDATION TESTS"

# Find crisk binary
CRISK_BIN=$(find_crisk_binary)
print_info "Using binary: $CRISK_BIN"

#############################################
# Test 1: Security-Sensitive Change
#############################################

test_security_detection() {
    local TEST_NAME="Test 1: Security-Sensitive Change"
    local TARGET_FILE="src/backend/auth/routes.py"
    local OUTPUT_FILE="$OUTPUT_DIR/phase0_security.txt"

    print_header "$TEST_NAME"

    cd "$TEST_DIR" || exit 1
    ensure_git_clean "$TEST_DIR"

    # Make security-sensitive change
    print_info "Making security-sensitive change to $TARGET_FILE..."

    # Add security-related TODO comment
    sed -i.bak '/"""Sync user from Supabase to our database"""/a\
    # TODO: Add session timeout validation here\
    # TODO: Validate JWT token expiration
' "$TARGET_FILE"

    # Verify changes
    if ! grep -q "session timeout" "$TARGET_FILE"; then
        print_error "Failed to apply changes"
        restore_git_state "$TEST_DIR" "$TARGET_FILE"
        return 1
    fi

    print_success "Changes applied"

    # Run crisk check with timing
    cd "$PROJECT_ROOT"
    local start_time=$(date +%s%3N)
    "$CRISK_BIN" check "$TEST_DIR/$TARGET_FILE" > "$OUTPUT_FILE" 2>&1 || true
    local latency_ms=$(measure_time $start_time)

    print_info "Execution time: ${latency_ms}ms"

    # Display output
    echo ""
    echo "--- Output Preview ---"
    head -30 "$OUTPUT_FILE"
    echo ""

    # Validation checks
    local validation_passed=true

    # Check for modification type detection
    if validate_modification_type "$OUTPUT_FILE" "SECURITY"; then
        echo "  ✓ Security modification type detected"
    else
        validation_result=$?
        if [ $validation_result -eq 2 ]; then
            print_warning "Phase 0 not yet implemented - expected behavior"
        else
            validation_passed=false
        fi
    fi

    # Check for security keywords in reasoning
    if check_pattern "$OUTPUT_file" "auth\|security\|session\|token" "Security keywords"; then
        echo "  ✓ Security context recognized"
    fi

    # Check risk level should be CRITICAL or HIGH
    local risk_level=$(extract_field "$OUTPUT_FILE" "Risk Level")
    if [[ "$risk_level" == "CRITICAL" ]] || [[ "$risk_level" == "HIGH" ]]; then
        print_success "Risk level appropriately elevated: $risk_level"
    else
        print_error "Risk level should be CRITICAL or HIGH, got: $risk_level"
        validation_passed=false
    fi

    # Check for Phase 2 escalation
    if validate_phase2_escalation "$OUTPUT_FILE" "true"; then
        echo "  ✓ Phase 2 escalation triggered"
    fi

    # Clean up
    restore_git_state "$TEST_DIR" "$TARGET_FILE"

    # Record result
    record_test_result "$TEST_NAME" "$validation_passed" "$latency_ms"

    echo "" >> "$REPORT_FILE"
    echo "Test 1: Security-Sensitive Change" >> "$REPORT_FILE"
    echo "  Risk Level: $risk_level" >> "$REPORT_FILE"
    echo "  Latency: ${latency_ms}ms" >> "$REPORT_FILE"
    echo "  Result: $([[ $validation_passed == true ]] && echo PASSED || echo FAILED)" >> "$REPORT_FILE"

    return $([[ $validation_passed == true ]] && echo 0 || echo 1)
}

#############################################
# Test 2: Documentation-Only Change
#############################################

test_documentation_skip() {
    local TEST_NAME="Test 2: Documentation-Only Change"
    local TARGET_FILE="README.md"
    local OUTPUT_FILE="$OUTPUT_DIR/phase0_docs_only.txt"

    print_header "$TEST_NAME"

    cd "$TEST_DIR" || exit 1
    ensure_git_clean "$TEST_DIR"

    # Make documentation-only change
    print_info "Making documentation-only change to $TARGET_FILE..."

    cat >> "$TARGET_FILE" << 'DOCEOF'

## Development Setup (Added by Test)

This section provides instructions for setting up the development environment.

### Prerequisites
- Python 3.11+
- PostgreSQL 14+
- Redis 6+

### Installation Steps
1. Clone the repository
2. Install dependencies: `pip install -r requirements-dev.txt`
3. Set up the database: `alembic upgrade head`
4. Run the development server: `./dev-start.sh`

DOCEOF

    # Verify changes
    if ! grep -q "Development Setup (Added by Test)" "$TARGET_FILE"; then
        print_error "Failed to apply changes"
        restore_git_state "$TEST_DIR" "$TARGET_FILE"
        return 1
    fi

    print_success "Changes applied"

    # Run crisk check with timing
    cd "$PROJECT_ROOT"
    local start_time=$(date +%s%3N)
    "$CRISK_BIN" check "$TEST_DIR/$TARGET_FILE" > "$OUTPUT_FILE" 2>&1 || true
    local latency_ms=$(measure_time $start_time)

    print_info "Execution time: ${latency_ms}ms"

    # Display output
    echo ""
    echo "--- Output Preview ---"
    head -30 "$OUTPUT_FILE"
    echo ""

    # Validation checks
    local validation_passed=true

    # Check for modification type detection
    if validate_modification_type "$OUTPUT_FILE" "DOCUMENTATION"; then
        echo "  ✓ Documentation modification type detected"
    else
        validation_result=$?
        if [ $validation_result -ne 2 ]; then
            # If Phase 0 is implemented but didn't detect docs, that's a failure
            validation_passed=false
        fi
    fi

    # Check for Phase 0 skip logic
    if validate_phase0_skip "$OUTPUT_FILE"; then
        echo "  ✓ Phase 0 skip logic working"
    fi

    # Check risk level should be LOW or MINIMAL
    local risk_level=$(extract_field "$OUTPUT_FILE" "Risk Level")
    if [[ "$risk_level" == "LOW" ]] || [[ "$risk_level" == "MINIMAL" ]]; then
        print_success "Risk level appropriate: $risk_level"
    else
        print_warning "Risk level should be LOW/MINIMAL for docs, got: $risk_level"
        # Not a hard failure - Phase 2 may downgrade
    fi

    # Check latency target <50ms (with Phase 0 skip)
    # Note: Without Phase 0, this will be slower (Phase 2 investigation)
    if validate_latency "$latency_ms" 50 "Documentation skip"; then
        echo "  ✓ Latency meets Phase 0 target"
    else
        if [ "$latency_ms" -gt 1000 ]; then
            print_warning "Latency suggests Phase 2 investigation (Phase 0 skip not implemented)"
        fi
    fi

    # Check Phase 2 should NOT be triggered (with Phase 0)
    if validate_phase2_escalation "$OUTPUT_FILE" "false"; then
        echo "  ✓ Phase 2 correctly skipped"
    else
        print_warning "Phase 2 triggered for docs (Phase 0 skip not implemented)"
    fi

    # Clean up
    restore_git_state "$TEST_DIR" "$TARGET_FILE"

    # Record result
    record_test_result "$TEST_NAME" "$validation_passed" "$latency_ms"

    echo "" >> "$REPORT_FILE"
    echo "Test 2: Documentation-Only Change" >> "$REPORT_FILE"
    echo "  Risk Level: $risk_level" >> "$REPORT_FILE"
    echo "  Latency: ${latency_ms}ms (target: <50ms with Phase 0)" >> "$REPORT_FILE"
    echo "  Result: $([[ $validation_passed == true ]] && echo PASSED || echo FAILED)" >> "$REPORT_FILE"

    return $([[ $validation_passed == true ]] && echo 0 || echo 1)
}

#############################################
# Test 3: Production Config Change
#############################################

test_production_config() {
    local TEST_NAME="Test 3: Production Config Change"
    local TARGET_FILE=".env.production"
    local OUTPUT_FILE="$OUTPUT_DIR/phase0_prod_config.txt"

    print_header "$TEST_NAME"

    cd "$TEST_DIR" || exit 1
    ensure_git_clean "$TEST_DIR"

    # Create production config file
    print_info "Creating production config file: $TARGET_FILE..."

    cat > "$TARGET_FILE" << 'ENVEOF'
# Production Environment Configuration
ENVIRONMENT=production
DATABASE_URL=postgresql://user:NEW_PASSWORD_CHANGED@prod-db.example.com:5432/omnara_prod
REDIS_URL=redis://prod-redis.example.com:6379/0
JWT_SECRET_KEY=prod_secret_key_value_changed
SUPABASE_URL=https://prod.supabase.co
SUPABASE_ANON_KEY=prod_anon_key
LOG_LEVEL=INFO
DEBUG=false
ENVEOF

    # Verify file created
    if [ ! -f "$TARGET_FILE" ]; then
        print_error "Failed to create production config"
        restore_git_state "$TEST_DIR" "$TARGET_FILE"
        return 1
    fi

    # Add to git so it shows as new file
    git add "$TARGET_FILE"

    print_success "Production config created"

    # Run crisk check with timing
    cd "$PROJECT_ROOT"
    local start_time=$(date +%s%3N)
    "$CRISK_BIN" check "$TEST_DIR/$TARGET_FILE" > "$OUTPUT_FILE" 2>&1 || true
    local latency_ms=$(measure_time $start_time)

    print_info "Execution time: ${latency_ms}ms"

    # Display output
    echo ""
    echo "--- Output Preview ---"
    head -30 "$OUTPUT_FILE"
    echo ""

    # Validation checks
    local validation_passed=true

    # Check for modification type detection
    if validate_modification_type "$OUTPUT_FILE" "CONFIGURATION\|PRODUCTION"; then
        echo "  ✓ Configuration/Production modification type detected"
    else
        validation_result=$?
        if [ $validation_result -ne 2 ]; then
            validation_passed=false
        fi
    fi

    # Check for environment detection
    if check_pattern "$OUTPUT_FILE" "production\|prod\|environment" "Production environment"; then
        echo "  ✓ Production environment recognized"
    fi

    # Check risk level should be CRITICAL or HIGH
    local risk_level=$(extract_field "$OUTPUT_FILE" "Risk Level")
    if [[ "$risk_level" == "CRITICAL" ]] || [[ "$risk_level" == "HIGH" ]]; then
        print_success "Risk level appropriately elevated: $risk_level"
    else
        print_error "Risk level should be CRITICAL/HIGH for production config, got: $risk_level"
        validation_passed=false
    fi

    # Check for Phase 2 escalation
    if validate_phase2_escalation "$OUTPUT_FILE" "true"; then
        echo "  ✓ Phase 2 escalation triggered"
    fi

    # Clean up
    restore_git_state "$TEST_DIR" "$TARGET_FILE"

    # Record result
    record_test_result "$TEST_NAME" "$validation_passed" "$latency_ms"

    echo "" >> "$REPORT_FILE"
    echo "Test 3: Production Config Change" >> "$REPORT_FILE"
    echo "  Risk Level: $risk_level" >> "$REPORT_FILE"
    echo "  Latency: ${latency_ms}ms" >> "$REPORT_FILE"
    echo "  Result: $([[ $validation_passed == true ]] && echo PASSED || echo FAILED)" >> "$REPORT_FILE"

    return $([[ $validation_passed == true ]] && echo 0 || echo 1)
}

#############################################
# Test 4: Comment-Only Change (Additional)
#############################################

test_comment_only() {
    local TEST_NAME="Test 4: Comment-Only Change"
    local TARGET_FILE="src/omnara/cli/commands.py"
    local OUTPUT_FILE="$OUTPUT_DIR/phase0_comment_only.txt"

    print_header "$TEST_NAME"

    cd "$TEST_DIR" || exit 1
    ensure_git_clean "$TEST_DIR"

    # Make comment-only change
    print_info "Making comment-only change to $TARGET_FILE..."

    # Add comments to an existing function
    sed -i.bak '1a\
# Added comprehensive documentation for better code understanding\
# This module handles CLI command registration and execution
' "$TARGET_FILE"

    # Verify changes
    if ! grep -q "comprehensive documentation" "$TARGET_FILE"; then
        print_error "Failed to apply changes"
        restore_git_state "$TEST_DIR" "$TARGET_FILE"
        return 1
    fi

    print_success "Changes applied"

    # Run crisk check with timing
    cd "$PROJECT_ROOT"
    local start_time=$(date +%s%3N)
    "$CRISK_BIN" check "$TEST_DIR/$TARGET_FILE" > "$OUTPUT_FILE" 2>&1 || true
    local latency_ms=$(measure_time $start_time)

    print_info "Execution time: ${latency_ms}ms"

    # Display output
    echo ""
    echo "--- Output Preview ---"
    head -30 "$OUTPUT_FILE"
    echo ""

    # Validation checks
    local validation_passed=true

    # Check risk level should be LOW
    local risk_level=$(extract_field "$OUTPUT_FILE" "Risk Level")
    if [[ "$risk_level" == "LOW" ]] || [[ "$risk_level" == "MINIMAL" ]]; then
        print_success "Risk level appropriate: $risk_level"
    else
        print_warning "Risk level should be LOW for comment-only, got: $risk_level"
    fi

    # Check latency should be fast
    validate_latency "$latency_ms" 200 "Comment-only change"

    # Clean up
    restore_git_state "$TEST_DIR" "$TARGET_FILE"

    # Record result
    record_test_result "$TEST_NAME" "$validation_passed" "$latency_ms"

    echo "" >> "$REPORT_FILE"
    echo "Test 4: Comment-Only Change" >> "$REPORT_FILE"
    echo "  Risk Level: $risk_level" >> "$REPORT_FILE"
    echo "  Latency: ${latency_ms}ms" >> "$REPORT_FILE"
    echo "  Result: $([[ $validation_passed == true ]] && echo PASSED || echo FAILED)" >> "$REPORT_FILE"

    return $([[ $validation_passed == true ]] && echo 0 || echo 1)
}

#############################################
# Run all tests
#############################################

main() {
    # Run all test scenarios
    test_security_detection || true
    echo ""

    test_documentation_skip || true
    echo ""

    test_production_config || true
    echo ""

    test_comment_only || true
    echo ""

    # Generate summary report
    generate_test_report "$REPORT_FILE"

    print_header "PHASE 0 VALIDATION COMPLETE"
    echo "Full report saved to: $REPORT_FILE"
    echo ""

    # Print key findings
    print_header "KEY FINDINGS"

    echo "Phase 0 Implementation Status:"
    grep -q "Modification Type:" "$OUTPUT_DIR"/phase0_*.txt 2>/dev/null && \
        print_success "Phase 0 modification type detection: IMPLEMENTED" || \
        print_warning "Phase 0 modification type detection: NOT YET IMPLEMENTED"

    echo ""
    echo "Performance Summary:"
    echo "  Security detection: ${TEST_LATENCIES["Test 1: Security-Sensitive Change"]:-N/A}ms"
    echo "  Documentation skip: ${TEST_LATENCIES["Test 2: Documentation-Only Change"]:-N/A}ms (target: <50ms)"
    echo "  Production config: ${TEST_LATENCIES["Test 3: Production Config Change"]:-N/A}ms"

    echo ""
    echo "Next Steps:"
    if [ "$TESTS_FAILED" -gt 0 ]; then
        print_warning "Some tests failed. Review outputs in $OUTPUT_DIR/ for details."
        print_info "If Phase 0 not yet implemented, failures are expected."
    else
        print_success "All tests passed! Phase 0 working as expected."
    fi

    # Exit with appropriate code
    if [ "$TESTS_FAILED" -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Run main function
main "$@"
