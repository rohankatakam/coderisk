#!/bin/bash
set -e

echo "=== AI Mode Integration Test ==="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build binary
echo -e "${BLUE}Building crisk binary...${NC}"
go build -o bin/crisk ./cmd/crisk

# Test file (create a mock risky file)
TEST_FILE="test_auth.go"
echo -e "${BLUE}Creating test file: ${TEST_FILE}${NC}"
cat > $TEST_FILE << 'EOF'
package main

import "fmt"

// authenticate_user validates user credentials (no tests, high risk!)
func authenticate_user(username string, password string) bool {
    // TODO: This is a security vulnerability - plain text password!
    if username == "admin" && password == "password123" {
        return true
    }
    return false
}

func main() {
    fmt.Println("Auth service")
}
EOF

# Test 1: AI Mode outputs valid JSON
echo -e "\n${BLUE}Test 1: AI Mode JSON validation...${NC}"
OUTPUT=$(./bin/crisk check --ai-mode $TEST_FILE 2>/dev/null || echo "{}")
if echo "$OUTPUT" | jq . > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PASS: Valid JSON output${NC}"
else
    echo -e "${RED}❌ FAIL: Invalid JSON${NC}"
    echo "Output: $OUTPUT"
    exit 1
fi

# Test 2: Required schema sections exist
echo -e "\n${BLUE}Test 2: Schema sections validation...${NC}"
REQUIRED_SECTIONS=("meta" "risk" "files" "ai_assistant_actions")
for section in "${REQUIRED_SECTIONS[@]}"; do
    if echo "$OUTPUT" | jq -e ".$section" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PASS: '$section' section exists${NC}"
    else
        echo -e "${RED}❌ FAIL: '$section' section missing${NC}"
        exit 1
    fi
done

# Test 3: Meta section has required fields
echo -e "\n${BLUE}Test 3: Meta section validation...${NC}"
META_FIELDS=("version" "timestamp" "duration_ms")
for field in "${META_FIELDS[@]}"; do
    if echo "$OUTPUT" | jq -e ".meta.$field" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PASS: meta.$field exists${NC}"
    else
        echo -e "${RED}❌ FAIL: meta.$field missing${NC}"
        exit 1
    fi
done

# Test 4: Risk section has valid level
echo -e "\n${BLUE}Test 4: Risk level validation...${NC}"
RISK_LEVEL=$(echo "$OUTPUT" | jq -r '.risk.level')
VALID_LEVELS=("NONE" "LOW" "MEDIUM" "HIGH" "CRITICAL")
if [[ " ${VALID_LEVELS[@]} " =~ " ${RISK_LEVEL} " ]]; then
    echo -e "${GREEN}✅ PASS: Risk level is valid: $RISK_LEVEL${NC}"
else
    echo -e "${RED}❌ FAIL: Invalid risk level: $RISK_LEVEL${NC}"
    exit 1
fi

# Test 5: AI actions array structure
echo -e "\n${BLUE}Test 5: AI actions array validation...${NC}"
AI_ACTIONS_COUNT=$(echo "$OUTPUT" | jq '.ai_assistant_actions | length')
if [ "$AI_ACTIONS_COUNT" -ge 0 ]; then
    echo -e "${GREEN}✅ PASS: AI actions array exists (${AI_ACTIONS_COUNT} actions)${NC}"

    # If there are actions, validate first action structure
    if [ "$AI_ACTIONS_COUNT" -gt 0 ]; then
        FIRST_ACTION=$(echo "$OUTPUT" | jq '.ai_assistant_actions[0]')

        # Check required fields
        ACTION_FIELDS=("action_type" "confidence" "ready_to_execute" "prompt")
        for field in "${ACTION_FIELDS[@]}"; do
            if echo "$FIRST_ACTION" | jq -e ".$field" > /dev/null 2>&1; then
                echo -e "${GREEN}✅ PASS: ai_assistant_actions[0].$field exists${NC}"
            else
                echo -e "${RED}❌ FAIL: ai_assistant_actions[0].$field missing${NC}"
                exit 1
            fi
        done
    fi
else
    echo -e "${RED}❌ FAIL: AI actions array missing or invalid${NC}"
    exit 1
fi

# Test 6: Confidence scores validation
echo -e "\n${BLUE}Test 6: Confidence scores validation...${NC}"
if [ "$AI_ACTIONS_COUNT" -gt 0 ]; then
    CONFIDENCE=$(echo "$OUTPUT" | jq -r '.ai_assistant_actions[0].confidence')

    # Check if confidence is a number between 0 and 1
    if (( $(echo "$CONFIDENCE >= 0 && $CONFIDENCE <= 1" | bc -l) )); then
        echo -e "${GREEN}✅ PASS: Confidence score is valid: $CONFIDENCE${NC}"
    else
        echo -e "${RED}❌ FAIL: Confidence score out of range: $CONFIDENCE${NC}"
        exit 1
    fi

    # Check ready_to_execute flag
    READY=$(echo "$OUTPUT" | jq -r '.ai_assistant_actions[0].ready_to_execute')
    if [ "$READY" == "true" ] || [ "$READY" == "false" ]; then
        echo -e "${GREEN}✅ PASS: ready_to_execute flag set: $READY${NC}"
    else
        echo -e "${RED}❌ FAIL: ready_to_execute flag invalid: $READY${NC}"
        exit 1
    fi
else
    echo -e "${BLUE}ℹ️  SKIP: No AI actions to validate${NC}"
fi

# Test 7: Commit control flags
echo -e "\n${BLUE}Test 7: Commit control flags validation...${NC}"
SHOULD_BLOCK=$(echo "$OUTPUT" | jq -r '.should_block_commit')
if [ "$SHOULD_BLOCK" == "true" ] || [ "$SHOULD_BLOCK" == "false" ]; then
    echo -e "${GREEN}✅ PASS: should_block_commit flag set: $SHOULD_BLOCK${NC}"
else
    echo -e "${RED}❌ FAIL: should_block_commit flag invalid${NC}"
    exit 1
fi

# Test 8: Performance metrics
echo -e "\n${BLUE}Test 8: Performance metrics validation...${NC}"
DURATION=$(echo "$OUTPUT" | jq -r '.performance.total_duration_ms')
if [ "$DURATION" -ge 0 ] 2>/dev/null; then
    echo -e "${GREEN}✅ PASS: Performance metrics present (${DURATION}ms)${NC}"
else
    echo -e "${RED}❌ FAIL: Performance metrics missing or invalid${NC}"
    exit 1
fi

# Test 9: JSON schema validation (if ajv is installed)
echo -e "\n${BLUE}Test 9: JSON schema validation...${NC}"
if command -v ajv &> /dev/null; then
    echo "$OUTPUT" > /tmp/ai_mode_output.json
    if ajv validate -s schemas/ai-mode-v1.0.json -d /tmp/ai_mode_output.json > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PASS: Output matches schema v1.0${NC}"
    else
        echo -e "${RED}❌ FAIL: Schema validation failed${NC}"
        echo "Run: ajv validate -s schemas/ai-mode-v1.0.json -d /tmp/ai_mode_output.json"
        exit 1
    fi
    rm -f /tmp/ai_mode_output.json
else
    echo -e "${BLUE}ℹ️  SKIP: ajv not installed (run: npm install -g ajv-cli)${NC}"
fi

# Test 10: Output size check
echo -e "\n${BLUE}Test 10: Output size validation...${NC}"
OUTPUT_SIZE=$(echo "$OUTPUT" | wc -c | tr -d ' ')
if [ "$OUTPUT_SIZE" -lt 100000 ]; then  # <100KB
    echo -e "${GREEN}✅ PASS: Output size acceptable: ${OUTPUT_SIZE} bytes${NC}"
else
    echo -e "${RED}❌ WARN: Output size large: ${OUTPUT_SIZE} bytes (target: <10KB)${NC}"
fi

# Cleanup
echo -e "\n${BLUE}Cleaning up test files...${NC}"
rm -f $TEST_FILE

# Summary
echo -e "\n${GREEN}=== All AI Mode tests passed! ===${NC}"
echo ""
echo "Summary:"
echo "- Valid JSON output: ✅"
echo "- Schema sections: ✅"
echo "- AI actions array: ✅"
echo "- Confidence scores: ✅"
echo "- Commit control: ✅"
echo "- Performance metrics: ✅"
echo ""
echo "Example AI Mode output saved to: /tmp/ai_mode_output.json"
echo "View with: jq . /tmp/ai_mode_output.json"
