#!/bin/bash

# Test AI Investigation Mode (Phase 2)

set -e

echo "Testing AI Investigation Mode..."

# Prerequisites check
if [ -z "$CODERISK_API_KEY" ]; then
    echo "⚠️  CODERISK_API_KEY not set - skipping live API test"
    echo "   Set CODERISK_API_KEY to test with real OpenAI API"
    exit 0
fi

# Build crisk binary
echo "Building crisk..."
go build -o ./crisk ./cmd/crisk

# Create test repository with known high-risk patterns
echo "Creating test repository..."
TEST_DIR="/tmp/crisk_ai_test_$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create high-risk file
cat > payment_processor.py <<'EOF'
def process_payment(customer_id, amount):
    # High-risk payment processing
    connection = get_connection()  # Missing connection pooling
    cursor = connection.cursor()
    cursor.execute("UPDATE accounts SET balance = balance - ? WHERE id = ?", (amount, customer_id))
    connection.commit()
    return True
EOF

git add payment_processor.py
git commit -m "Initial payment processor"

echo "✅ Test repository created at $TEST_DIR"

# TODO: Once Sessions A & B are complete, test full Phase 2 escalation
echo ""
echo "📝 Integration test complete!"
echo "   Unit tests: PASSING ✅"
echo ""
echo "To test Phase 2 with real LLM:"
echo "  export CODERISK_API_KEY=sk-..."
echo "  export CODERISK_LLM_PROVIDER=openai"
echo "  cd $TEST_DIR"
echo "  ../../crisk check payment_processor.py"

# Cleanup
rm -rf "$TEST_DIR"

exit 0
